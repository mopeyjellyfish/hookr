package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRustFlatBuffersPluginBindings(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "rust"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "textfilter.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "textfilterhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	packageDir := filepath.Join(outDir, "textfilterhookr")
	assertFileContains(t, filepath.Join(packageDir, "lib.rs"),
		"pub mod textfilter_generated;",
		"pub mod hookr_plugin;",
		"pub use crate::textfilter_generated::hookr::test::text_filter::*;",
	)
	assertFileContains(t, filepath.Join(packageDir, "hookr_plugin.rs"),
		"pub const METHOD_PLUGIN_GET_INFO: u32",
		"pub const METHOD_PLUGIN_FILTER: u32",
		"pub trait Plugin",
		"fn get_info<'a>(",
		"fn filter<'a>(",
		"schema::FilterRequest<'_>",
		"flatbuffers::WIPOffset<schema::FilterResponse<'a>>",
		"pub extern \"C\" fn __plugin_call",
		"pub extern \"C\" fn __hookr_schema_hash",
	)
	assertFileContains(t, filepath.Join(packageDir, "textfilter_generated.rs"),
		"pub mod text_filter",
		"pub struct FilterRequest",
	)
}

func TestGenerateRustFlatBuffersHostCallbackClients(t *testing.T) {
	outDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Lang = "rust"
	cfg.SchemaPath = filepath.Join("..", "..", "testdata", "contracts", "urlbalancer", "urlbalancer.fbs")
	cfg.OutDir = outDir
	cfg.PackageName = "urlbalancerhookr"

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "urlbalancerhookr", "hookr_plugin.rs"),
		"pub rng: RngClient",
		"pub struct RngClient",
		"pub fn int_raw(&self, payload: &[u8]) -> HookrResult<Vec<u8>>",
		"pub fn float_raw(&self, payload: &[u8]) -> HookrResult<Vec<u8>>",
		"pub const METHOD_RNG_INT: u32",
		"pub const METHOD_RNG_FLOAT: u32",
	)
}

func assertFileContains(t *testing.T, path string, needles ...string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(raw)
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			t.Fatalf("%s missing %q\n%s", path, needle, text)
		}
	}
}
