
# aliases
all: tinygo
tinygo: build/tinygo

.PHONY: all tinygo llvm-build llvm-source static run-test run-blinky run-blinky2 clean fmt gen-device gen-device-nrf gen-device-avr

TARGET ?= unix

ifeq ($(TARGET),unix)
# Regular *nix system.

else ifeq ($(TARGET),pca10040)
# PCA10040: nRF52832 development board
OBJCOPY = arm-none-eabi-objcopy
TGOFLAGS += -target $(TARGET)

else ifeq ($(TARGET),microbit)
# BBC micro:bit
OBJCOPY = arm-none-eabi-objcopy
TGOFLAGS += -target $(TARGET)

else ifeq ($(TARGET),reelboard)
# reel board
OBJCOPY = arm-none-eabi-objcopy
TGOFLAGS += -target $(TARGET)

else ifeq ($(TARGET),bluepill)
# "blue pill" development board
# See: https://wiki.stm32duino.com/index.php?title=Blue_Pill
OBJCOPY = arm-none-eabi-objcopy
TGOFLAGS += -target $(TARGET)

else ifeq ($(TARGET),arduino)
OBJCOPY = avr-objcopy
TGOFLAGS += -target $(TARGET)

else
$(error Unknown target)

endif

# Default build and source directories, as created by `make llvm-build`.
LLVM_BUILDDIR ?= llvm-build
CLANG_SRC ?= llvm/tools/clang
LLD_SRC ?= llvm/tools/lld

LLVM_COMPONENTS = all-targets analysis asmparser asmprinter bitreader bitwriter codegen core coroutines debuginfodwarf executionengine instrumentation interpreter ipo irreader linker lto mc mcjit objcarcopts option profiledata scalaropts support target

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    START_GROUP = -Wl,--start-group
    END_GROUP = -Wl,--end-group
endif

CLANG_LIBS = $(START_GROUP) $(abspath $(LLVM_BUILDDIR))/lib/libclang.a -lclangAnalysis -lclangARCMigrate -lclangAST -lclangASTMatchers -lclangBasic -lclangCodeGen -lclangCrossTU -lclangDriver -lclangDynamicASTMatchers -lclangEdit -lclangFormat -lclangFrontend -lclangFrontendTool -lclangHandleCXX -lclangHandleLLVM -lclangIndex -lclangLex -lclangParse -lclangRewrite -lclangRewriteFrontend -lclangSema -lclangSerialization -lclangStaticAnalyzerCheckers -lclangStaticAnalyzerCore -lclangStaticAnalyzerFrontend -lclangTooling -lclangToolingASTDiff -lclangToolingCore -lclangToolingInclusions -lclangToolingRefactor $(END_GROUP) -lstdc++

LLD_LIBS = $(START_GROUP) -llldCOFF -llldCommon -llldCore -llldDriver -llldELF -llldMachO -llldMinGW -llldReaderWriter -llldWasm -llldYAML $(END_GROUP)


# For static linking.
CGO_CPPFLAGS=$(shell $(LLVM_BUILDDIR)/bin/llvm-config --cppflags) -I$(abspath $(CLANG_SRC))/include -I$(abspath $(LLD_SRC))/include
CGO_CXXFLAGS=-std=c++11
CGO_LDFLAGS=-L$(LLVM_BUILDDIR)/lib $(CLANG_LIBS) $(LLD_LIBS) $(shell $(LLVM_BUILDDIR)/bin/llvm-config --ldflags --libs --system-libs $(LLVM_COMPONENTS))



run-test: build/test
	./build/test

run-blinky: run-blinky2
run-blinky2: build/blinky2
	./build/blinky2

ifeq ($(TARGET),pca10040)
flash-%: build/%.hex
	nrfjprog -f nrf52 --sectorerase --program $< --reset
else ifeq ($(TARGET),microbit)
flash-%: build/%.hex
	openocd -f interface/cmsis-dap.cfg -f target/nrf51.cfg -c 'program $< reset exit'
else ifeq ($(TARGET),reelboard)
flash-%: build/%.hex
	openocd -f interface/cmsis-dap.cfg -f target/nrf51.cfg -c 'program $< reset exit'
else ifeq ($(TARGET),arduino)
flash-%: build/%.hex
	avrdude -c arduino -p atmega328p -P /dev/ttyACM0 -U flash:w:$<
else ifeq ($(TARGET),bluepill)
flash-%: build/%.hex
	openocd -f interface/stlink-v2.cfg -f target/stm32f1x.cfg -c 'program $< reset exit'
endif

clean:
	@rm -rf build

