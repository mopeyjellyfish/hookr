package call

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
)

func TestRunTextFilter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "textfilter.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "plugin")
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	inputPath := filepath.Join(dir, "request.json")
	input := []byte(`{
		"input": "Bad inputs and bad habits",
		"blocked_terms": ["bad", "habits"],
		"replacement": "[filtered]",
		"case_sensitive": false,
		"max_replacements": 2
	}`)
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("write request: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := Run(Config{
		SchemaPath: filepath.Join(
			"..",
			"..",
			"testdata",
			"contracts",
			"textfilter",
			"textfilter.fbs",
		),
		PluginPath:    wasmPath,
		Method:        "Filter",
		InputPath:     inputPath,
		AllowUnsigned: true,
		Stdout:        &out,
		Stderr:        &errOut,
	})
	if err != nil {
		t.Fatalf("run call: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, out.String())
	}
	if got := resp["output"]; got != "[filtered] inputs and [filtered] habits" {
		t.Fatalf("output = %#v", got)
	}
	if got := resp["changed"]; got != true {
		t.Fatalf("changed = %#v, want true", got)
	}
	for _, want := range []string{"hookr: invoking Filter", "hookr: call succeeded for Filter"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q in %q", want, errOut.String())
		}
	}
}

func TestRunURLBalancerWithHostFixture(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "urlbalancer.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = filepath.Join(
		"..",
		"..",
		"testdata",
		"contracts",
		"urlbalancer",
		"plugin",
	)
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	inputPath := filepath.Join(dir, "request.json")
	input := []byte(`{
		"url": "https://example.com/api?q=1",
		"nodes": ["node-a", "node-b", "node-c"]
	}`)
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("write request: %v", err)
	}

	hostFixturePath := filepath.Join(dir, "host.json")
	hostFixture := []byte(`{
		"RngInt": { "response": { "value": 1 } },
		"RngFloat": { "response": { "value": 0.5 } }
	}`)
	if err := os.WriteFile(hostFixturePath, hostFixture, 0o600); err != nil {
		t.Fatalf("write host fixture: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := Run(Config{
		SchemaPath: filepath.Join(
			"..",
			"..",
			"testdata",
			"contracts",
			"urlbalancer",
			"urlbalancer.fbs",
		),
		PluginPath:      wasmPath,
		Method:          "Balance",
		InputPath:       inputPath,
		HostFixturePath: hostFixturePath,
		AllowUnsigned:   true,
		Stdout:          &out,
		Stderr:          &errOut,
	})
	if err != nil {
		t.Fatalf("run call: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, out.String())
	}
	if got := resp["selected_node"]; got != "node-b" {
		t.Fatalf("selected_node = %#v", got)
	}
	if got := resp["rng_int"]; got != float64(1) {
		t.Fatalf("rng_int = %#v", got)
	}
	for _, want := range []string{"hookr: invoking Balance", "hookr: loaded host fixture " + hostFixturePath, "hookr: call succeeded for Balance"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q in %q", want, errOut.String())
		}
	}
}

func TestSessionDefaultRequestJSONUsesSchemaFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "textfilter.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "plugin")
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	session, err := NewSession(Config{
		SchemaPath: filepath.Join(
			"..",
			"..",
			"testdata",
			"contracts",
			"textfilter",
			"textfilter.fbs",
		),
		PluginPath:    wasmPath,
		AllowUnsigned: true,
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	defer func() { _ = session.Close() }()

	req, err := session.DefaultRequestJSON("Filter")
	if err != nil {
		t.Fatalf("default request: %v", err)
	}
	for _, want := range []string{`"input"`, `"blocked_terms"`, `"replacement"`, `"case_sensitive"`, `"max_replacements"`} {
		if !strings.Contains(req, want) {
			t.Fatalf("request template missing %q in %q", want, req)
		}
	}
}

func TestSessionDebugInfo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "textfilter.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "plugin")
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	session, err := NewSession(Config{
		SchemaPath: filepath.Join(
			"..",
			"..",
			"testdata",
			"contracts",
			"textfilter",
			"textfilter.fbs",
		),
		PluginPath:    wasmPath,
		AllowUnsigned: true,
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	defer func() { _ = session.Close() }()

	info := session.DebugInfo()
	if info.ContractName == "" || info.ABIVersion == "" || !info.SchemaHashMatch {
		t.Fatalf("unexpected debug info: %#v", info)
	}
	if len(info.PluginMethodIDs) == 0 {
		t.Fatalf("expected plugin method ids, got %#v", info)
	}
}

func TestSessionInvokePrepared(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "textfilter.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "plugin")
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}

	session, err := NewSession(Config{
		SchemaPath: filepath.Join(
			"..",
			"..",
			"testdata",
			"contracts",
			"textfilter",
			"textfilter.fbs",
		),
		PluginPath:    wasmPath,
		AllowUnsigned: true,
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	defer func() { _ = session.Close() }()

	prepared, err := session.PrepareJSON("Filter", []byte(`{
		"input": "Bad inputs and bad habits",
		"blocked_terms": ["bad", "habits"],
		"replacement": "[filtered]",
		"case_sensitive": false,
		"max_replacements": 2
	}`))
	if err != nil {
		t.Fatalf("prepare json: %v", err)
	}

	result, err := session.InvokePrepared(context.Background(), prepared)
	if err != nil {
		t.Fatalf("invoke prepared: %v", err)
	}
	if result.Method.Name != "Filter" {
		t.Fatalf("result method = %q, want Filter", result.Method.Name)
	}
	if result.RequestBytes == 0 || result.ResponseBytes == 0 {
		t.Fatalf(
			"expected non-zero payload sizes, got req=%d resp=%d",
			result.RequestBytes,
			result.ResponseBytes,
		)
	}
	if len(result.Response) == 0 {
		t.Fatal("expected raw response bytes")
	}
}

func TestRunRequiresExplicitTrust(t *testing.T) {
	err := Run(Config{
		SchemaPath: filepath.Join(
			"..",
			"..",
			"testdata",
			"contracts",
			"textfilter",
			"textfilter.fbs",
		),
		PluginPath: "plugin.wasm",
		Method:     "Filter",
	})
	if err == nil || !strings.Contains(err.Error(), "plugin trust not configured") {
		t.Fatalf("expected trust configuration error, got %v", err)
	}
}
