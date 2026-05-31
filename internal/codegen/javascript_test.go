package codegen

import (
	"path/filepath"
	"testing"
)

func TestGenerateJavaScriptTypeScriptPluginBindings(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "js"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "textfilter.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "textfilterhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "hookr_plugin.ts"),
		"export const METHOD_PLUGIN_GET_INFO",
		"export const METHOD_PLUGIN_FILTER",
		"export interface HookrHostBridge",
		"export async function pluginCall",
		"export function hostCallRaw",
	)
	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "hookr", "test", "text-filter", "filter-request.ts"),
		"export class FilterRequest",
	)
	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "hookr", "test", "text-filter", "filter-response.ts"),
		"export class FilterResponse",
	)
}

func TestGenerateTypeScriptHostCallbackConstants(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "typescript"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "urlbalancer", "urlbalancer.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "urlbalancerhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "urlbalancerhookr", "hookr_plugin.ts"),
		"export const METHOD_RNG_INT",
		"export const METHOD_RNG_FLOAT",
		"hostCall(methodId: number, payload: Uint8Array): Uint8Array",
	)
}
