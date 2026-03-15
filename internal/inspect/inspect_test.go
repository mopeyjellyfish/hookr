package inspect

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
)

func TestRunRequiresSchemaPath(t *testing.T) {
	if err := Run(Config{}); err == nil {
		t.Fatal("expected schema path validation error")
	}
}

func TestRunPrintsContractDetails(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	err := Run(Config{
		SchemaPath:        "../../testdata/contracts/textfilter/textfilter.fbs",
		Package:           "textfilterhookr",
		OptionalAttribute: "hookr_optional",
		AllowUnsigned:     true,
		Stdout:            &out,
		Stderr:            &errOut,
	})
	if err != nil {
		t.Fatalf("run inspect: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"Contract: Textfilter",
		"Package: textfilterhookr",
		"Plugin Service: Plugin",
		"GetInfo",
		"Filter",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q in %q", want, text)
		}
	}
	if !strings.Contains(
		errOut.String(),
		"hookr: inspecting schema ../../testdata/contracts/textfilter/textfilter.fbs",
	) {
		t.Fatalf("expected stderr status, got %q", errOut.String())
	}
}

func TestRunPrintsWasmDetails(t *testing.T) {
	t.Parallel()

	wasmPath := filepath.Join(t.TempDir(), "textfilter.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "plugin")
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := Run(Config{
		SchemaPath:    "../../testdata/contracts/textfilter/textfilter.fbs",
		PluginPath:    wasmPath,
		Package:       "textfilterhookr",
		AllowUnsigned: true,
		Stdout:        &out,
		Stderr:        &errOut,
	})
	if err != nil {
		t.Fatalf("run inspect: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"Plugin Artifact:",
		"Plugin ABI: 2.0",
		"Schema Hash Match: true",
		"Plugin Methods:",
		"implemented",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q in %q", want, text)
		}
	}
	for _, want := range []string{"hookr: inspecting schema ../../testdata/contracts/textfilter/textfilter.fbs", "hookr: loading plugin " + wasmPath, "hookr: inspected plugin " + wasmPath} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q in %q", want, errOut.String())
		}
	}
}

func TestRunRequiresExplicitTrustWhenLoadingWasm(t *testing.T) {
	err := Run(Config{
		SchemaPath: "../../testdata/contracts/textfilter/textfilter.fbs",
		PluginPath: "plugin.wasm",
		Package:    "textfilterhookr",
	})
	if err == nil || !strings.Contains(err.Error(), "plugin trust not configured") {
		t.Fatalf("expected trust configuration error, got %v", err)
	}
}
