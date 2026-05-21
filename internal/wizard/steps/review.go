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
}

func NewReview() Step { return &Review{} }

func (*Review) Title() string { return "Review" }
func (*Review) Init() tea.Cmd { return nil }

func (r *Review) Set(engines []string, backend string) {
	r.Engines = engines
	r.Backend = backend
}

func (r *Review) Update(msg tea.Msg) (Step, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
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
	if engines == "" {
		engines = theme.Dim.Render("(ninguno)")
	}
	backend := theme.Body.Render(r.Backend)

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "",
		theme.Label10("Motores"), engines, "",
		theme.Label10("Backend"), backend, "",
		theme.Accent.Render("¿Instalar? [enter/y] sí  ·  [b/n] revisar"),
	)
	return theme.Box.Render(body)
}
