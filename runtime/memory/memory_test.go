package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMemory implements the Memory interface for testing
type MockMemory struct {
	Data       []byte
	ShouldFail bool
}

func (m *MockMemory) Read(offset, byteCount uint32) ([]byte, bool) {
	if m.ShouldFail {
		return nil, false
	}

	if int(offset+byteCount) > len(m.Data) {
		return nil, false
	}

	return m.Data[offset : offset+byteCount], true
}

func (m *MockMemory) Write(offset uint32, data []byte) bool {
	if m.ShouldFail {
		return false
	}

	requiredSize := int(offset) + len(data)
	if requiredSize > len(m.Data) {
		return false
	}

	copy(m.Data[offset:], data)
	return true
}

func TestRead(t *testing.T) {
	tests := []struct {
		name        string
		memory      *MockMemory
		fieldName   string
		offset      uint32
		byteCount   uint32
		shouldPanic bool
		expected    []byte
	}{
		{
			name:        "successful read",
			memory:      &MockMemory{Data: []byte("hello world")},
			fieldName:   "test field",
			offset:      0,
			byteCount:   5,
			shouldPanic: false,
			expected:    []byte("hello"),
		},
		{
			name:        "successful read with offset",
			memory:      &MockMemory{Data: []byte("hello world")},
			fieldName:   "test field",
			offset:      6,
			byteCount:   5,
			shouldPanic: false,
			expected:    []byte("world"),
		},
		{
			name:        "out of bounds read",
			memory:      &MockMemory{Data: []byte("hello")},
			fieldName:   "test field",
			offset:      10,
			byteCount:   5,
			shouldPanic: true,
			expected:    nil,
		},
		{
			name:        "read fails",
			memory:      &MockMemory{Data: []byte("hello"), ShouldFail: true},
			fieldName:   "test field",
			offset:      0,
			byteCount:   5,
			shouldPanic: true,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.Panics(t, func() {
					Read(tt.memory, tt.fieldName, tt.offset, tt.byteCount)
				}, "Read should panic on out of bounds or failure")
			} else {
				result := Read(tt.memory, tt.fieldName, tt.offset, tt.byteCount)
				assert.Equal(t, tt.expected, result, "Read should return the correct data")
			}
		})
	}
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name        string
		memory      *MockMemory
		fieldName   string
		offset      uint32
		data        []byte
		shouldPanic bool
		expected    []byte
	}{
		{
			name:        "successful write",
			memory:      &MockMemory{Data: make([]byte, 10)},
			fieldName:   "test field",
			offset:      0,
			data:        []byte("hello"),
			shouldPanic: false,
			expected:    []byte("hello\x00\x00\x00\x00\x00"),
		},
		{
			name:        "successful write with offset",
			memory:      &MockMemory{Data: make([]byte, 10)},
			fieldName:   "test field",
			offset:      5,
			data:        []byte("hello"),
			shouldPanic: false,
			expected:    []byte("\x00\x00\x00\x00\x00hello"),
		},
		{
			name:        "out of bounds write",
			memory:      &MockMemory{Data: make([]byte, 3)},
			fieldName:   "test field",
			offset:      0,
			data:        []byte("hello"),
			shouldPanic: true,
			expected:    nil,
		},
		{
			name:        "write fails",
			memory:      &MockMemory{Data: make([]byte, 10), ShouldFail: true},
			fieldName:   "test field",
			offset:      0,
			data:        []byte("hello"),
			shouldPanic: true,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.Panics(t, func() {
					Write(tt.memory, tt.fieldName, tt.offset, tt.data)
				}, "Write should panic on out of bounds or failure")
			} else {
				Write(tt.memory, tt.fieldName, tt.offset, tt.data)
				assert.Equal(t, tt.expected, tt.memory.Data, "Write should modify memory correctly")
			}
		})
	}
}

func TestReadString(t *testing.T) {
	tests := []struct {
		name        string
		memory      *MockMemory
		fieldName   string
		offset      uint32
		byteCount   uint32
		shouldPanic bool
		expected    string
	}{
		{
			name:        "successful string read",
			memory:      &MockMemory{Data: []byte("hello world")},
			fieldName:   "test field",
			offset:      0,
			byteCount:   5,
			shouldPanic: false,
			expected:    "hello",
		},
		{
			name:        "successful string read with offset",
			memory:      &MockMemory{Data: []byte("hello world")},
			fieldName:   "test field",
			offset:      6,
			byteCount:   5,
			shouldPanic: false,
			expected:    "world",
		},
		{
			name:        "out of bounds string read",
			memory:      &MockMemory{Data: []byte("hello")},
			fieldName:   "test field",
			offset:      10,
			byteCount:   5,
			shouldPanic: true,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.Panics(t, func() {
					ReadString(tt.memory, tt.fieldName, tt.offset, tt.byteCount)
				}, "ReadString should panic on out of bounds or failure")
			} else {
				result := ReadString(tt.memory, tt.fieldName, tt.offset, tt.byteCount)
				assert.Equal(t, tt.expected, result, "ReadString should return the correct string")
			}
		})
	}
}

func TestMockMemory(t *testing.T) {
	// Test the MockMemory implementation itself
	t.Run("mock memory read success", func(t *testing.T) {
		mem := &MockMemory{Data: []byte("hello world")}
		data, ok := mem.Read(0, 5)
		require.True(t, ok)
		assert.Equal(t, []byte("hello"), data)
	})

	t.Run("mock memory read failure", func(t *testing.T) {
		mem := &MockMemory{Data: []byte("hello"), ShouldFail: true}
		_, ok := mem.Read(0, 5)
		assert.False(t, ok)
	})

	t.Run("mock memory out of bounds read", func(t *testing.T) {
		mem := &MockMemory{Data: []byte("hello")}
		_, ok := mem.Read(10, 5)
		assert.False(t, ok)
	})

	t.Run("mock memory write success", func(t *testing.T) {
		mem := &MockMemory{Data: make([]byte, 10)}
		ok := mem.Write(0, []byte("hello"))
		require.True(t, ok)
		assert.Equal(t, []byte("hello\x00\x00\x00\x00\x00"), mem.Data)
	})

	t.Run("mock memory write failure", func(t *testing.T) {
		mem := &MockMemory{Data: make([]byte, 10), ShouldFail: true}
		ok := mem.Write(0, []byte("hello"))
		assert.False(t, ok)
	})

	t.Run("mock memory out of bounds write", func(t *testing.T) {
		mem := &MockMemory{Data: make([]byte, 3)}
		ok := mem.Write(0, []byte("hello"))
		assert.False(t, ok)
	})
}
