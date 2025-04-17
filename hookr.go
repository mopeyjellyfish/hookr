package hookr

import (
	"context"

	"github.com/mopeyjellyfish/hookr/host"
)

func NewPlugin(ctx context.Context, opts ...host.Option) (*host.Engine, error) {
	return host.New(ctx, opts...)
}
