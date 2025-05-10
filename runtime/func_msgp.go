package runtime

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/mopeyjellyfish/hookr/testdata/api"
)

type MsgpMarshaler interface {
	MarshalMsg([]byte) ([]byte, error)
}
type MsgpUnmarshaler interface {
	UnmarshalMsg([]byte) ([]byte, error)
}

type PluginFuncMsgp[In MsgpMarshaler, Out MsgpUnmarshaler] struct {
	Name string
	rt   *Runtime
}

func (p *PluginFuncMsgp[In, Out]) Call(ctx context.Context, input In) (Out, error) {
	var dataInput []byte
	var zero Out
	var err error

	if reflect.ValueOf(input).Kind() == reflect.Ptr && reflect.ValueOf(input).IsNil() {
		return zero, errors.New("input cannot be nil")
	}

	dataInput, err = input.MarshalMsg(nil)
	if err != nil {
		return zero, fmt.Errorf("failed to marshal input: %w", err)
	}

	d, err := p.rt.Invoke(ctx, p.Name, dataInput)
	if err != nil {
		return zero, err
	}
	if d == nil {
		return zero, nil
	}

	t := reflect.TypeOf(zero)

	var output Out

	// If it's a pointer type, create a new instance
	if t != nil && t.Kind() == reflect.Ptr {
		// Create a new instance of the element type
		elemType := t.Elem()
		newElem := reflect.New(elemType)
		output = newElem.Interface().(Out)
	} else {
		// For non-pointer types, use the zero value
		output = zero
	}

	_, err = output.UnmarshalMsg(d) // unmarshal the output
	if err != nil {
		return zero, fmt.Errorf("failed to unmarshal output: %w", err)
	}
	return output, nil
}

// Will create a new PluginFunc with the given name and engine.
// This is used to register the function with the host
func PluginFnMsgp[In MsgpMarshaler, Out MsgpUnmarshaler](
	rt *Runtime,
	name string,
) (*PluginFuncMsgp[In, Out], error) {
	if rt == nil {
		return nil, errors.New("engine cannot be nil")
	}
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	pFn := &PluginFuncMsgp[In, Out]{Name: name, rt: rt}
	return pFn, nil
}

// CallFnT is a generic function that accepts and returns specific types
// It handles marshaling/unmarshaling automatically
type CallFnT[In MsgpUnmarshaler, Out MsgpMarshaler] func(ctx context.Context, input In) (Out, error)

// Fn converts a strongly-typed GoFn to a byte-based CallFn allowing WASM plugins to call it.
// This allows for defining a strongly typed function, which can be called from WASM
// that will use a byte slice for input and output for communication.
func Fn[In MsgpUnmarshaler, Out MsgpMarshaler](fn CallFnT[In, Out]) CallFn {
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
		output, err := fn(ctx, input)
		if err != nil {
			return nil, err
		}

		return output.MarshalMsg(nil) // call output.Marshal to marshal the output
	}
}

// HostFunction is a wrapper for CallFnT that provides a name and a function
// This is used to register the function with the host
type HostFunction[In MsgpUnmarshaler, Out MsgpMarshaler] struct {
	name string
	fn   CallFnT[In, Out]
}

func (f *HostFunction[In, Out]) Fn() (name string, fn CallFn) {
	return f.name, Fn(f.fn)
}

func HostFnMsgp[In MsgpUnmarshaler, Out MsgpMarshaler](
	name string,
	fn CallFnT[In, Out],
) *HostFunction[In, Out] {
	return &HostFunction[In, Out]{name: name, fn: fn}
}

var _ HostFunc = &HostFunction[*api.EchoRequest, api.EchoResponse]{} // Compile time check to ensure HostFunction implements HostFunc
