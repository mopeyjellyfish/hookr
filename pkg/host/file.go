package host

import (
	"fmt"
	"os"
)

const MainModule = "main"

// Wasm is an interface that represents a WebAssembly module that can be loaded into a plugin.
type Wasm interface {
	GetData() (WasmData, error)
}

// WasmData is a struct that represents the data of a WebAssembly module.
type WasmData struct {
	Data []byte
	Hash string
	Name string
}

// WasmFile is a struct that represents a WebAssembly module that is stored in a file.
// We can possibly add more types of Wasm modules in the future, which have different sources.
type WasmFile struct {
	Path string
	Hash string
	Name string
	data WasmData
}

// GetData returns the WasmData for WasmData if the data has already been loaded into memory somewhere else
func (d *WasmData) GetData() (WasmData, error) {
	return *d, nil
}

// readData reads the data from the file and returns it as a WasmData struct
func (f *WasmFile) readData() (WasmData, error) {
	file, err := os.ReadFile(f.Path) // TODO: Look into caching perhaps if this is read a lot...
	if err != nil {
		cwdDir, errCwd := os.Getwd()
		if errCwd != nil {
			return WasmData{}, fmt.Errorf("error getting working directory when dealing with error reading engine file: %v", err)
		}
		return WasmData{}, fmt.Errorf("error reading path: %v cwd: %v", f.Path, cwdDir)
	} else {
		return WasmData{
			Data: file,
			Hash: f.Hash,
			Name: f.Name,
		}, nil
	}
}

// GetData returns the WasmData for the WasmFile
func (f *WasmFile) GetData() (WasmData, error) {
	return f.data, nil
}

// Verify checks if the WasmFile has been configured correctly and returns it if it is.
// Otherwise will return a specific error
func (f *WasmFile) Verify() (*WasmFile, error) {
	if f.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	data, err := f.readData()
	if err != nil {
		return nil, err
	}
	f.data = data
	return f, nil

}

type WasmFileOption func(*WasmFile)

// Name sets the Name field of a WasmFile
func Name(name string) WasmFileOption {
	return func(f *WasmFile) {
		if name != "" {
			f.Name = name
		} else {
			f.Name = MainModule
		}
	}
}

// Hash sets the expected Hash field of a WasmFile
func Hash(hash string) WasmFileOption {
	return func(f *WasmFile) {
		f.Hash = hash
	}
}

// NewFile creates a new WasmFile instance and verifies it.
// If the file or name is invalid, an error is returned.
// Otherwise the *WasmFile is returned.
func NewFile(path string, opts ...WasmFileOption) (*WasmFile, error) {
	newFile := WasmFile{
		Path: path,
		Name: MainModule,
		Hash: "",
	}
	for _, opt := range opts {
		opt(&newFile)
	}
	return newFile.Verify()
}
