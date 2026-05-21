package steps

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moshequantum/multiversa-cli/internal/credits"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type Install struct {
	Engines []string
	Backend string
	done    bool
}

func NewInstall() Step { return &Install{} }

func (*Install) Title() string { return "Install" }
func (*Install) Init() tea.Cmd { return nil }

func (i *Install) Set(engines []string, backend string) {
	i.Engines = engines
	i.Backend = backend
}

func (i *Install) Update(msg tea.Msg) (Step, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "enter", "q":
			i.done = true
			return i, tea.Quit
		}
	}
	return i, nil
}

func (i *Install) View() string {
	title := theme.Display.Render("Listo para instalar")
	note := theme.Body.Render(fmt.Sprintf("Motores: %v   ·   Backend: %s", i.Engines, i.Backend))
	stub := theme.Warn.Render("Descarga/instalación real: v0.2.")
	stub2 := theme.Dim.Render("Esta v0.1 valida wizard + atribución + manifest. Los stack managers se completan en la siguiente fase.")

	var creditsBuf strings.Builder
	credits.Print(&creditsBuf)

	hint := theme.Dim.Render("[enter] terminar")

	return theme.Box.Render(lipgloss.JoinVertical(lipgloss.Left,
		title, "", note, stub, stub2, "",
		creditsBuf.String(),
		hint,
	))
}
