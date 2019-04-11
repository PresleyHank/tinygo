package loader

import (
	"sync"
)

// RefMap is a convenient way to store opaque references that can be passed to
// C. It is useful if an API uses function pointers and you cannot pass a Go
// pointer but only a C pointer.
type RefMap struct {
	refs      map[uintptr]interface{}
	lastIndex uintptr
	lock      sync.Mutex
}

// Put stores a value in the map. It can later be retrieved using Get. It must
// be removed using Remove to avoid memory leaks.
func (m *RefMap) Put(v interface{}) uintptr {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.refs == nil {
		m.refs = make(map[uintptr]interface{}, 1)
	}
	m.lastIndex++
	if _, ok := m.refs[m.lastIndex]; ok {
		// sanity check
		panic("RefMap overflowed!")
	}
	m.refs[m.lastIndex] = v
	return m.lastIndex
}

// Get returns a stored value previously inserted with Put. Use the same
// reference as you got from Put.
func (m *RefMap) Get(ref uintptr) interface{} {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.refs[ref]
}

// Remove deletes a single reference from the map.
func (m *RefMap) Remove(ref uintptr) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.refs, ref)
}
