package host

import (
	"context"
	"reflect"

	"github.com/tinylib/msgp/msgp"
)

// CallFn is a function to be called by the CallHandler.
// It takes a context, a payload, and a serializer.
// It returns a byte slice and an error.
type CallFn func(ctx context.Context, payload []byte) ([]byte, error)

// GoFn is a generic function that accepts and returns specific types
// It handles marshaling/unmarshaling automatically
type GoFn[In msgp.Unmarshaler, Out msgp.Marshaler] func(input In) (Out, error)

type HostFunc interface {
	// Fn returns the name and function to be called
	Fn() (name string, fn CallFn)
}

// HostFunction is a wrapper for GoFn that provides a name and a function
// This is used to register the function with the host
type HostFunction[In msgp.Unmarshaler, Out msgp.Marshaler] struct {
	name string
	fn   GoFn[In, Out]
}

func (f *HostFunction[In, Out]) Fn() (name string, fn CallFn) {
	return f.name, Fn(f.fn)
}

func HostFn[In msgp.Unmarshaler, Out msgp.Marshaler](
	name string,
	fn GoFn[In, Out],
) *HostFunction[In, Out] {
	return &HostFunction[In, Out]{name: name, fn: fn}
}

// Fn converts a strongly-typed GoFn to a byte-based CallFn allowing WASM plugins to call it.
// This allows for defining a strongly typed function, which can be called from WASM
// that will use a byte slice for input and output for communication.
func Fn[In msgp.Unmarshaler, Out msgp.Marshaler](fn GoFn[In, Out]) CallFn {
	return func(ctx context.Context, payload []byte) ([]byte, error) {
		// Unmarshal the input from bytes
		var input In

		var zero In
		t := reflect.TypeOf(zero)

		// If it's a pointer type, create a new instance
		if t != nil && t.Kind() == reflect.Ptr {
			// Create a new instance of the element type
			elemType := t.Elem()
			newElem := reflect.New(elemType)
			input = newElem.Interface().(In)
		} else {
			// For non-pointer types, use the zero value
			input = zero
		}

		_, err := input.UnmarshalMsg(payload) // unmarshal the input
		if err != nil {
			return nil, err
		}

		// Call the go function
		output, err := fn(input)
		if err != nil {
			return nil, err
		}

		return output.MarshalMsg(nil) // call output.Marshal to marshal the output
	}
}

type CallFns = map[string]CallFn
