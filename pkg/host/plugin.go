package host

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"

	_ "embed"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

//go:embed runtime.wasm
var runtimeWasm []byte

type modules map[string]api.Module

type Func func(input []byte) ([]byte, error)

type Plugin struct {
	ctx   context.Context
	spec  *Spec
	cache wazero.CompilationCache

	moduleConfig wazero.ModuleConfig
	rtConfig     wazero.RuntimeConfig
	rt           wazero.Runtime
	hookr        api.Module // This is the runtime module handling I/O, common functions etc.
	main         api.Module // This is the main module for the plugin
	modules      modules
	mem          Memory
}

// LoadRuntimeConfig loads the runtime config into memory
func (p *Plugin) LoadRuntimeConfig() {
	p.rtConfig = wazero.NewRuntimeConfig().WithCompilationCache(p.cache) // TODO: Allow for overriding this perhaps?
}

// LoadRuntime loads the runtime into memory & configures the wasi if configured
func (p *Plugin) LoadRuntime() {
	p.rt = wazero.NewRuntimeWithConfig(p.ctx, p.rtConfig)
	if p.spec.Wasi {
		wasi_snapshot_preview1.MustInstantiate(p.ctx, p.rt)
	}
}

// LoadHookrModule loads the hookr module into the wasm runtime, enabling I/O and common functions for the main module & the go host code to interface with
// For example, I/O goes through the memory space of hookr and is read/written to by the main module & host code.
func (p *Plugin) LoadHookrModule() error {
	hookrCfg := wazero.NewModuleConfig().
		WithName("hookr").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStartFunctions().
		WithRandSource(rand.Reader).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime()

	hookr, err := p.rt.InstantiateWithConfig(p.ctx, runtimeWasm, hookrCfg)
	if err != nil {
		return fmt.Errorf("error instantiating hookr WASM module: %v", err)
	}
	p.hookr = hookr
	return nil
}

// WriteMem writes the input to be read by the hookr module
// Wrapper around the memory.Write function
func (p *Plugin) WriteMem(input []byte) (uint64, error) {
	return p.mem.Write(input)
}

// Free frees the memory from the hookr module
// Wrapper around the memory.Free function
func (p *Plugin) Free(ptr uint64) {
	offset := uint32(ptr >> 32)
	p.mem.Free(uint64(offset))
}

// ReadMem reads the memory from the hookr module
// Wrapper around the memory.Read function
func (p *Plugin) ReadMem(ptr uint64) ([]byte, error) {
	return p.mem.Read(ptr)
}

// GetMemory gets the memory helper struct for the plugin
func (p *Plugin) GetMemory() Memory {
	return p.mem
}

// CreateHostFunctionWrapper creates a wrapper around the host function to be called by the main module
// This needs to be in scope of the loop inside of LoadHostFuncs
func (p *Plugin) CreateHostFunctionWrapper(hf HostFunction) WasmFunc {
	return func(inPtr uint64) uint64 {
		return hf.Call(p.ctx, p.mem, inPtr)
	}
}

// LoadHostFuncs loads all of the host functions exposed by the service to
func (p *Plugin) LoadHostFuncs() {
	// Loop over host functions in the spec and add them to the env module
	builder := p.rt.NewHostModuleBuilder("env")
	for _, hf := range p.spec.HostFuncs {
		wasmFn := p.CreateHostFunctionWrapper(hf)
		builder = builder.NewFunctionBuilder().WithFunc(wasmFn).Export(hf.Name)
	}
	_, err := builder.Instantiate(p.ctx)
	if err != nil {
		log.Panicln("Error with env module and host function(s):", err)
	}
}

// LoadModules loads the modules from the spec into memory
// Returns an error if there is an issue loading the modules
// Uses the runtime to instantiate the modules
func (p *Plugin) LoadModules() error {
	moduleConfig := wazero.NewModuleConfig()
	modules := map[string]api.Module{}
	var main api.Module
	for _, wasm := range p.spec.Wasm { // Load each of the modules within the spec
		data, err := wasm.GetData()
		if err != nil {
			return fmt.Errorf("error getting data from wasm: %v", err)
		}
		_, ok := modules[data.Name]
		if ok {
			return fmt.Errorf("module with name %s already exists", data.Name)
		}

		moduleConfig.WithName(data.Name).
			WithStartFunctions().
			WithStdout(os.Stdout).
			WithStderr(os.Stderr).
			WithRandSource(rand.Reader).
			WithSysNanosleep().
			WithSysNanotime().
			WithSysWalltime()

		mod, err := p.rt.InstantiateWithConfig(p.ctx, data.Data, moduleConfig)
		if err != nil {
			return fmt.Errorf("error instantiating module '%s': %v", data.Name, err)
		}
		modules[data.Name] = mod
		if data.Name == MainModule {
			main = mod
		}
	}
	p.modules = modules
	p.main = main
	p.moduleConfig = moduleConfig
	return nil
}

