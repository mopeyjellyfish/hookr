package runtime

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/mopeyjellyfish/hookr/runtime/invoke"
	"github.com/mopeyjellyfish/hookr/runtime/logger"
	runtimememory "github.com/mopeyjellyfish/hookr/runtime/memory"
	"github.com/mopeyjellyfish/hookr/runtime/module"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/assemblyscript"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// NewRuntime returns a new wazero runtime which is called when the New method
// on hookr.Runtime is called. The result is closed upon wapc.Module Close.
type NewRuntime func(context.Context) (wazero.Runtime, error)

// functionStart is the name of the nullary function a module exports if it is a WASI Command Module.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/design/application-abi.md#current-unstable-abi
const fnStart = "_start"

// functionInitialize is the name of the function to initialize the runtime.
const fnInitialize = "_initialize"

// functionInit is the name of the nullary function that initializes hookr.
const fnHookrInit = "hookr_init"

// functionPluginCall is a callback required to be exported. Below is its signature in WebAssembly 1.0 (MVP) Text Format:
//
//	(func $__plugin_call (param $operation_len i32) (param $payload_len i32) (result (;errno;) i32))
const fnPluginCall = "__plugin_call"
const fnPluginCallV2 = "__plugin_call_v2"
const fnABIVersionV2 = "__hookr_abi_version_v2"
const fnSchemaHashV2 = "__hookr_schema_hash_v2"
const fnCapabilitiesV2 = "__hookr_capabilities_v2"

type Runtime struct {
	newRuntime    NewRuntime
	ctx           context.Context
	file          *File
	logger        logger.Logger
	stderr        io.Writer
	stdout        io.Writer
	rand          io.Reader
	callHandler   module.CallHandler
	callHandlerV2 module.CallHandlerV2

	hostFns              CallFns
	hostFnV2s            map[uint32]CallFn
	pluginCall           api.Function
	pluginCallV2         api.Function
	expectedHandshake    *runtimecontract.Handshake
	requiredCapabilities uint64
	expectedSchema       *runtimecontract.Schema
	pluginHandshake      *runtimecontract.Handshake
	moduleName           string
	r                    wazero.Runtime
	config               wazero.ModuleConfig
	hookr                api.Module
	plugin               api.Module
	compiled             wazero.CompiledModule

	invokeMu  sync.Mutex
	invokeCtx *invoke.Context
}

// Will initialize the wazero runtime
func (e *Runtime) InitRuntime() error {
	if e.newRuntime == nil {
		return errors.New("runtime not configured")
	}

	r, err := e.newRuntime(e.ctx)
	if err != nil {
		return err
	}
	e.r = r
	return nil
}

// InitHookr initializes the hookr host module and sets it to the wazero runtime.
func (e *Runtime) InitHookr() error {
	if e.r == nil {
		return errors.New("runtime not initialized")
	}
	hookr, err := module.New(e.ctx, e.r, e.fnHandler, e.fnHandlerV2, e.currentInvoke, e.logger)
	if err != nil {
		return err
	}
	e.hookr = hookr
	return nil
}

func (e *Runtime) fnHandler(ctx context.Context, operation string, payload []byte) ([]byte, error) {
	if e.callHandler != nil {
		return e.callHandler(ctx, operation, payload)
	}
	if fn, ok := e.hostFns[operation]; ok {
		return fn(ctx, payload)
	}
	return nil, fmt.Errorf("no handler registered for operation '%s'", operation)
}

func (e *Runtime) fnHandlerV2(ctx context.Context, methodID uint32, payload []byte) ([]byte, error) {
	if e.callHandlerV2 != nil {
		return e.callHandlerV2(ctx, methodID, payload)
	}
	if fn, ok := e.hostFnV2s[methodID]; ok {
		return fn(ctx, payload)
	}
	return nil, fmt.Errorf("no handler registered for method id %d", methodID)
}

// RegisterFunction registers a host function with the engine.
func (e *Runtime) RegisterFunction(name string, fn CallFn) {
	if e.hostFns == nil {
		e.hostFns = make(CallFns)
	}
	e.hostFns[name] = fn
}

