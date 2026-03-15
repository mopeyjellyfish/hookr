package runtime

import "context"

// CallFn is a host callback used by method-ID based Hookr plugins.
type CallFn func(ctx context.Context, payload []byte) ([]byte, error)
