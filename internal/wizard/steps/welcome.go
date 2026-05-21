package steps

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type Welcome struct{}

func NewWelcome() Step { return Welcome{} }

func (Welcome) Title() string { return "Welcome" }

func (Welcome) Init() tea.Cmd { return nil }

func (w Welcome) Update(msg tea.Msg) (Step, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "enter", " ":
			return w, Next
		}
	}
	return w, nil
}

func (w Welcome) View() string {
	title := theme.Display.Render("Multiversa") + " " + theme.Accent.Italic(true).Render("Lab")
	tagline := theme.Body.Render("Orquesta el stack agentic curado.")
	subtagline := theme.Dim.Render("Curated agentic stack — one command.")

	manifesto := strings.Join([]string{
		theme.Label10("Ética"),
		theme.Accent.Render("\"La IA propone, tú decides.\""),
		"",
		theme.Label10("Lo que NO somos"),
		theme.Body.Render("· Autores de los motores — los orquestamos."),
		theme.Body.Render("· Lock-in de agente, modelo, o suscripción."),
		"",
		theme.Label10("Lo que SÍ somos"),
		theme.Body.Render("· Curaduría + arquitectura + sistema de diseño."),
		theme.Body.Render("· Atribución built-in a cada creador upstream."),
	}, "\n")

	hint := theme.Dim.Render("[enter] continuar  ·  [ctrl-c] salir")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		tagline,
		subtagline,
		"",
		manifesto,
		"",
		hint,
	)

	return theme.Box.Render(body)
}