// RegisterMethod registers a method-ID based host callback for ABI v2 plugins.
func (e *Runtime) RegisterMethod(methodID uint32, fn CallFn) {
	if e.hostFnV2s == nil {
		e.hostFnV2s = make(map[uint32]CallFn)
	}
	e.hostFnV2s[methodID] = fn
}

// InitConfig initializes the wazero module config with the default settings.
func (e *Runtime) InitConfig() {
	cfg := wazero.NewModuleConfig().
		WithStartFunctions().
		WithStderr(e.stderr).
		WithStdout(e.stdout).
		WithRandSource(e.rand).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime()
	e.config = cfg
}

// Init initializes the engine by setting up the runtime, config, and hookr.
// It is called when the engine is created.
func (e *Runtime) Init() error {
	if err := e.InitRuntime(); err != nil {
		return err
	}

	e.InitConfig()

	if err := e.InitHookr(); err != nil {
		return err
	}

	return nil
}

// MemorySize returns the size of the memory for this instance.
// This is the size in bytes, not the number of pages.
func (e *Runtime) MemorySize() uint32 {
	if e.plugin == nil {
		return 0
	}
	return e.plugin.Memory().Size()
}

// Compile compiles the plugin module. It must be called after the runtime is
// initialized and before the module is instantiated.
func (e *Runtime) Compile() error {
	if e.r == nil {
		return errors.New("runtime not initialized")
	}
	if e.compiled != nil {
		return errors.New("plugin already compiled")
	}

	// Compile the plugin module
	d, err := e.file.GetData()
	if err != nil {
		return fmt.Errorf("failed to get data from file: %w", err)
	}
	compiled, err := e.r.CompileModule(e.ctx, d)
	if err != nil {
		return fmt.Errorf("failed to compile module: %w", err)
	}
	e.compiled = compiled
	return nil
}

// Instantiate instantiates the compiled module. It must be called after the
// module is compiled. It will also call the WASI and hookr start functions if
func (e *Runtime) Instantiate() error {
	if e.r == nil {
		return errors.New("runtime not initialized")
	}
	module, err := e.r.InstantiateModule(e.ctx, e.compiled, e.config.WithName(e.moduleName))
	if err != nil {
		return fmt.Errorf("failed to instantiate module: %w", err)
	}

	// Call any WASI or hookr start functions on instantiate.
	funcs := []string{fnStart, fnInitialize, fnHookrInit}
	for _, f := range funcs {
		exportedFunc := module.ExportedFunction(f)
		if exportedFunc != nil {
			ic := invoke.Context{Operation: f, PluginReq: nil}
			if _, err := e.callWithInvokeContext0(e.ctx, &ic, exportedFunc); err != nil {
				if exitErr, ok := err.(*sys.ExitError); ok {
					return fmt.Errorf("error calling %s: %w", f, exitErr)
				}
			}
		}
	}

	e.plugin = module

	e.pluginCall = module.ExportedFunction(fnPluginCall)
	e.pluginCallV2 = module.ExportedFunction(fnPluginCallV2)
	if e.pluginCall == nil && e.pluginCallV2 == nil {
		_ = e.plugin.Close(e.ctx)
		return fmt.Errorf("module %s didn't export %s or %s", e.moduleName, fnPluginCall, fnPluginCallV2)
	}
	if err := e.validateContractHandshake(); err != nil {
		_ = e.plugin.Close(e.ctx)
		return err
	}

	return nil
}

