package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSha256Hasher_Hash(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // SHA-256 of empty string
		},
		{
			name:     "hello world",
			data:     []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", // SHA-256 of "hello world"
		},
		{
			name:     "binary data",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04},
			expected: "08bb5e5d6eaac1049ede0893d30ed022b1a4d9b5b48db414871f51c9cb35283d", // SHA-256 of binary data
		},
	}

	hasher := Sha256Hasher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the actual Hash method
			got, err := hasher.Hash(tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSha256Hasher_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		data     []byte
		expected bool
	}{
		{
			name:     "valid empty data",
			hash:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			data:     []byte{},
			expected: true,
		},
		{
			name:     "valid hello world",
			hash:     "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			data:     []byte("hello world"),
			expected: true,
		},
		{
			name:     "invalid hash",
			hash:     "invalid_hash",
			data:     []byte("hello world"),
			expected: false,
		},
		{
			name:     "mismatched hash",
			hash:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // hash of empty string
			data:     []byte("hello world"),
			expected: false,
		},
		{
			name:     "case sensitivity",
			hash:     "B94D27B9934D3E08A52E52D7DA7DABFAC484EFE37A5380EE9088F7ACE2EFCDE9", // uppercase
			data:     []byte("hello world"),
			expected: false,
		},
	}

	hasher := Sha256Hasher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasher.IsValid(tt.hash, tt.data)
			assert.Equal(
				t,
				tt.expected,
				got,
				"Sha256Hasher.IsValid() = %v, want %v",
				got,
				tt.expected,
			)
		})
	}
}

func TestDefaultHasher_Hash(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
		wantErr  bool
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: "",
			wantErr:  false,
		},
		{
			name:     "some data",
			data:     []byte("hello world"),
			expected: "",
			wantErr:  false,
		},
	}

	hasher := DefaultHasher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hasher.Hash(tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDefaultHasher_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		data     []byte
		expected bool
	}{
		{
			name:     "empty hash, empty data",
			hash:     "",
			data:     []byte{},
			expected: true,
		},
		{
			name:     "empty hash, some data",
			hash:     "",
			data:     []byte("hello world"),
			expected: true,
		},
		{
			name:     "some hash, empty data",
			hash:     "some hash",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "some hash, some data",
			hash:     "some hash",
			data:     []byte("hello world"),
			expected: false,
		},
	}

	hasher := DefaultHasher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasher.IsValid(tt.hash, tt.data)
			assert.Equal(
				t,
				tt.expected,
				got,
				"DefaultHasher.IsValid() = %v, want %v",
				got,
				tt.expected,
			)
		})
	}
}

// TestHasher_Interface ensures that both Sha256Hasher and DefaultHasher implement the Hasher interface
func TestHasher_Interface(t *testing.T) {
	var _ Hasher = Sha256Hasher{}
	var _ Hasher = DefaultHasher{}
}
