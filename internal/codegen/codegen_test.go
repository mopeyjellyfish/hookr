package codegen

import (
	"reflect"
	"testing"
)

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

func TestParseFlags_IncludePaths(t *testing.T) {
	cfg, err := ParseFlags("hookr gen", []string{
		"-schema", "contract.fbs",
		"-out", "gen",
		"-package", "contractpkg",
		"-include", "schemas,common",
		"-I", "third_party",
	})
	if err != nil {
		t.Fatalf("ParseFlags returned error: %v", err)
	}

	want := []string{"schemas", "common", "third_party"}
	if !reflect.DeepEqual(cfg.IncludePaths, want) {
		t.Fatalf("include paths = %#v, want %#v", cfg.IncludePaths, want)
	}
}

func TestParseFlags_RustLang(t *testing.T) {
	cfg, err := ParseFlags("hookr gen", []string{
		"-schema", "contract.fbs",
		"-out", "gen",
		"-package", "contractpkg",
		"-lang", "rust",
	})
	if err != nil {
		t.Fatalf("ParseFlags returned error: %v", err)
	}
	if cfg.Lang != "rust" {
		t.Fatalf("lang = %q, want rust", cfg.Lang)
	}
}

func TestParseFlags_ZigLang(t *testing.T) {
	cfg, err := ParseFlags("hookr gen", []string{
		"-schema", "contract.fbs",
		"-out", "gen",
		"-package", "contractpkg",
		"-lang", "zig",
	})
	if err != nil {
		t.Fatalf("ParseFlags returned error: %v", err)
	}
	if cfg.Lang != "zig" {
		t.Fatalf("lang = %q, want zig", cfg.Lang)
	}
}
