package memory

import (
	"fmt"
)

type Memory interface {
	Read(offset, byteCount uint32) ([]byte, bool)
	Write(offset uint32, data []byte) bool
}

// ReadString is a convenience function that casts requireRead
func ReadString(mem Memory, fieldName string, offset, byteCount uint32) string {
	return string(Read(mem, fieldName, offset, byteCount))
}

// TryReadString is a non-panicking variant of ReadString.
func TryReadString(mem Memory, fieldName string, offset, byteCount uint32) (string, error) {
	buf, err := TryRead(mem, fieldName, offset, byteCount)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// Read is like api.Memory except that it panics if the offset and byteCount are out of range.
func Read(mem Memory, fieldName string, offset, byteCount uint32) []byte {
	buf, ok := mem.Read(offset, byteCount)
	if !ok {
		panic(fmt.Errorf("out of memory reading %s", fieldName))
	}
	return buf
}

// TryRead is like api.Memory except that it returns an error when the read is out of range.
func TryRead(mem Memory, fieldName string, offset, byteCount uint32) ([]byte, error) {
	buf, ok := mem.Read(offset, byteCount)
	if !ok {
		return nil, fmt.Errorf("out of memory reading %s", fieldName)
	}
	return buf, nil
}

// Write is like api.Memory except that it panics if the offset and byteCount are out of range.
func Write(mem Memory, fieldName string, offset uint32, data []byte) {
	if !mem.Write(offset, data) {
		panic(fmt.Errorf("out of memory writing %s", fieldName))
	}
}

// TryWrite is like api.Memory except that it returns an error when the write is out of range.
func TryWrite(mem Memory, fieldName string, offset uint32, data []byte) error {
	if !mem.Write(offset, data) {
		return fmt.Errorf("out of memory writing %s", fieldName)
	}
	return nil
}
