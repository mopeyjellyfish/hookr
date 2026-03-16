package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/version"
)

func TestGenCommandGeneratesFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "example.fbs")
	schema := []byte(`
namespace Example.TextFilter;

table FilterRequest {
  input:string;
}

table FilterResponse {
  output:string;
}

rpc_service Plugin {
  Filter(FilterRequest):FilterResponse;
}
`)
	if err := os.WriteFile(schemaPath, schema, 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	outDir := filepath.Join(tmpDir, "gen")
	cmd := NewRootCommand()
	cmd.SetArgs([]string{
		"gen",
		"--schema", schemaPath,
		"--out", outDir,
		"--package", "examplehookr",
	})
	cmd.SetOut(new(bytes.Buffer))
	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute gen command: %v", err)
	}

	packageDir := filepath.Join(outDir, "examplehookr")
	for _, name := range []string{
		"contract_meta_gen.go",
		"host_sdk_gen.go",
		"plugin_pdk_gen.go",
	} {
		if _, err := os.Stat(filepath.Join(packageDir, name)); err != nil {
			t.Fatalf("expected generated file %s: %v", name, err)
		}
	}
	if !strings.Contains(errBuf.String(), "hookr: generated package examplehookr") {
		t.Fatalf("expected generation status, got %q", errBuf.String())
	}
}

func TestBuildCommandStub(t *testing.T) {
	t.Parallel()

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"build"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected build command to report not implemented")
	}
}

func TestGenCommandIncludeFlag(t *testing.T) {
	t.Parallel()

	cmd := newGenCommand()
	if err := cmd.Flags().Set("include", "schemas"); err != nil {
		t.Fatalf("set include flag: %v", err)
	}
	if err := cmd.ParseFlags([]string{"-I", "shared"}); err != nil {
		t.Fatalf("parse shorthand include flag: %v", err)
	}
}

func TestInspectCommandIncludeFlag(t *testing.T) {
	t.Parallel()

	cmd := newInspectCommand()
	if err := cmd.Flags().Set("include", "schemas"); err != nil {
		t.Fatalf("set include flag: %v", err)
	}
	if err := cmd.ParseFlags([]string{"-I", "shared"}); err != nil {
		t.Fatalf("parse shorthand include flag: %v", err)
	}
}

func TestVersionCommandPrintsVersion(t *testing.T) {
	prev := version.Value
	version.Value = "v1.2.3-test"
	t.Cleanup(func() {
		version.Value = prev
	})

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"version"})
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version command: %v", err)
	}

	if got := out.String(); got != "v1.2.3-test\n" {
		t.Fatalf("version output = %q, want %q", got, "v1.2.3-test\n")
	}
}
