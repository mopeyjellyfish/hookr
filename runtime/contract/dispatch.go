package contract

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrMethodHandlerMissing = errors.New("method handler cannot be nil")
	ErrMethodNotFound       = errors.New("method not found")
)

// Handler is the low-level runtime call shape used by generated wrappers.
type Handler func(ctx context.Context, payload []byte) ([]byte, error)

// HostMethod maps a generated method ID/name to an implementation.
type HostMethod struct {
	ID      MethodID
	Name    string
	Handler Handler
}

// HostRegistry dispatches host callbacks by method ID.
type HostRegistry struct {
	byID   map[MethodID]Handler
	byName map[string]MethodID
}

func NewHostRegistry(methods ...HostMethod) (*HostRegistry, error) {
	reg := &HostRegistry{
		byID:   make(map[MethodID]Handler, len(methods)),
		byName: make(map[string]MethodID, len(methods)),
	}
	for _, method := range methods {
		if method.Handler == nil {
			return nil, fmt.Errorf(
				"%w (id=%d name=%q)",
				ErrMethodHandlerMissing,
				method.ID,
				method.Name,
			)
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

func (r *HostRegistry) Call(ctx context.Context, id MethodID, payload []byte) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: %d", ErrMethodNotFound, id)
	}
	handler, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrMethodNotFound, id)
	}
	return handler(ctx, payload)
}

func (r *HostRegistry) MethodID(name string) (MethodID, bool) {
	if r == nil {
		return 0, false
	}
	id, ok := r.byName[name]
	return id, ok
}
