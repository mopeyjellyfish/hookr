package memory

import (
	"encoding/binary"
	"encoding/json"
)

//go:wasmimport hookr hookr_alloc
func hookrAlloc(size uint64) uint64

//go:wasmimport hookr hookr_free
func hookrFree(ptr uint64)

//go:wasmimport hookr hookr_write_u64
func hookrWriteU64(ptr uint64, value uint64)

//go:wasmimport hookr hookr_write_u8
func hookrWriteU8_(uint64, uint32)
func hookrWriteU8(p uint64, v uint8) {
	hookrWriteU8_(p, uint32(v))
}

//go:wasmimport hookr hookr_read_u64
func hookrReadU64(ptr uint64) uint64

//go:wasmimport hookr hookr_read_u8
func hookrReadU8_(uint64) uint32
func hookrReadU8(p uint64) uint8 {
	return uint8(hookrReadU8_(p))
}

func Free(ptr uint32) {
	hookrFree(uint64(ptr))
}

// Write writes the contents of buf to the memory at the given offset.
// The caller is expected to allocate the memory before calling this function.
func Write(offset uint64, buf []byte) {
	length := len(buf)
	chunkCount := length >> 3

	for chunkIdx := 0; chunkIdx < chunkCount; chunkIdx++ {
		i := chunkIdx << 3
		x := binary.LittleEndian.Uint64(buf[i : i+8])
		hookrWriteU64(offset+uint64(i), x)
	}

	remainder := length & 7
	remainderOffset := chunkCount << 3
	for index := remainderOffset; index < (remainder + remainderOffset); index++ {
		hookrWriteU8(offset+uint64(index), buf[index])
	}
}

// Read reads the contents of the memory at the given offset into buf.
func Read(offset uint64, buf []byte) {
	length := len(buf)
	chunkCount := length >> 3

	for chunkIdx := 0; chunkIdx < chunkCount; chunkIdx++ {
		i := chunkIdx << 3
		binary.LittleEndian.PutUint64(buf[i:i+8], hookrReadU64(offset+uint64(i)))
	}

	remainder := length & 7
	remainderOffset := chunkCount << 3
	for index := remainderOffset; index < (remainder + remainderOffset); index++ {
		buf[index] = hookrReadU8(offset + uint64(index))
	}
}

// WriteString writes the contents of s to the memory and returns a pointer to the memory.
// The caller is expected to free the memory after it is done using it.
// If this pointer is passed back to the host, the host is expected to free the memory.
func WriteString(s string) uint64 {
	size := len(s)
	ptr := hookrAlloc(uint64(size))
	Write(ptr, []byte(s))
	return (uint64(ptr) << uint64(32)) | uint64(size)
}

// ReadString reads the contents of the memory at the given pointer and returns it as a string.
// The host will be responsible for freeing the memory.
func ReadString(ptr uint64) string {
	offset := uint32(ptr >> 32)
	size := uint32(ptr)
	buf := make([]byte, size)
	Read(uint64(offset), buf)
	return string(buf)
}

// WriteBytes writes the contents of data to the memory and returns a pointer to the memory.
// The caller is expected to free the memory after it is done using it.
// If this pointer is passed back to the host, the host is expected to free the memory.
func WriteBytes(data []byte) uint64 {
	size := len(data)
	ptr := hookrAlloc(uint64(size))
	Write(ptr, data)
	return (uint64(ptr) << uint64(32)) | uint64(size)
}

// ReadBytes reads the contents of the memory at the given pointer and returns it as a byte slice.
// The host will be responsible for freeing the memory.
func ReadBytes(ptr uint64) []byte {
	offset := uint32(ptr >> 32)
	size := uint32(ptr)
	buf := make([]byte, size)
	Read(uint64(offset), buf)
	Free(offset)
	return buf
}

// WriteJson writes the contents of data to the memory and returns a pointer to the memory.
// The caller is expected to free the memory after it is done using it.
func WriteJson(data interface{}) uint64 {
	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return WriteBytes(bytes)
}

// ReadJson reads the contents of the memory at the given pointer and unmarshals it into v.
func ReadJson(ptr uint64, v interface{}) error {
	bytes := ReadBytes(ptr)
	if err := json.Unmarshal(bytes, v); err != nil {
		return err
	}
	return nil
}
