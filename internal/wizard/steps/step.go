// Package steps holds the individual wizard screens. Each step is a
// self-contained Bubble Tea sub-model.
package steps

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Step is the contract each wizard screen satisfies. Returning a NextMsg from
// Update advances the wizard; returning BackMsg goes back.
type Step interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Step, tea.Cmd)
	View() string
	Title() string
}

type NextMsg struct{}
type BackMsg struct{}

func Next() tea.Msg { return NextMsg{} }
func Back() tea.Msg { return BackMsg{} }
