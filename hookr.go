package hookr

import (
	"context"
	"errors"
	"io"

	"github.com/mopeyjellyfish/hookr/host"
	"github.com/mopeyjellyfish/hookr/host/logger"
	"github.com/mopeyjellyfish/hookr/host/module"
	"github.com/tinylib/msgp/msgp"
)

// Plugin is an interface that defines the methods for a plugin.
// It includes methods for invoking the plugin, closing it, and initializing it.
type Plugin interface {
	Invoke(ctx context.Context, operation string, payload []byte) ([]byte, error)
	Close(ctx context.Context) error
	RegisterFunction(name string, fn CallFn)
}

// CallFn is a function to be called by the CallHandler.
// It takes a context, a payload, and returns a byte slice and an error.
type CallFn func(ctx context.Context, payload []byte) ([]byte, error)

// CallFnT is a generic function that accepts and returns specific types.
// It handles marshaling/unmarshaling automatically.
type CallFnMsgP[In msgp.Unmarshaler, Out msgp.Marshaler] func(ctx context.Context, input In) (Out, error)

// HostFunc is an interface for host functions that can be registered with a plugin.
type HostFunc interface {
	// Fn returns the name and function to be called
	Fn() (name string, fn CallFn)
}

// PluginFunc is an interface for plugin functions that can be called from the host.
type PluginFunc[In, Out any] interface {
	// Call takes an input of type In and returns an output of type Out
	Call(input In) (Out, error)
}

// Option is a function that configures a Plugin.
type Option host.Option

// WithFile returns an Option that configures the Plugin to use the specified WASM file.
func WithFile(path string, opts ...host.FileOption) Option {
	return Option(host.WithFile(path, opts...))
}

// WithHash returns a FileOption that configures hash verification for a WASM file.
func WithHash(hash string) host.FileOption {
	return host.WithHash(hash)
}

// WithHasher returns a FileOption that configures the hash algorithm.
func WithHasher(hasher host.Hasher) host.FileOption {
	return host.WithHasher(hasher)
}

// WithLogger returns an Option that configures the logger for the Plugin.
func WithLogger(l logger.Logger) Option {
	return Option(host.WithLogger(l))
}

// WithStderr returns an Option that configures stderr for the Plugin.
func WithStderr(w io.Writer) Option {
	return Option(host.WithStderr(w))
}

// WithStdout returns an Option that configures stdout for the Plugin.
func WithStdout(w io.Writer) Option {
	return Option(host.WithStdout(w))
}

// WithRandSource returns an Option that configures the random source for the Plugin.
func WithRandSource(r io.Reader) Option {
	return Option(host.WithRandSource(r))
}

// WithCallHandler returns an Option that configures the call handler for the Plugin.
func WithCallHandler(h module.CallHandler) Option {
	return Option(host.WithCallHandler(h))
}

// WithHostFns returns an Option that configures the host functions for the Plugin.
func WithHostFns(fns ...HostFunc) Option {
	// Convert our HostFunc interface to host.HostFunc
	hostFns := make([]host.HostFunc, len(fns))
	for i, fn := range fns {
		hostFns[i] = hostFuncWrapper{fn}
	}
	return Option(host.WithHostFns(hostFns...))
}

// hostFuncWrapper adapts our HostFunc to host.HostFunc
type hostFuncWrapper struct {
	hf HostFunc
}

func (w hostFuncWrapper) Fn() (name string, fn host.CallFn) {
	name, callFn := w.hf.Fn()
	return name, host.CallFn(callFn)
}

// HostFn creates a new typed host function.
func HostFn[In msgp.Unmarshaler, Out msgp.Marshaler](
	name string,
	fn CallFnMsgP[In, Out],
) HostFunc {
	hostFn := host.HostFn(name, host.CallFnT[In, Out](fn))
	return hostFuncAdapter{hostFn}
}

// hostFuncAdapter adapts host.HostFunc to our HostFunc
type hostFuncAdapter struct {
	hf host.HostFunc
}

func (a hostFuncAdapter) Fn() (name string, fn CallFn) {
	name, hostFn := a.hf.Fn()
	return name, CallFn(hostFn)
}

// HostFnByte creates a new byte-based host function.
func HostFnByte(name string, fn CallFn) HostFunc {
	hostFn := host.HostFnByte(name, host.CallFn(fn))
	return hostFuncAdapter{hostFn}
}

// Fn converts a typed function to a byte-based CallFn.
func Fn[In msgp.Unmarshaler, Out msgp.Marshaler](fn CallFnMsgP[In, Out]) CallFn {
	return CallFn(host.Fn(host.CallFnT[In, Out](fn)))
}

// PluginFn creates a typed plugin function.
func PluginFn[In msgp.Marshaler, Out msgp.Unmarshaler](
	p Plugin,
	name string,
) (PluginFunc[In, Out], error) {
	// Need to cast the Plugin to *host.Engine to use host.PluginFn
	if engine, ok := p.(*engineWrapper); ok {
		fn, err := host.PluginFn[In, Out](engine.engine, name)
		if err != nil {
			return nil, err
		}
		return pluginFuncWrapper[In, Out]{fn}, nil
	}
	return nil, errors.New("invalid plugin type")
}

// pluginFuncWrapper adapts host.PluginFunc to our PluginFunc
type pluginFuncWrapper[In, Out any] struct {
	pf host.PluginFunc[In, Out]
}

func (w pluginFuncWrapper[In, Out]) Call(input In) (Out, error) {
	return w.pf.Call(input)
}

// PluginFnByte creates a byte-based plugin function.
func PluginFnByte(p Plugin, name string) (PluginFunc[[]byte, []byte], error) {
	if engine, ok := p.(*engineWrapper); ok {
		fn, err := host.PluginFnByte(engine.engine, name)
		if err != nil {
			return nil, err
		}
		return pluginFuncWrapper[[]byte, []byte]{fn}, nil
	}
	return nil, errors.New("invalid plugin type")
}

// engineWrapper wraps a host.Engine to implement our Plugin interface
type engineWrapper struct {
	engine *host.Engine
}

func (e *engineWrapper) Invoke(ctx context.Context, operation string, payload []byte) ([]byte, error) {
	return e.engine.Invoke(ctx, operation, payload)
}

func (e *engineWrapper) Close(ctx context.Context) error {
	return e.engine.Close(ctx)
}

func (e *engineWrapper) RegisterFunction(name string, fn CallFn) {
	e.engine.RegisterFunction(name, host.CallFn(fn))
}

// NewPlugin creates a new Plugin with the given options.
func NewPlugin(ctx context.Context, opts ...Option) (Plugin, error) {
	// Convert our options to host.Option
	hostOpts := make([]host.Option, len(opts))
	for i, opt := range opts {
		hostOpts[i] = host.Option(opt)
	}

	engine, err := host.New(ctx, hostOpts...)
	if err != nil {
		return nil, err
	}

	return &engineWrapper{engine}, nil
}
