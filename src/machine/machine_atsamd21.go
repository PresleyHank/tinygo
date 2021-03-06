// +build sam,atsamd21

// Peripheral abstraction layer for the atsamd21.
//
// Datasheet:
// http://ww1.microchip.com/downloads/en/DeviceDoc/SAMD21-Family-DataSheet-DS40001882D.pdf
//
package machine

import (
	"bytes"
	"device/arm"
	"device/sam"
	"encoding/binary"
	"errors"
	"unsafe"
)

const CPU_FREQUENCY = 48000000

type GPIOMode uint8

const (
	GPIO_ANALOG         = 1
	GPIO_SERCOM         = 2
	GPIO_SERCOM_ALT     = 3
	GPIO_TIMER          = 4
	GPIO_TIMER_ALT      = 5
	GPIO_COM            = 6
	GPIO_AC_CLK         = 7
	GPIO_DIGITAL        = 8
	GPIO_INPUT          = 9
	GPIO_INPUT_PULLUP   = 10
	GPIO_OUTPUT         = 11
	GPIO_PWM            = GPIO_TIMER
	GPIO_PWM_ALT        = GPIO_TIMER_ALT
	GPIO_INPUT_PULLDOWN = 12
)

// Hardware pins
const (
	PA00 = 0
	PA01 = 1
	PA02 = 2
	PA03 = 3
	PA04 = 4
	PA05 = 5
	PA06 = 6
	PA07 = 7
	PA08 = 8
	PA09 = 9
	PA10 = 10
	PA11 = 11
	PA12 = 12
	PA13 = 13
	PA14 = 14
	PA15 = 15
	PA16 = 16
	PA17 = 17
	PA18 = 18
	PA19 = 19
	PA20 = 20
	PA21 = 21
	PA22 = 22
	PA23 = 23
	PA24 = 24
	PA25 = 25
	PA26 = 26
	PA27 = 27
	PA28 = 28
	PA29 = 29
	PA30 = 30
	PA31 = 31
	PB00 = 32
	PB01 = 33
	PB02 = 34
	PB03 = 35
	PB04 = 36
	PB05 = 37
	PB06 = 38
	PB07 = 39
	PB08 = 40
	PB09 = 41
	PB10 = 42
	PB11 = 43
	PB12 = 44
	PB13 = 45
	PB14 = 46
	PB15 = 47
	PB16 = 48
	PB17 = 49
	PB18 = 50
	PB19 = 51
	PB20 = 52
	PB21 = 53
	PB22 = 54
	PB23 = 55
	PB24 = 56
	PB25 = 57
	PB26 = 58
	PB27 = 59
	PB28 = 60
	PB29 = 61
	PB30 = 62
	PB31 = 63
)

// Configure this pin with the given configuration.
func (p GPIO) Configure(config GPIOConfig) {
	switch config.Mode {
	case GPIO_OUTPUT:
		if p.Pin < 32 {
			sam.PORT.DIRSET0 = (1 << p.Pin)
			// output is also set to input enable so pin can read back its own value
			p.setPinCfg(sam.PORT_PINCFG0_INEN)
		} else {
			sam.PORT.DIRSET1 = (1 << (p.Pin - 32))
			// output is also set to input enable so pin can read back its own value
			p.setPinCfg(sam.PORT_PINCFG0_INEN)
		}

	case GPIO_INPUT:
		if p.Pin < 32 {
			sam.PORT.DIRCLR0 = (1 << p.Pin)
			p.setPinCfg(sam.PORT_PINCFG0_INEN)
		} else {
			sam.PORT.DIRCLR1 = (1<<p.Pin - 32)
			p.setPinCfg(sam.PORT_PINCFG0_INEN)
		}

	case GPIO_INPUT_PULLDOWN:
		if p.Pin < 32 {
			sam.PORT.DIRCLR0 = (1 << p.Pin)
			sam.PORT.OUTCLR0 = (1 << p.Pin)
			p.setPinCfg(sam.PORT_PINCFG0_INEN | sam.PORT_PINCFG0_PULLEN)
		} else {
			sam.PORT.DIRCLR1 = (1<<p.Pin - 32)
			sam.PORT.OUTCLR1 = (1<<p.Pin - 32)
			p.setPinCfg(sam.PORT_PINCFG0_INEN | sam.PORT_PINCFG0_PULLEN)
		}

	case GPIO_INPUT_PULLUP:
		if p.Pin < 32 {
			sam.PORT.DIRCLR0 = (1 << p.Pin)
			sam.PORT.OUTSET0 = (1 << p.Pin)
			p.setPinCfg(sam.PORT_PINCFG0_INEN | sam.PORT_PINCFG0_PULLEN)
		} else {
			sam.PORT.DIRCLR1 = (1<<p.Pin - 32)
			sam.PORT.OUTSET1 = (1<<p.Pin - 32)
			p.setPinCfg(sam.PORT_PINCFG0_INEN | sam.PORT_PINCFG0_PULLEN)
		}

	case GPIO_SERCOM:
		if p.Pin&1 > 0 {
			// odd pin, so save the even pins
			val := p.getPMux() & sam.PORT_PMUX0_PMUXE_Msk
			p.setPMux(val | (GPIO_SERCOM << sam.PORT_PMUX0_PMUXO_Pos))
		} else {
			// even pin, so save the odd pins
			val := p.getPMux() & sam.PORT_PMUX0_PMUXO_Msk
			p.setPMux(val | (GPIO_SERCOM << sam.PORT_PMUX0_PMUXE_Pos))
		}
		// enable port config
		p.setPinCfg(sam.PORT_PINCFG0_PMUXEN | sam.PORT_PINCFG0_DRVSTR | sam.PORT_PINCFG0_INEN)

	case GPIO_SERCOM_ALT:
		if p.Pin&1 > 0 {
			// odd pin, so save the even pins
			val := p.getPMux() & sam.PORT_PMUX0_PMUXE_Msk
			p.setPMux(val | (GPIO_SERCOM_ALT << sam.PORT_PMUX0_PMUXO_Pos))
		} else {
			// even pin, so save the odd pins
			val := p.getPMux() & sam.PORT_PMUX0_PMUXO_Msk
			p.setPMux(val | (GPIO_SERCOM_ALT << sam.PORT_PMUX0_PMUXE_Pos))
		}
		// enable port config
		p.setPinCfg(sam.PORT_PINCFG0_PMUXEN | sam.PORT_PINCFG0_DRVSTR)

	case GPIO_COM:
		if p.Pin&1 > 0 {
			// odd pin, so save the even pins
			val := p.getPMux() & sam.PORT_PMUX0_PMUXE_Msk
			p.setPMux(val | (GPIO_COM << sam.PORT_PMUX0_PMUXO_Pos))
		} else {
			// even pin, so save the odd pins
			val := p.getPMux() & sam.PORT_PMUX0_PMUXO_Msk
			p.setPMux(val | (GPIO_COM << sam.PORT_PMUX0_PMUXE_Pos))
		}
		// enable port config
		p.setPinCfg(sam.PORT_PINCFG0_PMUXEN)
	case GPIO_ANALOG:
		if p.Pin&1 > 0 {
			// odd pin, so save the even pins
			val := p.getPMux() & sam.PORT_PMUX0_PMUXE_Msk
			p.setPMux(val | (GPIO_ANALOG << sam.PORT_PMUX0_PMUXO_Pos))
		} else {
			// even pin, so save the odd pins
			val := p.getPMux() & sam.PORT_PMUX0_PMUXO_Msk
			p.setPMux(val | (GPIO_COM << sam.PORT_PMUX0_PMUXE_Pos))
		}
		// enable port config
		p.setPinCfg(sam.PORT_PINCFG0_PMUXEN | sam.PORT_PINCFG0_DRVSTR)
	}
}

// Get returns the current value of a GPIO pin.
func (p GPIO) Get() bool {
	if p.Pin < 32 {
		return (sam.PORT.IN0>>p.Pin)&1 > 0
	} else {
		return (sam.PORT.IN1>>(p.Pin-32))&1 > 0
	}
}

// Set the pin to high or low.
// Warning: only use this on an output pin!
func (p GPIO) Set(high bool) {
	if p.Pin < 32 {
		if high {
			sam.PORT.OUTSET0 = (1 << p.Pin)
		} else {
			sam.PORT.OUTCLR0 = (1 << p.Pin)
		}
	} else {
		if high {
			sam.PORT.OUTSET1 = (1 << (p.Pin - 32))
		} else {
			sam.PORT.OUTCLR1 = (1 << (p.Pin - 32))
		}
	}
}

// Return the register and mask to enable a given GPIO pin. This can be used to
// implement bit-banged drivers.
func (p GPIO) PortMaskSet() (*uint32, uint32) {
	if p.Pin < 32 {
		return (*uint32)(&sam.PORT.OUTSET0), 1 << p.Pin
	} else {
		return (*uint32)(&sam.PORT.OUTSET1), 1 << (p.Pin - 32)
	}
}

// Return the register and mask to disable a given port. This can be used to
// implement bit-banged drivers.
func (p GPIO) PortMaskClear() (*uint32, uint32) {
	if p.Pin < 32 {
		return (*uint32)(&sam.PORT.OUTCLR0), 1 << p.Pin
	} else {
		return (*uint32)(&sam.PORT.OUTCLR1), 1 << (p.Pin - 32)
	}
}

// getPMux returns the value for the correct PMUX register for this pin.
func (p GPIO) getPMux() sam.RegValue8 {
	return getPMux(p.Pin)
}

// setPMux sets the value for the correct PMUX register for this pin.
func (p GPIO) setPMux(val sam.RegValue8) {
	setPMux(p.Pin, val)
}

// getPinCfg returns the value for the correct PINCFG register for this pin.
func (p GPIO) getPinCfg() sam.RegValue8 {
	return getPinCfg(p.Pin)
}

// setPinCfg sets the value for the correct PINCFG register for this pin.
func (p GPIO) setPinCfg(val sam.RegValue8) {
	setPinCfg(p.Pin, val)
}

// InitADC initializes the ADC.
func InitADC() {
	// ADC Bias Calibration
	// #define ADC_FUSES_BIASCAL_ADDR      (NVMCTRL_OTP4 + 4)
	// #define ADC_FUSES_BIASCAL_Pos       3            /**< \brief (NVMCTRL_OTP4) ADC Bias Calibration */
	// #define ADC_FUSES_BIASCAL_Msk       (0x7u << ADC_FUSES_BIASCAL_Pos)
	// #define ADC_FUSES_BIASCAL(value)    ((ADC_FUSES_BIASCAL_Msk & ((value) << ADC_FUSES_BIASCAL_Pos)))
	// #define ADC_FUSES_LINEARITY_0_ADDR  NVMCTRL_OTP4
	// #define ADC_FUSES_LINEARITY_0_Pos   27           /**< \brief (NVMCTRL_OTP4) ADC Linearity bits 4:0 */
	// #define ADC_FUSES_LINEARITY_0_Msk   (0x1Fu << ADC_FUSES_LINEARITY_0_Pos)
	// #define ADC_FUSES_LINEARITY_0(value) ((ADC_FUSES_LINEARITY_0_Msk & ((value) << ADC_FUSES_LINEARITY_0_Pos)))
	// #define ADC_FUSES_LINEARITY_1_ADDR  (NVMCTRL_OTP4 + 4)
	// #define ADC_FUSES_LINEARITY_1_Pos   0            /**< \brief (NVMCTRL_OTP4) ADC Linearity bits 7:5 */
	// #define ADC_FUSES_LINEARITY_1_Msk   (0x7u << ADC_FUSES_LINEARITY_1_Pos)
	// #define ADC_FUSES_LINEARITY_1(value) ((ADC_FUSES_LINEARITY_1_Msk & ((value) << ADC_FUSES_LINEARITY_1_Pos)))

	biasFuse := *(*uint32)(unsafe.Pointer(uintptr(0x00806020) + 4))
	bias := sam.RegValue16(uint16(biasFuse>>3) & uint16(0x7))

	// ADC Linearity bits 4:0
	linearity0Fuse := *(*uint32)(unsafe.Pointer(uintptr(0x00806020)))
	linearity := sam.RegValue16(uint16(linearity0Fuse>>27) & uint16(0x1f))

	// ADC Linearity bits 7:5
	linearity1Fuse := *(*uint32)(unsafe.Pointer(uintptr(0x00806020) + 4))
	linearity |= sam.RegValue16(uint16(linearity1Fuse)&uint16(0x7)) << 5

	// set calibration
	sam.ADC.CALIB = (bias << 8) | linearity

	// Wait for synchronization
	waitADCSync()

	// Divide Clock by 32 with 12 bits resolution as default
	sam.ADC.CTRLB = (sam.ADC_CTRLB_PRESCALER_DIV32 << sam.ADC_CTRLB_PRESCALER_Pos) |
		(sam.ADC_CTRLB_RESSEL_12BIT << sam.ADC_CTRLB_RESSEL_Pos)

	// Sampling Time Length
	sam.ADC.SAMPCTRL = 5

	// Wait for synchronization
	waitADCSync()

	// Use internal ground
	sam.ADC.INPUTCTRL = (sam.ADC_INPUTCTRL_MUXNEG_GND << sam.ADC_INPUTCTRL_MUXNEG_Pos)

	// Averaging (see datasheet table in AVGCTRL register description)
	sam.ADC.AVGCTRL = (sam.ADC_AVGCTRL_SAMPLENUM_1 << sam.ADC_AVGCTRL_SAMPLENUM_Pos) |
		(0x0 << sam.ADC_AVGCTRL_ADJRES_Pos)

	// Analog Reference is AREF pin (3.3v)
	sam.ADC.INPUTCTRL |= (sam.ADC_INPUTCTRL_GAIN_DIV2 << sam.ADC_INPUTCTRL_GAIN_Pos)

	// 1/2 VDDANA = 0.5 * 3V3 = 1.65V
	sam.ADC.REFCTRL |= (sam.ADC_REFCTRL_REFSEL_INTVCC1 << sam.ADC_REFCTRL_REFSEL_Pos)
}

