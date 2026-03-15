package tickloop

import (
	"context"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	tickloophookr "github.com/mopeyjellyfish/hookr/testdata/contracts/tickloop/gen/tickloophookr"
)

func BenchmarkTickLoopTick(b *testing.B) {
	wasmPath := buildPluginBenchmark(b, "plugin")
	rt := openRuntimeBenchmark(b, wasmPath)
	defer closeRuntimeBenchmark(b, rt)

	req := &tickloophookr.TickRequestT{
		Tick:      1,
		DtMicros:  16666,
		StateHash: 1000,
		Events:    3,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.Tick = uint64(i + 1)
		if _, err := rt.Tick(context.Background(), req); err != nil {
			b.Fatalf("tick: %v", err)
		}
	}
}

func BenchmarkTickLoopTickView(b *testing.B) {
	wasmPath := buildPluginBenchmark(b, "plugin")
	rt := openRuntimeBenchmark(b, wasmPath)
	defer closeRuntimeBenchmark(b, rt)

	req := &tickloophookr.TickRequestT{
		Tick:      1,
		DtMicros:  16666,
		StateHash: 1000,
		Events:    3,
	}

	var nextStateHash uint64
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.Tick = uint64(i + 1)
		if err := rt.TickView(context.Background(), req, func(resp *tickloophookr.TickResponse) error {
			nextStateHash = resp.NextStateHash()
			return nil
		}); err != nil {
			b.Fatalf("tick view: %v", err)
		}
	}
	if nextStateHash == 0 {
		b.Fatal("expected non-zero next state hash")
	}
}

func BenchmarkTickLoopWarmup(b *testing.B) {
	wasmPath := buildPluginBenchmark(b, "plugin")
	rt := openRuntimeBenchmark(b, wasmPath)
	defer closeRuntimeBenchmark(b, rt)

	req := &tickloophookr.WarmupRequestT{TargetTicks: 64}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := rt.Warmup(context.Background(), req); err != nil {
			b.Fatalf("warmup: %v", err)
		}
	}
}

func buildPluginBenchmark(b *testing.B, pluginDir string) string {
	b.Helper()
	_, filename, _, ok := goruntime.Caller(0)
	if !ok {
		b.Fatalf("resolve benchmark file path")
	}
	contractDir := filepath.Dir(filename)
	outFile := filepath.Join(b.TempDir(), pluginDir+".wasm")

	cfg := buildkit.DefaultConfig()
	cfg.PluginPath = filepath.Join(contractDir, pluginDir)
	cfg.OutputPath = outFile
	if err := buildkit.Build(cfg); err != nil {
		b.Fatalf("build plugin %s: %v", pluginDir, err)
	}
	return outFile
}

func openRuntimeBenchmark(b *testing.B, wasmPath string) *tickloophookr.Runtime {
	b.Helper()
	rt, err := tickloophookr.Open(context.Background(), tickloophookr.Config{
		PluginPath:  wasmPath,
		FileOptions: []hookrruntime.FileOption{hookrruntime.WithAllowUnsigned()},
		Host: tickloophookr.Host{
			Rng: testHost{},
		},
	})
	if err != nil {
		b.Fatalf("open runtime: %v", err)
	}
	return rt
}

func closeRuntimeBenchmark(b *testing.B, rt *tickloophookr.Runtime) {
	b.Helper()
	if err := rt.Close(context.Background()); err != nil {
		b.Fatalf("close runtime: %v", err)
	}
}