// Invoke calls the plugin function with the given operation and payload.
func (e *Runtime) Invoke(ctx context.Context, operation string, payload []byte) ([]byte, error) {
	if e.plugin == nil {
		return nil, errors.New("plugin not initialized")
	}
	if e.pluginCall == nil {
		return nil, errors.New("plugin does not support string-operation ABI; use InvokeMethod for ABI v2 plugins")
	}

	ic := invoke.Context{Operation: operation, PluginReq: payload}
	results, err := e.callWithInvokeContext2(
		ctx,
		&ic,
		e.pluginCall,
		uint64(len(operation)),
		uint64(len(payload)),
	)
	if err != nil {
		return nil, fmt.Errorf("error while making %s call: %w", operation, err)
	}
	if ic.PluginErr != "" { // guestErr is not nil if the guest called "__plugin_error".
		return nil, errors.New(ic.PluginErr)
	}

	result := results[0]
	success := result == 1

	if success { // guestResp is not nil if the guest called "__plugin_response".
		return ic.PluginResp, nil
	}

	return nil, fmt.Errorf("call to %q was unsuccessful", operation)
}

// InvokeMethod calls a plugin method-ID endpoint with the given payload.
func (e *Runtime) InvokeMethod(ctx context.Context, methodID uint32, payload []byte) ([]byte, error) {
	if e.plugin == nil {
		return nil, errors.New("plugin not initialized")
	}
	if e.pluginCallV2 == nil {
		return nil, errors.New("plugin does not support ABI v2 method calls")
	}

	ic := invoke.Context{PluginReq: payload}
	results, err := e.callWithInvokeContext2(
		ctx,
		&ic,
		e.pluginCallV2,
		uint64(methodID),
		uint64(len(payload)),
	)
	if err != nil {
		return nil, fmt.Errorf("error while making method %d call: %w", methodID, err)
	}
	if ic.PluginErr != "" {
		return nil, errors.New(ic.PluginErr)
	}

	if len(results) > 0 && results[0] == 1 {
		return ic.PluginResp, nil
	}
	return nil, fmt.Errorf("call to method %d was unsuccessful", methodID)
}

func (e *Runtime) validateContractHandshake() error {
	strict := e.expectedHandshake != nil || e.requiredCapabilities != 0

	if e.pluginCallV2 == nil {
		if !strict {
			return nil
		}
		return errors.New("contract handshake requested, but plugin does not export __plugin_call_v2")
	}

	abiFn := e.plugin.ExportedFunction(fnABIVersionV2)
	hashFn := e.plugin.ExportedFunction(fnSchemaHashV2)
	capsFn := e.plugin.ExportedFunction(fnCapabilitiesV2)
	if abiFn == nil || hashFn == nil {
		if !strict {
			return nil
		}
		return errors.New("contract handshake requested, but plugin does not export ABI v2 handshake functions")
	}

	abiResults, err := abiFn.Call(e.ctx)
	if err != nil {
		if !strict {
			return nil
		}
		return fmt.Errorf("failed to call %s: %w", fnABIVersionV2, err)
	}
	if len(abiResults) == 0 {
		if !strict {
			return nil
		}
		return fmt.Errorf("plugin %s returned no results", fnABIVersionV2)
	}
	versionPacked := uint32(abiResults[0])
	pluginABIMajor := uint16(versionPacked >> 16)
	pluginABIMinor := uint16(versionPacked & 0xFFFF)

	hashResults, err := hashFn.Call(e.ctx)
	if err != nil {
		if !strict {
			return nil
		}
		return fmt.Errorf("failed to call %s: %w", fnSchemaHashV2, err)
	}
	if len(hashResults) == 0 {
		if !strict {
			return nil
		}
		return fmt.Errorf("plugin %s returned no results", fnSchemaHashV2)
	}

	hashPtr, hashLen := unpackPtrLenU64(hashResults[0])
	if hashLen != runtimecontract.SchemaHashLen {
		if !strict {
			return nil
		}
		return fmt.Errorf("plugin schema hash has invalid length: got %d want %d", hashLen, runtimecontract.SchemaHashLen)
	}
	hash := runtimememory.Read(e.plugin.Memory(), "schema_hash", hashPtr, hashLen)
	var schemaHash [runtimecontract.SchemaHashLen]byte
	copy(schemaHash[:], hash)

	pluginHandshake := runtimecontract.Handshake{
		ABIMajor:   pluginABIMajor,
		ABIMinor:   pluginABIMinor,
		SchemaHash: schemaHash,
	}
	if capsFn != nil {
		capsResults, err := capsFn.Call(e.ctx)
		if err != nil {
			if strict {
				return fmt.Errorf("failed to call %s: %w", fnCapabilitiesV2, err)
			}
		} else if len(capsResults) == 0 {
			if strict {
				return fmt.Errorf("plugin %s returned no results", fnCapabilitiesV2)
			}
		} else {
			pluginHandshake.Capabilities = capsResults[0]
		}
	}

	e.pluginHandshake = &pluginHandshake

	requiredCaps := e.requiredCapabilities
	if e.expectedHandshake != nil {
		requiredCaps |= e.expectedHandshake.Capabilities
	}
	if missing := requiredCaps &^ pluginHandshake.Capabilities; missing != 0 {
		return fmt.Errorf("%w: missing_bits=0x%x", runtimecontract.ErrCapabilityMismatch, missing)
	}

	if e.expectedHandshake != nil {
		hostHandshake := *e.expectedHandshake
		hostHandshake.Capabilities = requiredCaps
		if err := runtimecontract.ValidateHandshake(hostHandshake, pluginHandshake); err != nil {
			return fmt.Errorf("contract handshake validation failed: %w", err)
		}
	}

	return nil
}

