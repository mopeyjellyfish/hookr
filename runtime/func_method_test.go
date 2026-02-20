package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginFnMethodBadParams(t *testing.T) {
	fn, err := PluginFnMethod(nil, 1)
	require.Error(t, err, "expected error when engine is nil")
	require.Nil(t, fn, "function should be nil on error")
}

func TestPluginFuncMethodNilRuntime(t *testing.T) {
	p := PluginFuncMethod{}
	data, err := p.Call(context.Background(), nil)
	require.Error(t, err, "expected error with nil runtime")
	require.Nil(t, data, "data should be nil on error")
}
