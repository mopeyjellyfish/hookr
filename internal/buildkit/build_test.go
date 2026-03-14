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
	if cfg.Target != "wasip1" {
		t.Fatalf("target = %q", cfg.Target)
	}
	if cfg.BuildMode != "c-shared" {
		t.Fatalf("build mode = %q", cfg.BuildMode)
	}
	if cfg.Scheduler != "none" {
		t.Fatalf("scheduler = %q", cfg.Scheduler)
	}
	if !cfg.NoDebug {
		t.Fatal("expected no debug enabled")
	}
}

func TestConfigValidate(t *testing.T) {
	if err := (Config{}).Validate(); err == nil || !strings.Contains(err.Error(), "plugin path") {
		t.Fatalf("expected plugin path validation error, got %v", err)
	}
	if err := (Config{PluginPath: "./plugin"}).Validate(); err == nil || !strings.Contains(err.Error(), "output path") {
		t.Fatalf("expected output path validation error, got %v", err)
	}
	if err := (Config{PluginPath: "./plugin", OutputPath: "./plugin.wasm"}).Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
}

func TestBuildUsesConfiguredTinyGo(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "plugin.wasm")
	logPath := filepath.Join(tmpDir, "tinygo.log")
	scriptPath := filepath.Join(tmpDir, "tinygo")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" >" + logPath + "\nout=''\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"-o\" ]; then\n    shift\n    out=\"$1\"\n  fi\n  shift\ndone\n: > \"$out\"\n"
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is unix-only")
	}
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write tinygo stub: %v", err)
	}

	cfg := DefaultConfig()
	cfg.PluginPath = "./plugin"
	cfg.OutputPath = outPath
	cfg.TinyGoPath = scriptPath
	var stderr bytes.Buffer
	cfg.Stderr = &stderr
	if err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read tinygo log: %v", err)
	}
	args := string(logData)
	for _, want := range []string{"build", "-o", outPath, "-target=wasip1", "-buildmode=c-shared", "-scheduler=none", "--no-debug", "./plugin"} {
		if !strings.Contains(args, want) {
			t.Fatalf("tinygo args missing %q in %q", want, args)
		}
	}
	for _, want := range []string{"hookr: building plugin ./plugin -> " + outPath, "hookr: built plugin " + outPath} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q in %q", want, stderr.String())
		}
	}
}

func TestBuildFindTinyGoError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PluginPath = "./plugin"
	cfg.OutputPath = "./plugin.wasm"
	cfg.TinyGoPath = filepath.Join(t.TempDir(), "missing-tinygo")
	if err := Build(cfg); err == nil || !strings.Contains(err.Error(), "find tinygo") {
		t.Fatalf("expected tinygo lookup error, got %v", err)
	}
}
