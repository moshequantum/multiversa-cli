package main

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// TestWorkspaceShowPrintsScript verifies that --show dumps the embedded
// bash script to the writer and exits cleanly. We look for a distinctive
// marker (ssh-keygen) that the real setup script must contain.
func TestWorkspaceShowPrintsScript(t *testing.T) {
	var buf bytes.Buffer
	err := runWorkspace(workspaceOpts{showOnly: true, out: &buf})
	if err != nil {
		t.Fatalf("runWorkspace(--show) returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ssh-keygen") {
		t.Fatalf("expected --show output to contain 'ssh-keygen', got:\n%s", out)
	}
}

// TestWorkspaceModelSatisfiesTeaModel checks the contract at compile
// time + runtime: NewWorkspaceModel must return a value that the Bubble
// Tea program loop can consume.
func TestWorkspaceModelSatisfiesTeaModel(t *testing.T) {
	var m tea.Model = NewWorkspaceModel(detect.Report{}, nil)
	if m.Init() != nil {
		// Init may return nil — this is just a smoke check.
		t.Logf("Init returned a non-nil cmd, that's fine")
	}
	if m.View() == "" {
		t.Fatalf("View() must not be empty for a fresh model")
	}
}

// TestWorkspaceEscEmitsCancel walks the model through an Esc key press
// and asserts that the resulting command emits a CancelMsg, marking
// the run as cancelled (which the host translates to exit code 2).
func TestWorkspaceEscEmitsCancel(t *testing.T) {
	m := NewWorkspaceModel(detect.Report{}, nil)
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	wm, ok := model.(WorkspaceModel)
	if !ok {
		t.Fatalf("expected WorkspaceModel, got %T", model)
	}
	if !wm.cancelled {
		t.Fatalf("expected cancelled=true after Esc, got false")
	}
	if cmd == nil {
		t.Fatalf("expected non-nil command after Esc")
	}
	if _, ok := cmd().(tui.CancelMsg); !ok {
		t.Fatalf("expected CancelMsg, got %T", cmd())
	}
}

// TestWorkspaceConfirmHonorsStrictYes verifies that pressing enter with
// "y" buffered marks the model as confirmed, while any other input
// (including blank) cancels — matching the no-default-yes rule.
func TestWorkspaceConfirmHonorsStrictYes(t *testing.T) {
	t.Run("strict yes confirms", func(t *testing.T) {
		m := NewWorkspaceModel(detect.Report{}, nil)
		m.input = "y"
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		wm := model.(WorkspaceModel)
		if !wm.confirmed {
			t.Fatalf("expected confirmed=true for input 'y'")
		}
	})
	t.Run("blank cancels", func(t *testing.T) {
		m := NewWorkspaceModel(detect.Report{}, nil)
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		wm := model.(WorkspaceModel)
		if wm.confirmed {
			t.Fatalf("expected confirmed=false for blank input")
		}
		if !wm.cancelled {
			t.Fatalf("expected cancelled=true for blank input")
		}
	})
}

// TestWorkspacePrereqMissingNonInteractive checks the plain-stdin
// branch: when prerequisites are missing, runWorkspaceNonInteractive
// must return a non-nil error so the cobra layer surfaces exit code 1.
// We feed it a stub Report whose Tools list omits git and ssh.
func TestWorkspacePrereqMissingNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	report := detect.Report{Tools: []detect.Tool{
		{Name: "go", Installed: true},
	}}
	missing := requiredMissing(report, []string{"git", "ssh"})
	if len(missing) != 2 {
		t.Fatalf("setup error: expected 2 missing tools, got %d", len(missing))
	}
	err := runWorkspaceNonInteractive(&buf, report, missing)
	if err == nil {
		t.Fatalf("expected prereq-missing error, got nil")
	}
	if !strings.Contains(buf.String(), "Prerrequisitos faltantes") {
		t.Fatalf("expected 'Prerrequisitos faltantes' in output, got:\n%s", buf.String())
	}
}

// TestWorkspaceRequiredMissing covers the helper in isolation: a tool
// marked Installed=true must be reported as present, otherwise missing.
func TestWorkspaceRequiredMissing(t *testing.T) {
	r := detect.Report{Tools: []detect.Tool{
		{Name: "git", Installed: true},
		{Name: "ssh", Installed: false},
	}}
	missing := requiredMissing(r, []string{"git", "ssh"})
	if len(missing) != 1 || missing[0] != "ssh" {
		t.Fatalf("expected only 'ssh' missing, got %v", missing)
	}
}
