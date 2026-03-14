package devhost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFixture(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "host.json")
	data := []byte(`{
		"Ping": { "response": { "ok": true } },
		"RngInt": [
			{ "value": 1 },
			{ "value": 3 }
		]
	}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	fixture, err := LoadFixture(path)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	if len(fixture.Methods) != 2 {
		t.Fatalf("fixture methods = %d, want 2", len(fixture.Methods))
	}
	if got := string(fixture.Methods["Ping"].Response); got == "" {
		t.Fatal("expected Ping response fixture")
	}
	if got := len(fixture.Methods["RngInt"].Responses); got != 2 {
		t.Fatalf("RngInt responses = %d, want 2", got)
	}
}

func TestParseMethodFixture_FlexibleShapes(t *testing.T) {
	t.Parallel()

	for name, raw := range map[string]string{
		"direct":   `{"value":1}`,
		"single":   `{"response":{"value":1}}`,
		"sequence": `[{"value":1},{"value":2}]`,
	} {
		fixture, err := parseMethodFixture(json.RawMessage(raw))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(fixture.Response) == 0 && len(fixture.Responses) == 0 {
			t.Fatalf("%s: expected parsed response content", name)
		}
	}
}
