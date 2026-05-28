package main

import (
	"bytes"
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
