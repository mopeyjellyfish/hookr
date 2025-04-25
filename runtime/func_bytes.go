package runtime

import (
	"context"
	"errors"
)

// PluginFuncByte is a function that takes a byte slice and returns a byte slice
type PluginFuncByte struct {
	Name string
	rt   *Runtime
}

// Call takes an input of type In and returns an output of type Out
// These will always be []byte in and out.
func (p PluginFuncByte) Call(ctx context.Context, input []byte) ([]byte, error) {
	out, err := p.rt.Invoke(ctx, p.Name, input)
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
	rt *Runtime,
	name string,
) (*PluginFuncByte, error) {
	if rt == nil {
		return nil, errors.New("engine cannot be nil")
	}
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	pFn := &PluginFuncByte{Name: name, rt: rt}
	return pFn, nil
}

// HostFuncByte is a wrapper for CallFn that provides a name and a function
type HostFuncByte struct {
	name string
	fn   CallFn
}

// Fn returns the name and function to be called
// This is used to register the function with the host
func (f HostFuncByte) Fn() (name string, fn CallFn) {
	return f.name, f.fn
}

// HostFnByte is a function that takes a byte slice and returns a byte slice
// It is used to call a function that takes a byte slice and returns a byte slice
func HostFnByte(name string, fn CallFn) *HostFuncByte {
	return &HostFuncByte{
		name: name,
		fn:   fn,
	}
}

var (
	_ HostFunc                   = HostFuncByte{} // Compile time check to ensure HostFuncByte implements HostFunc
	_ PluginFunc[[]byte, []byte] = PluginFuncByte{}
)
