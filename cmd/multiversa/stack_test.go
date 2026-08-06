package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// TestStackDryRunPrintsPlans confirms the v0.3.0 dry-run contract still
// holds: planned tools are listed, no execution happens, exit is clean.
func TestStackDryRunPrintsPlans(t *testing.T) {
	var buf bytes.Buffer
	err := runStack(stackOpts{dryRun: true, out: &buf})
	if err != nil {
		t.Fatalf("runStack(--dry-run) returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "multiversa stack") {
		t.Errorf("expected header in dry-run output; got:\n%s", out)
	}
	if !strings.Contains(out, "Dry run") {
		t.Errorf("expected dry-run sentinel in output; got:\n%s", out)
	}
}

// TestStackOnlyFiltersInNonTTY checks that --only restricts the planned
// set even when running through the non-TTY path. We assert the output
// contains the requested tool's display name and not the others.
func TestStackOnlyFiltersInNonTTY(t *testing.T) {
	var buf bytes.Buffer
	err := runStack(stackOpts{dryRun: true, only: []string{"docker"}, out: &buf})
	if err != nil {
		t.Fatalf("runStack(--only=docker) returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(strings.ToLower(out), "docker") {
		t.Errorf("expected docker row in --only=docker output; got:\n%s", out)
	}
	// Tools NOT in the filter must not appear as rows. We look for the
	// padded ID prefix the row printer uses to avoid false negatives
	// from incidental substrings.
	for _, id := range []string{"rust", "python", "node", "pnpm"} {
		padded := lipglossPad(id, 10)
		if strings.Contains(out, padded) {
			t.Errorf("expected %q to be filtered out by --only=docker; got:\n%s", id, out)
		}
	}
}

// TestStackModelImplementsTeaModel locks in the Bubble Tea contract for
// stackModel — Init/Update/View must compile against tea.Model.
func TestStackModelImplementsTeaModel(t *testing.T) {
	planned, report := planStack(stackOpts{only: []string{"docker"}})
	m := newStackModel(report, planned)
	var _ tea.Model = m
	if got := m.View(); got == "" {
		t.Error("expected non-empty View() for fresh stackModel")
	}
}

// TestStackEscEmitsCancelMsg confirms Esc returns a tea.Cmd that, when
// invoked, yields a tui.CancelMsg — the contract that lets the host
// program translate cancellation into exit code 2.
func TestStackEscEmitsCancelMsg(t *testing.T) {
	planned, report := planStack(stackOpts{only: []string{"docker"}})
	m := newStackModel(report, planned)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd from Esc keypress")
	}
	msg := cmd()
	if _, ok := msg.(tui.CancelMsg); !ok {
		t.Errorf("expected tui.CancelMsg from Esc; got %T", msg)
	}
}

// TestStackNonInteractive_CorruptProfileNotOverwritten is the regression
// test for the data-loss bug: runStackNonInteractive used to do
// `prof, _ := profile.Load()` and unconditionally `prof.Save()` at the
// end. When ~/.multiversa/profile.toml exists but fails to parse,
// profile.Load returns a zero-value Profile — saving it would silently
// replace the corrupt-but-possibly-recoverable file with a "clean"
// empty one, permanently destroying whatever level/locale/
// installed_engines it held. This test uses a throwaway HOME (never the
// real one) and asserts the corrupt file is left byte-for-byte
// untouched, with a warning surfaced to the user instead.
func TestStackNonInteractive_CorruptProfileNotOverwritten(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	profDir := filepath.Join(home, ".multiversa")
	if err := os.MkdirAll(profDir, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", profDir, err)
	}
	profPath := filepath.Join(profDir, "profile.toml")
	const corrupt = "level = \"expert\"\nlocale = [this is not valid toml\n"
	if err := os.WriteFile(profPath, []byte(corrupt), 0o600); err != nil {
		t.Fatalf("writing corrupt profile: %v", err)
	}

	var buf bytes.Buffer
	// --only=go and --yes: "go" is guaranteed installed in this test's
	// own environment (the test binary was built with it), so the loop
	// hits the "skipped" branch and never shells out to a real
	// installer — this test only cares about profile persistence.
	if err := runStack(stackOpts{yes: true, only: []string{"go"}, out: &buf}); err != nil {
		t.Fatalf("runStack returned error: %v", err)
	}

	got, err := os.ReadFile(profPath)
	if err != nil {
		t.Fatalf("profile.toml vanished after runStack: %v", err)
	}
	if string(got) != corrupt {
		t.Errorf("corrupt profile.toml was overwritten by runStack;\ngot:\n%s\nwant (unchanged):\n%s", got, corrupt)
	}

	out := buf.String()
	if !strings.Contains(out, "no se pudo leer el perfil existente") {
		t.Errorf("expected a warning about the unreadable profile in output; got:\n%s", out)
	}
}

// TestStackModel_CorruptProfileNotOverwritten covers the same data-loss
// bug in the Bubble Tea path: stackModel captured profile.Load()'s error
// into profErr but never checked it before m.prof.Save(). This asserts
// newStackModel correctly derives profSavable=false for a corrupt file,
// and that driving the install queue to completion does not touch disk.
func TestStackModel_CorruptProfileNotOverwritten(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	profDir := filepath.Join(home, ".multiversa")
	if err := os.MkdirAll(profDir, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", profDir, err)
	}
	profPath := filepath.Join(profDir, "profile.toml")
	const corrupt = "level = \"expert\"\nlocale = [this is not valid toml\n"
	if err := os.WriteFile(profPath, []byte(corrupt), 0o600); err != nil {
		t.Fatalf("writing corrupt profile: %v", err)
	}

	planned, report := planStack(stackOpts{only: []string{"go"}})
	m := newStackModel(report, planned)
	if m.profSavable {
		t.Fatal("expected profSavable=false when profile.toml fails to parse")
	}

	// Drive the model as if an install queue just finished, which is
	// where the buggy unconditional Save() used to fire.
	m.phase = phaseInstall
	m.queue = nil
	m.cursor = 0
	m.installCount = 1
	m.advanceInstall(0)

	got, err := os.ReadFile(profPath)
	if err != nil {
		t.Fatalf("profile.toml vanished after advanceInstall: %v", err)
	}
	if string(got) != corrupt {
		t.Errorf("stack TUI overwrote corrupt profile.toml;\ngot:\n%s\nwant (unchanged):\n%s", got, corrupt)
	}

	if !strings.Contains(m.View(), "perfil existente ilegible") {
		t.Errorf("expected the done-phase View() to warn about the unreadable profile; got:\n%s", m.View())
	}
}

// Sanity: planStack should run without panicking on an empty filter and
// return a slice + a populated detect.Report.
func TestPlanStackBaseline(t *testing.T) {
	planned, report := planStack(stackOpts{})
	if len(planned) == 0 {
		t.Error("expected planStack to return some tools from the registry")
	}
	if report.OS.Kind == "" {
		t.Error("expected detect.Report to have a non-empty OS kind")
	}
	_ = detect.Report{} // keep the import live
}
