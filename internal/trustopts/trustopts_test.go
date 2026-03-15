package trustopts

import (
	"os"
	"testing"

	"github.com/mopeyjellyfish/hookr/runtime"
)

func TestBuildRequiresExplicitTrust(t *testing.T) {
	t.Parallel()

	opts, err := Build("", false)
	if err == nil {
		t.Fatal("expected trust configuration error")
	}
	if opts != nil {
		t.Fatalf("opts = %#v, want nil", opts)
	}
}

func TestBuildWithHash(t *testing.T) {
	t.Parallel()

	path := writeTempPluginFile(t)
	opts, err := Build("deadbeef", false)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("len(opts) = %d, want 1", len(opts))
	}
	if _, err := runtime.NewFile(path, opts...); err == nil {
		t.Fatal("expected mismatched hash verification error")
	}
}

func TestBuildWithAllowUnsigned(t *testing.T) {
	t.Parallel()

	path := writeTempPluginFile(t)
	opts, err := Build("", true)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("len(opts) = %d, want 1", len(opts))
	}
	if _, err := runtime.NewFile(path, opts...); err != nil {
		t.Fatalf("NewFile with allow-unsigned: %v", err)
	}
}

func writeTempPluginFile(t *testing.T) string {
	t.Helper()

	path := t.TempDir() + "/plugin.wasm"
	if err := os.WriteFile(path, []byte("wasm"), 0o600); err != nil {
		t.Fatalf("write temp plugin: %v", err)
	}
	return path
}
