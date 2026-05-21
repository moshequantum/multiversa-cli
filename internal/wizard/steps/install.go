package steps

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moshequantum/multiversa-cli/internal/credits"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type engineStatus int

const (
	stPending engineStatus = iota
	stRunning
	stDone
	stError
)

// installPlan returns the shell command Multiversa would execute for each
// engine. v0.1 displays these as a dry-run; v0.2 will execute them.
func installPlan(id string) string {
	switch id {
	case "engram":
		return "go install github.com/Gentleman-Programming/engram/cmd/engram@latest"
	case "graphify":
		return "pipx install graphify"
	case "gentle-ai":
		return "go install github.com/Gentleman-Programming/gentle-ai/cmd/gentle@latest"
	case "gentle-pi":
		return "npm install -g gentle-pi"
	case "codegraph":
		return "npm install -g codegraph"
	case "mirofish":
		return "docker pull ghcr.io/666ghj/mirofish:latest   # AGPL · external-only"
	default:
		return "# unknown engine: " + id
	}
}

type Install struct {
	Engines  []string
	Backend  string
	width    int
	statuses map[string]engineStatus
	current  int
	spinner  spinner.Model
	finished bool
}

// tickInstallMsg is fired when the simulated install for engine ID finishes.
type tickInstallMsg struct{ id string }

func NewInstall() Step {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Chartreuse)
	return &Install{
		statuses: map[string]engineStatus{},
		spinner:  sp,
	}
}

func (*Install) Title() string { return "Install" }

func (i *Install) Set(engines []string, backend string) {
	i.Engines = engines
	i.Backend = backend
	for _, id := range engines {
		if _, ok := i.statuses[id]; !ok {
			i.statuses[id] = stPending
		}
	}
}

func (i *Install) Init() tea.Cmd {
	if len(i.Engines) == 0 {
		i.finished = true
		return nil
	}
	i.statuses[i.Engines[0]] = stRunning
	return tea.Batch(i.spinner.Tick, i.simulate(i.Engines[0]))
}

// simulate returns a Cmd that pretends to install `id` after a short delay.
// v0.2 will replace this with real exec.Command invocations.
func (i *Install) simulate(id string) tea.Cmd {
	return tea.Tick(700*time.Millisecond, func(time.Time) tea.Msg {
		return tickInstallMsg{id: id}
	})
}

func (i *Install) Update(msg tea.Msg) (Step, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		i.width = m.Width
		return i, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		i.spinner, cmd = i.spinner.Update(msg)
		return i, cmd

	case tickInstallMsg:
		i.statuses[m.id] = stDone
		i.current++
		if i.current < len(i.Engines) {
			next := i.Engines[i.current]
			i.statuses[next] = stRunning
			return i, i.simulate(next)
		}
		i.finished = true
		return i, nil

	case tea.KeyMsg:
		if i.finished {
			switch m.String() {
			case "enter", "q", "esc":
				return i, tea.Quit
			}
		}
	}
	return i, nil
}

func (i *Install) View() string {
	title := theme.Display.Render("Instalando")
	if i.finished {
		title = theme.Display.Render("Listo")
	}

	mode := theme.Warn.Render("MODO DRY-RUN · v0.1") + " " +
		theme.Dim.Render("— se imprimen los comandos pero no se ejecutan todavía.")

	rows := []string{}
	for _, id := range i.Engines {
		rows = append(rows, i.renderRow(id))
	}
	if len(rows) == 0 {
		rows = append(rows, theme.Dim.Render("(ningún motor seleccionado)"))
	}

	backendLine := theme.Label10("Backend") + " " + theme.Body.Render(i.Backend)
	if i.Backend == "" {
		backendLine = theme.Label10("Backend") + " " + theme.Dim.Render("local (default)")
	}

	var creditsBuf strings.Builder
	credits.Print(&creditsBuf)

	hint := theme.Dim.Render("[enter] terminar")
	if !i.finished {
		hint = theme.Dim.Render("Instalando...  (ctrl-c para cancelar)")
	}

	return theme.Frame(i.width, lipgloss.JoinVertical(lipgloss.Left,
		title,
		mode,
		"",
		theme.Label10("Motores"),
		strings.Join(rows, "\n"),
		"",
		backendLine,
		"",
		theme.Divider,
		creditsBuf.String(),
		hint,
	))
}

func (i *Install) renderRow(id string) string {
	st := i.statuses[id]
	var icon string
	switch st {
	case stPending:
		icon = theme.Dim.Render("○ ")
	case stRunning:
		icon = i.spinner.View() + " "
	case stDone:
		icon = theme.Accent.Render("✓ ")
	case stError:
		icon = theme.Warn.Render("✗ ")
	}
	name := theme.Body.Render(pad(id, 12))
	cmd := theme.Dim.Render(fmt.Sprintf("$ %s", installPlan(id)))
	return icon + name + cmd
}
