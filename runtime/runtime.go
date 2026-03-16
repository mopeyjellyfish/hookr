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
//	(func $__plugin_call (param $method_id i32) (param $payload_len i32) (result (;errno;) i32))
const (
	fnPluginCall   = "__plugin_call"
	fnABIVersion   = "__hookr_abi_version"
	fnSchemaHash   = "__hookr_schema_hash"
	fnCapabilities = "__hookr_capabilities"
	fnMethods      = "__hookr_methods"
	maxMethodsLen  = 64 * 1024
)

var (
	newWazeroRuntime          = wazero.NewRuntime
	instantiateWASI           = wasi_snapshot_preview1.Instantiate
	newAssemblyscriptExporter = assemblyscript.NewFunctionExporter
	callFunction              = func(ctx context.Context, fn api.Function) ([]uint64, error) {
		return fn.Call(ctx)
	}
)

type Runtime struct {
	newRuntime NewRuntime
	ctx        context.Context
	file       *File
	logger     logger.Logger
	stderr     io.Writer
	stdout     io.Writer
	rand       io.Reader

	hostMethods       map[uint32]CallFn
	pluginCall        api.Function
	expectedHandshake *runtimecontract.Handshake
	expectedSchema    *runtimecontract.Schema
	pluginHandshake   *runtimecontract.Handshake
	pluginMethods     map[uint32]struct{}
	moduleName        string
	r                 wazero.Runtime
	config            wazero.ModuleConfig
	hookr             api.Module
	plugin            api.Module
	compiled          wazero.CompiledModule

	invokeMu sync.Mutex
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
	hookr, err := module.New(e.ctx, e.r, e.methodHandler, e.currentInvoke, e.logger)
	if err != nil {
		return err
	}
	e.hookr = hookr
	return nil
}

func (e *Runtime) methodHandler(
	ctx context.Context,
	methodID uint32,
	payload []byte,
) ([]byte, error) {
	if fn, ok := e.hostMethods[methodID]; ok {
		return fn(ctx, payload)
	}
	return nil, fmt.Errorf("no handler registered for method id %d", methodID)
}

