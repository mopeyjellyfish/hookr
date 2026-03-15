package memory

import (
	"fmt"
	"math"
)

// Uint32FromInt safely converts an int to uint32, checking for negative values and overflow
func Uint32FromInt(v int) (uint32, error) {
	if v < 0 {
		return 0, fmt.Errorf("negative integer: %d cannot be represented as uint32", v)
	}
	if v > math.MaxUint32 {
		return 0, fmt.Errorf("integer overflow: %d cannot be represented as uint32", v)
	}
	return uint32(v), nil
}

// Uint32FromUint64 safely converts a uint64 to uint32, checking for overflow.
func Uint32FromUint64(v uint64) (uint32, error) {
	if v > math.MaxUint32 {
		return 0, fmt.Errorf("integer overflow: %d cannot be represented as uint32", v)
	}
	return uint32(v), nil
}

// Uint16FromUint32 safely converts a uint32 to uint16, checking for overflow.
func Uint16FromUint32(v uint32) (uint16, error) {
	if v > math.MaxUint16 {
		return 0, fmt.Errorf("integer overflow: %d cannot be represented as uint16", v)
	}
	return uint16(v), nil
}
