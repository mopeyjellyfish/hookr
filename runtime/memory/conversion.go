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
