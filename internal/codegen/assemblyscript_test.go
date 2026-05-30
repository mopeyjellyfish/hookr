package codegen

import (
	"path/filepath"
	"testing"
)

func TestGenerateAssemblyScriptPluginBindings(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "assemblyscript"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "textfilter.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "textfilterhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "hookr_plugin.ts"),
		"export const METHOD_PLUGIN_GET_INFO: u32",
		"export const METHOD_PLUGIN_FILTER: u32",
		"@external(\"hookr\", \"__plugin_request\")",
		"export function __plugin_call",
		"export function __hookr_schema_hash",
		"export function hostCallRaw",
	)
}

func TestGenerateAssemblyScriptHostCallbackConstants(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "assemblyscript"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "urlbalancer", "urlbalancer.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "urlbalancerhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "urlbalancerhookr", "hookr_plugin.ts"),
		"export const METHOD_RNG_INT: u32",
		"export const METHOD_RNG_FLOAT: u32",
		"@external(\"hookr\", \"__host_call\")",
	)
}
