package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// ProgressState is the lifecycle of a single ProgressItem.
type ProgressState int

const (
	// Pending — not started yet. Rendered dim.
	Pending ProgressState = iota
	// Running — in flight. Rendered with the accent color.
	Running
	// Done — completed successfully. Rendered with ✓.
	Done
	// Skipped — intentionally bypassed (e.g. already installed).
	Skipped
	// Failed — completed with error. Rendered as ✗ in warn style.
	Failed
)

// ProgressItem is one row in a ProgressList.
type ProgressItem struct {
	Label string
	Note  string
	State ProgressState
}

// ProgressList renders a vertical stack of items with state glyphs.
// Like Selector, it is stateless rendering — the host owns state
// transitions and feeds in the slice each frame.
type ProgressList struct {
	Items []ProgressItem
}

// Render returns the multi-line view for the progress list.
func (p ProgressList) Render() string {
	var b strings.Builder
	for _, it := range p.Items {
		glyph, styled := stateGlyph(it.State)
		b.WriteString("  ")
		b.WriteString(glyph)
		b.WriteString(" ")
		b.WriteString(styled.Render(it.Label))
		if it.Note != "" {
			b.WriteString("  ")
			b.WriteString(theme.Dim.Render(it.Note))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Counts returns done/skipped/failed/pending counts.
func (p ProgressList) Counts() (done, skipped, failed, pending int) {
	for _, it := range p.Items {
		switch it.State {
		case Done:
			done++
		case Skipped:
			skipped++
		case Failed:
			failed++
		case Pending, Running:
			pending++
		}
	}
	return
}

// stateGlyph picks the glyph + style for a state. Returns the
// pre-styled glyph string AND the appropriate body style for the
// label, so callers stay consistent across all wizards.
func stateGlyph(s ProgressState) (string, lipgloss.Style) {
	switch s {
	case Done:
		return theme.Accent.Render("✓"), theme.Body
	case Skipped:
		return theme.Dim.Render("·"), theme.Dim
	case Failed:
		return theme.Warn.Render("✗"), theme.Warn
	case Running:
		return theme.Accent.Render("◐"), theme.Accent
	default: // Pending
		return theme.Dim.Render("·"), theme.Dim
	}
}
