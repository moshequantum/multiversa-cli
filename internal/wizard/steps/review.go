package steps

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type Review struct {
	Engines []string
	Backend string
	width   int
}

func NewReview() Step { return &Review{} }

func (*Review) Title() string { return "Review" }
func (*Review) Init() tea.Cmd { return nil }

func (r *Review) Set(engines []string, backend string) {
	r.Engines = engines
	r.Backend = backend
}

func (r *Review) Update(msg tea.Msg) (Step, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = m.Width
		return r, nil
	case tea.KeyMsg:
		switch m.String() {
		case "enter", "y", "Y":
			return r, Next
		case "b", "n", "N":
			return r, Back
		}
	}
	return r, nil
}

func (r *Review) View() string {
	title := theme.Display.Render("Confirma")
	engines := theme.Body.Render(strings.Join(r.Engines, ", "))
	if r.Engines == nil || len(r.Engines) == 0 {
		engines = theme.Dim.Render("(ninguno)")
	}
	backend := theme.Body.Render(r.Backend)
	if r.Backend == "" {
		backend = theme.Dim.Render("local (default)")
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "",
		theme.Label10("Motores"), engines, "",
		theme.Label10("Backend"), backend, "",
		theme.Accent.Render("¿Instalar? [enter/y] sí  ·  [b/n] revisar"),
	)
	return theme.Frame(r.width, body)
}
