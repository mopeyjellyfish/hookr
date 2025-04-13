package host

import (
	"errors"
	"fmt"
	"os"
)

// Hasher hashes and validate strings
type Hasher interface {
	// Hash returns a hash for the provided string
	Hash(raw string) (string, error)

	// IsValid checks if a hash and a string match
	IsValid(hash string, raw string) bool
}

// DefaultHasher is a default implementation of the Hasher interface.
// It does not actually hash anything, but is a placeholder for future implementations.
// It is used when no specific hashing algorithm is provided and no verification is needed.
type DefaultHasher struct{}

func (h DefaultHasher) Hash(raw string) (string, error) {
	return "", nil
}

func (h DefaultHasher) IsValid(hash, raw string) bool {
	has, _ := h.Hash(raw)
	return has == hash
}

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

	if !f.hasher.IsValid(f.Hash, string(data)) {
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
