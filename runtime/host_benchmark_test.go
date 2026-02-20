package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/mopeyjellyfish/hookr/testdata/api"
	"github.com/stretchr/testify/require"
)

const (
	v2MethodHello = 1
	v2MethodEcho  = 2
	v2MethodVowel = 3
)

func BenchmarkInvokeBytesVowel(b *testing.B) {
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
	fn, err := PluginFnByte(p, "vowel")
	require.NotNil(b, fn, "plugin function should not be nil")
	require.NoError(b, err, "failed to create plugin function")
	d, err := fn.Call(context.Background(), payload) // confirm the call works
	require.NoError(b, err, "failed to call plugin function")
	require.NotNil(b, d, "plugin function should return a value")
	b.ResetTimer() // Reset timer to exclude setup time
	b.Run("Vowel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fn.Call(context.Background(), payload)
		}
	})
}

func BenchmarkInvokeMsgP(b *testing.B) {
	ctx := context.Background()
	hostFn := HostFnSerial("hello", Hello)
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
	fn, err := PluginFnSerial[*api.EchoRequest, *api.EchoResponse](p, "echo")
	require.NotNil(b, fn, "plugin function should not be nil")
	require.NoError(b, err, "failed to create plugin function")
	d, err := fn.Call(context.Background(), payload) // confirm the call works
	require.NoError(b, err, "failed to call plugin function")
	require.NotNil(b, d, "plugin function should return a value")
	b.ResetTimer() // Reset timer to exclude setup time
	b.Run("Echo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fn.Call(context.Background(), payload)
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
	d, err := fn.Call(context.Background(), payload) // confirm the call works
	require.NoError(b, err, "failed to call plugin function")
	require.NotNil(b, d, "plugin function should return a value")
	b.ResetTimer() // Reset timer to exclude setup time
	b.Run("Echo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fn.Call(context.Background(), payload)
		}
	})
}

func BenchmarkInvokeV2BytesVowel(b *testing.B) {
	ctx := context.Background()
	p, err := New(ctx, WithFile(SIMPLE_V2_WASM))
	require.NoError(b, err, "failed to create module")
	require.NotNil(b, p, "plugin should not be nil")
	defer func() {
		err := p.Close(ctx)
		require.NoError(b, err, "failed to close module")
	}()

	payload := []byte(
		"Who controls the past controls the future; who controls the present controls the past.",
	)
	fn, err := PluginFnMethod(p, v2MethodVowel)
	require.NotNil(b, fn, "plugin function should not be nil")
	require.NoError(b, err, "failed to create plugin function")
	d, err := fn.Call(context.Background(), payload) // confirm the call works
	require.NoError(b, err, "failed to call plugin function")
	require.NotNil(b, d, "plugin function should return a value")
	b.ResetTimer()
	b.Run("Vowel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fn.Call(context.Background(), payload)
		}
	})
}

func BenchmarkInvokeV2BytesEcho(b *testing.B) {
	ctx := context.Background()
	hostFn := HostFnMethod(v2MethodHello, HelloByte)
	p, err := New(ctx, WithFile(SIMPLE_V2_WASM), WithHostMethodFns(hostFn))
	require.NoError(b, err, "failed to create module")
	require.NotNil(b, p, "plugin should not be nil")
	defer func() {
		err := p.Close(ctx)
		require.NoError(b, err, "failed to close module")
	}()

	payload := []byte(
		"Who controls the past controls the future; who controls the present controls the past.",
	)
	fn, err := PluginFnMethod(p, v2MethodEcho)
	require.NotNil(b, fn, "plugin function should not be nil")
	require.NoError(b, err, "failed to create plugin function")
	d, err := fn.Call(context.Background(), payload)
	require.NoError(b, err, "failed to call plugin function")
	require.NotNil(b, d, "plugin function should return a value")
	b.ResetTimer()
	b.Run("Echo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fn.Call(context.Background(), payload)
		}
	})
}

func BenchmarkInvokeV2BytesEchoDirect(b *testing.B) {
	ctx := context.Background()
	hostHandler := func(ctx context.Context, methodID uint32, input []byte) ([]byte, error) {
		switch methodID {
		case v2MethodHello:
			return HelloByte(ctx, input)
		default:
			return nil, errors.New("unknown method")
		}
	}

	p, err := New(ctx, WithFile(SIMPLE_V2_WASM), WithCallHandlerV2(hostHandler))
	require.NoError(b, err, "failed to create module")
	require.NotNil(b, p, "plugin should not be nil")
	defer func() {
		err := p.Close(ctx)
		require.NoError(b, err, "failed to close module")
	}()

	payload := []byte(
		"Who controls the past controls the future; who controls the present controls the past.",
	)
	fn, err := PluginFnMethod(p, v2MethodEcho)
	require.NotNil(b, fn, "plugin function should not be nil")
	require.NoError(b, err, "failed to create plugin function")
	d, err := fn.Call(context.Background(), payload)
	require.NoError(b, err, "failed to call plugin function")
	require.NotNil(b, d, "plugin function should return a value")
	b.ResetTimer()
	b.Run("Echo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fn.Call(context.Background(), payload)
		}
	})
}
