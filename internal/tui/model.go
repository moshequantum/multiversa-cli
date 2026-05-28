// Package tui hosts the shared Bubble Tea primitives used by every
// Multiversa wizard. It defines the Step contract that screens
// satisfy and the messages used to navigate between them. Concrete
// step types still live in each wizard's own package (e.g.
// internal/wizard/steps for the `init` wizard) — this package only
// owns the abstractions.
//
// The Step interface was originally introduced under
// internal/wizard/steps; v0.4.0 lifts it here so detect/stack/
// workspace/usb/lab wizards can share the same shape.
package tui

import tea "github.com/charmbracelet/bubbletea"

// Step is the contract every wizard screen satisfies. Returning a
// NextMsg from Update advances the wizard; BackMsg goes back;
// CancelMsg exits the program with status 2 (user cancel) so scripts
// can distinguish cancellation from a real failure.
type Step interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Step, tea.Cmd)
	View() string
	Title() string
}

// NextMsg signals that the current step is done; the wizard should
// advance to the next one.
type NextMsg struct{}

// BackMsg signals that the user wants to revisit the previous step.
type BackMsg struct{}

// CancelMsg signals that the user wants to abort the wizard. The
// host program is expected to translate this into exit code 2.
type CancelMsg struct{}

// Next returns a NextMsg as a tea.Msg so callers can use it as a
// command return value.
func Next() tea.Msg { return NextMsg{} }

// Back returns a BackMsg as a tea.Msg.
func Back() tea.Msg { return BackMsg{} }

// Cancel returns a CancelMsg as a tea.Msg.
func Cancel() tea.Msg { return CancelMsg{} }
