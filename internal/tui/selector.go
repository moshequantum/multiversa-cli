package tui

import (
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// SelectorItem is one row in a Selector list. Disabled rows are
// rendered dim and can't be chosen; Marker is an optional glyph the
// caller can use to flag installed/completed items.
type SelectorItem struct {
	Label    string
	Hint     string
	Marker   string // e.g. "✓" for installed
	Disabled bool
}

// Selector is a stateless renderer for a vertical list with a cursor.
// It is intentionally pure: no Bubble Tea wiring lives here. Wizards
// own their tea.Model and call Render() from their View().
type Selector struct {
	Items  []SelectorItem
	Cursor int
}

// Render returns the multi-line view for the selector.
func (s Selector) Render() string {
	var b strings.Builder
	for i, it := range s.Items {
		marker := " "
		if i == s.Cursor {
			marker = theme.Accent.Render(">")
		}
		row := marker + " " + it.Marker
		if it.Disabled {
			b.WriteString(theme.Dim.Render(row + " " + it.Label))
		} else {
			b.WriteString(theme.Body.Render(row + " " + it.Label))
		}
		if it.Hint != "" {
			b.WriteString("  ")
			b.WriteString(theme.Dim.Render(it.Hint))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// MoveDown advances the cursor, skipping disabled items.
func (s *Selector) MoveDown() {
	for i := s.Cursor + 1; i < len(s.Items); i++ {
		if !s.Items[i].Disabled {
			s.Cursor = i
			return
		}
	}
}

// MoveUp retreats the cursor, skipping disabled items.
func (s *Selector) MoveUp() {
	for i := s.Cursor - 1; i >= 0; i-- {
		if !s.Items[i].Disabled {
			s.Cursor = i
			return
		}
	}
}

// Selected returns the currently focused item or nil if the cursor
// is out of bounds (empty list).
func (s Selector) Selected() *SelectorItem {
	if s.Cursor < 0 || s.Cursor >= len(s.Items) {
		return nil
	}
	return &s.Items[s.Cursor]
}
