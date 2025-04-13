package host

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/tinylib/msgp/msgp"
)

type PluginFunc[In msgp.Marshaler, Out msgp.Unmarshaler] struct {
	Name string
	e    *Engine
}

func (p *PluginFunc[In, Out]) Call(input In) (Out, error) {
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

	d, err := p.e.Invoke(p.e.ctx, p.Name, dataInput)
	if err != nil {
		return zero, fmt.Errorf("error invoking plugin: %w", err)
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
) (*PluginFunc[In, Out], error) {
	if e == nil {
		return nil, errors.New("engine cannot be nil")
	}
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	pFn := &PluginFunc[In, Out]{Name: name, e: e}
	return pFn, nil
}
