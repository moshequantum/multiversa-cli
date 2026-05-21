package steps

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type Consent struct {
	accepted bool
}

func NewConsent() Step { return &Consent{} }

func (*Consent) Title() string { return "Consent" }
func (*Consent) Init() tea.Cmd { return nil }

func (c *Consent) Update(msg tea.Msg) (Step, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "y", "Y":
			c.accepted = true
			return c, Next
		case "n", "N", "esc":
			return c, tea.Quit
		case "b":
			return c, Back
		}
	}
	return c, nil
}

func (c *Consent) View() string {
	title := theme.Display.Render("La IA propone, tú decides.")
	body := strings.Join([]string{
		theme.Body.Render("Multiversa nunca toma acciones irreversibles sin tu aprobación explícita."),
		theme.Body.Render("Cada paso del asistente te muestra qué hace antes de hacerlo."),
		"",
		theme.Label10("AGPL-3.0"),
		theme.Body.Render("Si más adelante eliges MiroFish, lo invocamos como servicio externo."),
		theme.Body.Render("Multiversa nunca embebe código AGPL — su licencia es viral."),
	}, "\n")

	prompt := theme.Accent.Render("¿Aceptas el contrato? [y/n]")
	hint := theme.Dim.Render("[b] atrás")

	return theme.Box.Render(lipgloss.JoinVertical(lipgloss.Left,
		title, "", body, "", prompt, hint,
	))
}
