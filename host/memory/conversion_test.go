package memory

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUint32FromUint64(t *testing.T) {
	tests := []struct {
		name      string
		input     uint64
		expected  uint32
		expectErr bool
	}{
		{
			name:      "zero value",
			input:     0,
			expected:  0,
			expectErr: false,
		},
		{
			name:      "valid value",
			input:     123456,
			expected:  123456,
			expectErr: false,
		},
		{
			name:      "maximum uint32",
			input:     math.MaxUint32,
			expected:  math.MaxUint32,
			expectErr: false,
		},
		{
			name:      "overflow - just above max",
			input:     uint64(math.MaxUint32) + 1,
			expected:  0,
			expectErr: true,
		},
		{
			name:      "overflow - large value",
			input:     math.MaxUint64,
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Uint32(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "integer overflow")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUint32FromInt(t *testing.T) {
	tests := []struct {
		name      string
		input     int
		expected  uint32
		expectErr bool
		errType   string
	}{
		{
			name:      "zero value",
			input:     0,
			expected:  0,
			expectErr: false,
		},
		{
			name:      "valid positive value",
			input:     123456,
			expected:  123456,
			expectErr: false,
		},
		{
			name:      "maximum valid value",
			input:     math.MaxUint32,
			expected:  math.MaxUint32,
			expectErr: false,
		},
		{
			name:      "negative value",
			input:     -1,
			expected:  0,
			expectErr: true,
			errType:   "negative integer",
		},
		{
			name:      "large negative value",
			input:     -1000000,
			expected:  0,
			expectErr: true,
			errType:   "negative integer",
		},
	}

	// Add overflow test only if int is larger than uint32 on this platform
	if math.MaxInt > math.MaxUint32 {
		tests = append(tests, struct {
			name      string
			input     int
			expected  uint32
			expectErr bool
			errType   string
		}{
			name:      "overflow value",
			input:     math.MaxUint32 + 1,
			expected:  0,
			expectErr: true,
			errType:   "integer overflow",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Uint32FromInt(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errType)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