// Configure configures a ADCPin to be able to be used to read data.
func (a ADC) Configure() {
	GPIO{a.Pin}.Configure(GPIOConfig{Mode: GPIO_ANALOG})
	return
}

// Get returns the current value of a ADC pin, in the range 0..0xffff.
func (a ADC) Get() uint16 {
	ch := a.getADCChannel()

	// Selection for the positive ADC input
	sam.ADC.INPUTCTRL &^= sam.ADC_INPUTCTRL_MUXPOS_Msk
	waitADCSync()
	sam.ADC.INPUTCTRL |= sam.RegValue(ch << sam.ADC_INPUTCTRL_MUXPOS_Pos)
	waitADCSync()

	// Enable ADC
	sam.ADC.CTRLA |= sam.ADC_CTRLA_ENABLE
	waitADCSync()

	// Start conversion
	sam.ADC.SWTRIG |= sam.ADC_SWTRIG_START
	waitADCSync()

	// Clear the Data Ready flag
	sam.ADC.INTFLAG = sam.ADC_INTFLAG_RESRDY
	waitADCSync()

	// Start conversion again, since first conversion after reference voltage changed is invalid.
	sam.ADC.SWTRIG |= sam.ADC_SWTRIG_START
	waitADCSync()

	// Waiting for conversion to complete
	for (sam.ADC.INTFLAG & sam.ADC_INTFLAG_RESRDY) == 0 {
	}
	val := sam.ADC.RESULT

	// Disable ADC
	sam.ADC.CTRLA &^= sam.ADC_CTRLA_ENABLE
	waitADCSync()

	return uint16(val)
}

func (a ADC) getADCChannel() uint8 {
	switch a.Pin {
	case PA02:
		return 0
	case PB08:
		return 2
	case PB09:
		return 3
	case PA04:
		return 4
	case PA05:
		return 5
	case PA06:
		return 6
	case PA07:
		return 7
	case PB02:
		return 10
	case PB03:
		return 11
	case PA09:
		return 17
	case PA11:
		return 19
	default:
		return 0
	}
}

func waitADCSync() {
	for (sam.ADC.STATUS & sam.ADC_STATUS_SYNCBUSY) > 0 {
	}
}

// UART on the SAMD21.
type UART struct {
	Buffer *RingBuffer
	Bus    *sam.SERCOM_USART_Type
}

var (
	// UART0 is actually a USB CDC interface.
	UART0 = USBCDC{Buffer: NewRingBuffer()}

	// The first hardware serial port on the SAMD21. Uses the SERCOM0 interface.
	UART1 = UART{Bus: sam.SERCOM0_USART, Buffer: NewRingBuffer()}
)

const (
	sampleRate16X  = 16
	lsbFirst       = 1
	sercomRXPad0   = 0
	sercomRXPad1   = 1
	sercomRXPad2   = 2
	sercomRXPad3   = 3
	sercomTXPad0   = 0 // Only for UART
	sercomTXPad2   = 1 // Only for UART
	sercomTXPad023 = 2 // Only for UART with TX on PAD0, RTS on PAD2 and CTS on PAD3

	spiTXPad0SCK1 = 0
	spiTXPad2SCK3 = 1
	spiTXPad3SCK1 = 2
	spiTXPad0SCK3 = 3
)

// Configure the UART.
func (uart UART) Configure(config UARTConfig) {
	// Default baud rate to 115200.
	if config.BaudRate == 0 {
		config.BaudRate = 115200
	}

	// determine pins
	if config.TX == 0 {
		// use default pins
		config.TX = UART_TX_PIN
		config.RX = UART_RX_PIN
	}

	// determine pads
	var txpad, rxpad int
	switch config.TX {
	case UART_TX_PIN:
		txpad = sercomTXPad2
	case D10:
		txpad = sercomTXPad2
	case D11:
		txpad = sercomTXPad0
	default:
		panic("Invalid TX pin for UART")
	}

	switch config.RX {
	case UART_RX_PIN:
		rxpad = sercomRXPad3
	case D10:
		rxpad = sercomRXPad2
	case D11:
		rxpad = sercomRXPad0
	case D12:
		rxpad = sercomRXPad3
	case D13:
		rxpad = sercomRXPad1
	default:
		panic("Invalid RX pin for UART")
	}

	// configure pins
	GPIO{config.TX}.Configure(GPIOConfig{Mode: GPIO_SERCOM})
	GPIO{config.RX}.Configure(GPIOConfig{Mode: GPIO_SERCOM})

	// reset SERCOM0
	uart.Bus.CTRLA |= sam.SERCOM_USART_CTRLA_SWRST
	for (uart.Bus.CTRLA&sam.SERCOM_USART_CTRLA_SWRST) > 0 ||
		(uart.Bus.SYNCBUSY&sam.SERCOM_USART_SYNCBUSY_SWRST) > 0 {
	}

	// set UART mode/sample rate
	// SERCOM_USART_CTRLA_MODE(mode) |
	// SERCOM_USART_CTRLA_SAMPR(sampleRate);
	uart.Bus.CTRLA = (sam.SERCOM_USART_CTRLA_MODE_USART_INT_CLK << sam.SERCOM_USART_CTRLA_MODE_Pos) |
		(1 << sam.SERCOM_USART_CTRLA_SAMPR_Pos) // sample rate of 16x

	// Set baud rate
	uart.SetBaudRate(config.BaudRate)

	// setup UART frame
	// SERCOM_USART_CTRLA_FORM( (parityMode == SERCOM_NO_PARITY ? 0 : 1) ) |
	// dataOrder << SERCOM_USART_CTRLA_DORD_Pos;
	uart.Bus.CTRLA |= (0 << sam.SERCOM_USART_CTRLA_FORM_Pos) | // no parity
		(lsbFirst << sam.SERCOM_USART_CTRLA_DORD_Pos) // data order

	// set UART stop bits/parity
	// SERCOM_USART_CTRLB_CHSIZE(charSize) |
	// 	nbStopBits << SERCOM_USART_CTRLB_SBMODE_Pos |
	// 	(parityMode == SERCOM_NO_PARITY ? 0 : parityMode) << SERCOM_USART_CTRLB_PMODE_Pos; //If no parity use default value
	uart.Bus.CTRLB |= (0 << sam.SERCOM_USART_CTRLB_CHSIZE_Pos) | // 8 bits is 0
		(0 << sam.SERCOM_USART_CTRLB_SBMODE_Pos) | // 1 stop bit is zero
		(0 << sam.SERCOM_USART_CTRLB_PMODE_Pos) // no parity

	// set UART pads. This is not same as pins...
	//  SERCOM_USART_CTRLA_TXPO(txPad) |
	//   SERCOM_USART_CTRLA_RXPO(rxPad);
	uart.Bus.CTRLA |= sam.RegValue((txpad << sam.SERCOM_USART_CTRLA_TXPO_Pos) |
		(rxpad << sam.SERCOM_USART_CTRLA_RXPO_Pos))

	// Enable Transceiver and Receiver
	//sercom->USART.CTRLB.reg |= SERCOM_USART_CTRLB_TXEN | SERCOM_USART_CTRLB_RXEN ;
	uart.Bus.CTRLB |= (sam.SERCOM_USART_CTRLB_TXEN | sam.SERCOM_USART_CTRLB_RXEN)

	// Enable USART1 port.
	// sercom->USART.CTRLA.bit.ENABLE = 0x1u;
	uart.Bus.CTRLA |= sam.SERCOM_USART_CTRLA_ENABLE
	for (uart.Bus.SYNCBUSY & sam.SERCOM_USART_SYNCBUSY_ENABLE) > 0 {
	}

	// setup interrupt on receive
	uart.Bus.INTENSET = sam.SERCOM_USART_INTENSET_RXC

	// Enable RX IRQ.
	if config.TX == UART_TX_PIN {
		// UART0
		arm.EnableIRQ(sam.IRQ_SERCOM0)
	} else {
		// UART1
		arm.EnableIRQ(sam.IRQ_SERCOM1)
	}
}

// SetBaudRate sets the communication speed for the UART.
func (uart UART) SetBaudRate(br uint32) {
	// Asynchronous fractional mode (Table 24-2 in datasheet)
	//   BAUD = fref / (sampleRateValue * fbaud)
	// (multiply by 8, to calculate fractional piece)
	// uint32_t baudTimes8 = (SystemCoreClock * 8) / (16 * baudrate);
	baud := (CPU_FREQUENCY * 8) / (sampleRate16X * br)

	// sercom->USART.BAUD.FRAC.FP   = (baudTimes8 % 8);
	// sercom->USART.BAUD.FRAC.BAUD = (baudTimes8 / 8);
	uart.Bus.BAUD = sam.RegValue16(((baud % 8) << sam.SERCOM_USART_BAUD_FRAC_MODE_FP_Pos) |
		((baud / 8) << sam.SERCOM_USART_BAUD_FRAC_MODE_BAUD_Pos))
}

// WriteByte writes a byte of data to the UART.
func (uart UART) WriteByte(c byte) error {
	// wait until ready to receive
	for (uart.Bus.INTFLAG & sam.SERCOM_USART_INTFLAG_DRE) == 0 {
	}
	uart.Bus.DATA = sam.RegValue16(c)
	return nil
}

//go:export SERCOM0_IRQHandler
func handleUART1() {
	// should reset IRQ
	UART1.Receive(byte((UART1.Bus.DATA & 0xFF)))
	UART1.Bus.INTFLAG |= sam.SERCOM_USART_INTFLAG_RXC
}

// I2C on the SAMD21.
type I2C struct {
	Bus *sam.SERCOM_I2CM_Type
}

// I2CConfig is used to store config info for I2C.
type I2CConfig struct {
	Frequency uint32
	SCL       uint8
	SDA       uint8
}

const (
	// Default rise time in nanoseconds, based on 4.7K ohm pull up resistors
	riseTimeNanoseconds = 125

	// wire bus states
	wireUnknownState = 0
	wireIdleState    = 1
	wireOwnerState   = 2
	wireBusyState    = 3

	// wire commands
	wireCmdNoAction    = 0
	wireCmdRepeatStart = 1
	wireCmdRead        = 2
	wireCmdStop        = 3
)

const i2cTimeout = 1000

