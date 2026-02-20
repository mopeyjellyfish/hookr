package runtime

import (
	"context"
	"errors"
)

// PluginFuncMethod is a method-ID based plugin function wrapper for ABI v2 calls.
type PluginFuncMethod struct {
	ID uint32
	rt *Runtime
}

func (p PluginFuncMethod) Call(ctx context.Context, input []byte) ([]byte, error) {
	if p.rt == nil {
		return nil, errors.New("engine cannot be nil")
	}
	out, err := p.rt.InvokeMethod(ctx, p.ID, input)
	if err != nil || out == nil {
		return nil, err
	}
	return out, nil
}

// PluginFnMethod creates a method-ID based plugin function wrapper.
func PluginFnMethod(rt *Runtime, methodID uint32) (*PluginFuncMethod, error) {
	if rt == nil {
		return nil, errors.New("engine cannot be nil")
	}
	return &PluginFuncMethod{ID: methodID, rt: rt}, nil
}
