package host

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

// Hasher hashes and validates byte arrays.
// It provides functionality for computing cryptographic hashes of data
// and validating data against a previously computed hash.
//
// There are two built-in implementations:
//   - Sha256Hasher: Uses the SHA-256 algorithm for secure hashing
//   - DefaultHasher: A no-op implementation that performs no actual hashing
//
// Example:
//
//	// Using Sha256Hasher
//	hasher := Sha256Hasher{}
//
//	// Compute hash of a file
//	data, _ := os.ReadFile("plugin.wasm")
//	hash, _ := hasher.Hash(data)
//	fmt.Println("SHA-256 hash:", hash)
//
//	// Verify file integrity
//	isValid := hasher.IsValid(hash, data)
//	fmt.Println("Is valid:", isValid)
type Hasher interface {
	// Hash returns a hash for the provided data
	Hash(data []byte) (string, error)

	// IsValid checks if a hash and data match
	IsValid(hash string, data []byte) bool
}

// Sha256Hasher is an implementation of the Hasher interface.
// It uses SHA-256 hashing algorithm to hash and validate data.
// This provides a secure way to verify the integrity of WebAssembly modules.
//
// Example:
//
//	hasher := Sha256Hasher{}
//	hash, _ := hasher.Hash(wasmBytes)
//	fmt.Println("Use this hash for verification:", hash)
type Sha256Hasher struct{}

func (h Sha256Hasher) Hash(data []byte) (string, error) {
	// Implement SHA-256 hashing logic here
	hasher := sha256.New()
	_, err := hasher.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (h Sha256Hasher) IsValid(hash string, data []byte) bool {
	has, _ := h.Hash(data)
	return has == hash
}

var _ Hasher = Sha256Hasher{} // ensure Sha256Hasher implements the Hasher interface

// DefaultHasher is a default implementation of the Hasher interface.
// It does not actually hash anything, but is a placeholder for future implementations.
// It is used when no specific hashing algorithm is provided and no verification is needed.
//
// The Hash method always returns an empty string, and IsValid only returns true
// when the hash is empty.
//
// Example:
//
//	hasher := DefaultHasher{}
//
//	// Always returns empty string
//	hash, _ := hasher.Hash([]byte("data"))
//
//	// Returns true only when hash is empty
//	isValid := hasher.IsValid("", []byte("data")) // true
//	isValid = hasher.IsValid("some-hash", []byte("data")) // false
type DefaultHasher struct{}

func (h DefaultHasher) Hash(raw []byte) (string, error) {
	return "", nil
}

func (h DefaultHasher) IsValid(hash string, raw []byte) bool {
	has, _ := h.Hash(raw)
	return has == hash // if a hash is provided we do want to make sure this fails as it won't be equal to nothing..
}

var _ Hasher = DefaultHasher{} // ensure DefaultHasher implements the Hasher interface

// File is a struct that represents a WebAssembly module that is stored in a file.
// We can possibly add more types of Wasm modules in the future, which have different sources.
type File struct {
	Path   string
	Hash   string
	Name   string
	hasher Hasher
	data   []byte
}

// GetData returns the WasmData for WasmData if the data has already been loaded into memory somewhere else
func (f *File) GetData() ([]byte, error) {
	return f.data, nil
}

// readData reads the data from the file and returns it as a WasmData struct
func (f *File) load() ([]byte, error) {
	file, err := os.ReadFile(f.Path) // TODO: Look into caching perhaps if this is read a lot...
	if err != nil {
		cwdDir, errCwd := os.Getwd()
		if errCwd != nil {
			return nil, fmt.Errorf(
				"error getting working directory when dealing with error reading engine file: %v",
				err,
			)
		}
		return nil, fmt.Errorf("error reading path: %v cwd: %v", f.Path, cwdDir)
	} else {
		return file, nil
	}
}

// Verify checks if the File has been configured correctly and returns it if it is.
// Otherwise will return a specific error
func (f *File) Verify() (*File, error) {
	if f.Path == "" {
		return nil, errors.New("path is required")
	}
	data, err := f.load()
	if err != nil {
		return nil, err
	}

	if !f.hasher.IsValid(f.Hash, data) { // optionally check the hash
		return nil, fmt.Errorf("hash does not match for %s", f.Path)
	}

	f.data = data
	return f, nil
}

type FileOption func(*File)

// WithHash sets the expected Hash field of a File
func WithHash(hash string) FileOption {
	return func(f *File) {
		f.Hash = hash
	}
}

// WithHasher sets the Hasher to use for validating the hash of the File
func WithHasher(hasher Hasher) FileOption {
	return func(f *File) {
		f.hasher = hasher
	}
}

// NewFile creates a new File instance and verifies it.
// If the file or name is invalid, an error is returned.
// Otherwise the *File is returned.
func NewFile(path string, opts ...FileOption) (*File, error) {
	newFile := File{
		Path:   path,
		Name:   "",
		Hash:   "",
		hasher: DefaultHasher{},
	}
	for _, opt := range opts {
		opt(&newFile)
	}
	return newFile.Verify()
}
