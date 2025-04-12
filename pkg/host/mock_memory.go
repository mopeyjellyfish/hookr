package host

import (
	"fmt"
	"sort"
)

// MockMemory is a memory mock implementation of WASM memory interface, allows for manipulation of memory for testing
// Enables testing of the WASM memory interface without needing to load a WASM module
// This is useful for testing things such as calling host functions and checking for error handling etc.
type MockMemory struct {
	memory    map[uint64][]byte
	readErr   error
	writeErr  error
	mallocErr error
}

func NewMockMemory() *MockMemory {
	return &MockMemory{
		memory: make(map[uint64][]byte),
	}
}

// SetReadError allows for setting an error for the next time read is called
func (m *MockMemory) SetReadError(err error) {
	m.readErr = err
}

// SetWriteError allows for setting an error for the next time write is called
func (m *MockMemory) SetWriteError(err error) {
	m.writeErr = err
}

// SetMallocError allows for setting an error for the next time malloc is called
func (m *MockMemory) SetMallocError(err error) {
	m.mallocErr = err
}

// Malloc allocates memory inside of a map for use in testing
func (m *MockMemory) Malloc(size uint64) (uint64, error) {
	if m.mallocErr != nil {
		err := m.mallocErr
		m.mallocErr = nil
		return 0, err
	}

	var keys []uint64
	for k := range m.memory {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	if len(keys) == 0 {
		m.memory[1] = make([]byte, size)
		return 1, nil
	}

	newKey := keys[len(keys)-1] + 1
	m.memory[newKey] = make([]byte, size)
	return newKey, nil
}

// Free will remove the key from the map, this is used to simulate freeing memory
func (m *MockMemory) Free(ptr uint64) {
	delete(m.memory, ptr)
}

func (m *MockMemory) WriteBytes(offset uint64, data []byte) (uint64, error) {
	if m.writeErr != nil {
		err := m.writeErr
		m.writeErr = nil
		return 0, err
	}
	if offset == 0 {
		return 0, fmt.Errorf("error writing output: no offset provided")
	}
	m.memory[offset] = data
	return offset, nil
}

// Write will write the input to the map at the provided pointer
func (m *MockMemory) Write(input []byte) (uint64, error) {
	if m.writeErr != nil {
		err := m.writeErr
		m.writeErr = nil
		return 0, err
	}
	ptr, err := m.Malloc(uint64(len(input)))
	if err != nil {
		return 0, fmt.Errorf("error calling alloc: %v", err)
	}
	m.memory[ptr] = input
	return ptr, nil
}

// Read will read the data from the map at the provided pointer
func (m *MockMemory) Read(ptr uint64) ([]byte, error) {
	if m.readErr != nil {
		err := m.readErr
		m.readErr = nil
		return nil, err
	}
	if ptr == 0 {
		return nil, fmt.Errorf("error reading output: no pointer provided")
	}
	mem, ok := m.memory[ptr]
	if !ok {
		return nil, fmt.Errorf("error reading memory")
	}
	buffer := make([]byte, len(mem))
	copy(buffer, mem)
	m.Free(ptr)
	return buffer, nil
}
