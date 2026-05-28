// Package steps holds the individual wizard screens for the `init`
// wizard. Each step is a self-contained Bubble Tea sub-model.
//
// As of v0.4.0 the Step contract and navigation messages live in
// internal/tui. This file re-exports them as type aliases so the
// existing init wizard keeps compiling unchanged while
// detect/stack/workspace/usb/lab share the same abstractions.
package steps

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// Step is an alias for tui.Step. Kept for backward compatibility with
// the existing init wizard. New wizards should import tui directly.
type Step = tui.Step

// NextMsg is an alias for tui.NextMsg.
type NextMsg = tui.NextMsg

// BackMsg is an alias for tui.BackMsg.
type BackMsg = tui.BackMsg

// Next emits a NextMsg as a tea.Cmd-compatible value.
func Next() tea.Msg { return tui.Next() }

// Back emits a BackMsg as a tea.Cmd-compatible value.
func Back() tea.Msg { return tui.Back() }
