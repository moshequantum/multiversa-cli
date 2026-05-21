package steps

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moshequantum/multiversa-cli/internal/backends"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type Backend struct {
	cursor   int
	choice   string
	backends []backends.Backend
}

func NewBackend() Step {
	bs := backends.List()
	return &Backend{backends: bs, choice: "local"}
}

func (*Backend) Title() string  { return "Backend" }
func (*Backend) Init() tea.Cmd  { return nil }
func (b *Backend) Choice() string { return b.choice }

func (b *Backend) Update(msg tea.Msg) (Step, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return b, nil
	}
	switch k.String() {
	case "up", "k":
		if b.cursor > 0 {
			b.cursor--
		}
	case "down", "j":
		if b.cursor < len(b.backends)-1 {
			b.cursor++
		}
	case "enter":
		b.choice = b.backends[b.cursor].ID()
		return b, Next
	case "b":
		return b, Back
	}
	return b, nil
}

func (b *Backend) View() string {
	title := theme.Display.Render("¿Backend opcional?")
	subtitle := theme.Dim.Render("Local SQLite es el default. Los backends remotos son para sync multi-dispositivo.")

	var rows []string
	for i, bk := range b.backends {
		cursor := "  "
		if i == b.cursor {
			cursor = theme.Accent.Render("▸ ")
		}
		rows = append(rows, cursor+theme.Body.Render(pad(bk.ID(), 12))+theme.Dim.Render(bk.DisplayName()))
	}

	hint := theme.Dim.Render("[↑↓] mover  ·  [enter] elegir  ·  [b] atrás")

	return theme.Box.Render(lipgloss.JoinVertical(lipgloss.Left,
		title, subtitle, "",
		strings.Join(rows, "\n"),
		"", hint,
	))
}
