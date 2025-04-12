package host

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

type Memory interface {
	Malloc(size uint64) (uint64, error)
	Free(ptr uint64)
	Write(input []byte) (uint64, error)
	WriteBytes(offset uint64, data []byte) (uint64, error)
	Read(ptr uint64) ([]byte, error)
}

// WasmMemory Memory object is used to read and write memory from the hookr module for I/O
// Provides some higher level helpers around allocating, freeing,
type WasmMemory struct {
	memory   api.Memory
	freeFn   api.Function
	mallocFn api.Function
	ctx      context.Context
}

// Malloc allocates memory inside of the hookr module for use by the loaded main module
// Returns a uint64 that is the pointer to the starting location of the allocated memory
//
// Example:
//
// offset, err := mem.Malloc(uint64(size))
//
//	if err != nil {
//		return 0, fmt.Errorf("error calling alloc: %v", err)
//	}
func (m *WasmMemory) Malloc(size uint64) (uint64, error) {
	ptr, err := m.mallocFn.Call(m.ctx, size)
	if err != nil {
		return 0, fmt.Errorf("error calling alloc: %v", err)
	}
	return ptr[0], nil
}

// Free frees the memory from the main module
// Example:
//
// offset, err := mem.Malloc(uint64(size))
//
//	if err != nil {
//		return 0, fmt.Errorf("error calling alloc: %v", err)
//	}
//	p.Free(offset)
func (m *WasmMemory) Free(ptr uint64) {
	_, _ = m.freeFn.Call(m.ctx, ptr)
}

// WriteBytes writes data to a specific offset in the memory
// Returns the offset and size bit shifted into a single pointer
func (m *WasmMemory) WriteBytes(offset uint64, data []byte) (uint64, error) {
	size := len(data)
	ok := m.memory.Write(uint32(offset), data)
	if !ok {
		return 0, fmt.Errorf("error writing input to memory")
	}
	return (uint64(offset) << uint64(32)) | uint64(size), nil
}

// Write writes the input to be read by the main module
// Returns a uint64 that is the 32bit pointer to the starting location and a 32bit number of bytes written packed into a single uint64
// This uint64 can be passed to the main module to be used to read an input from hookr's module inside the wasm runtime
//
// Example:
//
//	ptr, err := p.WriteMem([]byte("Hello, World!"))
//	offset := uint32(ptr >> 32) // offset is the 32bit pointer to the starting location
//	size := uint32(ptr) // size is the 32bit number of bytes written
func (m *WasmMemory) Write(input []byte) (uint64, error) {
	size := len(input)
	if size == 0 {
		return 0, fmt.Errorf("error writing input: no input provided")
	}
	offset, err := m.Malloc(uint64(size))
	if err != nil {
		return 0, fmt.Errorf("error calling alloc: %v", err)
	}
	return m.WriteBytes(offset, input)
}

// Read reads the output from the main module
// Returns the output and an error if there is an issue reading the output
// Example:
//
//	ptr, err := p.WriteInput([]byte("Hello, World!"))
//	output, err := p.ReadMem(ptr)
func (m *WasmMemory) Read(ptr uint64) ([]byte, error) {
	if ptr == 0 {
		return nil, fmt.Errorf("error reading output: no pointer provided")
	}
	offset := uint32(ptr >> 32)
	len := uint32(ptr)
	mem, ok := m.memory.Read(offset, len)
	if !ok {
		return nil, fmt.Errorf("error reading memory")
	}
	buffer := make([]byte, len)
	copy(buffer, mem)
	m.Free(uint64(offset))
	return buffer, nil
}

// NewMemory creates a new memory object to be used to read and write memory from the hookr module for I/O
func NewMemory(ctx context.Context, memory api.Memory, freeFn api.Function, mallocFn api.Function) Memory {
	return &WasmMemory{
		memory:   memory,
		freeFn:   freeFn,
		mallocFn: mallocFn,
		ctx:      ctx,
	}
}
