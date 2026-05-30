package textfilter

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/codegen"
	"github.com/mopeyjellyfish/hookr/runtime"
	textfilterhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/textfilter/gen/textfilterhookr"
)

func TestTextFilterRustPluginE2E(t *testing.T) {
	requireRustWasip1(t)

	dir := t.TempDir()
	cfg := codegen.DefaultConfig()
	cfg.Lang = "rust"
	cfg.SchemaPath = "textfilter.fbs"
	cfg.OutDir = dir
	cfg.PackageName = "textfilterhookr"
	if err := codegen.Generate(cfg); err != nil {
		t.Fatalf("generate rust bindings: %v", err)
	}

	crateDir := filepath.Join(dir, "textfilterhookr")
	writeRustTextFilterCrate(t, crateDir)

	cmd := exec.Command("cargo", "build", "--release", "--target", "wasm32-wasip1")
	cmd.Dir = crateDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cargo build rust plugin: %v\n%s", err, out)
	}

	wasmPath := filepath.Join(crateDir, "target", "wasm32-wasip1", "release", "textfilterhookr.wasm")
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
	if info.Name != "rust-textfilter" {
		t.Fatalf("plugin name = %q, want rust-textfilter", info.Name)
	}

	resp, err := rt.Filter(ctx, &textfilterhookr.FilterRequestT{Input: "bad input"})
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if resp.Output != "bad input" {
		t.Fatalf("output = %q, want original input", resp.Output)
	}
	if resp.Changed {
		t.Fatalf("changed = true, want false")
	}
}

func requireRustWasip1(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed; skipping Rust plugin conformance test")
	}
	if _, err := exec.LookPath("rustup"); err != nil {
		t.Skip("rustup not installed; skipping Rust plugin conformance test")
	}
	out, err := exec.Command("rustup", "target", "list", "--installed").Output()
	if err != nil {
		t.Skipf("cannot list installed Rust targets: %v", err)
	}
	if !strings.Contains(string(out), "wasm32-wasip1") {
		t.Skip("Rust target wasm32-wasip1 is not installed")
	}
}

func writeRustTextFilterCrate(t *testing.T, crateDir string) {
	t.Helper()
	cargo := `[package]
name = "textfilterhookr"
version = "0.0.0"
edition = "2021"

[lib]
path = "lib.rs"
crate-type = ["cdylib"]

[dependencies]
flatbuffers = "25"

[profile.release]
panic = "abort"
`
	if err := os.WriteFile(filepath.Join(crateDir, "Cargo.toml"), []byte(cargo), 0o600); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}

	impl := `

struct TextFilterPlugin;

impl hookr_plugin::Plugin for TextFilterPlugin {
    fn get_info<'a>(
        &mut self,
        _ctx: &mut hookr_plugin::PluginContext,
        _req: schema::Empty<'_>,
        builder: &mut flatbuffers::FlatBufferBuilder<'a>,
    ) -> hookr_plugin::HookrResult<flatbuffers::WIPOffset<schema::PluginInfo<'a>>> {
        let name = builder.create_string("rust-textfilter");
        let version = builder.create_string("0.0.0");
        let description = builder.create_string("Rust Hookr text filter fixture");
        Ok(schema::PluginInfo::create(builder, &schema::PluginInfoArgs {
            name: Some(name),
            version: Some(version),
            description: Some(description),
        }))
    }

    fn filter<'a>(
        &mut self,
        _ctx: &mut hookr_plugin::PluginContext,
        req: schema::FilterRequest<'_>,
        builder: &mut flatbuffers::FlatBufferBuilder<'a>,
    ) -> hookr_plugin::HookrResult<flatbuffers::WIPOffset<schema::FilterResponse<'a>>> {
        let output = builder.create_string(req.input().unwrap_or(""));
        let reason = builder.create_string("");
        Ok(schema::FilterResponse::create(builder, &schema::FilterResponseArgs {
            output: Some(output),
            changed: false,
            replacements: 0,
            blocked: false,
            reason: Some(reason),
        }))
    }
}

#[no_mangle]
pub extern "C" fn hookr_init() {
    hookr_plugin::register_plugin(TextFilterPlugin, &[]);
}
`
	libPath := filepath.Join(crateDir, "lib.rs")
	f, err := os.OpenFile(libPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open lib.rs: %v", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			t.Fatalf("close lib.rs: %v", closeErr)
		}
	}()
	if _, err := f.WriteString(impl); err != nil {
		t.Fatalf("append Rust plugin implementation: %v", err)
	}
}
