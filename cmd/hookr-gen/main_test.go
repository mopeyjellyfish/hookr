package main

import "testing"

func TestToExportedIdentifier(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"hello", "Hello"},
		{"hello_world", "HelloWorld"},
		{"1st-method", "M1stMethod"},
		{"", "Unnamed"},
		{"***", "Unnamed"},
	}

	for _, tt := range tests {
		if got := toExportedIdentifier(tt.in); got != tt.want {
			t.Fatalf("toExportedIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