// Configure is intended to setup the I2C interface.
func (i2c I2C) Configure(config I2CConfig) {
	// Default I2C bus speed is 100 kHz.
	if config.Frequency == 0 {
		config.Frequency = TWI_FREQ_100KHZ
	}

	// reset SERCOM3
	i2c.Bus.CTRLA |= sam.SERCOM_I2CM_CTRLA_SWRST
	for (i2c.Bus.CTRLA&sam.SERCOM_I2CM_CTRLA_SWRST) > 0 ||
		(i2c.Bus.SYNCBUSY&sam.SERCOM_I2CM_SYNCBUSY_SWRST) > 0 {
	}

	// Set i2c master mode
	//SERCOM_I2CM_CTRLA_MODE( I2C_MASTER_OPERATION )
	i2c.Bus.CTRLA = (sam.SERCOM_I2CM_CTRLA_MODE_I2C_MASTER << sam.SERCOM_I2CM_CTRLA_MODE_Pos) // |

	i2c.SetBaudRate(config.Frequency)

	// Enable I2CM port.
	// sercom->USART.CTRLA.bit.ENABLE = 0x1u;
	i2c.Bus.CTRLA |= sam.SERCOM_I2CM_CTRLA_ENABLE
	for (i2c.Bus.SYNCBUSY & sam.SERCOM_I2CM_SYNCBUSY_ENABLE) > 0 {
	}

	// set bus idle mode
	i2c.Bus.STATUS |= (wireIdleState << sam.SERCOM_I2CM_STATUS_BUSSTATE_Pos)
	for (i2c.Bus.SYNCBUSY & sam.SERCOM_I2CM_SYNCBUSY_SYSOP) > 0 {
	}

	// enable pins
	GPIO{SDA_PIN}.Configure(GPIOConfig{Mode: GPIO_SERCOM})
	GPIO{SCL_PIN}.Configure(GPIOConfig{Mode: GPIO_SERCOM})
}

// SetBaudRate sets the communication speed for the I2C.
func (i2c I2C) SetBaudRate(br uint32) {
	// Synchronous arithmetic baudrate, via Arduino SAMD implementation:
	// SystemCoreClock / ( 2 * baudrate) - 5 - (((SystemCoreClock / 1000000) * WIRE_RISE_TIME_NANOSECONDS) / (2 * 1000));
	baud := CPU_FREQUENCY/(2*br) - 5 - (((CPU_FREQUENCY / 1000000) * riseTimeNanoseconds) / (2 * 1000))
	i2c.Bus.BAUD = sam.RegValue(baud)
}

