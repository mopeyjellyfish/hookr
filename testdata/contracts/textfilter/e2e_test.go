package textfilter

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	"github.com/mopeyjellyfish/hookr/runtime"
	textfilterhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/textfilter/gen/textfilterhookr"
)

func TestTextFilterE2E(t *testing.T) {
	t.Parallel()

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
