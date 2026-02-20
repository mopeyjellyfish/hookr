package runtime

import (
	"io"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
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

// WithCallHandlerV2 sets the method-ID based call handler for ABI v2 plugins.
func WithCallHandlerV2(callHandler module.CallHandlerV2) Option {
	return func(e *Runtime) error {
		e.callHandlerV2 = callHandler
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

// WithHostMethodFns sets method-ID based host callbacks callable from ABI v2 plugins.
func WithHostMethodFns(fns ...HostMethod) Option {
	return func(e *Runtime) error {
		for _, fn := range fns {
			id, caller := fn.FnMethod()
			e.RegisterMethod(id, caller)
		}
		return nil
	}
}

// WithContractHandshake enables ABI v2 contract validation during plugin initialization.
func WithContractHandshake(handshake runtimecontract.Handshake) Option {
	return func(e *Runtime) error {
		e.expectedHandshake = &handshake
		return nil
	}
}

// WithContractSchemaHash enables ABI v2 contract validation for the provided schema hash.
func WithContractSchemaHash(schemaHash [runtimecontract.SchemaHashLen]byte) Option {
	return WithContractHandshake(runtimecontract.NewHandshake(schemaHash))
}

// WithContractCapabilities requires the plugin handshake to include the provided capability bits.
func WithContractCapabilities(capabilities uint64) Option {
	return func(e *Runtime) error {
		e.requiredCapabilities = capabilities
		return nil
	}
}

// WithContractSchema validates and configures expected schema + handshake requirements.
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
