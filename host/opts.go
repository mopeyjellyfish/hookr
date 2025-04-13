package host

import (
	"io"

	"github.com/mopeyjellyfish/hookr/host/logger"
	"github.com/mopeyjellyfish/hookr/host/module"
)

type HookrOption func(*Engine) error

// WithFile sets the file for the engine.
func WithFile(file string, opts ...FileOption) HookrOption {
	return func(e *Engine) error {
		f, err := NewFile(file, opts...)
		if err != nil {
			return err
		}
		e.file = f
		return nil
	}
}

// WithLogger sets the logger for the engine.
func WithLogger(logger logger.Logger) HookrOption {
	return func(e *Engine) error {
		e.logger = logger
		return nil
	}
}

// WithStdout sets the stdout writer for the engine.
func WithStdout(stdout io.Writer) HookrOption {
	return func(e *Engine) error {
		e.stdout = stdout
		return nil
	}
}

// WithStderr sets the stderr writer for the engine.
func WithStderr(stderr io.Writer) HookrOption {
	return func(e *Engine) error {
		e.stderr = stderr
		return nil
	}
}

// WithRandSource sets the random source for the runtime.
func WithRandSource(rand io.Reader) HookrOption {
	return func(e *Engine) error {
		e.rand = rand
		return nil
	}
}

// WithNewRuntime sets the runtime for the engine.
func WithNewRuntime(newRuntime NewRuntime) HookrOption {
	return func(e *Engine) error {
		e.newRuntime = newRuntime
		return nil
	}
}

// WithCallHandler sets the call handler for the engine.
func WithCallHandler(callHandler module.CallHandler) HookrOption {
	return func(e *Engine) error {
		e.callHandler = callHandler
		return nil
	}
}

// WithHostFns sets the host functions which are callable from the plugin.
func WithHostFns(fns ...HostFunc) HookrOption {
	return func(e *Engine) error {
		for _, fn := range fns {
			name, caller := fn.Fn()
			e.RegisterFunction(name, caller)
		}
		return nil
	}
}
