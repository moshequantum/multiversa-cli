// Tests for `multiversa detect` covering the non-TTY fallback and the
// Bubble Tea model wiring. We intentionally do not spin up a real
// tea.Program here — those are heavy and flaky in CI. Instead we
// exercise Init/Update/View directly to confirm the model satisfies
// the tea.Model contract and reacts to keyboard input.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// TestDetectNonTTYFallback locks in v0.3.0 backward compatibility:
// when stdout is not a terminal, runDetect must emit the plain
// renderer output verbatim — including the "multiversa detect"
// header that scripts and the /lab-setup skill scrape.
func TestDetectNonTTYFallback(t *testing.T) {
	var buf bytes.Buffer
	if err := runDetect(&buf, false); err != nil {
		t.Fatalf("runDetect returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "multiversa detect") {
		t.Fatalf("expected plain renderer header in non-TTY output, got:\n%s", out)
	}
	// Sanity: the plain renderer always prints a Ready: summary line.
	if !strings.Contains(out, "Ready:") {
		t.Fatalf("expected Ready summary in non-TTY fallback, got:\n%s", out)
	}
}

// TestDetectJSONEnvelope locks in the machine contract of
// `multiversa detect --json`: one envelope, schema multiversa.detect/v1,
// ok=true, and a data payload carrying report + summary whose totals
// agree with the report's own counters.
func TestDetectJSONEnvelope(t *testing.T) {
	report := detect.Report{
		OS: detect.OSInfo{Kind: "linux", Arch: "amd64", Distro: "debian", PkgMgr: "apt"},
		Tools: []detect.Tool{
			{Name: "go", Installed: true, Version: "go1.26", Category: "language"},
			{Name: "node", Installed: false, Category: "language"},
			{Name: "npm", Installed: true, Advisory: true, Warn: true, Category: "pkgmgr"},
		},
		Multiversa: detect.MultiversaState{
			Engines: []detect.EngineState{
				{ID: "engram", Name: "Engram", Author: "Gentleman-Programming", Installed: true},
				{ID: "graphify", Name: "Graphify", Author: "Safi Shamsi", Installed: false},
			},
		},
	}

	var buf bytes.Buffer
	if err := emitDetectJSON(&buf, report); err != nil {
		t.Fatalf("emitDetectJSON returned error: %v", err)
	}

	var env struct {
		Schema  string `json:"schema"`
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Report  detect.Report  `json:"report"`
			Summary detect.Summary `json:"summary"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if env.Schema != "multiversa.detect/v1" {
		t.Fatalf("expected schema multiversa.detect/v1, got %q", env.Schema)
	}
	if env.Command != "detect" || !env.OK {
		t.Fatalf("unexpected envelope command/ok: %q / %v", env.Command, env.OK)
	}
	if env.Data.Summary.ToolsReady != 1 || env.Data.Summary.ToolsTotal != 2 {
		t.Fatalf("summary tools mismatch: %+v", env.Data.Summary)
	}
	if env.Data.Summary.EnginesReady != 1 || env.Data.Summary.EnginesTotal != 2 {
		t.Fatalf("summary engines mismatch: %+v", env.Data.Summary)
	}
	if len(env.Data.Report.Tools) != 3 || env.Data.Report.OS.PkgMgr != "apt" {
		t.Fatalf("report did not round-trip: %+v", env.Data.Report)
	}
}

// TestDetectModelSatisfiesTeaModel is a compile-time assertion that
// the model honors the tea.Model interface. If this ever stops
// compiling, the dual-mode contract is broken.
func TestDetectModelSatisfiesTeaModel(t *testing.T) {
	var _ tea.Model = NewDetectModel(detect.Run(), tui.Standard)
}

// TestDetectModelInitUpdateView exercises the three tea.Model methods
// without a running program. View must render the canonical header
// text after a WindowSizeMsg sets the width.
func TestDetectModelInitUpdateView(t *testing.T) {
	m := NewDetectModel(detect.Run(), tui.Standard)
	if cmd := m.Init(); cmd != nil {
		// Init should be a no-op for detect (no async work).
		t.Fatalf("expected nil Init cmd, got %T", cmd)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	view := updated.View()
	if !strings.Contains(view, "multiversa detect") {
		t.Fatalf("View missing header, got:\n%s", view)
	}
	if !strings.Contains(view, "Categorías") {
		t.Fatalf("View missing categories label, got:\n%s", view)
	}
}

// TestDetectModelQuitKeys confirms that both q and esc send tea.Quit.
// We compare the returned tea.Cmd to tea.Quit by invoking it: tea.Quit
// is a closure that returns a tea.QuitMsg, which is the only contract
// we can assert on without hitting unexported internals.
func TestDetectModelQuitKeys(t *testing.T) {
	for _, key := range []string{"q", "esc"} {
		m := NewDetectModel(detect.Run(), tui.Standard)
		_, cmd := m.Update(tea.KeyMsg{Type: keyTypeFor(key), Runes: runesFor(key)})
		if cmd == nil {
			t.Fatalf("key %q: expected tea.Quit cmd, got nil", key)
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Fatalf("key %q: expected tea.QuitMsg, got %T", key, msg)
		}
	}
}

// keyTypeFor maps the small key set we test to tea.KeyType. Bubble
// Tea's KeyMsg.String() honors both Runes (for printable chars) and
// Type (for named keys like esc); we have to feed it the right shape
// or msg.String() returns the wrong value.
func keyTypeFor(k string) tea.KeyType {
	switch k {
	case "esc":
		return tea.KeyEsc
	default:
		return tea.KeyRunes
	}
}

func runesFor(k string) []rune {
	if k == "esc" {
		return nil
	}
	return []rune(k)
}
