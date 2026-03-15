package flatc

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildBFBSArgs_Includes(t *testing.T) {
	args := buildBFBSArgs("contract.fbs", "/tmp/out", []string{"schemas", "common ", ""})
	want := []string{
		"--binary",
		"--schema",
		"-I",
		"schemas",
		"-I",
		"common",
		"-o",
		"/tmp/out",
		"contract.fbs",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("buildBFBSArgs() = %#v, want %#v", args, want)
	}
}

func TestBuildGoArgs_Includes(t *testing.T) {
	args := buildGoArgs(GoOptions{
		SchemaPath:   "contract.fbs",
		OutDir:       "/tmp/out",
		PackageName:  "contractpkg",
		ObjectAPI:    true,
		IncludePaths: []string{"schemas", "common"},
	})
	want := []string{
		"--go",
		"--go-namespace", "contractpkg",
		"--gen-object-api",
		"-I", "schemas",
		"-I", "common",
		"-o", "/tmp/out",
		"contract.fbs",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("buildGoArgs() = %#v, want %#v", args, want)
	}
}

func TestEncodeDecodeJSON(t *testing.T) {
	t.Parallel()

	runner, err := New("")
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	schemaPath := filepath.Join("..", "..", "testdata", "contracts", "textfilter", "textfilter.fbs")
	raw := []byte(
		`{"input":"bad input","blocked_terms":["bad"],"replacement":"[x]","case_sensitive":false,"max_replacements":1}`,
	)
	bin, err := runner.EncodeJSON(schemaPath, nil, "Hookr.Test.TextFilter.FilterRequest", raw)
	if err != nil {
		t.Fatalf("encode json: %v", err)
	}
	if len(bin) == 0 {
		t.Fatal("expected encoded binary output")
	}

	decoded, err := runner.DecodeJSON(schemaPath, nil, "Hookr.Test.TextFilter.FilterRequest", bin)
	if err != nil {
		t.Fatalf("decode json: %v", err)
	}
	for _, want := range [][]byte{
		[]byte(`"input": "bad input"`),
		[]byte(`"blocked_terms": [`),
		[]byte(`"replacement": "[x]"`),
	} {
		if !bytes.Contains(decoded, want) {
			t.Fatalf("decoded json missing %q in %s", want, decoded)
		}
	}
}
