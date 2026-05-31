package codegen

import (
	"path/filepath"
	"testing"
)

func TestGenerateCPluginBindings(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "c"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "textfilter.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "textfilterhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "hookr_plugin.h"),
		"#define METHOD_PLUGIN_GET_INFO",
		"#define METHOD_PLUGIN_FILTER",
		"typedef bool (*hookr_dispatch_fn)",
		"hookr_host_call_raw",
	)
	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "hookr_plugin.c"),
		"HOOKR_EXPORT(\"__plugin_call\")",
		"HOOKR_EXPORT(\"__hookr_schema_hash\")",
		"hookr_register_plugin",
		"hookr_default_method_bytes",
	)
}

func TestGenerateCHostCallbackConstants(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "c"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "urlbalancer", "urlbalancer.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "urlbalancerhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "urlbalancerhookr", "hookr_plugin.h"),
		"#define METHOD_RNG_INT",
		"#define METHOD_RNG_FLOAT",
	)
	assertFileContains(t, filepath.Join(outDir, "urlbalancerhookr", "hookr_plugin.c"),
		"HOOKR_IMPORT(\"hookr\", \"__host_call\")",
	)
}
