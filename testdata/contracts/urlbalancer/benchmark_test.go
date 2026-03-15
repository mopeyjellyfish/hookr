package urlbalancer

import (
	"context"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	urlbalancerhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/urlbalancer/gen/urlbalancerhookr"
)

func BenchmarkURLBalancerBalance(b *testing.B) {
	wasmPath := buildURLBalancerBenchmarkPlugin(b)
	rt := openURLBalancerBenchmarkRuntime(b, wasmPath)
	defer closeURLBalancerBenchmarkRuntime(b, rt)

	req := &urlbalancerhookr.BalanceRequestT{
		Url:   "https://example.com/api?q=1",
		Nodes: []string{"node-a", "node-b", "node-c"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := rt.Balance(context.Background(), req); err != nil {
			b.Fatalf("balance: %v", err)
		}
	}
}

func buildURLBalancerBenchmarkPlugin(b *testing.B) string {
	b.Helper()
	_, filename, _, ok := goruntime.Caller(0)
	if !ok {
		b.Fatalf("resolve benchmark file path")
	}
	contractDir := filepath.Dir(filename)
	outFile := filepath.Join(b.TempDir(), "urlbalancer.wasm")

	cfg := buildkit.DefaultConfig()
	cfg.PluginPath = filepath.Join(contractDir, "plugin")
	cfg.OutputPath = outFile
	if err := buildkit.Build(cfg); err != nil {
		b.Fatalf("build plugin: %v", err)
	}
	return outFile
}

func openURLBalancerBenchmarkRuntime(b *testing.B, wasmPath string) *urlbalancerhookr.Runtime {
	b.Helper()
	rt, err := urlbalancerhookr.Open(context.Background(), urlbalancerhookr.Config{
		PluginPath:  wasmPath,
		FileOptions: []hookrruntime.FileOption{hookrruntime.WithAllowUnsigned()},
		Host:        testHost{},
	})
	if err != nil {
		b.Fatalf("open runtime: %v", err)
	}
	return rt
}

func closeURLBalancerBenchmarkRuntime(b *testing.B, rt *urlbalancerhookr.Runtime) {
	b.Helper()
	if err := rt.Close(context.Background()); err != nil {
		b.Fatalf("close runtime: %v", err)
	}
}
