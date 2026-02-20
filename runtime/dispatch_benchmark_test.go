package runtime

import (
	"context"
	"errors"
	"testing"
)

func BenchmarkDispatchByName(b *testing.B) {
	rt := &Runtime{}
	rt.RegisterFunction("echo", func(ctx context.Context, payload []byte) ([]byte, error) {
		return payload, nil
	})

	payload := []byte("benchmark-payload")
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rt.fnHandler(ctx, "echo", payload)
	}
}

func BenchmarkDispatchByMethodID(b *testing.B) {
	rt := &Runtime{}
	rt.RegisterMethod(1, func(ctx context.Context, payload []byte) ([]byte, error) {
		return payload, nil
	})

	payload := []byte("benchmark-payload")
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rt.fnHandlerV2(ctx, 1, payload)
	}
}

func BenchmarkDispatchByMethodIDDirect(b *testing.B) {
	rt := &Runtime{}
	errUnknown := errors.New("unknown method")
	rt.callHandlerV2 = func(ctx context.Context, methodID uint32, payload []byte) ([]byte, error) {
		switch methodID {
		case 1:
			return payload, nil
		default:
			return nil, errUnknown
		}
	}

	payload := []byte("benchmark-payload")
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rt.fnHandlerV2(ctx, 1, payload)
	}
}