// RegisterMethod registers a method-ID based host callback.
func (e *Runtime) RegisterMethod(methodID uint32, fn CallFn) {
	if e.hostMethods == nil {
		e.hostMethods = make(map[uint32]CallFn)
	}
	e.hostMethods[methodID] = fn
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
	if e.file == nil {
		return errors.New("plugin file not configured")
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
	if e.compiled == nil {
		return errors.New("plugin not compiled")
	}
	pluginModule, err := e.r.InstantiateModule(e.ctx, e.compiled, e.config.WithName(e.moduleName))
	if err != nil {
		return fmt.Errorf("failed to instantiate module: %w", err)
	}
	closeModule := true
	defer func() {
		if closeModule {
			_ = pluginModule.Close(e.ctx)
		}
	}()

	// Call any WASI or hookr start functions on instantiate.
	funcs := []string{fnStart, fnInitialize, fnHookrInit}
	for _, f := range funcs {
		exportedFunc := pluginModule.ExportedFunction(f)
		if exportedFunc != nil {
			ic := invoke.Context{Operation: f, PluginReq: nil}
			if _, err := e.callWithInvokeContext0(e.ctx, &ic, exportedFunc); err != nil {
				if exitErr, ok := err.(*sys.ExitError); ok {
					return fmt.Errorf("error calling %s: %w", f, exitErr)
				}
				return fmt.Errorf("error calling %s: %w", f, err)
			}
		}
	}

	e.plugin = pluginModule

	e.pluginCall = pluginModule.ExportedFunction(fnPluginCall)
	if e.pluginCall == nil {
		_ = e.plugin.Close(e.ctx)
		return fmt.Errorf("module %s didn't export %s", e.moduleName, fnPluginCall)
	}
	if err := e.validateContractHandshake(); err != nil {
		_ = e.plugin.Close(e.ctx)
		return err
	}
	closeModule = false

	return nil
}

// InvokeMethod calls a plugin method-ID endpoint with the given payload.
func (e *Runtime) InvokeMethod(
	ctx context.Context,
	methodID uint32,
	payload []byte,
) ([]byte, error) {
	var response []byte
	if err := e.InvokeMethodWithResponse(ctx, methodID, payload, func(data []byte) error {
		response = data
		return nil
	}); err != nil {
		return nil, err
	}
	return response, nil
}

// InvokeMethodWithResponse calls a plugin method-ID endpoint and invokes fn with
// the response bytes before returning to the caller.
func (e *Runtime) InvokeMethodWithResponse(
	ctx context.Context,
	methodID uint32,
	payload []byte,
	fn func([]byte) error,
) error {
	if e.plugin == nil {
		return errors.New("plugin not initialized")
	}
	if invoke.From(ctx) != nil {
		return errors.New("reentrant plugin invocation is not supported")
	}

	var (
		results   []uint64
		err       error
		resp      []byte
		pluginErr string
	)

	e.invokeMu.Lock()
	ic := &invoke.Context{}
	ic.Operation = ""
	ic.PluginReq = payload
	ic.PluginResp = nil
	ic.PluginErr = ""
	ic.HostResp = nil
	ic.HostErr = nil
	ctx = invoke.New(ctx, ic)
	defer func() {
		ic.PluginReq = nil
		ic.PluginResp = nil
		ic.PluginErr = ""
		ic.HostResp = nil
		ic.HostErr = nil
		e.invokeMu.Unlock()
	}()

	results, err = e.pluginCall.Call(ctx, uint64(methodID), uint64(len(payload)))
	resp = ic.PluginResp
	pluginErr = ic.PluginErr

	if err != nil {
		return fmt.Errorf("error while making method %d call: %w", methodID, err)
	}
	if pluginErr != "" {
		return errors.New(pluginErr)
	}

	if len(results) > 0 && results[0] == 1 {
		if fn != nil {
			return fn(resp)
		}
		return nil
	}
	return fmt.Errorf("call to method %d was unsuccessful", methodID)
}

func (e *Runtime) validateContractHandshake() error {
	strict := e.expectedHandshake != nil

	if e.pluginCall == nil {
		if !strict {
			return nil
		}
		return errors.New("contract handshake requested, but plugin does not export __plugin_call")
	}

	abiFn := e.plugin.ExportedFunction(fnABIVersion)
	hashFn := e.plugin.ExportedFunction(fnSchemaHash)
	capsFn := e.plugin.ExportedFunction(fnCapabilities)
	methodsFn := e.plugin.ExportedFunction(fnMethods)
	if abiFn == nil || hashFn == nil {
		if !strict {
			return nil
		}
		return errors.New(
			"contract handshake requested, but plugin does not export handshake functions",
		)
	}

	abiResults, err := abiFn.Call(e.ctx)
	if err != nil {
		if !strict {
			return nil
		}
		return fmt.Errorf("failed to call %s: %w", fnABIVersion, err)
	}
	if len(abiResults) == 0 {
		if !strict {
			return nil
		}
		return fmt.Errorf("plugin %s returned no results", fnABIVersion)
	}
	pluginABIMajor, pluginABIMinor, err := decodeABIVersion(abiResults[0])
	if err != nil {
		if !strict {
			return nil
		}
		return fmt.Errorf("plugin %s returned invalid ABI version: %w", fnABIVersion, err)
	}

	hashResults, err := hashFn.Call(e.ctx)
	if err != nil {
		if !strict {
			return nil
		}
		return fmt.Errorf("failed to call %s: %w", fnSchemaHash, err)
	}
	if len(hashResults) == 0 {
		if !strict {
			return nil
		}
		return fmt.Errorf("plugin %s returned no results", fnSchemaHash)
	}

	hashPtr, hashLen := unpackPtrLenU64(hashResults[0])
	if hashLen != runtimecontract.SchemaHashLen {
		if !strict {
			return nil
		}
		return fmt.Errorf(
			"plugin schema hash has invalid length: got %d want %d",
			hashLen,
			runtimecontract.SchemaHashLen,
		)
	}
	hash, err := runtimememory.TryRead(e.plugin.Memory(), "schema_hash", hashPtr, hashLen)
	if err != nil {
		if !strict {
			return nil
		}
		return err
	}
	var schemaHash [runtimecontract.SchemaHashLen]byte
	copy(schemaHash[:], hash)

	pluginHandshake := runtimecontract.Handshake{
		ABIMajor:   pluginABIMajor,
		ABIMinor:   pluginABIMinor,
		SchemaHash: schemaHash,
	}
	if capsFn != nil {
		capsResults, err := capsFn.Call(e.ctx)
		switch {
		case err != nil:
			if strict {
				return fmt.Errorf("failed to call %s: %w", fnCapabilities, err)
			}
		case len(capsResults) == 0:
			if strict {
				return fmt.Errorf("plugin %s returned no results", fnCapabilities)
			}
		default:
			pluginHandshake.Capabilities = capsResults[0]
		}
	}
	if methodsFn != nil {
		methods, err := e.loadPluginMethods(methodsFn)
		if err != nil {
			return err
		}
		e.pluginMethods = methods
	}
	if e.expectedSchema != nil {
		if methodsFn == nil {
			return errors.New(
				"contract schema validation requested, but plugin does not export __hookr_methods",
			)
		}
		for _, method := range e.expectedSchema.Methods {
			if method.Optional {
				continue
			}
			if _, ok := e.pluginMethods[uint32(method.ID)]; !ok {
				return fmt.Errorf(
					"plugin does not implement required method %s (%d)",
					method.Name,
					method.ID,
				)
			}
		}
	}

	e.pluginHandshake = &pluginHandshake

	var requiredCaps uint64
	if e.expectedHandshake != nil {
		requiredCaps = e.expectedHandshake.Capabilities
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
	var errs []error
	errs = append(
		errs,
		closeNamed(ctx, "plugin", e.plugin),
		closeNamed(ctx, "hookr", e.hookr),
		closeNamed(ctx, "runtime", e.r),
	)
	return errors.Join(errs...)
}

type closer interface {
	Close(context.Context) error
}

func closeNamed(ctx context.Context, name string, c closer) error {
	if c == nil {
		return nil
	}
	if err := c.Close(ctx); err != nil {
		return fmt.Errorf("error closing %s: %w", name, err)
	}
	return nil
}

// DefaultRuntime implements NewRuntime by returning a wazero runtime with WASI
// and AssemblyScript host functions instantiated.
func DefaultRuntime(ctx context.Context) (wazero.Runtime, error) {
	r := newWazeroRuntime(ctx)

	if _, err := instantiateWASI(ctx, r); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}

	// This disables the abort message as no other engines write it.
	envBuilder := r.NewHostModuleBuilder("env")
	newAssemblyscriptExporter().WithAbortMessageDisabled().ExportFunctions(envBuilder)
	if _, err := envBuilder.Instantiate(ctx); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}
	return r, nil
}