// Tx does a single I2C transaction at the specified address.
// It clocks out the given address, writes the bytes in w, reads back len(r)
// bytes and stores them in r, and generates a stop condition on the bus.
func (i2c I2C) Tx(addr uint16, w, r []byte) error {
	var err error
	if len(w) != 0 {
		// send start/address for write
		i2c.sendAddress(addr, true)

		// wait until transmission complete
		timeout := i2cTimeout
		for (i2c.Bus.INTFLAG & sam.SERCOM_I2CM_INTFLAG_MB) == 0 {
			timeout--
			if timeout == 0 {
				return errors.New("I2C timeout on ready to write data")
			}
		}

		// ACK received (0: ACK, 1: NACK)
		if (i2c.Bus.STATUS & sam.SERCOM_I2CM_STATUS_RXNACK) > 0 {
			return errors.New("I2C write error: expected ACK not NACK")
		}

		// write data
		for _, b := range w {
			err = i2c.WriteByte(b)
			if err != nil {
				return err
			}
		}

		err = i2c.signalStop()
		if err != nil {
			return err
		}
	}
	if len(r) != 0 {
		// send start/address for read
		i2c.sendAddress(addr, false)

		// wait transmission complete
		for (i2c.Bus.INTFLAG & sam.SERCOM_I2CM_INTFLAG_SB) == 0 {
			// If the slave NACKS the address, the MB bit will be set.
			// In that case, send a stop condition and return error.
			if (i2c.Bus.INTFLAG & sam.SERCOM_I2CM_INTFLAG_MB) > 0 {
				i2c.Bus.CTRLB |= (wireCmdStop << sam.SERCOM_I2CM_CTRLB_CMD_Pos) // Stop condition
				return errors.New("I2C read error: expected ACK not NACK")
			}
		}

		// ACK received (0: ACK, 1: NACK)
		if (i2c.Bus.STATUS & sam.SERCOM_I2CM_STATUS_RXNACK) > 0 {
			return errors.New("I2C read error: expected ACK not NACK")
		}

		// read first byte
		r[0] = i2c.readByte()
		for i := 1; i < len(r); i++ {
			// Send an ACK
			i2c.Bus.CTRLB &^= sam.SERCOM_I2CM_CTRLB_ACKACT

			i2c.signalRead()

			// Read data and send the ACK
			r[i] = i2c.readByte()
		}

		// Send NACK to end transmission
		i2c.Bus.CTRLB |= sam.SERCOM_I2CM_CTRLB_ACKACT

		err = i2c.signalStop()
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteByte writes a single byte to the I2C bus.
func (i2c I2C) WriteByte(data byte) error {
	// Send data byte
	i2c.Bus.DATA = sam.RegValue8(data)

	// wait until transmission successful
	timeout := i2cTimeout
	for (i2c.Bus.INTFLAG & sam.SERCOM_I2CM_INTFLAG_MB) == 0 {
		// check for bus error
		if (sam.SERCOM3_I2CM.STATUS & sam.SERCOM_I2CM_STATUS_BUSERR) > 0 {
			return errors.New("I2C bus error")
		}
		timeout--
		if timeout == 0 {
			return errors.New("I2C timeout on write data")
		}
	}

	if (i2c.Bus.STATUS & sam.SERCOM_I2CM_STATUS_RXNACK) > 0 {
		return errors.New("I2C write error: expected ACK not NACK")
	}

	return nil
}

// sendAddress sends the address and start signal
func (i2c I2C) sendAddress(address uint16, write bool) error {
	data := (address << 1)
	if !write {
		data |= 1 // set read flag
	}

	// wait until bus ready
	timeout := i2cTimeout
	for (i2c.Bus.STATUS&(wireIdleState<<sam.SERCOM_I2CM_STATUS_BUSSTATE_Pos)) == 0 &&
		(i2c.Bus.STATUS&(wireOwnerState<<sam.SERCOM_I2CM_STATUS_BUSSTATE_Pos)) == 0 {
		timeout--
		if timeout == 0 {
			return errors.New("I2C timeout on bus ready")
		}
	}
	i2c.Bus.ADDR = sam.RegValue(data)

	return nil
}

func (i2c I2C) signalStop() error {
	i2c.Bus.CTRLB |= (wireCmdStop << sam.SERCOM_I2CM_CTRLB_CMD_Pos) // Stop command
	timeout := i2cTimeout
	for (i2c.Bus.SYNCBUSY & sam.SERCOM_I2CM_SYNCBUSY_SYSOP) > 0 {
		timeout--
		if timeout == 0 {
			return errors.New("I2C timeout on signal stop")
		}
	}
	return nil
}

func (i2c I2C) signalRead() error {
	i2c.Bus.CTRLB |= (wireCmdRead << sam.SERCOM_I2CM_CTRLB_CMD_Pos) // Read command
	timeout := i2cTimeout
	for (i2c.Bus.SYNCBUSY & sam.SERCOM_I2CM_SYNCBUSY_SYSOP) > 0 {
		timeout--
		if timeout == 0 {
			return errors.New("I2C timeout on signal read")
		}
	}
	return nil
}

func (i2c I2C) readByte() byte {
	for (i2c.Bus.INTFLAG & sam.SERCOM_I2CM_INTFLAG_SB) == 0 {
	}
	return byte(i2c.Bus.DATA)
}

// SPI
type SPI struct {
	Bus *sam.SERCOM_SPI_Type
}

// SPIConfig is used to store config info for SPI.
type SPIConfig struct {
	Frequency uint32
	SCK       uint8
	MOSI      uint8
	MISO      uint8
	LSBFirst  bool
	Mode      uint8
}

// Configure is intended to setup the SPI interface.
func (spi SPI) Configure(config SPIConfig) {
	config.SCK = SPI0_SCK_PIN
	config.MOSI = SPI0_MOSI_PIN
	config.MISO = SPI0_MISO_PIN

	doPad := spiTXPad2SCK3
	diPad := sercomRXPad0

	// set default frequency
	if config.Frequency == 0 {
		config.Frequency = 4000000
	}

	// Disable SPI port.
	spi.Bus.CTRLA &^= sam.SERCOM_SPI_CTRLA_ENABLE
	for (spi.Bus.SYNCBUSY & sam.SERCOM_SPI_SYNCBUSY_ENABLE) > 0 {
	}

	// enable pins
	GPIO{config.SCK}.Configure(GPIOConfig{Mode: GPIO_SERCOM_ALT})
	GPIO{config.MOSI}.Configure(GPIOConfig{Mode: GPIO_SERCOM_ALT})
	GPIO{config.MISO}.Configure(GPIOConfig{Mode: GPIO_SERCOM_ALT})

	// reset SERCOM
	spi.Bus.CTRLA |= sam.SERCOM_SPI_CTRLA_SWRST
	for (spi.Bus.CTRLA&sam.SERCOM_SPI_CTRLA_SWRST) > 0 ||
		(spi.Bus.SYNCBUSY&sam.SERCOM_SPI_SYNCBUSY_SWRST) > 0 {
	}

	// set bit transfer order
	dataOrder := 0
	if config.LSBFirst {
		dataOrder = 1
	}

	// Set SPI master
	spi.Bus.CTRLA = (sam.SERCOM_SPI_CTRLA_MODE_SPI_MASTER << sam.SERCOM_SPI_CTRLA_MODE_Pos) |
		sam.RegValue(doPad<<sam.SERCOM_SPI_CTRLA_DOPO_Pos) |
		sam.RegValue(diPad<<sam.SERCOM_SPI_CTRLA_DIPO_Pos) |
		sam.RegValue(dataOrder<<sam.SERCOM_SPI_CTRLA_DORD_Pos)

	spi.Bus.CTRLB |= (0 << sam.SERCOM_SPI_CTRLB_CHSIZE_Pos) | // 8bit char size
		sam.SERCOM_SPI_CTRLB_RXEN // receive enable

	for (spi.Bus.SYNCBUSY & sam.SERCOM_SPI_SYNCBUSY_CTRLB) > 0 {
	}

	// set mode
	switch config.Mode {
	case 0:
		spi.Bus.CTRLA &^= sam.SERCOM_SPI_CTRLA_CPHA
		spi.Bus.CTRLA &^= sam.SERCOM_SPI_CTRLA_CPOL
	case 1:
		spi.Bus.CTRLA |= sam.SERCOM_SPI_CTRLA_CPHA
		spi.Bus.CTRLA &^= sam.SERCOM_SPI_CTRLA_CPOL
	case 2:
		spi.Bus.CTRLA &^= sam.SERCOM_SPI_CTRLA_CPHA
		spi.Bus.CTRLA |= sam.SERCOM_SPI_CTRLA_CPOL
	case 3:
		spi.Bus.CTRLA |= sam.SERCOM_SPI_CTRLA_CPHA | sam.SERCOM_SPI_CTRLA_CPOL
	default: // to mode 0
		spi.Bus.CTRLA &^= sam.SERCOM_SPI_CTRLA_CPHA
		spi.Bus.CTRLA &^= sam.SERCOM_SPI_CTRLA_CPOL
	}

	// Set synch speed for SPI
	baudRate := (CPU_FREQUENCY / (2 * config.Frequency)) - 1
	spi.Bus.BAUD = sam.RegValue8(baudRate)

	// Enable SPI port.
	spi.Bus.CTRLA |= sam.SERCOM_SPI_CTRLA_ENABLE
	for (spi.Bus.SYNCBUSY & sam.SERCOM_SPI_SYNCBUSY_ENABLE) > 0 {
	}
}

// Transfer writes/reads a single byte using the SPI interface.
func (spi SPI) Transfer(w byte) (byte, error) {
	// write data
	spi.Bus.DATA = sam.RegValue(w)

	// wait for receive
	for (spi.Bus.INTFLAG & sam.SERCOM_SPI_INTFLAG_RXC) == 0 {
	}

	// return data
	return byte(spi.Bus.DATA), nil
}

// PWM
const period = 0xFFFF

// InitPWM initializes the PWM interface.
func InitPWM() {
	// turn on timer clocks used for PWM
	sam.PM.APBCMASK |= sam.PM_APBCMASK_TCC0_ | sam.PM_APBCMASK_TCC1_ | sam.PM_APBCMASK_TCC2_

	// Use GCLK0 for TCC0/TCC1
	sam.GCLK.CLKCTRL = sam.RegValue16((sam.GCLK_CLKCTRL_ID_TCC0_TCC1 << sam.GCLK_CLKCTRL_ID_Pos) |
		(sam.GCLK_CLKCTRL_GEN_GCLK0 << sam.GCLK_CLKCTRL_GEN_Pos) |
		sam.GCLK_CLKCTRL_CLKEN)
	for (sam.GCLK.STATUS & sam.GCLK_STATUS_SYNCBUSY) > 0 {
	}

	// Use GCLK0 for TCC2/TC3
	sam.GCLK.CLKCTRL = sam.RegValue16((sam.GCLK_CLKCTRL_ID_TCC2_TC3 << sam.GCLK_CLKCTRL_ID_Pos) |
		(sam.GCLK_CLKCTRL_GEN_GCLK0 << sam.GCLK_CLKCTRL_GEN_Pos) |
		sam.GCLK_CLKCTRL_CLKEN)
	for (sam.GCLK.STATUS & sam.GCLK_STATUS_SYNCBUSY) > 0 {
	}
}

// Configure configures a PWM pin for output.
func (pwm PWM) Configure() {
	// figure out which TCCX timer for this pin
	timer := pwm.getTimer()

	// disable timer
	timer.CTRLA &^= sam.TCC_CTRLA_ENABLE
	// Wait for synchronization
	for (timer.SYNCBUSY & sam.TCC_SYNCBUSY_ENABLE) > 0 {
	}

	// Use "Normal PWM" (single-slope PWM)
	timer.WAVE |= sam.TCC_WAVE_WAVEGEN_NPWM
	// Wait for synchronization
	for (timer.SYNCBUSY & sam.TCC_SYNCBUSY_WAVE) > 0 {
	}

	// Set the period (the number to count to (TOP) before resetting timer)
	//TCC0->PER.reg = period;
	timer.PER = period
	// Wait for synchronization
	for (timer.SYNCBUSY & sam.TCC_SYNCBUSY_PER) > 0 {
	}

	// Set pin as output
	sam.PORT.DIRSET0 = (1 << pwm.Pin)
	// Set pin to low
	sam.PORT.OUTCLR0 = (1 << pwm.Pin)

	// Enable the port multiplexer for pin
	pwm.setPinCfg(sam.PORT_PINCFG0_PMUXEN)

	// Connect TCCX timer to pin.
	// we normally use the F channel aka ALT
	pwmConfig := GPIO_PWM_ALT

	// in the case of PA6 or PA7 we have to use E channel
	if pwm.Pin == 6 || pwm.Pin == 7 {
		pwmConfig = GPIO_PWM
	}

	if pwm.Pin&1 > 0 {
		// odd pin, so save the even pins
		val := pwm.getPMux() & sam.PORT_PMUX0_PMUXE_Msk
		pwm.setPMux(val | sam.RegValue8(pwmConfig<<sam.PORT_PMUX0_PMUXO_Pos))
	} else {
		// even pin, so save the odd pins
		val := pwm.getPMux() & sam.PORT_PMUX0_PMUXO_Msk
		pwm.setPMux(val | sam.RegValue8(pwmConfig<<sam.PORT_PMUX0_PMUXE_Pos))
	}
}

// Set turns on the duty cycle for a PWM pin using the provided value.
func (pwm PWM) Set(value uint16) {
	// figure out which TCCX timer for this pin
	timer := pwm.getTimer()

	// disable output
	timer.CTRLA &^= sam.TCC_CTRLA_ENABLE

	// Wait for synchronization
	for (timer.SYNCBUSY & sam.TCC_SYNCBUSY_ENABLE) > 0 {
	}

	// Set PWM signal to output duty cycle
	pwm.setChannel(sam.RegValue(value))

	// Wait for synchronization on all channels
	for (timer.SYNCBUSY & (sam.TCC_SYNCBUSY_CC0 |
		sam.TCC_SYNCBUSY_CC1 |
		sam.TCC_SYNCBUSY_CC2 |
		sam.TCC_SYNCBUSY_CC3)) > 0 {
	}

	// enable
	timer.CTRLA |= sam.TCC_CTRLA_ENABLE
	// Wait for synchronization
	for (timer.SYNCBUSY & sam.TCC_SYNCBUSY_ENABLE) > 0 {
	}
}

// getPMux returns the value for the correct PMUX register for this pin.
func (pwm PWM) getPMux() sam.RegValue8 {
	return getPMux(pwm.Pin)
}

// setPMux sets the value for the correct PMUX register for this pin.
func (pwm PWM) setPMux(val sam.RegValue8) {
	setPMux(pwm.Pin, val)
}

// getPinCfg returns the value for the correct PINCFG register for this pin.
func (pwm PWM) getPinCfg() sam.RegValue8 {
	return getPinCfg(pwm.Pin)
}

// setPinCfg sets the value for the correct PINCFG register for this pin.
func (pwm PWM) setPinCfg(val sam.RegValue8) {
	setPinCfg(pwm.Pin, val)
}

// getPMux returns the value for the correct PMUX register for this pin.
func getPMux(p uint8) sam.RegValue8 {
	pin := p >> 1
	switch pin {
	case 0:
		return sam.PORT.PMUX0_0
	case 1:
		return sam.PORT.PMUX0_1
	case 2:
		return sam.PORT.PMUX0_2
	case 3:
		return sam.PORT.PMUX0_3
	case 4:
		return sam.PORT.PMUX0_4
	case 5:
		return sam.PORT.PMUX0_5
	case 6:
		return sam.PORT.PMUX0_6
	case 7:
		return sam.PORT.PMUX0_7
	case 8:
		return sam.PORT.PMUX0_8
	case 9:
		return sam.PORT.PMUX0_9
	case 10:
		return sam.PORT.PMUX0_10
	case 11:
		return sam.PORT.PMUX0_11
	case 12:
		return sam.PORT.PMUX0_12
	case 13:
		return sam.PORT.PMUX0_13
	case 14:
		return sam.PORT.PMUX0_14
	case 15:
		return sam.PORT.PMUX0_15
	case 16:
		return sam.RegValue8(sam.PORT.PMUX1_0>>0) & 0xff
	case 17:
		return sam.RegValue8(sam.PORT.PMUX1_0>>8) & 0xff
	case 18:
		return sam.RegValue8(sam.PORT.PMUX1_0>>16) & 0xff
	case 19:
		return sam.RegValue8(sam.PORT.PMUX1_0>>24) & 0xff
	case 20:
		return sam.RegValue8(sam.PORT.PMUX1_4>>0) & 0xff
	case 21:
		return sam.RegValue8(sam.PORT.PMUX1_4>>8) & 0xff
	case 22:
		return sam.RegValue8(sam.PORT.PMUX1_4>>16) & 0xff
	case 23:
		return sam.RegValue8(sam.PORT.PMUX1_4>>24) & 0xff
	case 24:
		return sam.RegValue8(sam.PORT.PMUX1_8>>0) & 0xff
	case 25:
		return sam.RegValue8(sam.PORT.PMUX1_8>>8) & 0xff
	case 26:
		return sam.RegValue8(sam.PORT.PMUX1_8>>16) & 0xff
	case 27:
		return sam.RegValue8(sam.PORT.PMUX1_8>>24) & 0xff
	case 28:
		return sam.RegValue8(sam.PORT.PMUX1_12>>0) & 0xff
	case 29:
		return sam.RegValue8(sam.PORT.PMUX1_12>>8) & 0xff
	case 30:
		return sam.RegValue8(sam.PORT.PMUX1_12>>16) & 0xff
	case 31:
		return sam.RegValue8(sam.PORT.PMUX1_12>>24) & 0xff
	default:
		return 0
	}
}

// setPMux sets the value for the correct PMUX register for this pin.
func setPMux(p uint8, val sam.RegValue8) {
	pin := p >> 1
	switch pin {
	case 0:
		sam.PORT.PMUX0_0 = val
	case 1:
		sam.PORT.PMUX0_1 = val
	case 2:
		sam.PORT.PMUX0_2 = val
	case 3:
		sam.PORT.PMUX0_3 = val
	case 4:
		sam.PORT.PMUX0_4 = val
	case 5:
		sam.PORT.PMUX0_5 = val
	case 6:
		sam.PORT.PMUX0_6 = val
	case 7:
		sam.PORT.PMUX0_7 = val
	case 8:
		sam.PORT.PMUX0_8 = val
	case 9:
		sam.PORT.PMUX0_9 = val
	case 10:
		sam.PORT.PMUX0_10 = val
	case 11:
		sam.PORT.PMUX0_11 = val
	case 12:
		sam.PORT.PMUX0_12 = val
	case 13:
		sam.PORT.PMUX0_13 = val
	case 14:
		sam.PORT.PMUX0_14 = val
	case 15:
		sam.PORT.PMUX0_15 = val
	case 16:
		sam.PORT.PMUX1_0 = (sam.PORT.PMUX1_0 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 17:
		sam.PORT.PMUX1_0 = (sam.PORT.PMUX1_0 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 18:
		sam.PORT.PMUX1_0 = (sam.PORT.PMUX1_0 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 19:
		sam.PORT.PMUX1_0 = (sam.PORT.PMUX1_0 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 20:
		sam.PORT.PMUX1_4 = (sam.PORT.PMUX1_4 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 21:
		sam.PORT.PMUX1_4 = (sam.PORT.PMUX1_4 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 22:
		sam.PORT.PMUX1_4 = (sam.PORT.PMUX1_4 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 23:
		sam.PORT.PMUX1_4 = (sam.PORT.PMUX1_4 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 24:
		sam.PORT.PMUX1_8 = (sam.PORT.PMUX1_8 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 25:
		sam.PORT.PMUX1_8 = (sam.PORT.PMUX1_8 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 26:
		sam.PORT.PMUX1_8 = (sam.PORT.PMUX1_8 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 27:
		sam.PORT.PMUX1_8 = (sam.PORT.PMUX1_8 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 28:
		sam.PORT.PMUX1_12 = (sam.PORT.PMUX1_12 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 29:
		sam.PORT.PMUX1_12 = (sam.PORT.PMUX1_12 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 30:
		sam.PORT.PMUX1_12 = (sam.PORT.PMUX1_12 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 31:
		sam.PORT.PMUX1_12 = (sam.PORT.PMUX1_12 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	}
}

// getPinCfg returns the value for the correct PINCFG register for this pin.
func getPinCfg(p uint8) sam.RegValue8 {
	switch p {
	case 0:
		return sam.PORT.PINCFG0_0
	case 1:
		return sam.PORT.PINCFG0_1
	case 2:
		return sam.PORT.PINCFG0_2
	case 3:
		return sam.PORT.PINCFG0_3
	case 4:
		return sam.PORT.PINCFG0_4
	case 5:
		return sam.PORT.PINCFG0_5
	case 6:
		return sam.PORT.PINCFG0_6
	case 7:
		return sam.PORT.PINCFG0_7
	case 8:
		return sam.PORT.PINCFG0_8
	case 9:
		return sam.PORT.PINCFG0_9
	case 10:
		return sam.PORT.PINCFG0_10
	case 11:
		return sam.PORT.PINCFG0_11
	case 12:
		return sam.PORT.PINCFG0_12
	case 13:
		return sam.PORT.PINCFG0_13
	case 14:
		return sam.PORT.PINCFG0_14
	case 15:
		return sam.PORT.PINCFG0_15
	case 16:
		return sam.PORT.PINCFG0_16
	case 17:
		return sam.PORT.PINCFG0_17
	case 18:
		return sam.PORT.PINCFG0_18
	case 19:
		return sam.PORT.PINCFG0_19
	case 20:
		return sam.PORT.PINCFG0_20
	case 21:
		return sam.PORT.PINCFG0_21
	case 22:
		return sam.PORT.PINCFG0_22
	case 23:
		return sam.PORT.PINCFG0_23
	case 24:
		return sam.PORT.PINCFG0_24
	case 25:
		return sam.PORT.PINCFG0_25
	case 26:
		return sam.PORT.PINCFG0_26
	case 27:
		return sam.PORT.PINCFG0_27
	case 28:
		return sam.PORT.PINCFG0_28
	case 29:
		return sam.PORT.PINCFG0_29
	case 30:
		return sam.PORT.PINCFG0_30
	case 31:
		return sam.PORT.PINCFG0_31
	case 32: // PB00
		return sam.RegValue8(sam.PORT.PINCFG1_0>>0) & 0xff
	case 33: // PB01
		return sam.RegValue8(sam.PORT.PINCFG1_0>>8) & 0xff
	case 34: // PB02
		return sam.RegValue8(sam.PORT.PINCFG1_0>>16) & 0xff
	case 35: // PB03
		return sam.RegValue8(sam.PORT.PINCFG1_0>>24) & 0xff
	case 37: // PB04
		return sam.RegValue8(sam.PORT.PINCFG1_4>>0) & 0xff
	case 38: // PB05
		return sam.RegValue8(sam.PORT.PINCFG1_4>>8) & 0xff
	case 39: // PB06
		return sam.RegValue8(sam.PORT.PINCFG1_4>>16) & 0xff
	case 40: // PB07
		return sam.RegValue8(sam.PORT.PINCFG1_4>>24) & 0xff
	case 41: // PB08
		return sam.RegValue8(sam.PORT.PINCFG1_8>>0) & 0xff
	case 42: // PB09
		return sam.RegValue8(sam.PORT.PINCFG1_8>>8) & 0xff
	case 43: // PB10
		return sam.RegValue8(sam.PORT.PINCFG1_8>>16) & 0xff
	case 44: // PB11
		return sam.RegValue8(sam.PORT.PINCFG1_8>>24) & 0xff
	case 45: // PB12
		return sam.RegValue8(sam.PORT.PINCFG1_12>>0) & 0xff
	case 46: // PB13
		return sam.RegValue8(sam.PORT.PINCFG1_12>>8) & 0xff
	case 47: // PB14
		return sam.RegValue8(sam.PORT.PINCFG1_12>>16) & 0xff
	case 48: // PB15
		return sam.RegValue8(sam.PORT.PINCFG1_12>>24) & 0xff
	case 49: // PB16
		return sam.RegValue8(sam.PORT.PINCFG1_16>>0) & 0xff
	case 50: // PB17
		return sam.RegValue8(sam.PORT.PINCFG1_16>>8) & 0xff
	case 51: // PB18
		return sam.RegValue8(sam.PORT.PINCFG1_16>>16) & 0xff
	case 52: // PB19
		return sam.RegValue8(sam.PORT.PINCFG1_16>>24) & 0xff
	case 53: // PB20
		return sam.RegValue8(sam.PORT.PINCFG1_20>>0) & 0xff
	case 54: // PB21
		return sam.RegValue8(sam.PORT.PINCFG1_20>>8) & 0xff
	case 55: // PB22
		return sam.RegValue8(sam.PORT.PINCFG1_20>>16) & 0xff
	case 56: // PB23
		return sam.RegValue8(sam.PORT.PINCFG1_20>>24) & 0xff
	case 57: // PB24
		return sam.RegValue8(sam.PORT.PINCFG1_24>>0) & 0xff
	case 58: // PB25
		return sam.RegValue8(sam.PORT.PINCFG1_24>>8) & 0xff
	case 59: // PB26
		return sam.RegValue8(sam.PORT.PINCFG1_24>>16) & 0xff
	case 60: // PB27
		return sam.RegValue8(sam.PORT.PINCFG1_24>>24) & 0xff
	case 61: // PB28
		return sam.RegValue8(sam.PORT.PINCFG1_28>>0) & 0xff
	case 62: // PB29
		return sam.RegValue8(sam.PORT.PINCFG1_28>>8) & 0xff
	case 63: // PB30
		return sam.RegValue8(sam.PORT.PINCFG1_28>>16) & 0xff
	case 64: // PB31
		return sam.RegValue8(sam.PORT.PINCFG1_28>>24) & 0xff
	default:
		return 0
	}
}

// setPinCfg sets the value for the correct PINCFG register for this pin.
func setPinCfg(p uint8, val sam.RegValue8) {
	switch p {
	case 0:
		sam.PORT.PINCFG0_0 = val
	case 1:
		sam.PORT.PINCFG0_1 = val
	case 2:
		sam.PORT.PINCFG0_2 = val
	case 3:
		sam.PORT.PINCFG0_3 = val
	case 4:
		sam.PORT.PINCFG0_4 = val
	case 5:
		sam.PORT.PINCFG0_5 = val
	case 6:
		sam.PORT.PINCFG0_6 = val
	case 7:
		sam.PORT.PINCFG0_7 = val
	case 8:
		sam.PORT.PINCFG0_8 = val
	case 9:
		sam.PORT.PINCFG0_9 = val
	case 10:
		sam.PORT.PINCFG0_10 = val
	case 11:
		sam.PORT.PINCFG0_11 = val
	case 12:
		sam.PORT.PINCFG0_12 = val
	case 13:
		sam.PORT.PINCFG0_13 = val
	case 14:
		sam.PORT.PINCFG0_14 = val
	case 15:
		sam.PORT.PINCFG0_15 = val
	case 16:
		sam.PORT.PINCFG0_16 = val
	case 17:
		sam.PORT.PINCFG0_17 = val
	case 18:
		sam.PORT.PINCFG0_18 = val
	case 19:
		sam.PORT.PINCFG0_19 = val
	case 20:
		sam.PORT.PINCFG0_20 = val
	case 21:
		sam.PORT.PINCFG0_21 = val
	case 22:
		sam.PORT.PINCFG0_22 = val
	case 23:
		sam.PORT.PINCFG0_23 = val
	case 24:
		sam.PORT.PINCFG0_24 = val
	case 25:
		sam.PORT.PINCFG0_25 = val
	case 26:
		sam.PORT.PINCFG0_26 = val
	case 27:
		sam.PORT.PINCFG0_27 = val
	case 28:
		sam.PORT.PINCFG0_28 = val
	case 29:
		sam.PORT.PINCFG0_29 = val
	case 30:
		sam.PORT.PINCFG0_30 = val
	case 31:
		sam.PORT.PINCFG0_31 = val
	case 32: // PB00
		sam.PORT.PINCFG1_0 = (sam.PORT.PINCFG1_0 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 33: // PB01
		sam.PORT.PINCFG1_0 = (sam.PORT.PINCFG1_0 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 34: // PB02
		sam.PORT.PINCFG1_0 = (sam.PORT.PINCFG1_0 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 35: // PB03
		sam.PORT.PINCFG1_0 = (sam.PORT.PINCFG1_0 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 36: // PB04
		sam.PORT.PINCFG1_4 = (sam.PORT.PINCFG1_4 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 37: // PB05
		sam.PORT.PINCFG1_4 = (sam.PORT.PINCFG1_4 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 38: // PB06
		sam.PORT.PINCFG1_4 = (sam.PORT.PINCFG1_4 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 39: // PB07
		sam.PORT.PINCFG1_4 = (sam.PORT.PINCFG1_4 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 40: // PB08
		sam.PORT.PINCFG1_8 = (sam.PORT.PINCFG1_8 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 41: // PB09
		sam.PORT.PINCFG1_8 = (sam.PORT.PINCFG1_8 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 42: // PB10
		sam.PORT.PINCFG1_8 = (sam.PORT.PINCFG1_8 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 43: // PB11
		sam.PORT.PINCFG1_8 = (sam.PORT.PINCFG1_8 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 44: // PB12
		sam.PORT.PINCFG1_12 = (sam.PORT.PINCFG1_12 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 45: // PB13
		sam.PORT.PINCFG1_12 = (sam.PORT.PINCFG1_12 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 46: // PB14
		sam.PORT.PINCFG1_12 = (sam.PORT.PINCFG1_12 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 47: // PB15
		sam.PORT.PINCFG1_12 = (sam.PORT.PINCFG1_12 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 48: // PB16
		sam.PORT.PINCFG1_16 = (sam.PORT.PINCFG1_16 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 49: // PB17
		sam.PORT.PINCFG1_16 = (sam.PORT.PINCFG1_16 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 50: // PB18
		sam.PORT.PINCFG1_16 = (sam.PORT.PINCFG1_16 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 51: // PB19
		sam.PORT.PINCFG1_16 = (sam.PORT.PINCFG1_16 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 52: // PB20
		sam.PORT.PINCFG1_20 = (sam.PORT.PINCFG1_20 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 53: // PB21
		sam.PORT.PINCFG1_20 = (sam.PORT.PINCFG1_20 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 54: // PB22
		sam.PORT.PINCFG1_20 = (sam.PORT.PINCFG1_20 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 55: // PB23
		sam.PORT.PINCFG1_20 = (sam.PORT.PINCFG1_20 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 56: // PB24
		sam.PORT.PINCFG1_24 = (sam.PORT.PINCFG1_24 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 57: // PB25
		sam.PORT.PINCFG1_24 = (sam.PORT.PINCFG1_24 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 58: // PB26
		sam.PORT.PINCFG1_24 = (sam.PORT.PINCFG1_24 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 59: // PB27
		sam.PORT.PINCFG1_24 = (sam.PORT.PINCFG1_24 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	case 60: // PB28
		sam.PORT.PINCFG1_28 = (sam.PORT.PINCFG1_28 &^ (0xff << 0)) | (sam.RegValue(val) << 0)
	case 61: // PB29
		sam.PORT.PINCFG1_28 = (sam.PORT.PINCFG1_28 &^ (0xff << 8)) | (sam.RegValue(val) << 8)
	case 62: // PB30
		sam.PORT.PINCFG1_28 = (sam.PORT.PINCFG1_28 &^ (0xff << 16)) | (sam.RegValue(val) << 16)
	case 63: // PB31
		sam.PORT.PINCFG1_28 = (sam.PORT.PINCFG1_28 &^ (0xff << 24)) | (sam.RegValue(val) << 24)
	}
}

// getTimer returns the timer to be used for PWM on this pin
func (pwm PWM) getTimer() *sam.TCC_Type {
	switch pwm.Pin {
	case 6:
		return sam.TCC1
	case 7:
		return sam.TCC1
	case 8:
		return sam.TCC1
	case 9:
		return sam.TCC1
	case 14:
		return sam.TCC0
	case 15:
		return sam.TCC0
	case 16:
		return sam.TCC0
	case 17:
		return sam.TCC0
	case 18:
		return sam.TCC0
	case 19:
		return sam.TCC0
	case 20:
		return sam.TCC0
	case 21:
		return sam.TCC0
	default:
		return nil // not supported on this pin
	}
}

// setChannel sets the value for the correct channel for PWM on this pin
func (pwm PWM) setChannel(val sam.RegValue) {
	switch pwm.Pin {
	case 6:
		pwm.getTimer().CC0 = val
	case 7:
		pwm.getTimer().CC1 = val
	case 8:
		pwm.getTimer().CC0 = val
	case 9:
		pwm.getTimer().CC1 = val
	case 14:
		pwm.getTimer().CC0 = val
	case 15:
		pwm.getTimer().CC1 = val
	case 16:
		pwm.getTimer().CC2 = val
	case 17:
		pwm.getTimer().CC3 = val
	case 18:
		pwm.getTimer().CC2 = val
	case 19:
		pwm.getTimer().CC3 = val
	case 20:
		pwm.getTimer().CC2 = val
	case 21:
		pwm.getTimer().CC3 = val
	default:
		return // not supported on this pin
	}
}

// USBCDC is the USB CDC aka serial over USB interface on the SAMD21.
type USBCDC struct {
	Buffer *RingBuffer
}

// WriteByte writes a byte of data to the USB CDC interface.
func (usbcdc USBCDC) WriteByte(c byte) error {
	// Supposedly to handle problem with Windows USB serial ports?
	if usbLineInfo.lineState > 0 {
		// set the data
		udd_ep_in_cache_buffer[usb_CDC_ENDPOINT_IN][0] = c

		usbEndpointDescriptors[usb_CDC_ENDPOINT_IN].DeviceDescBank[1].ADDR =
			sam.RegValue(uintptr(unsafe.Pointer(&udd_ep_in_cache_buffer[usb_CDC_ENDPOINT_IN])))

		// clean multi packet size of bytes already sent
		usbEndpointDescriptors[usb_CDC_ENDPOINT_IN].DeviceDescBank[1].PCKSIZE &^=
			sam.RegValue(usb_DEVICE_PCKSIZE_MULTI_PACKET_SIZE_Mask << usb_DEVICE_PCKSIZE_MULTI_PACKET_SIZE_Pos)

		// set count of bytes to be sent
		usbEndpointDescriptors[usb_CDC_ENDPOINT_IN].DeviceDescBank[1].PCKSIZE |=
			sam.RegValue((1&usb_DEVICE_PCKSIZE_BYTE_COUNT_Mask)<<usb_DEVICE_PCKSIZE_BYTE_COUNT_Pos) |
				sam.RegValue(epPacketSize(64)<<usb_DEVICE_PCKSIZE_SIZE_Pos)

		// ack transfer complete flag
		setEPINTFLAG(usb_CDC_ENDPOINT_IN, sam.USB_DEVICE_EPINTFLAG_TRCPT1)

		// send data by setting bank ready
		setEPSTATUSSET(usb_CDC_ENDPOINT_IN, sam.USB_DEVICE_EPSTATUSSET_BK1RDY)

		// wait for transfer to complete
		for (getEPINTFLAG(usb_CDC_ENDPOINT_IN) & sam.USB_DEVICE_EPINTFLAG_TRCPT1) == 0 {
		}
	}

	return nil
}

const (
	// these are SAMD21 specific.
	usb_DEVICE_PCKSIZE_BYTE_COUNT_Pos  = 0
	usb_DEVICE_PCKSIZE_BYTE_COUNT_Mask = 0x3FFF

	usb_DEVICE_PCKSIZE_SIZE_Pos  = 28
	usb_DEVICE_PCKSIZE_SIZE_Mask = 0x7

	usb_DEVICE_PCKSIZE_MULTI_PACKET_SIZE_Pos  = 14
	usb_DEVICE_PCKSIZE_MULTI_PACKET_SIZE_Mask = 0x3FFF
)

var (
	usbEndpointDescriptors [8]usbDeviceDescriptor

	udd_ep_in_cache_buffer  [7][128]uint8
	udd_ep_out_cache_buffer [7][128]uint8

	isEndpointHalt        = false
	isRemoteWakeUpEnabled = false
	endPoints             = []uint32{usb_ENDPOINT_TYPE_CONTROL,
		(usb_ENDPOINT_TYPE_INTERRUPT | usbEndpointIn),
		(usb_ENDPOINT_TYPE_BULK | usbEndpointOut),
		(usb_ENDPOINT_TYPE_BULK | usbEndpointIn)}

	usbConfiguration uint8
	usbSetInterface  uint8
	usbLineInfo      = cdcLineInfo{115200, 0x00, 0x00, 0x08, 0x00}
)

// Configure the USB CDC interface. The config is here for compatibility with the UART interface.
func (usbcdc USBCDC) Configure(config UARTConfig) {
	// reset USB interface
	sam.USB_DEVICE.CTRLA |= sam.USB_DEVICE_CTRLA_SWRST
	for (sam.USB_DEVICE.SYNCBUSY&sam.USB_DEVICE_SYNCBUSY_SWRST) > 0 ||
		(sam.USB_DEVICE.SYNCBUSY&sam.USB_DEVICE_SYNCBUSY_ENABLE) > 0 {
	}

	sam.USB_DEVICE.DESCADD = sam.RegValue(uintptr(unsafe.Pointer(&usbEndpointDescriptors)))

	// configure pins
	GPIO{USBCDC_DM_PIN}.Configure(GPIOConfig{Mode: GPIO_COM})
	GPIO{USBCDC_DP_PIN}.Configure(GPIOConfig{Mode: GPIO_COM})

	// performs pad calibration from store fuses
	handlePadCalibration()

	// run in standby
	sam.USB_DEVICE.CTRLA |= sam.USB_DEVICE_CTRLA_RUNSTDBY

	// set full speed
	sam.USB_DEVICE.CTRLB |= (sam.USB_DEVICE_CTRLB_SPDCONF_FS << sam.USB_DEVICE_CTRLB_SPDCONF_Pos)

	// attach
	sam.USB_DEVICE.CTRLB &^= sam.USB_DEVICE_CTRLB_DETACH

	// enable interrupt for end of reset
	sam.USB_DEVICE.INTENSET |= sam.USB_DEVICE_INTENSET_EORST

	// enable interrupt for start of frame
	sam.USB_DEVICE.INTENSET |= sam.USB_DEVICE_INTENSET_SOF

	// enable USB
	sam.USB_DEVICE.CTRLA |= sam.USB_DEVICE_CTRLA_ENABLE

	// enable IRQ
	arm.EnableIRQ(sam.IRQ_USB)
}

func handlePadCalibration() {
	// Load Pad Calibration data from non-volatile memory
	// This requires registers that are not included in the SVD file.
	// Modeled after defines from samd21g18a.h and nvmctrl.h:
	//
	// #define NVMCTRL_OTP4 0x00806020
	//
	// #define USB_FUSES_TRANSN_ADDR       (NVMCTRL_OTP4 + 4)
	// #define USB_FUSES_TRANSN_Pos        13           /**< \brief (NVMCTRL_OTP4) USB pad Transn calibration */
	// #define USB_FUSES_TRANSN_Msk        (0x1Fu << USB_FUSES_TRANSN_Pos)
	// #define USB_FUSES_TRANSN(value)     ((USB_FUSES_TRANSN_Msk & ((value) << USB_FUSES_TRANSN_Pos)))

	// #define USB_FUSES_TRANSP_ADDR       (NVMCTRL_OTP4 + 4)
	// #define USB_FUSES_TRANSP_Pos        18           /**< \brief (NVMCTRL_OTP4) USB pad Transp calibration */
	// #define USB_FUSES_TRANSP_Msk        (0x1Fu << USB_FUSES_TRANSP_Pos)
	// #define USB_FUSES_TRANSP(value)     ((USB_FUSES_TRANSP_Msk & ((value) << USB_FUSES_TRANSP_Pos)))

	// #define USB_FUSES_TRIM_ADDR         (NVMCTRL_OTP4 + 4)
	// #define USB_FUSES_TRIM_Pos          23           /**< \brief (NVMCTRL_OTP4) USB pad Trim calibration */
	// #define USB_FUSES_TRIM_Msk          (0x7u << USB_FUSES_TRIM_Pos)
	// #define USB_FUSES_TRIM(value)       ((USB_FUSES_TRIM_Msk & ((value) << USB_FUSES_TRIM_Pos)))
	//
	fuse := *(*uint32)(unsafe.Pointer(uintptr(0x00806020) + 4))
	calibTransN := sam.RegValue16(uint16(fuse>>13) & uint16(0x1f))
	calibTransP := sam.RegValue16(uint16(fuse>>18) & uint16(0x1f))
	calibTrim := sam.RegValue16(uint16(fuse>>23) & uint16(0x7))

	if calibTransN == 0x1f {
		calibTransN = 5
	}
	sam.USB_DEVICE.PADCAL |= (calibTransN << sam.USB_DEVICE_PADCAL_TRANSN_Pos)

	if calibTransP == 0x1f {
		calibTransP = 29
	}
	sam.USB_DEVICE.PADCAL |= (calibTransP << sam.USB_DEVICE_PADCAL_TRANSP_Pos)

	if calibTrim == 0x7 {
		calibTransN = 3
	}
	sam.USB_DEVICE.PADCAL |= (calibTrim << sam.USB_DEVICE_PADCAL_TRIM_Pos)
}

//go:export USB_IRQHandler
func handleUSB() {
	// reset all interrupt flags
	flags := sam.USB_DEVICE.INTFLAG
	sam.USB_DEVICE.INTFLAG = flags

	// End of reset
	if (flags & sam.USB_DEVICE_INTFLAG_EORST) > 0 {
		// Configure control endpoint
		initEndpoint(0, usb_ENDPOINT_TYPE_CONTROL)

		// Enable Setup-Received interrupt
		setEPINTENSET(0, sam.USB_DEVICE_EPINTENSET_RXSTP)

		usbConfiguration = 0

		// ack the End-Of-Reset interrupt
		sam.USB_DEVICE.INTFLAG = sam.USB_DEVICE_INTFLAG_EORST
	}

	// Start of frame
	if (flags & sam.USB_DEVICE_INTFLAG_SOF) > 0 {
		// if you want to blink LED showing traffic, this would be the place...
	}

	// Endpoint 0 Setup interrupt
	if getEPINTFLAG(0)&sam.USB_DEVICE_EPINTFLAG_RXSTP > 0 {
		// ack setup received
		setEPINTFLAG(0, sam.USB_DEVICE_EPINTFLAG_RXSTP)

		// parse setup
		setup := newUSBSetup(udd_ep_out_cache_buffer[0][:])

		// Clear the Bank 0 ready flag on Control OUT
		setEPSTATUSCLR(0, sam.USB_DEVICE_EPSTATUSCLR_BK0RDY)

		ok := false
		if (setup.bmRequestType & usb_REQUEST_TYPE) == usb_REQUEST_STANDARD {
			// Standard Requests
			ok = handleStandardSetup(setup)
		} else {
			// Class Interface Requests
			if setup.wIndex == usb_CDC_ACM_INTERFACE {
				ok = cdcSetup(setup)
			}
		}

		if ok {
			// set Bank1 ready
			setEPSTATUSSET(0, sam.USB_DEVICE_EPSTATUSSET_BK1RDY)
		} else {
			// Stall endpoint
			setEPSTATUSSET(0, sam.USB_DEVICE_EPINTFLAG_STALL1)
		}

		if getEPINTFLAG(0)&sam.USB_DEVICE_EPINTFLAG_STALL1 > 0 {
			// ack the stall
			setEPINTFLAG(0, sam.USB_DEVICE_EPINTFLAG_STALL1)

			// clear stall request
			setEPINTENCLR(0, sam.USB_DEVICE_EPINTENCLR_STALL1)
		}
	}

	// Now the actual transfer handlers
	eptInts := sam.USB_DEVICE.EPINTSMRY & 0xFE // Remove endpoint number 0 (setup)
	var i uint32
	for i = 1; i < uint32(len(endPoints)); i++ {
		// Check if endpoint has a pending interrupt
		if eptInts&(1<<i) > 0 {
			// yes, so handle flags
			epFlags := getEPINTFLAG(i)
			setEPINTFLAG(i, epFlags)

			// Endpoint Transfer Complete Interrupt
			if (epFlags & sam.USB_DEVICE_EPINTFLAG_TRCPT0) > 0 {
				handleEndpoint(i)
			}
		}
	}
}

func initEndpoint(ep, config uint32) {
	switch config {
	case usb_ENDPOINT_TYPE_INTERRUPT | usbEndpointIn:
		// set packet size
		usbEndpointDescriptors[ep].DeviceDescBank[1].PCKSIZE |=
			sam.RegValue(epPacketSize(64) << usb_DEVICE_PCKSIZE_SIZE_Pos)

		// set data buffer address
		usbEndpointDescriptors[ep].DeviceDescBank[1].ADDR =
			sam.RegValue(uintptr(unsafe.Pointer(&udd_ep_in_cache_buffer[ep])))

		// set endpoint type
		setEPCFG(ep, getEPCFG(ep)|((usb_ENDPOINT_TYPE_INTERRUPT+1)<<sam.USB_DEVICE_EPCFG_EPTYPE1_Pos))

	case usb_ENDPOINT_TYPE_BULK | usbEndpointOut:
		// set packet size
		usbEndpointDescriptors[ep].DeviceDescBank[0].PCKSIZE |=
			sam.RegValue(epPacketSize(64) << usb_DEVICE_PCKSIZE_SIZE_Pos)

		// set data buffer address
		usbEndpointDescriptors[ep].DeviceDescBank[0].ADDR =
			sam.RegValue(uintptr(unsafe.Pointer(&udd_ep_out_cache_buffer[ep])))

		// set endpoint type
		setEPCFG(ep, getEPCFG(ep)|((usb_ENDPOINT_TYPE_BULK+1)<<sam.USB_DEVICE_EPCFG_EPTYPE0_Pos))

		// ack the current transfer
		setEPINTENSET(ep, sam.USB_DEVICE_EPINTENSET_TRCPT0)

		// ready for next transfer
		setEPSTATUSCLR(ep, sam.USB_DEVICE_EPSTATUSCLR_BK0RDY)

	case usb_ENDPOINT_TYPE_INTERRUPT | usbEndpointOut:
		// TODO: not really anything, seems like...

	case usb_ENDPOINT_TYPE_BULK | usbEndpointIn:
		// set packet size
		usbEndpointDescriptors[ep].DeviceDescBank[1].PCKSIZE |=
			sam.RegValue(epPacketSize(64) << usb_DEVICE_PCKSIZE_SIZE_Pos)

		// set data buffer address
		usbEndpointDescriptors[ep].DeviceDescBank[1].ADDR =
			sam.RegValue(uintptr(unsafe.Pointer(&udd_ep_in_cache_buffer[ep])))

		// set endpoint type
		setEPCFG(ep, getEPCFG(ep)|((usb_ENDPOINT_TYPE_BULK+1)<<sam.USB_DEVICE_EPCFG_EPTYPE1_Pos))

		// NAK on endpoint IN, the bank is not yet filled in.
		setEPSTATUSCLR(ep, sam.USB_DEVICE_EPSTATUSCLR_BK1RDY)

	case usb_ENDPOINT_TYPE_CONTROL:
		// Control OUT
		// set packet size
		usbEndpointDescriptors[ep].DeviceDescBank[0].PCKSIZE |=
			sam.RegValue(epPacketSize(64) << usb_DEVICE_PCKSIZE_SIZE_Pos)

		// set data buffer address
		usbEndpointDescriptors[ep].DeviceDescBank[0].ADDR =
			sam.RegValue(uintptr(unsafe.Pointer(&udd_ep_out_cache_buffer[ep])))

		// set endpoint type
		setEPCFG(ep, getEPCFG(ep)|((usb_ENDPOINT_TYPE_CONTROL+1)<<sam.USB_DEVICE_EPCFG_EPTYPE0_Pos))

		// Control IN
		// set packet size
		usbEndpointDescriptors[ep].DeviceDescBank[1].PCKSIZE |=
			sam.RegValue(epPacketSize(64) << usb_DEVICE_PCKSIZE_SIZE_Pos)

		// set data buffer address
		usbEndpointDescriptors[ep].DeviceDescBank[1].ADDR =
			sam.RegValue(uintptr(unsafe.Pointer(&udd_ep_in_cache_buffer[ep])))

		// set endpoint type
		setEPCFG(ep, getEPCFG(ep)|((usb_ENDPOINT_TYPE_CONTROL+1)<<sam.USB_DEVICE_EPCFG_EPTYPE1_Pos))

		// Prepare OUT endpoint for receive
		// set multi packet size for expected number of receive bytes on control OUT
		usbEndpointDescriptors[ep].DeviceDescBank[0].PCKSIZE |=
			sam.RegValue(64 << usb_DEVICE_PCKSIZE_MULTI_PACKET_SIZE_Pos)

		// set byte count to zero, we have not received anything yet
		usbEndpointDescriptors[ep].DeviceDescBank[0].PCKSIZE &^=
			sam.RegValue(usb_DEVICE_PCKSIZE_BYTE_COUNT_Mask << usb_DEVICE_PCKSIZE_BYTE_COUNT_Pos)

		// NAK on endpoint OUT to show we are ready to receive control data
		setEPSTATUSSET(ep, sam.USB_DEVICE_EPSTATUSSET_BK0RDY)
	}
}

func handleStandardSetup(setup usbSetup) bool {
	switch setup.bRequest {
	case usb_GET_STATUS:
		buf := []byte{0, 0}

		if setup.bmRequestType != 0 { // endpoint
			// TODO: actually check if the endpoint in question is currently halted
			if isEndpointHalt {
				buf[0] = 1
			}
		}

		sendUSBPacket(0, buf)
		return true

	case usb_CLEAR_FEATURE:
		if setup.wValueL == 1 { // DEVICEREMOTEWAKEUP
			isRemoteWakeUpEnabled = false
		} else if setup.wValueL == 0 { // ENDPOINTHALT
			isEndpointHalt = false
		}
		sendZlp(0)
		return true

	case usb_SET_FEATURE:
		if setup.wValueL == 1 { // DEVICEREMOTEWAKEUP
			isRemoteWakeUpEnabled = true
		} else if setup.wValueL == 0 { // ENDPOINTHALT
			isEndpointHalt = true
		}
		sendZlp(0)
		return true

	case usb_SET_ADDRESS:
		// set packet size 64 with auto Zlp after transfer
		usbEndpointDescriptors[0].DeviceDescBank[1].PCKSIZE =
			sam.RegValue(epPacketSize(64)<<usb_DEVICE_PCKSIZE_SIZE_Pos) |
				sam.RegValue(1<<31) // autozlp

		// ack the transfer is complete from the request
		setEPINTFLAG(0, sam.USB_DEVICE_EPINTFLAG_TRCPT1)

		// set bank ready for data
		setEPSTATUSSET(0, sam.USB_DEVICE_EPSTATUSSET_BK1RDY)

		// wait for transfer to complete
		for (getEPINTFLAG(0) & sam.USB_DEVICE_EPINTFLAG_TRCPT1) == 0 {
		}

		// last, set the device address to that requested by host
		sam.USB_DEVICE.DADD |= sam.RegValue8(setup.wValueL)
		sam.USB_DEVICE.DADD |= sam.USB_DEVICE_DADD_ADDEN

		return true

	case usb_GET_DESCRIPTOR:
		sendDescriptor(setup)
		return true

	case usb_SET_DESCRIPTOR:
		return false

	case usb_GET_CONFIGURATION:
		buff := []byte{usbConfiguration}
		sendUSBPacket(0, buff)
		return true

	case usb_SET_CONFIGURATION:
		if setup.bmRequestType&usb_REQUEST_RECIPIENT == usb_REQUEST_DEVICE {
			for i := 1; i < len(endPoints); i++ {
				initEndpoint(uint32(i), endPoints[i])
			}

			usbConfiguration = setup.wValueL

			// Enable interrupt for CDC control messages from host (OUT packet)
			setEPINTENSET(usb_CDC_ENDPOINT_ACM, sam.USB_DEVICE_EPINTENSET_TRCPT1)

			// Enable interrupt for CDC data messages from host
			setEPINTENSET(usb_CDC_ENDPOINT_OUT, sam.USB_DEVICE_EPINTENSET_TRCPT0)

			sendZlp(0)
			return true
		} else {
			return false
		}

	case usb_GET_INTERFACE:
		buff := []byte{usbSetInterface}
		sendUSBPacket(0, buff)
		return true

	case usb_SET_INTERFACE:
		usbSetInterface = setup.wValueL

		sendZlp(0)
		return true

	default:
		return true
	}
}

func cdcSetup(setup usbSetup) bool {
	if setup.bmRequestType == usb_REQUEST_DEVICETOHOST_CLASS_INTERFACE {
		if setup.bRequest == usb_CDC_GET_LINE_CODING {
			buf := bytes.NewBuffer(make([]byte, 0, 7))
			binary.Write(buf, binary.LittleEndian, usbLineInfo.dwDTERate)
			binary.Write(buf, binary.LittleEndian, usbLineInfo.bCharFormat)
			binary.Write(buf, binary.LittleEndian, usbLineInfo.bParityType)
			binary.Write(buf, binary.LittleEndian, usbLineInfo.bDataBits)

			sendUSBPacket(0, buf.Bytes())
			return true
		}
	}

	if setup.bmRequestType == usb_REQUEST_HOSTTODEVICE_CLASS_INTERFACE {
		if setup.bRequest == usb_CDC_SET_LINE_CODING {
			buf := bytes.NewBuffer(receiveUSBControlPacket())
			binary.Read(buf, binary.LittleEndian, &(usbLineInfo.dwDTERate))
			binary.Read(buf, binary.LittleEndian, &(usbLineInfo.bCharFormat))
			binary.Read(buf, binary.LittleEndian, &(usbLineInfo.bParityType))
			binary.Read(buf, binary.LittleEndian, &(usbLineInfo.bDataBits))
		}

		if setup.bRequest == usb_CDC_SET_CONTROL_LINE_STATE {
			usbLineInfo.lineState = setup.wValueL
		}

		if setup.bRequest == usb_CDC_SET_LINE_CODING || setup.bRequest == usb_CDC_SET_CONTROL_LINE_STATE {
			// auto-reset into the bootloader
			if usbLineInfo.dwDTERate == 1200 && (usbLineInfo.lineState&0x01) == 0 {
				// TODO: system reset
			} else {
				// TODO: cancel any reset
			}
		}

		if setup.bRequest == usb_CDC_SEND_BREAK {
			// TODO: something with this value?
			// breakValue = ((uint16_t)setup.wValueH << 8) | setup.wValueL;
			// return false;
		}
		return true
	}
	return false
}

func sendUSBPacket(ep uint32, data []byte) {
	copy(udd_ep_in_cache_buffer[ep][:], data)

	// Set endpoint address for sending data
	usbEndpointDescriptors[ep].DeviceDescBank[1].ADDR =
		sam.RegValue(uintptr(unsafe.Pointer(&udd_ep_in_cache_buffer[ep])))

	// clear multi-packet size which is total bytes already sent
	usbEndpointDescriptors[ep].DeviceDescBank[1].PCKSIZE &^=
		sam.RegValue(usb_DEVICE_PCKSIZE_MULTI_PACKET_SIZE_Mask << usb_DEVICE_PCKSIZE_MULTI_PACKET_SIZE_Pos)

	// set byte count, which is total number of bytes to be sent
	usbEndpointDescriptors[ep].DeviceDescBank[1].PCKSIZE |=
		sam.RegValue((len(data) & usb_DEVICE_PCKSIZE_BYTE_COUNT_Mask) << usb_DEVICE_PCKSIZE_BYTE_COUNT_Pos)
}

func receiveUSBControlPacket() []byte {
	// set ready to receive data
	setEPSTATUSCLR(0, sam.USB_DEVICE_EPSTATUSCLR_BK0RDY)

	// read the data
	bytesread := armRecvCtrlOUT(0)

	// return the data
	data := make([]byte, 0, bytesread)
	copy(data, udd_ep_out_cache_buffer[0][:bytesread])
	return data
}

func armRecvCtrlOUT(ep uint32) uint32 {
	// Set output address to receive data
	usbEndpointDescriptors[ep].DeviceDescBank[0].ADDR =
		sam.RegValue(uintptr(unsafe.Pointer(&udd_ep_out_cache_buffer[ep])))

	// set multi-packet size which is total expected number of bytes to receive.
	usbEndpointDescriptors[ep].DeviceDescBank[0].PCKSIZE |=
		sam.RegValue(8<<usb_DEVICE_PCKSIZE_MULTI_PACKET_SIZE_Pos) |
			sam.RegValue(epPacketSize(64)<<usb_DEVICE_PCKSIZE_SIZE_Pos)

	// clear byte count of bytes received so far.
	usbEndpointDescriptors[ep].DeviceDescBank[0].PCKSIZE &^=
		sam.RegValue(usb_DEVICE_PCKSIZE_BYTE_COUNT_Mask << usb_DEVICE_PCKSIZE_BYTE_COUNT_Pos)

	// clear ready state to start transfer
	setEPSTATUSCLR(ep, sam.USB_DEVICE_EPSTATUSCLR_BK0RDY)

	// Wait until OUT transfer is ready.
	for (getEPSTATUS(ep) & sam.USB_DEVICE_EPSTATUS_BK0RDY) == 0 {
	}

	// Wait until OUT transfer is completed.
	for (getEPINTFLAG(ep) & sam.USB_DEVICE_EPINTFLAG_TRCPT0) == 0 {
	}

	// return number of bytes received
	return uint32((usbEndpointDescriptors[ep].DeviceDescBank[0].PCKSIZE >>
		usb_DEVICE_PCKSIZE_BYTE_COUNT_Pos) & usb_DEVICE_PCKSIZE_BYTE_COUNT_Mask)
}

// sendDescriptor creates and sends the various USB descriptor types that
// can be requested by the host.
func sendDescriptor(setup usbSetup) {
	switch setup.wValueH {
	case usb_CONFIGURATION_DESCRIPTOR_TYPE:
		sendConfiguration(setup)
		return
	case usb_DEVICE_DESCRIPTOR_TYPE:
		if setup.wLength == 8 {
			// composite descriptor requested, so only send 8 bytes
			dd := NewDeviceDescriptor(0xEF, 0x02, 0x01, 64, usb_VID, usb_PID, 0x100, usb_IMANUFACTURER, usb_IPRODUCT, usb_ISERIAL, 1)
			sendUSBPacket(0, dd.Bytes()[:8])
		} else {
			// complete descriptor requested so send entire packet
			dd := NewDeviceDescriptor(0x00, 0x00, 0x00, 64, usb_VID, usb_PID, 0x100, usb_IMANUFACTURER, usb_IPRODUCT, usb_ISERIAL, 1)
			sendUSBPacket(0, dd.Bytes())
		}
		return

	case usb_STRING_DESCRIPTOR_TYPE:
		switch setup.wValueL {
		case 0:
			b := make([]byte, 4)
			b[0] = byte(usb_STRING_LANGUAGE[0] >> 8)
			b[1] = byte(usb_STRING_LANGUAGE[0] & 0xff)
			b[2] = byte(usb_STRING_LANGUAGE[1] >> 8)
			b[3] = byte(usb_STRING_LANGUAGE[1] & 0xff)
			sendUSBPacket(0, b)

		case usb_IPRODUCT:
			prod := []byte(usb_STRING_PRODUCT)
			b := make([]byte, len(prod)*2+2)
			b[0] = byte(len(prod)*2 + 2)
			b[1] = 0x03

			for i, val := range prod {
				b[i*2] = 0
				b[i*2+1] = val
			}

			sendUSBPacket(0, b)

		case usb_IMANUFACTURER:
			prod := []byte(usb_STRING_MANUFACTURER)
			b := make([]byte, len(prod)*2+2)
			b[0] = byte(len(prod)*2 + 2)
			b[1] = 0x03

			for i, val := range prod {
				b[i*2] = 0
				b[i*2+1] = val
			}

			sendUSBPacket(0, b)

		case usb_ISERIAL:
			// TODO: allow returning a product serial number
			sendZlp(0)
		}

		// send final zero length packet and return
		sendZlp(0)
		return
	}

	// do not know how to handle this message, so return zero
	sendZlp(0)
	return
}

// sendConfiguration creates and sends the configuration packet to the host.
func sendConfiguration(setup usbSetup) {
	if setup.wLength == 9 {
		sz := uint16(configDescriptorSize + cdcSize)
		config := NewConfigDescriptor(sz, 2)
		sendUSBPacket(0, config.Bytes())
	} else {
		iad := NewIADDescriptor(0, 2, usb_CDC_COMMUNICATION_INTERFACE_CLASS, usb_CDC_ABSTRACT_CONTROL_MODEL, 0)

		cif := NewInterfaceDescriptor(usb_CDC_ACM_INTERFACE, 1, usb_CDC_COMMUNICATION_INTERFACE_CLASS, usb_CDC_ABSTRACT_CONTROL_MODEL, 0)

		header := NewCDCCSInterfaceDescriptor(usb_CDC_HEADER, usb_CDC_V1_10&0xFF, (usb_CDC_V1_10>>8)&0x0FF)

		controlManagement := NewACMFunctionalDescriptor(usb_CDC_ABSTRACT_CONTROL_MANAGEMENT, 6)

		functionalDescriptor := NewCDCCSInterfaceDescriptor(usb_CDC_UNION, usb_CDC_ACM_INTERFACE, usb_CDC_DATA_INTERFACE)

		callManagement := NewCMFunctionalDescriptor(usb_CDC_CALL_MANAGEMENT, 1, 1)

		cifin := NewEndpointDescriptor((usb_CDC_ENDPOINT_ACM | usbEndpointIn), usb_ENDPOINT_TYPE_INTERRUPT, 0x10, 0x10)

		dif := NewInterfaceDescriptor(usb_CDC_DATA_INTERFACE, 2, usb_CDC_DATA_INTERFACE_CLASS, 0, 0)

		in := NewEndpointDescriptor((usb_CDC_ENDPOINT_OUT | usbEndpointOut), usb_ENDPOINT_TYPE_BULK, usbEndpointPacketSize, 0)

		out := NewEndpointDescriptor((usb_CDC_ENDPOINT_IN | usbEndpointIn), usb_ENDPOINT_TYPE_BULK, usbEndpointPacketSize, 0)

		cdc := NewCDCDescriptor(iad,
			cif,
			header,
			controlManagement,
			functionalDescriptor,
			callManagement,
			cifin,
			dif,
			in,
			out)

		sz := uint16(configDescriptorSize + cdcSize)
		config := NewConfigDescriptor(sz, 2)

		buf := make([]byte, 0, sz)
		buf = append(buf, config.Bytes()...)
		buf = append(buf, cdc.Bytes()...)

		sendUSBPacket(0, buf)
	}
}

func handleEndpoint(ep uint32) {
	// get data
	count := int((usbEndpointDescriptors[ep].DeviceDescBank[0].PCKSIZE >>
		usb_DEVICE_PCKSIZE_BYTE_COUNT_Pos) & usb_DEVICE_PCKSIZE_BYTE_COUNT_Mask)

	// move to ring buffer
	for i := 0; i < count; i++ {
		UART0.Receive(byte((udd_ep_out_cache_buffer[ep][i] & 0xFF)))
	}

	// set ready for next data
	setEPSTATUSCLR(ep, sam.USB_DEVICE_EPSTATUSCLR_BK0RDY)

}

func sendZlp(ep uint32) {
	usbEndpointDescriptors[ep].DeviceDescBank[1].PCKSIZE &^=
		sam.RegValue(usb_DEVICE_PCKSIZE_BYTE_COUNT_Mask << usb_DEVICE_PCKSIZE_BYTE_COUNT_Pos)
}

func epPacketSize(size uint16) uint32 {
	switch size {
	case 8:
		return 0
	case 16:
		return 1
	case 32:
		return 2
	case 64:
		return 3
	case 128:
		return 4
	case 256:
		return 5
	case 512:
		return 6
	case 1023:
		return 7
	default:
		return 0
	}
}

func getEPCFG(ep uint32) sam.RegValue8 {
	switch ep {
	case 0:
		return sam.USB_DEVICE.EPCFG0
	case 1:
		return sam.USB_DEVICE.EPCFG1
	case 2:
		return sam.USB_DEVICE.EPCFG2
	case 3:
		return sam.USB_DEVICE.EPCFG3
	case 4:
		return sam.USB_DEVICE.EPCFG4
	case 5:
		return sam.USB_DEVICE.EPCFG5
	case 6:
		return sam.USB_DEVICE.EPCFG6
	case 7:
		return sam.USB_DEVICE.EPCFG7
	default:
		return 0
	}
}

func setEPCFG(ep uint32, val sam.RegValue8) {
	switch ep {
	case 0:
		sam.USB_DEVICE.EPCFG0 = val
	case 1:
		sam.USB_DEVICE.EPCFG1 = val
	case 2:
		sam.USB_DEVICE.EPCFG2 = val
	case 3:
		sam.USB_DEVICE.EPCFG3 = val
	case 4:
		sam.USB_DEVICE.EPCFG4 = val
	case 5:
		sam.USB_DEVICE.EPCFG5 = val
	case 6:
		sam.USB_DEVICE.EPCFG6 = val
	case 7:
		sam.USB_DEVICE.EPCFG7 = val
	default:
		return
	}
}

func setEPSTATUSCLR(ep uint32, val sam.RegValue8) {
	switch ep {
	case 0:
		sam.USB_DEVICE.EPSTATUSCLR0 = val
	case 1:
		sam.USB_DEVICE.EPSTATUSCLR1 = val
	case 2:
		sam.USB_DEVICE.EPSTATUSCLR2 = val
	case 3:
		sam.USB_DEVICE.EPSTATUSCLR3 = val
	case 4:
		sam.USB_DEVICE.EPSTATUSCLR4 = val
	case 5:
		sam.USB_DEVICE.EPSTATUSCLR5 = val
	case 6:
		sam.USB_DEVICE.EPSTATUSCLR6 = val
	case 7:
		sam.USB_DEVICE.EPSTATUSCLR7 = val
	default:
		return
	}
}

func setEPSTATUSSET(ep uint32, val sam.RegValue8) {
	switch ep {
	case 0:
		sam.USB_DEVICE.EPSTATUSSET0 = val
	case 1:
		sam.USB_DEVICE.EPSTATUSSET1 = val
	case 2:
		sam.USB_DEVICE.EPSTATUSSET2 = val
	case 3:
		sam.USB_DEVICE.EPSTATUSSET3 = val
	case 4:
		sam.USB_DEVICE.EPSTATUSSET4 = val
	case 5:
		sam.USB_DEVICE.EPSTATUSSET5 = val
	case 6:
		sam.USB_DEVICE.EPSTATUSSET6 = val
	case 7:
		sam.USB_DEVICE.EPSTATUSSET7 = val
	default:
		return
	}
}

func getEPSTATUS(ep uint32) sam.RegValue8 {
	switch ep {
	case 0:
		return sam.USB_DEVICE.EPSTATUS0
	case 1:
		return sam.USB_DEVICE.EPSTATUS1
	case 2:
		return sam.USB_DEVICE.EPSTATUS2
	case 3:
		return sam.USB_DEVICE.EPSTATUS3
	case 4:
		return sam.USB_DEVICE.EPSTATUS4
	case 5:
		return sam.USB_DEVICE.EPSTATUS5
	case 6:
		return sam.USB_DEVICE.EPSTATUS6
	case 7:
		return sam.USB_DEVICE.EPSTATUS7
	default:
		return 0
	}
}

func getEPINTFLAG(ep uint32) sam.RegValue8 {
	switch ep {
	case 0:
		return sam.USB_DEVICE.EPINTFLAG0
	case 1:
		return sam.USB_DEVICE.EPINTFLAG1
	case 2:
		return sam.USB_DEVICE.EPINTFLAG2
	case 3:
		return sam.USB_DEVICE.EPINTFLAG3
	case 4:
		return sam.USB_DEVICE.EPINTFLAG4
	case 5:
		return sam.USB_DEVICE.EPINTFLAG5
	case 6:
		return sam.USB_DEVICE.EPINTFLAG6
	case 7:
		return sam.USB_DEVICE.EPINTFLAG7
	default:
		return 0
	}
}

func setEPINTFLAG(ep uint32, val sam.RegValue8) {
	switch ep {
	case 0:
		sam.USB_DEVICE.EPINTFLAG0 = val
	case 1:
		sam.USB_DEVICE.EPINTFLAG1 = val
	case 2:
		sam.USB_DEVICE.EPINTFLAG2 = val
	case 3:
		sam.USB_DEVICE.EPINTFLAG3 = val
	case 4:
		sam.USB_DEVICE.EPINTFLAG4 = val
	case 5:
		sam.USB_DEVICE.EPINTFLAG5 = val
	case 6:
		sam.USB_DEVICE.EPINTFLAG6 = val
	case 7:
		sam.USB_DEVICE.EPINTFLAG7 = val
	default:
		return
	}
}

func setEPINTENCLR(ep uint32, val sam.RegValue8) {
	switch ep {
	case 0:
		sam.USB_DEVICE.EPINTENCLR0 = val
	case 1:
		sam.USB_DEVICE.EPINTENCLR1 = val
	case 2:
		sam.USB_DEVICE.EPINTENCLR2 = val
	case 3:
		sam.USB_DEVICE.EPINTENCLR3 = val
	case 4:
		sam.USB_DEVICE.EPINTENCLR4 = val
	case 5:
		sam.USB_DEVICE.EPINTENCLR5 = val
	case 6:
		sam.USB_DEVICE.EPINTENCLR6 = val
	case 7:
		sam.USB_DEVICE.EPINTENCLR7 = val
	default:
		return
	}
}

func setEPINTENSET(ep uint32, val sam.RegValue8) {
	switch ep {
	case 0:
		sam.USB_DEVICE.EPINTENSET0 = val
	case 1:
		sam.USB_DEVICE.EPINTENSET1 = val
	case 2:
		sam.USB_DEVICE.EPINTENSET2 = val
	case 3:
		sam.USB_DEVICE.EPINTENSET3 = val
	case 4:
		sam.USB_DEVICE.EPINTENSET4 = val
	case 5:
		sam.USB_DEVICE.EPINTENSET5 = val
	case 6:
		sam.USB_DEVICE.EPINTENSET6 = val
	case 7:
		sam.USB_DEVICE.EPINTENSET7 = val
	default:
		return
	}
}
