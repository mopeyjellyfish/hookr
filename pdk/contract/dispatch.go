package contract

import (
	"errors"
	"fmt"
)

var (
	ErrMethodHandlerMissing = errors.New("method handler cannot be nil")
	ErrMethodNotFound       = errors.New("method not found")
)

// Handler is the low-level plugin call shape used by generated wrappers.
type Handler func(payload []byte) ([]byte, error)

// PluginMethod maps a generated method ID/name to an implementation.
type PluginMethod struct {
	ID      MethodID
	Name    string
	Handler Handler
}

// Registry dispatches plugin exports by method ID.
type Registry struct {
	byID   map[MethodID]Handler
	byName map[string]MethodID
}

func NewRegistry(methods ...PluginMethod) (*Registry, error) {
	reg := &Registry{
		byID:   make(map[MethodID]Handler, len(methods)),
		byName: make(map[string]MethodID, len(methods)),
	}
	for _, method := range methods {
		if method.Handler == nil {
			return nil, fmt.Errorf("%w (id=%d name=%q)", ErrMethodHandlerMissing, method.ID, method.Name)
		}
		if _, ok := reg.byID[method.ID]; ok {
			return nil, fmt.Errorf("%w (%d)", ErrMethodIDDuplicate, method.ID)
		}
		if method.Name != "" {
			if _, ok := reg.byName[method.Name]; ok {
				return nil, fmt.Errorf("%w (%s)", ErrMethodNameDuplicate, method.Name)
			}
			reg.byName[method.Name] = method.ID
		}
		reg.byID[method.ID] = method.Handler
	}
	return reg, nil
}

func (r *Registry) Call(id MethodID, payload []byte) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: %d", ErrMethodNotFound, id)
	}
	handler, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrMethodNotFound, id)
	}
	return handler(payload)
}

func (r *Registry) MethodID(name string) (MethodID, bool) {
	if r == nil {
		return 0, false
	}
	id, ok := r.byName[name]
	return id, ok
}
