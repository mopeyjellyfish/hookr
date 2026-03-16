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
