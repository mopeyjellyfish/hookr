package runtime

import (
	"io"

	"github.com/mopeyjellyfish/hookr/runtime/logger"
	"github.com/mopeyjellyfish/hookr/runtime/module"
)

type Option func(*Runtime) error

// WithFile sets the file for the engine.
func WithFile(file string, opts ...FileOption) Option {
	return func(e *Runtime) error {
		f, err := NewFile(file, opts...)
		if err != nil {
			return err
		}
		e.file = f
		return nil
	}
}

// WithLogger sets the logger for the engine.
func WithLogger(logger logger.Logger) Option {
	return func(e *Runtime) error {
		e.logger = logger
		return nil
	}
}

// WithStdout sets the stdout writer for the engine.
func WithStdout(stdout io.Writer) Option {
	return func(e *Runtime) error {
		e.stdout = stdout
		return nil
	}
}

// WithStderr sets the stderr writer for the engine.
func WithStderr(stderr io.Writer) Option {
	return func(e *Runtime) error {
		e.stderr = stderr
		return nil
	}
}

// WithRandSource sets the random source for the runtime.
func WithRandSource(rand io.Reader) Option {
	return func(e *Runtime) error {
		e.rand = rand
		return nil
	}
}

// WithCallHandler sets the call handler for the engine.
func WithCallHandler(callHandler module.CallHandler) Option {
	return func(e *Runtime) error {
		e.callHandler = callHandler
		return nil
	}
}

// WithHostFns sets the host functions which are callable from the plugin.
func WithHostFns(fns ...HostFunc) Option {
	return func(e *Runtime) error {
		for _, fn := range fns {
			name, caller := fn.Fn()
			e.RegisterFunction(name, caller)
		}
		return nil
	}
}
