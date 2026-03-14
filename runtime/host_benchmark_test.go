package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	methodHello = 1
	methodEcho  = 2
	methodVowel = 3
)

func BenchmarkInvokeMethodBytesVowel(b *testing.B) {
	ctx := context.Background()
	p, err := New(ctx, WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()))
	require.NoError(b, err)
	require.NotNil(b, p)
	defer func() {
		require.NoError(b, p.Close(ctx))
	}()

	payload := []byte(
		"Who controls the past controls the future; who controls the present controls the past.",
	)
	d, err := p.InvokeMethod(context.Background(), methodVowel, payload)
	require.NoError(b, err)
	require.NotNil(b, d)
	b.ResetTimer()
	b.Run("Vowel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = p.InvokeMethod(context.Background(), methodVowel, payload)
		}
	})
}

func BenchmarkInvokeMethodBytesEcho(b *testing.B) {
	ctx := context.Background()
	hostFn := HostFnMethod(methodHello, HelloByte)
	p, err := New(ctx, WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()), WithHostMethodFns(hostFn))
	require.NoError(b, err)
	require.NotNil(b, p)
	defer func() {
		require.NoError(b, p.Close(ctx))
	}()

	payload := []byte(
		"Who controls the past controls the future; who controls the present controls the past.",
	)
	d, err := p.InvokeMethod(context.Background(), methodEcho, payload)
	require.NoError(b, err)
	require.NotNil(b, d)
	b.ResetTimer()
	b.Run("Echo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = p.InvokeMethod(context.Background(), methodEcho, payload)
		}
	})
}
