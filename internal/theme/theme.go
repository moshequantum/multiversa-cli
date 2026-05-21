// Package theme holds the Multiversa visual tokens used across the CLI.
// Carbon background, Chartreuse single accent, Ivory text. Never two accents
// at hero scale.
package theme

import "github.com/charmbracelet/lipgloss"

const (
	Carbon     = lipgloss.Color("#0A0A0F")
	Surface    = lipgloss.Color("#121217")
	Chartreuse = lipgloss.Color("#BDEB34")
	Ivory      = lipgloss.Color("#FAFCE8")
	Muted      = lipgloss.Color("#7A7A82")
	Faint      = lipgloss.Color("#3A3A42")
	Danger     = lipgloss.Color("#FF6B6B")
)

var (
	Display = lipgloss.NewStyle().Foreground(Ivory).Italic(true).Bold(true)
	Body    = lipgloss.NewStyle().Foreground(Ivory)
	Accent  = lipgloss.NewStyle().Foreground(Chartreuse).Bold(true)
	Label   = lipgloss.NewStyle().Foreground(Muted).Bold(true)
	Dim     = lipgloss.NewStyle().Foreground(Faint)
	Warn    = lipgloss.NewStyle().Foreground(Danger).Bold(true)

	Box = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Chartreuse).
		Padding(1, 2)

	Divider = lipgloss.NewStyle().
		Foreground(Faint).
		Render("───────────────────────────────────────")
)

// Frame renders content inside the canonical Multiversa box, fitting the box
// to the available terminal width with a 2-column outer margin on each side.
// Falls back to width=80 when the terminal size is unknown.
func Frame(width int, content string) string {
	if width <= 0 {
		width = 80
	}
	// Box border + padding consume 2 (border) + 4 (padding) = 6 cols.
	// Outer margin of 2 each side = 4 cols.
	inner := width - 10
	if inner < 20 {
		inner = 20
	}
	margin := lipgloss.NewStyle().MarginLeft(2).MarginRight(2)
	return margin.Render(Box.Width(inner).Render(content))
}

// HardWrap rewraps long lines to fit within width. Strips trailing spaces.
func HardWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}

func Label10(s string) string {
	return Label.Render(uppercase(s))
}

func uppercase(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			r = r - 'a' + 'A'
		}
		out = append(out, r)
	}
	return string(out)
}
