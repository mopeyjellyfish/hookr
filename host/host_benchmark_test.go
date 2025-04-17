package host

import (
	"context"
	"testing"

	"github.com/mopeyjellyfish/hookr/testdata/api"
	"github.com/stretchr/testify/require"
)

func BenchmarkInvoke(b *testing.B) {
	ctx := context.Background()
	hostFn := HostFn("hello", Hello)
	p, err := New(ctx, WithFile(SIMPLE_WASM), WithHostFns(hostFn))
	require.NoError(b, err, "failed to create module")
	require.NotNil(b, p, "plugin should not be nil")
	defer func() {
		err := p.Close(ctx)
		require.NoError(b, err, "failed to close module")
	}()

	payload := &api.EchoRequest{
		Data: "Who controls the past controls the future; who controls the present controls the past.",
	}
	fn, err := PluginFn[*api.EchoRequest, *api.EchoResponse](p, "echo")
	require.NotNil(b, fn, "plugin function should not be nil")
	require.NoError(b, err, "failed to create plugin function")
	d, err := fn.Call(payload) // confirm the call works
	require.NoError(b, err, "failed to call plugin function")
	require.NotNil(b, d, "plugin function should return a value")
	b.ResetTimer() // Reset timer to exclude setup time
	b.Run("Echo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fn.Call(payload)
		}
	})
}

func BenchmarkInvokeBytes(b *testing.B) {
	ctx := context.Background()
	hostFn := HostFnByte("helloByte", HelloByte)
	p, err := New(ctx, WithFile(SIMPLE_WASM), WithHostFns(hostFn))
	require.NoError(b, err, "failed to create module")
	require.NotNil(b, p, "plugin should not be nil")
	defer func() {
		err := p.Close(ctx)
		require.NoError(b, err, "failed to close module")
	}()

	payload := []byte(
		"Who controls the past controls the future; who controls the present controls the past.",
	)
	fn, err := PluginFnByte(p, "echoByte")
	require.NotNil(b, fn, "plugin function should not be nil")
	require.NoError(b, err, "failed to create plugin function")
	d, err := fn.Call(payload) // confirm the call works
	require.NoError(b, err, "failed to call plugin function")
	require.NotNil(b, d, "plugin function should return a value")
	b.ResetTimer() // Reset timer to exclude setup time
	b.Run("Echo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fn.Call(payload)
		}
	})
}