func (e *Runtime) Close(ctx context.Context) error {
	if e.plugin != nil {
		if err := e.plugin.Close(ctx); err != nil {
			return fmt.Errorf("error closing plugin: %w", err)
		}
	}

	if e.hookr != nil {
		if err := e.hookr.Close(ctx); err != nil {
			return fmt.Errorf("error closing hookr: %w", err)
		}
	}

	if e.r != nil {
		if err := e.r.Close(ctx); err != nil {
			return fmt.Errorf("error closing runtime: %w", err)
		}
	}

	return nil
}

// DefaultRuntime implements NewRuntime by returning a wazero runtime with WASI
// and AssemblyScript host functions instantiated.
func DefaultRuntime(ctx context.Context) (wazero.Runtime, error) {
	r := wazero.NewRuntime(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}

	// This disables the abort message as no other engines write it.
	envBuilder := r.NewHostModuleBuilder("env")
	assemblyscript.NewFunctionExporter().WithAbortMessageDisabled().ExportFunctions(envBuilder)
	if _, err := envBuilder.Instantiate(ctx); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}
	return r, nil
}

func New(ctx context.Context, opts ...Option) (*Runtime, error) {
	e := &Runtime{
		newRuntime: DefaultRuntime,
		ctx:        ctx,
		stderr:     os.Stderr,
		stdout:     os.Stdout,
		rand:       rand.Reader,
		logger:     logger.Default,
	}

	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, err
		}
	}

	if err := e.Init(); err != nil {
		return nil, err
	}

	if err := e.Compile(); err != nil {
		return nil, err
	}

	if err := e.Instantiate(); err != nil {
		return nil, err
	}

	return e, nil
}

func unpackPtrLenU64(encoded uint64) (ptr uint32, dataLen uint32) {
	ptr = uint32(encoded >> 32)
	dataLen = uint32(encoded & 0xFFFFFFFF)
	return ptr, dataLen
}

func (e *Runtime) currentInvoke() *invoke.Context {
	return e.invokeCtx
}

func (e *Runtime) callWithInvokeContext0(
	ctx context.Context,
	ic *invoke.Context,
	fn api.Function,
) ([]uint64, error) {
	e.invokeMu.Lock()
	e.invokeCtx = ic
	defer func() {
		e.invokeCtx = nil
		e.invokeMu.Unlock()
	}()
	return fn.Call(ctx)
}

func (e *Runtime) callWithInvokeContext2(
	ctx context.Context,
	ic *invoke.Context,
	fn api.Function,
	arg0 uint64,
	arg1 uint64,
) ([]uint64, error) {
	e.invokeMu.Lock()
	e.invokeCtx = ic
	defer func() {
		e.invokeCtx = nil
		e.invokeMu.Unlock()
	}()
	return fn.Call(ctx, arg0, arg1)
}
