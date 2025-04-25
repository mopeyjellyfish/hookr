package runtime

import (
	"context"
)

type PluginFunc[In, Out any] interface {
	// Call takes an input of type In and returns an output of type Out
	// It returns an error if the call fails
	Call(ctx context.Context, input In) (Out, error)
}
