package tui

import (
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// LayerStatus is one layer entry in the Sidebar of the meta-wizard.
// It represents a vertical group of related Steps (Técnica /
// Identitaria / Operacional), each with its own per-step progress.
type LayerStatus struct {
	Name      string // e.g. "Técnica"
	Tagline   string // optional one-line description rendered dim
	Steps     []ProgressItem
	IsCurrent bool // true when the cursor is inside this layer
}

// Sidebar renders the layered overview shown on the left of
// `multiversa lab`. It is intentionally text-only — no fixed-width
// padding — so the wizard host can wrap it in any layout primitive
// (lipgloss.JoinHorizontal, etc.) without fighting alignment.
type Sidebar struct {
	Layers []LayerStatus
}

// Render returns the multi-line view for the sidebar.
func (s Sidebar) Render() string {
	var b strings.Builder
	for i, l := range s.Layers {
		header := l.Name
		if l.IsCurrent {
			header = theme.Accent.Render("● " + l.Name)
		} else {
			header = theme.Body.Render("○ " + l.Name)
		}
		b.WriteString(header)
		b.WriteByte('\n')
		if l.Tagline != "" {
			b.WriteString("  ")
			b.WriteString(theme.Dim.Render(l.Tagline))
			b.WriteByte('\n')
		}
		for _, st := range l.Steps {
			glyph, _ := stateGlyph(st.State)
			b.WriteString("    ")
			b.WriteString(glyph)
			b.WriteByte(' ')
			b.WriteString(theme.Body.Render(st.Label))
			b.WriteByte('\n')
		}
		if i < len(s.Layers)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
