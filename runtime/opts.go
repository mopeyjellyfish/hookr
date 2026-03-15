package runtime

import (
	"io"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/mopeyjellyfish/hookr/runtime/logger"
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
func WithLogger(logFn logger.Logger) Option {
	return func(e *Runtime) error {
		e.logger = logFn
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

// WithHostMethodFns sets method-ID based host callbacks callable from method-ID plugins.
func WithHostMethodFns(fns ...HostMethod) Option {
	return func(e *Runtime) error {
		for _, fn := range fns {
			id, caller := fn.FnMethod()
			e.RegisterMethod(id, caller)
		}
		return nil
	}
}

// WithContractSchema validates and configures expected schema + handshake requirements.
//
//nolint:gocritic // Keep the public API value-based to avoid forcing pointer ownership on callers.
func WithContractSchema(schema runtimecontract.Schema) Option {
	return func(e *Runtime) error {
		if err := schema.Validate(); err != nil {
			return err
		}
		e.expectedSchema = &schema
		handshake := runtimecontract.NewHandshake(schema.SchemaHash)
		handshake.Capabilities = schema.Capabilities
		e.expectedHandshake = &handshake
		return nil
	}
}
