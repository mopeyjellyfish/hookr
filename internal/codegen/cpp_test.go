package codegen

import (
	"path/filepath"
	"testing"
)

func TestGenerateCppFlatBuffersPluginBindings(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "cpp"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "textfilter.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "textfilterhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "hookr_plugin.hpp"),
		"constexpr std::uint32_t METHOD_PLUGIN_GET_INFO",
		"constexpr std::uint32_t METHOD_PLUGIN_FILTER",
		"RegisterPlugin",
		"HostCallRaw",
		"export_name(\"__plugin_call\")",
		"export_name(\"__hookr_schema_hash\")",
	)
	assertFileContains(t, filepath.Join(outDir, "textfilterhookr", "textfilter_generated.h"),
		"struct FilterRequest",
		"struct FilterResponse",
	)
}

func TestGenerateCppFlatBuffersHostCallbackConstants(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "cpp"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "urlbalancer", "urlbalancer.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "urlbalancerhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "urlbalancerhookr", "hookr_plugin.hpp"),
		"constexpr std::uint32_t METHOD_RNG_INT",
		"constexpr std::uint32_t METHOD_RNG_FLOAT",
		"import_name(\"__host_call\")",
	)
}