fmt:
	@go fmt . ./compiler ./interp ./loader ./ir ./src/device/arm ./src/examples/* ./src/machine ./src/os ./src/runtime ./src/sync ./src/syscall
	@go fmt ./testdata/*.go

test: build/lib/clang/include/stdint.h
	@go test -v .

gen-device: gen-device-avr gen-device-nrf gen-device-sam gen-device-stm32

gen-device-avr:
	./tools/gen-device-avr.py lib/avr/packs/atmega src/device/avr/
	./tools/gen-device-avr.py lib/avr/packs/tiny src/device/avr/
	go fmt ./src/device/avr

gen-device-nrf:
	./tools/gen-device-svd.py lib/nrfx/mdk/ src/device/nrf/ --source=https://github.com/NordicSemiconductor/nrfx/tree/master/mdk
	go fmt ./src/device/nrf

gen-device-sam:
	./tools/gen-device-svd.py lib/cmsis-svd/data/Atmel/ src/device/sam/ --source=https://github.com/posborne/cmsis-svd/tree/master/data/Atmel
	go fmt ./src/device/sam

gen-device-stm32:
	./tools/gen-device-svd.py lib/cmsis-svd/data/STMicro/ src/device/stm32/ --source=https://github.com/posborne/cmsis-svd/tree/master/data/STMicro
	go fmt ./src/device/stm32


# Get LLVM sources.
llvm/README.txt:
	git clone -b release_80 https://github.com/llvm-mirror/llvm.git llvm
llvm/tools/clang/README.txt:
	git clone -b release_80 https://github.com/llvm-mirror/clang.git llvm/tools/clang
llvm/tools/lld/README.md:
	git clone -b release_80 https://github.com/llvm-mirror/lld.git llvm/tools/lld
llvm-source: llvm/README.txt llvm/tools/clang/README.txt llvm/tools/lld/README.md

# Configure LLVM.
llvm-build/build.ninja: llvm-source
	mkdir -p llvm-build; cd llvm-build; cmake -G Ninja ../llvm "-DLLVM_TARGETS_TO_BUILD=X86;ARM;AArch64;WebAssembly" "-DLLVM_EXPERIMENTAL_TARGETS_TO_BUILD=AVR" -DCMAKE_BUILD_TYPE=Release -DLLVM_ENABLE_ASSERTIONS=OFF -DLIBCLANG_BUILD_STATIC=ON

# Build LLVM.
llvm-build: llvm-build/build.ninja
	cd llvm-build; ninja


# Build the Go compiler.
build/tinygo:
	@mkdir -p build
	go build -o build/tinygo .

build/lib/clang/include/stdint.h:
	@mkdir -p build/lib/clang/include
	cp -p $(abspath $(CLANG_SRC))/lib/Headers/*.h build/lib/clang/include

static:
	CGO_CPPFLAGS="$(CGO_CPPFLAGS)" CGO_CXXFLAGS="$(CGO_CXXFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o build/tinygo -tags byollvm .

static-test: build/lib/clang/include/stdint.h
	CGO_CPPFLAGS="$(CGO_CPPFLAGS)" CGO_CXXFLAGS="$(CGO_CXXFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go test -v -tags byollvm .

release: static build/lib/clang/include/stdint.h gen-device
	@mkdir -p build/lib/CMSIS/CMSIS
	@mkdir -p build/lib/compiler-rt/lib
	@mkdir -p build/lib/nrfx
	@mkdir -p build/pkg/armv6m-none-eabi
	@mkdir -p build/pkg/armv7m-none-eabi
	@mkdir -p build/pkg/armv7em-none-eabi
	@echo copying source files
	@cp -rp lib/CMSIS/CMSIS/Include      build/lib/CMSIS/CMSIS
	@cp -rp lib/CMSIS/README.md          build/lib/CMSIS
	@cp -rp lib/compiler-rt/lib/builtins build/lib/compiler-rt/lib
	@cp -rp lib/compiler-rt/LICENSE.TXT  build/lib/compiler-rt
	@cp -rp lib/compiler-rt/README.txt   build/lib/compiler-rt
	@cp -rp lib/nrfx/*                   build/lib/nrfx
	@cp -rp src                          build/src
	@cp -rp targets                      build/targets
	./build/tinygo build-builtins -target=armv6m-none-eabi  -o build/pkg/armv6m-none-eabi/compiler-rt.a
	./build/tinygo build-builtins -target=armv7m-none-eabi  -o build/pkg/armv7m-none-eabi/compiler-rt.a
	./build/tinygo build-builtins -target=armv7em-none-eabi -o build/pkg/armv7em-none-eabi/compiler-rt.a
	tar --transform s/^build/tinygo/ -czf release.tar.gz build

# Binary that can run on the host.
build/%: src/examples/% src/examples/%/*.go build/tinygo src/runtime/*.go
	./build/tinygo build $(TGOFLAGS) -size=short -o $@ $(subst src/,,$<)

# ELF file that can run on a microcontroller.
build/%.elf: src/examples/% src/examples/%/*.go build/tinygo src/runtime/*.go
	./build/tinygo build $(TGOFLAGS) -size=short -o $@ $(subst src/,,$<)

# Convert executable to Intel hex file (for flashing).
build/%.hex: build/%.elf
	$(OBJCOPY) -O ihex $^ $@
