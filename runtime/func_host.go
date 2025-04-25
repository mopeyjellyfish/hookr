package runtime

import (
	"context"
)

// CallFn is a function to be called by the CallHandler.
// It takes a context, a payload, and a serializer.
// It returns a byte slice and an error.
type CallFn func(ctx context.Context, payload []byte) ([]byte, error)

type CallFns = map[string]CallFn

type HostFunc interface {
	// Fn returns the name and function to be called
	Fn() (name string, fn CallFn)
}
