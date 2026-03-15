package tickloop

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	tickloophookr "github.com/mopeyjellyfish/hookr/testdata/contracts/tickloop/gen/tickloophookr"
	goruntime "runtime"
)

type testHost struct{}

func (testHost) Int(_ context.Context, req *tickloophookr.RngIntRequestT) (*tickloophookr.RngIntResponseT, error) {
	if req == nil {
		return &tickloophookr.RngIntResponseT{Value: 0}, nil
	}
	if req.Min < 0 && req.Max >= 2 {
		return &tickloophookr.RngIntResponseT{Value: 2}, nil
	}
	if req.Min < 0 && req.Max <= 1 {
		return &tickloophookr.RngIntResponseT{Value: -1}, nil
	}
	return &tickloophookr.RngIntResponseT{Value: req.Min}, nil
}

func TestTickLoopE2E_WithWarmupPlugin(t *testing.T) {
	t.Parallel()

	wasmPath := buildPlugin(t, "plugin")
	rt := openRuntime(t, wasmPath)
	defer closeRuntime(t, rt)
	if !rt.SupportsWarmup() {
		t.Fatalf("expected plugin to report Warmup support")
	}

	info, err := rt.GetInfo(context.Background(), &tickloophookr.EmptyT{})
	if err != nil {
		t.Fatalf("get info: %v", err)
	}
	if info.Name != "tickloop" {
		t.Fatalf("info name = %q, want tickloop", info.Name)
	}

	tickResp, err := rt.Tick(context.Background(), &tickloophookr.TickRequestT{
		Tick:      99,
		DtMicros:  16666,
		StateHash: 1000,
		Events:    3,
	})
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if tickResp.NextStateHash != 1005 {
		t.Fatalf("next state hash = %d, want 1005", tickResp.NextStateHash)
	}
	if tickResp.Actions != 5 {
		t.Fatalf("actions = %d, want 5", tickResp.Actions)
	}
	if tickResp.JitterBucket != 2 {
		t.Fatalf("jitter bucket = %d, want 2", tickResp.JitterBucket)
	}
	if !tickResp.ContinueRun {
		t.Fatalf("continue_run = false, want true")
	}
	if tickResp.Note == "" {
		t.Fatalf("expected non-empty tick note")
	}

	var viewStateHash uint64
	err = rt.TickView(context.Background(), &tickloophookr.TickRequestT{
		Tick:      100,
		DtMicros:  16666,
		StateHash: 1005,
		Events:    1,
	}, func(resp *tickloophookr.TickResponse) error {
		viewStateHash = resp.NextStateHash()
		if !resp.ContinueRun() {
			t.Fatal("continue_run = false, want true")
		}
		if string(resp.Note()) == "" {
			t.Fatal("expected non-empty tick note")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tick view: %v", err)
	}
	if viewStateHash != 1008 {
		t.Fatalf("tick view next state hash = %d, want 1008", viewStateHash)
	}

	warmupResp, err := rt.Warmup(context.Background(), &tickloophookr.WarmupRequestT{
		TargetTicks: 64,
	})
	if err != nil {
		t.Fatalf("warmup: %v", err)
	}
	if !warmupResp.Ready {
		t.Fatalf("warmup ready = false, want true")
	}
	if warmupResp.RecommendedBatch != 32 {
		t.Fatalf("recommended batch = %d, want 32", warmupResp.RecommendedBatch)
	}
}

func TestTickLoopE2E_WithoutWarmupPlugin(t *testing.T) {
	t.Parallel()

	wasmPath := buildPlugin(t, "plugin_nowarmup")
	rt := openRuntime(t, wasmPath)
	defer closeRuntime(t, rt)
	if rt.SupportsWarmup() {
		t.Fatalf("expected plugin without optional method to report no Warmup support")
	}

	tickResp, err := rt.Tick(context.Background(), &tickloophookr.TickRequestT{
		Tick:      1,
		DtMicros:  16666,
		StateHash: 7,
		Events:    2,
	})
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if tickResp.NextStateHash != 9 {
		t.Fatalf("next state hash = %d, want 9", tickResp.NextStateHash)
	}
	if tickResp.Actions != 2 {
		t.Fatalf("actions = %d, want 2", tickResp.Actions)
	}

	_, err = rt.Warmup(context.Background(), &tickloophookr.WarmupRequestT{
		TargetTicks: 16,
	})
	if err == nil {
		t.Fatalf("expected warmup to fail for plugin without optional warmup implementation")
	}
	if !strings.Contains(err.Error(), "method not found") {
		t.Fatalf("warmup error = %q, want contains 'method not found'", err.Error())
	}
}

func buildPlugin(t *testing.T, pluginDir string) string {
	t.Helper()

	_, filename, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatalf("resolve test file path")
	}
	contractDir := filepath.Dir(filename)
	outFile := filepath.Join(t.TempDir(), pluginDir+".wasm")

	cfg := buildkit.DefaultConfig()
	cfg.PluginPath = filepath.Join(contractDir, pluginDir)
	cfg.OutputPath = outFile
	if err := buildkit.Build(cfg); err != nil {
		t.Fatalf("build plugin %s: %v", pluginDir, err)
	}
	return outFile
}

func openRuntime(t *testing.T, wasmPath string) *tickloophookr.Runtime {
	t.Helper()
	rt, err := tickloophookr.Open(context.Background(), tickloophookr.Config{
		PluginPath:  wasmPath,
		FileOptions: []hookrruntime.FileOption{hookrruntime.WithAllowUnsigned()},
		Host: tickloophookr.Host{
			Rng: testHost{},
		},
	})
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	return rt
}

func closeRuntime(t *testing.T, rt *tickloophookr.Runtime) {
	t.Helper()
	if err := rt.Close(context.Background()); err != nil {
		t.Fatalf("close runtime: %v", err)
	}
}
