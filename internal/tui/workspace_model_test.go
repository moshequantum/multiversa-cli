package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/detect"
)

func TestWorkspaceModelSatisfiesTeaModel(t *testing.T) {
	var m tea.Model = NewWorkspaceModel(detect.Report{}, nil)
	if m.Init() != nil {
		t.Logf("Init returned a non-nil cmd, that's fine")
	}
	if m.View() == "" {
		t.Fatalf("View() must not be empty for a fresh model")
	}
}

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
	if _, ok := cmd().(CancelMsg); !ok {
		t.Fatalf("expected CancelMsg, got %T", cmd())
	}
}

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
