package runtime

import (
	"context"
	"testing"
)

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
		_, _ = rt.methodHandler(ctx, 1, payload)
	}
}
