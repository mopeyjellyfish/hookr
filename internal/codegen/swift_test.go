package codegen

import (
	"path/filepath"
	"testing"
)

func TestGenerateSwiftFlatBuffersPluginBindings(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "swift"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "textfilter.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "textfilterhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "HookrPlugin.swift"),
		"public let METHOD_PLUGIN_GET_INFO: UInt32",
		"public let METHOD_PLUGIN_FILTER: UInt32",
		"@_extern(wasm, module: \"hookr\", name: \"__plugin_request\")",
		"@_expose(wasm, \"__plugin_call\")",
		"public func hostCallRaw",
	)
	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "textfilter_generated.swift"),
		"struct Hookr_Test_TextFilter_FilterRequest",
		"struct Hookr_Test_TextFilter_FilterResponse",
	)
}

func TestGenerateSwiftHostCallbackConstants(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "swift"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "urlbalancer", "urlbalancer.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "urlbalancerhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "urlbalancerhookr", "HookrPlugin.swift"),
		"public let METHOD_RNG_INT: UInt32",
		"public let METHOD_RNG_FLOAT: UInt32",
		"@_extern(wasm, module: \"hookr\", name: \"__host_call\")",
	)
}
