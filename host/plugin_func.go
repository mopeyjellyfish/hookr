package host

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/tinylib/msgp/msgp"
)

type PluginFunc[In, Out any] interface {
	// Call takes an input of type In and returns an output of type Out
	// It returns an error if the call fails
	Call(input In) (Out, error)
}

type PluginFuncT[In msgp.Marshaler, Out msgp.Unmarshaler] struct {
	Name string
	e    *Engine
	ctx  context.Context
}

func (p *PluginFuncT[In, Out]) Call(input In) (Out, error) {
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

	d, err := p.e.Invoke(p.ctx, p.Name, dataInput)
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
func PluginFn[In msgp.Marshaler, Out msgp.Unmarshaler](
	e *Engine,
	name string,
) (PluginFunc[In, Out], error) {
	if e == nil {
		return nil, errors.New("engine cannot be nil")
	}
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	pFn := &PluginFuncT[In, Out]{Name: name, e: e, ctx: e.ctx}
	return pFn, nil
}

// PluginFuncByte is a function that takes a byte slice and returns a byte slice
type PluginFuncByte struct {
	Name string
	e    *Engine
	ctx  context.Context
}

// Call takes an input of type In and returns an output of type Out
// These will always be []byte in and out.
func (p PluginFuncByte) Call(input []byte) ([]byte, error) {
	out, err := p.e.Invoke(p.ctx, p.Name, input)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	return out, nil
}

// PluginFnByte creates a new PluginFunc with the given name and engine.
// This will always be a byte slice in and out.
func PluginFnByte(
	e *Engine,
	name string,
) (PluginFunc[[]byte, []byte], error) {
	if e == nil {
		return nil, errors.New("engine cannot be nil")
	}
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	pFn := &PluginFuncByte{Name: name, e: e, ctx: e.ctx}
	return pFn, nil
}

var _ PluginFunc[[]byte, []byte] = PluginFuncByte{}