// LoadMemory loads all of the memory related functions for use in the main module
func (p *Plugin) LoadMemory() {
	memory := p.hookr.ExportedMemory("memory")
	freeFn := p.hookr.ExportedFunction("free")
	mallocFn := p.hookr.ExportedFunction("malloc")

	if memory == nil {
		panic("Error getting memory from module")
	}
	if freeFn == nil {
		panic("Error getting free function from module")
	}
	if mallocFn == nil {
		panic("Error getting malloc function from module")
	}

	p.mem = NewMemory(p.ctx, memory, freeFn, mallocFn)
}

// Load will load the plugin into memory
// Returns an error if there is an issue loading the plugin
// Loads a wazero config and runtime
// Uses the provided spec to load the WASM modules
func (p *Plugin) Load() (*Plugin, error) {
	p.LoadRuntimeConfig()
	p.LoadRuntime()
	err := p.LoadHookrModule()
	if err != nil {
		return nil, fmt.Errorf("error hookr module: %v", err)
	}
	p.LoadMemory()
	p.LoadHostFuncs()
	err = p.LoadModules()
	if err != nil {
		return nil, fmt.Errorf("error loading modules: %v", err)
	}
	return p, nil
}

// GetError gets the error from the main module
// Returns the error and an error if there is an issue getting the error
// Example:
//
//	error, err := p.GetError()
func (p *Plugin) GetError() ([]byte, error) { // TODO: Get the error from the main module
	return nil, nil
}

// Memory gets the memory from the main module
func (p *Plugin) Memory(name string) api.Memory { // TODO: Read the memory from the main module
	return p.main.ExportedMemory(name)
}

// Call gets the function and then calls it with the input
func (p *Plugin) Call(name string, input []byte) ([]byte, error) {
	fn, err := p.GetFunction(name)
	if err != nil {
		return nil, err
	}
	return fn(input)
}

// CallFunction calls a function in the main module, writes the input and reads a copy of the output
// The memory for the input is freed after the function is called
// The memory for the output is freed after the output is read
// Returns the output and an error if there is an issue calling the function
func (p *Plugin) CallFunction(callFn api.Function, input []byte) ([]byte, error) {
	if callFn == nil {
		return nil, fmt.Errorf("function does not exist")
	}
	name := callFn.Definition().Name()
	var params []uint64
	if len(input) > 0 {
		ptr, err := p.WriteMem(input)
		if err != nil {
			return nil, fmt.Errorf("error writing input %s: %v", name, err)
		}
		params = append(params, ptr)
	}

	result, err := callFn.Call(p.ctx, params...)

	if exitErr, ok := err.(*sys.ExitError); ok {
		if exitErr.ExitCode() != 0 {
			return nil, fmt.Errorf("error calling function %s: exit code %d", name, exitErr.ExitCode())
		}
	} else {
		if err != nil {
			return nil, fmt.Errorf("error calling function %s: %v", name, err)
		}
	}

	if len(result) == 0 {
		return nil, nil
	}
	resultPtr := result[0]
	if resultPtr == 1 {
		return nil, fmt.Errorf("error calling function %s: error", name)
	}
	if resultPtr == 0 { // Nothing returned from function
		return nil, nil
	}
	data, err := p.ReadMem(resultPtr)
	if err != nil {
		return nil, fmt.Errorf("error reading output: %v", err)
	}
	return data, nil
}

// GetFunction gets a function from the main module by name
// Returns a function which can be called with an input
// Wraps the function defined by the name if it exists which enables the plugin pkg to manage the inputs and outputs
func (p *Plugin) GetFunction(name string) (Func, error) {
	callFn := p.main.ExportedFunction(name)
	if callFn == nil {
		return nil, fmt.Errorf("function %s does not exist", name)
	} else if n := len(callFn.Definition().ResultTypes()); n > 1 {
		return nil, fmt.Errorf("function %s has %v results, expected 0 or 1", name, n)
	}
	return func(input []byte) ([]byte, error) {
		return p.CallFunction(callFn, input)
	}, nil
}

// Exists checks for existence of a function in the main module
func (p *Plugin) Exists(name string) bool { // Does a function exist in the main module
	return p.main.ExportedFunction(name) != nil
}

// NewFromSpec creates a new plugin with the given ctx & spec
func NewFromSpec(ctx context.Context, spec *Spec) (*Plugin, error) {
	plugin := Plugin{
		ctx:   ctx,
		spec:  spec,
		cache: wazero.NewCompilationCache(),
	}
	return plugin.Load()
}

// New creates a new plugin with the given ctx, file & opts for constructing the spec
func New(ctx context.Context, file string, opts ...SpecOption) (*Plugin, error) {
	spec, err := NewSpecFromFile(file, opts...)
	if err != nil {
		return nil, err
	}
	return NewFromSpec(ctx, spec)
}
