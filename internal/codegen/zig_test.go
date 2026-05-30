package codegen

import (
	"path/filepath"
	"testing"
)

func TestGenerateZigFlatBuffersPluginBindings(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "zig"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "textfilter.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "textfilterhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "hookr_plugin.zig"),
		"pub const METHOD_PLUGIN_GET_INFO: u32",
		"pub const METHOD_PLUGIN_FILTER: u32",
		"pub export fn __plugin_call",
		"pub export fn __hookr_schema_hash",
		"pub fn hostCallRaw",
		"pub fn registerPlugin",
	)
}

func TestGenerateZigFlatBuffersHostCallbackConstants(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "zig"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "urlbalancer", "urlbalancer.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "urlbalancerhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "urlbalancerhookr", "hookr_plugin.zig"),
		"pub const METHOD_RNG_INT: u32",
		"pub const METHOD_RNG_FLOAT: u32",
		"extern \"hookr\" fn __host_call",
	)
}
