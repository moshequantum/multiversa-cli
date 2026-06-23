package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/stack"
)

func TestStackModelImplementsTeaModel(t *testing.T) {
	planned := []stack.ToolPlan{}
	report := detect.Report{}
	m := NewStackModel(report, planned)
	var _ tea.Model = m
	if got := m.View(); got == "" {
		t.Error("expected non-empty View() for fresh StackModel")
	}
}

func TestStackEscEmitsCancelMsg(t *testing.T) {
	planned := []stack.ToolPlan{}
	report := detect.Report{}
	m := NewStackModel(report, planned)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd from Esc keypress")
	}
	msg := cmd()
	if _, ok := msg.(CancelMsg); !ok {
		t.Errorf("expected CancelMsg from Esc; got %T", msg)
	}
}
