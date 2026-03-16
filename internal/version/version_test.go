package version

import "testing"

func TestCurrentUsesInjectedValue(t *testing.T) {
	prev := Value
	Value = "v9.9.9"
	t.Cleanup(func() {
		Value = prev
	})

	if got := Current(); got != "v9.9.9" {
		t.Fatalf("Current() = %q, want %q", got, "v9.9.9")
	}
}

func TestFormatDevelVersion(t *testing.T) {
	t.Parallel()

	if got := formatDevelVersion("0123456789abcdef", false); got != "devel-0123456789ab" {
		t.Fatalf("formatDevelVersion() = %q", got)
	}

	if got := formatDevelVersion("0123456789abcdef", true); got != "devel-0123456789ab-dirty" {
		t.Fatalf("formatDevelVersion() dirty = %q", got)
	}
}

func TestFallbackInfo(t *testing.T) {
	prev := Value
	Value = ""
	t.Cleanup(func() {
		Value = prev
	})

	if got := fallbackInfo(); got.Version != "dev" {
		t.Fatalf("fallbackInfo().Version = %q, want %q", got.Version, "dev")
	}
}
