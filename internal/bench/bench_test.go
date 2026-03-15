package bench

import (
	"bytes"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.PackagePath == "" || cfg.Bench == "" || cfg.Run == "" || cfg.Count != 1 {
		t.Fatalf("unexpected default config: %#v", cfg)
	}
}

func TestWriterOrDefault(t *testing.T) {
	var custom bytes.Buffer
	var fallback bytes.Buffer
	if got := writerOrDefault(&custom, &fallback); got != &custom {
		t.Fatal("expected custom writer")
	}
	if got := writerOrDefault(nil, &fallback); got != &fallback {
		t.Fatal("expected fallback writer")
	}
}

func TestRunRequiresPackagePath(t *testing.T) {
	if err := Run(Config{}); err == nil {
		t.Fatal("expected package path validation error")
	}
}
