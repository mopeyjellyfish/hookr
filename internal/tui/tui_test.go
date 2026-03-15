package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	"github.com/mopeyjellyfish/hookr/internal/call"
)

func TestNewModelView(t *testing.T) {
	t.Parallel()

	session := buildTextFilterSession(t)
	defer func() {
		_ = session.Close()
	}()

	m := newModel(Config{
		SchemaPath: filepath.Join(
			"..",
			"..",
			"testdata",
			"contracts",
			"textfilter",
			"textfilter.fbs",
		),
		PluginPath: "plugin.wasm",
	}, session)
	if m.selected.Name != "Filter" {
		t.Fatalf("selected method = %q, want Filter", m.selected.Name)
	}
	view := m.View()
	for _, want := range []string{
		"Hookr TUI",
		"schema: textfilter.fbs",
		"plugin: textfilter.wasm",
		"focus: methods",
		"Ready",
		"Textfilter",
		"Filter",
		"GetInfo",
		"up/down",
		"tab",
		"editor",
		"call",
		"loop",
		"quit",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q in %q", want, view)
		}
	}
}

func TestModelSelectionAndSchemaPrefill(t *testing.T) {
	t.Parallel()

	session := buildTextFilterSession(t)
	defer func() {
		_ = session.Close()
	}()

	m := newModel(Config{}, session)
	if m.selected.Name != "Filter" {
		t.Fatalf("selected method = %q, want Filter", m.selected.Name)
	}
	for _, want := range []string{`"input"`, `"blocked_terms"`, `"replacement"`} {
		if !strings.Contains(m.requestValue(), want) {
			t.Fatalf("expected request prefill to contain %q in %q", want, m.requestValue())
		}
	}
}

func TestModelResetUsesSchemaPrefill(t *testing.T) {
	t.Parallel()

	session := buildTextFilterSession(t)
	defer func() {
		_ = session.Close()
	}()

	m := newModel(Config{}, session)
	m.requests[m.selected.Name] = `{"input":"x"}`
	m.renderRequest()
	updated, _ := m.Update(keyMsg("r"))
	tuiModel := updated.(model)
	if !strings.Contains(tuiModel.requestValue(), `"input"`) {
		t.Fatalf(
			"expected reset request to include schema-derived fields, got %q",
			tuiModel.requestValue(),
		)
	}
}

func TestArrowKeysSelectMethods(t *testing.T) {
	t.Parallel()

	session := buildTextFilterSession(t)
	defer func() {
		_ = session.Close()
	}()

	m := newModel(Config{}, session)
	if m.selected.Name != "Filter" {
		t.Fatalf("selected method = %q, want Filter", m.selected.Name)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	tuiModel := updated.(model)
	if tuiModel.selected.Name != "GetInfo" {
		t.Fatalf("selected method after up = %q, want GetInfo", tuiModel.selected.Name)
	}
}

func TestTabCyclesFocus(t *testing.T) {
	t.Parallel()

	session := buildTextFilterSession(t)
	defer func() {
		_ = session.Close()
	}()

	m := newModel(Config{}, session)
	if m.focus != focusMethods {
		t.Fatalf("initial focus = %v, want methods", m.focus)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	tuiModel := updated.(model)
	if tuiModel.focus != focusRequest {
		t.Fatalf("focus after tab = %v, want request", tuiModel.focus)
	}
	updated, _ = tuiModel.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	tuiModel = updated.(model)
	if tuiModel.focus != focusMethods {
		t.Fatalf("focus after shift+tab = %v, want methods", tuiModel.focus)
	}
}

func TestReadFileModTime(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "plugin.wasm")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if got := readFileModTime(path); got.IsZero() {
		t.Fatal("expected non-zero mod time")
	}
}

func buildTextFilterSession(t *testing.T) *call.Session {
	t.Helper()
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "textfilter.wasm")
	buildCfg := buildkit.DefaultConfig()
	buildCfg.PluginPath = filepath.Join("..", "..", "testdata", "contracts", "textfilter", "plugin")
	buildCfg.OutputPath = wasmPath
	if err := buildkit.Build(buildCfg); err != nil {
		t.Fatalf("build plugin: %v", err)
	}
	session, err := call.NewSession(call.Config{
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
	return session
}

func keyMsg(s string) tea.KeyMsg {
	runes := []rune(s)
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: runes}
}
