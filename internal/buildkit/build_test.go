package buildkit

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BuildMode != "c-shared" {
		t.Fatalf("build mode = %q", cfg.BuildMode)
	}
}

func TestConfigValidate(t *testing.T) {
	if err := (Config{}).Validate(); err == nil || !strings.Contains(err.Error(), "plugin path") {
		t.Fatalf("expected plugin path validation error, got %v", err)
	}
	if err := (Config{PluginPath: "./plugin"}).Validate(); err == nil ||
		!strings.Contains(err.Error(), "output path") {
		t.Fatalf("expected output path validation error, got %v", err)
	}
	if err := (Config{PluginPath: "./plugin", OutputPath: "./plugin.wasm"}).Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
}

func TestBuildUsesConfiguredGo(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "plugin.wasm")
	argsPath := filepath.Join(tmpDir, "go.args")
	envPath := filepath.Join(tmpDir, "go.env")
	scriptPath := filepath.Join(tmpDir, "go")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" >" + argsPath + "\nprintf '%s\n' \"$GOOS\" \"$GOARCH\" >" + envPath + "\nout=''\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"-o\" ]; then\n    shift\n    out=\"$1\"\n  fi\n  shift\ndone\n: > \"$out\"\n"
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is unix-only")
	}
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write go stub: %v", err)
	}

	cfg := DefaultConfig()
	cfg.PluginPath = "./plugin"
	cfg.OutputPath = outPath
	cfg.GoPath = scriptPath
	var stderr bytes.Buffer
	cfg.Stderr = &stderr
	if err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	logData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read go args log: %v", err)
	}
	args := string(logData)
	for _, want := range []string{"build", "-o", outPath, "-buildmode=c-shared", "./plugin"} {
		if !strings.Contains(args, want) {
			t.Fatalf("go args missing %q in %q", want, args)
		}
	}
	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read go env log: %v", err)
	}
	if got := string(envData); got != "wasip1\nwasm\n" {
		t.Fatalf("env = %q, want %q", got, "wasip1\nwasm\n")
	}
	for _, want := range []string{"hookr: building plugin ./plugin -> " + outPath, "hookr: built plugin " + outPath} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q in %q", want, stderr.String())
		}
	}
}

func TestBuildFindGoError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PluginPath = "./plugin"
	cfg.OutputPath = "./plugin.wasm"
	cfg.GoPath = filepath.Join(t.TempDir(), "missing-go")
	if err := Build(cfg); err == nil || !strings.Contains(err.Error(), "find go") {
		t.Fatalf("expected go lookup error, got %v", err)
	}
}
