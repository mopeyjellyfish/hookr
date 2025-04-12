package host

import (
	"fmt"
)

// Spec is everything that is needed to run a plugin
// These are all the `Wasm` modules that are part of the plugin and whether or not the plugin uses WASI
type Spec struct {
	Wasm      []Wasm         // All of the WASM modules that are part of this plugin
	HostFuncs []HostFunction // All of the host functions that are part of this plugin
	Wasi      bool           // Whether or not this plugin uses WASI - default is true
}

// AddHostFunc will add a host function to the existing spec
// Returns the spec for chaining
func (s *Spec) AddHostFunc(hf HostFunction) *Spec {
	s.HostFuncs = append(s.HostFuncs, hf)
	return s
}

type SpecOption func(*Spec)

// Wasi is an option for the spec to set whether or not the plugin uses WASI
func Wasi(wasi bool) SpecOption {
	return func(s *Spec) {
		s.Wasi = wasi
	}
}

// WithHostFuncs is an option for the spec to set the host functions that are part of the plugin
func WithHostFuncs(hostFuncs []HostFunction) SpecOption {
	return func(s *Spec) {
		s.HostFuncs = hostFuncs
	}
}

// WithHostFunc is an option for the spec to set a single host function that is part of the plugin
func WithHostFunc(hostFunc HostFunction) SpecOption {
	return func(s *Spec) {
		s.HostFuncs = append(s.HostFuncs, hostFunc)
	}
}

// NewSpec will create a new spec with the given WASM modules
func NewSpec(wasm []Wasm, opts ...SpecOption) *Spec {
	spec := &Spec{
		Wasm: wasm,
		Wasi: true, // Defaults to enabled
	}
	for _, opt := range opts {
		opt(spec)
	}
	return spec
}

// NewSpecFromFile will create a new spec from the given WASM file with default settings
func NewSpecFromFile(wasmFile string, opts ...SpecOption) (*Spec, error) {
	wasm, err := NewFile(wasmFile)
	if err != nil {
		return nil, fmt.Errorf("error creating spec: %v", err)
	}
	return NewSpec([]Wasm{wasm}, opts...), nil
}

// NewSpecFromFiles will create a new spec from the given WASM files with default settings
func NewSpecFromFiles(wasmFiles []string, opts ...SpecOption) (*Spec, error) {
	if len(wasmFiles) == 0 {
		return nil, fmt.Errorf("no wasm files provided")
	}
	wasm := make([]Wasm, len(wasmFiles))
	for i, f := range wasmFiles {
		wasmFile, err := NewFile(f)
		if err != nil {
			return nil, fmt.Errorf("error creating spec: %v", err)
		}
		wasm[i] = wasmFile
	}
	return NewSpec(wasm, opts...), nil
}
