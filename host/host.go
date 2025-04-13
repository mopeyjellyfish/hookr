package host

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mopeyjellyfish/hookr/host/invoke"
	"github.com/mopeyjellyfish/hookr/host/logger"
	"github.com/mopeyjellyfish/hookr/host/module"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/assemblyscript"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// NewRuntime returns a new wazero runtime which is called when the New method
// on hookr.Engine is called. The result is closed upon wapc.Module Close.
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

type Engine struct {
	newRuntime  NewRuntime
	ctx         context.Context
	file        *File
	logger      logger.Logger
	stderr      io.Writer
	stdout      io.Writer
	rand        io.Reader
	callHandler module.CallHandler

	hostFns    CallFns
	pluginCall api.Function
	moduleName string
	r          wazero.Runtime
	config     wazero.ModuleConfig
	hookr      api.Module
	plugin     api.Module
	compiled   wazero.CompiledModule
}

// Will initialize the wazero runtime
func (e *Engine) InitRuntime() error {
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
func (e *Engine) InitHookr() error {
	if e.r == nil {
		return errors.New("runtime not initialized")
	}
	hookr, err := module.New(e.ctx, e.r, e.fnHandler, e.logger)
	if err != nil {
		return err
	}
	e.hookr = hookr
	return nil
}

func (e *Engine) fnHandler(ctx context.Context, operation string, payload []byte) ([]byte, error) {
	if e.callHandler != nil {
		return e.callHandler(ctx, operation, payload)
	}
	if fn, ok := e.hostFns[operation]; ok {
		return fn(ctx, payload)
	}
	return nil, fmt.Errorf("no handler registered for operation '%s'", operation)
}

// RegisterFunction registers a host function with the engine.
func (e *Engine) RegisterFunction(name string, fn CallFn) {
	if e.hostFns == nil {
		e.hostFns = make(CallFns)
	}
	e.hostFns[name] = fn
}

// InitConfig initializes the wazero module config with the default settings.
func (e *Engine) InitConfig() {
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
func (e *Engine) Init() error {
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
func (e *Engine) MemorySize() uint32 {
	if e.plugin == nil {
		return 0
	}
	return e.plugin.Memory().Size()
}

// Compile compiles the plugin module. It must be called after the runtime is
// initialized and before the module is instantiated.
func (e *Engine) Compile() error {
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
func (e *Engine) Instantiate() error {
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
			ictx := invoke.New(e.ctx, &ic)
			if _, err := exportedFunc.Call(ictx); err != nil {
				if exitErr, ok := err.(*sys.ExitError); ok {
					if exitErr.ExitCode() != 0 {
						err := module.Close(e.ctx)
						if err != nil {
							return fmt.Errorf("error closing module: %w", err)
						}

						return fmt.Errorf("module closed with exit_code(%d)", exitErr.ExitCode())
					}
				} else {
					err := module.Close(e.ctx)
					if err != nil {
						return fmt.Errorf("error closing module: %w", err)
					}

					return fmt.Errorf("error calling %s: %v", f, err)
				}
			}
		}
	}

	e.plugin = module

	if e.pluginCall = module.ExportedFunction(fnPluginCall); e.pluginCall == nil {
		_ = e.plugin.Close(e.ctx)
		return fmt.Errorf("module %s didn't export function %s", e.moduleName, fnPluginCall)
	}

	return nil
}

// Invoke calls the plugin function with the given operation and payload.
func (e *Engine) Invoke(ctx context.Context, operation string, payload []byte) ([]byte, error) {
	if e.plugin == nil {
		return nil, errors.New("plugin not initialized")
	}

	ic := invoke.Context{Operation: operation, PluginReq: payload}
	ctx = invoke.New(ctx, &ic)

	results, err := e.pluginCall.Call(ctx, uint64(len(operation)), uint64(len(payload)))
	if err != nil {
		return nil, fmt.Errorf("error invoking guest: %w", err)
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

func (e *Engine) Close(ctx context.Context) error {
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

func New(ctx context.Context, opts ...HookrOption) (*Engine, error) {
	e := &Engine{
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
