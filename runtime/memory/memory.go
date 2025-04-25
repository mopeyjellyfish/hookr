package memory

import (
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

// ReadString is a convenience function that casts requireRead
func ReadString(mem api.Memory, fieldName string, offset, byteCount uint32) string {
	return string(Read(mem, fieldName, offset, byteCount))
}

// Read is like api.Memory except that it panics if the offset and byteCount are out of range.
func Read(mem api.Memory, fieldName string, offset, byteCount uint32) []byte {
	buf, ok := mem.Read(offset, byteCount)
	if !ok {
		panic(fmt.Errorf("out of memory reading %s", fieldName))
	}
	return buf
}

// Write is like api.Memory except that it panics if the offset and byteCount are out of range.
func Write(mem api.Memory, fieldName string, offset uint32, data []byte) {
	if !mem.Write(offset, data) {
		panic(fmt.Errorf("out of memory writing %s", fieldName))
	}
}
