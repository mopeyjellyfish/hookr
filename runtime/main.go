package main

// #include <stdlib.h>
import "C"

import (
	"unsafe"
)

//export hookr_alloc
func hookr_alloc(size uint64) uint64 {
	len := C.ulong(size)
	ptr := unsafe.Pointer(C.malloc(len))
	return uint64(uintptr(ptr))
}

//export hookr_free
func hookr_free(ptr uint64) {
	C.free(unsafe.Pointer(uintptr(ptr)))
}

//export hookr_write_u64
func hookr_write_u64(ptr uint64, value uint64) {
	ptrToMemory := unsafe.Pointer(uintptr(ptr)) // this is the start of the memory to write to

	for i := 0; i < 8; i++ {
		// Calculate the address for the current byte.
		currentBytePtr := (*byte)(unsafe.Pointer(uintptr(ptrToMemory) + uintptr(i)))
		// Assign the ith byte of the uint64 value to the memory location.
		*currentBytePtr = byte(value >> (8 * i))
	}
}

//export hookr_write_u8
func hookr_write_u8(ptr uint64, value uint8) {
	// Convert ptr (uint64) to unsafe.Pointer for manipulation.
	ptrToMemory := unsafe.Pointer(uintptr(ptr))

	// Write the uint8 value to the specified memory location.
	*(*uint8)(ptrToMemory) = value
}

//export hookr_read_u64
func hookr_read_u64(ptr uint64) uint64 {
	// Convert ptr (uint64) to unsafe.Pointer for manipulation.
	ptrToMemory := unsafe.Pointer(uintptr(ptr))

	var value uint64
	// Read bytes from memory and assemble them into a uint64 value.
	for i := 0; i < 8; i++ {
		// Calculate the address for the current byte.
		currentBytePtr := (*byte)(unsafe.Pointer(uintptr(ptrToMemory) + uintptr(i)))
		// Read the ith byte and shift it to its correct position in the uint64 value.
		value |= uint64(*currentBytePtr) << (8 * i)
	}

	return value
}

//export hookr_read_u8
func hookr_read_u8(ptr uint64) uint8 {
	// Convert ptr (uint64) to unsafe.Pointer for manipulation.
	ptrToMemory := unsafe.Pointer(uintptr(ptr))

	// Read and return the uint8 value from the specified memory location.
	return *(*uint8)(ptrToMemory)
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
