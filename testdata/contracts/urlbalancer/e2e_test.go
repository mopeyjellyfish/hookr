package urlbalancer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	urlbalancerhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/urlbalancer/gen/urlbalancerhookr"
)

type testHost struct{}

func (testHost) Int(_ context.Context, _ *urlbalancerhookr.RngIntRequestT) (*urlbalancerhookr.RngIntResponseT, error) {
	return &urlbalancerhookr.RngIntResponseT{Value: 1}, nil
}

func (testHost) Float(_ context.Context, _ *urlbalancerhookr.RngFloatRequestT) (*urlbalancerhookr.RngFloatResponseT, error) {
	return &urlbalancerhookr.RngFloatResponseT{Value: 0.5}, nil
}

func TestURLBalancerE2E(t *testing.T) {
	t.Parallel()

	outFile := filepath.Join(t.TempDir(), "urlbalancer.wasm")
	cfg := buildkit.DefaultConfig()
	cfg.PluginPath = "./plugin"
	cfg.OutputPath = outFile
	if err := buildkit.Build(cfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	ctx := context.Background()
	rt, err := urlbalancerhookr.Open(ctx, urlbalancerhookr.Config{
		PluginPath:  outFile,
		FileOptions: []hookrruntime.FileOption{hookrruntime.WithAllowUnsigned()},
		Host: urlbalancerhookr.Host{
			Rng: testHost{},
		},
	})
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	defer func() {
		if closeErr := rt.Close(ctx); closeErr != nil {
			t.Fatalf("close runtime: %v", closeErr)
		}
	}()

	info, err := rt.GetInfo(ctx, &urlbalancerhookr.EmptyT{})
	if err != nil {
		t.Fatalf("get info: %v", err)
	}
	if info.Name != "urlbalancer" {
		t.Fatalf("info name = %q, want urlbalancer", info.Name)
	}

	resp, err := rt.Balance(ctx, &urlbalancerhookr.BalanceRequestT{
		Url:   "https://example.com/api?q=1",
		Nodes: []string{"node-a", "node-b", "node-c"},
	})
	if err != nil {
		t.Fatalf("balance: %v", err)
	}

	if !resp.Valid {
		t.Fatalf("expected valid response, got %#v", resp)
	}
	if resp.SelectedNode != "node-b" {
		t.Fatalf("selected node = %q, want node-b", resp.SelectedNode)
	}
	if resp.SelectedIndex != 1 {
		t.Fatalf("selected index = %d, want 1", resp.SelectedIndex)
	}
	if resp.RngFloat != 0.5 {
		t.Fatalf("rng float = %v, want 0.5", resp.RngFloat)
	}
	if resp.Host != "example.com" {
		t.Fatalf("host = %q, want example.com", resp.Host)
	}
	if resp.Error != "" {
		t.Fatalf("unexpected response error: %q", resp.Error)
	}
}
