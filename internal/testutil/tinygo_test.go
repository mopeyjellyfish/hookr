package testutil

import "testing"

func TestParseTinyGoVersion(t *testing.T) {
	got, ok := parseTinyGoVersion("tinygo version 0.41.0 darwin/arm64 (using go version go1.26.3)")
	if !ok {
		t.Fatal("expected tinygo version to parse")
	}
	if got != (version{major: 0, minor: 41, patch: 0}) {
		t.Fatalf("version = %#v", got)
	}
}

func TestCompareVersions(t *testing.T) {
	min := mustParseVersion(MinTinyGoVersion)
	for _, tc := range []struct {
		name string
		v    version
		want int
	}{
		{name: "older minor", v: version{major: 0, minor: 40, patch: 1}, want: -1},
		{name: "same", v: version{major: 0, minor: 41, patch: 0}, want: 0},
		{name: "newer patch", v: version{major: 0, minor: 41, patch: 1}, want: 1},
		{name: "newer minor", v: version{major: 0, minor: 42, patch: 0}, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := compareVersions(tc.v, min)
			switch {
			case tc.want < 0 && got >= 0:
				t.Fatalf("compareVersions(%#v, %#v) = %d, want negative", tc.v, min, got)
			case tc.want == 0 && got != 0:
				t.Fatalf("compareVersions(%#v, %#v) = %d, want zero", tc.v, min, got)
			case tc.want > 0 && got <= 0:
				t.Fatalf("compareVersions(%#v, %#v) = %d, want positive", tc.v, min, got)
			}
		})
	}
}
