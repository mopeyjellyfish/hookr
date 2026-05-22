package textfilter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	"github.com/mopeyjellyfish/hookr/internal/testutil"
	"github.com/mopeyjellyfish/hookr/runtime"
	textfilterhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/textfilter/gen/textfilterhookr"
)

func TestTextFilterE2E(t *testing.T) {
	t.Parallel()
	testutil.RequireTinyGo(t)

	wasmPath := filepath.Join(t.TempDir(), "textfilter.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = "./plugin"
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	ctx := context.Background()
	rt, err := textfilterhookr.Open(ctx, textfilterhookr.Config{
		PluginPath:  wasmPath,
		FileOptions: []runtime.FileOption{runtime.WithAllowUnsigned()},
	})
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	defer func() {
		if closeErr := rt.Close(ctx); closeErr != nil {
			t.Fatalf("close runtime: %v", closeErr)
		}
	}()

	info, err := rt.GetInfo(ctx, &textfilterhookr.EmptyT{})
	if err != nil {
		t.Fatalf("get info: %v", err)
	}
	if info.Name != "textfilter" {
		t.Fatalf("plugin name = %q, want textfilter", info.Name)
	}

	req := &textfilterhookr.FilterRequestT{
		Input:           "Bad inputs and bad habits",
		BlockedTerms:    []string{"bad", "habits"},
		Replacement:     "[filtered]",
		CaseSensitive:   false,
		MaxReplacements: 2,
	}
	resp, err := rt.Filter(ctx, req)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}

	if !resp.Changed {
		t.Fatalf("expected changed response, got %#v", resp)
	}
	if resp.Replacements != 2 {
		t.Fatalf("replacements = %d, want 2", resp.Replacements)
	}
	if resp.Output != "[filtered] inputs and [filtered] habits" {
		t.Fatalf("output = %q", resp.Output)
	}
	if resp.Blocked {
		t.Fatalf("blocked = true, want false")
	}
	if resp.Reason != "" {
		t.Fatalf("reason = %q, want empty", resp.Reason)
	}
}

func TestTextFilterE2E_LiveReload(t *testing.T) {
	t.Parallel()
	testutil.RequireTinyGo(t)

	wasmPath := filepath.Join(t.TempDir(), "textfilter.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = "./plugin"
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	original, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read plugin: %v", err)
	}

	reloaded := make(chan struct{}, 1)
	ctx := context.Background()
	rt, err := textfilterhookr.Open(ctx, textfilterhookr.Config{
		PluginPath:  wasmPath,
		FileOptions: []runtime.FileOption{runtime.WithAllowUnsigned()},
		Reload: &textfilterhookr.ReloadConfig{
			Debounce: 25 * time.Millisecond,
			OnReload: func(
				ctx context.Context,
				next *textfilterhookr.Runtime,
				event runtime.ReloadEvent,
			) error {
				info, err := next.GetInfo(ctx, &textfilterhookr.EmptyT{})
				if err != nil {
					return err
				}
				if info.Name != "textfilter" {
					t.Fatalf("reloaded plugin name = %q, want textfilter", info.Name)
				}
				select {
				case reloaded <- struct{}{}:
				default:
				}
				return nil
			},
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

	if err := os.WriteFile(wasmPath, original, 0o644); err != nil {
		t.Fatalf("rewrite plugin: %v", err)
	}

	select {
	case <-reloaded:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for live reload")
	}

	resp, err := rt.Filter(ctx, &textfilterhookr.FilterRequestT{
		Input:           "bad habits",
		BlockedTerms:    []string{"bad"},
		Replacement:     "[filtered]",
		CaseSensitive:   false,
		MaxReplacements: 1,
	})
	if err != nil {
		t.Fatalf("filter after reload: %v", err)
	}
	if resp.Output != "[filtered] habits" {
		t.Fatalf("output after reload = %q", resp.Output)
	}
}