func New(ctx context.Context, opts ...Option) (_ *Runtime, err error) {
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
	defer func() {
		if err != nil {
			_ = e.Close(ctx)
		}
	}()

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

func unpackPtrLenU64(encoded uint64) (ptr, dataLen uint32) {
	return api.DecodeU32(encoded >> 32), api.DecodeU32(encoded)
}

func decodeABIVersion(encoded uint64) (major, minor uint16, err error) {
	versionPacked, err := runtimememory.Uint32FromUint64(encoded)
	if err != nil {
		return 0, 0, err
	}
	major = uint16(byte(versionPacked>>24))<<8 | uint16(byte(versionPacked>>16))
	minor = uint16(byte(versionPacked>>8))<<8 | uint16(byte(versionPacked))
	return major, minor, nil
}

func (e *Runtime) currentInvoke() *invoke.Context {
	return nil
}

func (e *Runtime) loadPluginMethods(fn api.Function) (map[uint32]struct{}, error) {
	results, err := callFunction(e.ctx, fn)
	if err != nil {
		return nil, fmt.Errorf("failed to call %s: %w", fnMethods, err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("plugin %s returned no results", fnMethods)
	}
	ptr, dataLen := unpackPtrLenU64(results[0])
	if dataLen == 0 {
		return map[uint32]struct{}{}, nil
	}
	if dataLen%4 != 0 {
		return nil, fmt.Errorf("plugin methods payload has invalid length: got %d", dataLen)
	}
	if dataLen > maxMethodsLen {
		return nil, fmt.Errorf("plugin methods payload too large: got %d", dataLen)
	}
	raw, err := runtimememory.TryRead(e.plugin.Memory(), "methods", ptr, dataLen)
	if err != nil {
		return nil, err
	}
	return decodePluginMethodSet(raw)
}

func decodePluginMethodSet(raw []byte) (map[uint32]struct{}, error) {
	if len(raw) == 0 {
		return map[uint32]struct{}{}, nil
	}
	if len(raw)%4 != 0 {
		return nil, fmt.Errorf("plugin methods payload has invalid length: got %d", len(raw))
	}
	if len(raw) > maxMethodsLen {
		return nil, fmt.Errorf("plugin methods payload too large: got %d", len(raw))
	}
	methods := make(map[uint32]struct{}, len(raw)/4)
	for i := 0; i < len(raw); i += 4 {
		methodID := uint32(raw[i]) |
			uint32(raw[i+1])<<8 |
			uint32(raw[i+2])<<16 |
			uint32(raw[i+3])<<24
		methods[methodID] = struct{}{}
	}
	return methods, nil
}

func (e *Runtime) callWithInvokeContext0(
	ctx context.Context,
	ic *invoke.Context,
	fn api.Function,
) ([]uint64, error) {
	e.invokeMu.Lock()
	defer func() {
		e.invokeMu.Unlock()
	}()
	return fn.Call(invoke.New(ctx, ic))
}

func (e *Runtime) callWithInvokeContext2(
	ctx context.Context,
	ic *invoke.Context,
	fn api.Function,
	arg0 uint64,
	arg1 uint64,
) ([]uint64, error) {
	e.invokeMu.Lock()
	defer func() {
		e.invokeMu.Unlock()
	}()
	return fn.Call(invoke.New(ctx, ic), arg0, arg1)
}
