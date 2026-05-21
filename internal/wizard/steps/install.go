package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moshequantum/multiversa-cli/internal/credits"
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
	"github.com/moshequantum/multiversa-cli/internal/stack"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type engineStatus int

const (
	stPending engineStatus = iota
	stPrereqMissing
	stRunning
	stDone
	stError
	stSkipped
)

type Install struct {
	Engines          []string
	Backend          string
	DryRun           bool
	AgplAcknowledged bool
	width            int
	statuses         map[string]engineStatus
	results          map[string]xexec.Result
	current          int
	spinner          spinner.Model
	finished         bool
}

// installResultMsg is fired by runEngine when an engine finishes installing.
type installResultMsg struct {
	id     string
	result xexec.Result
	status engineStatus
}

func NewInstall() Step {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Chartreuse)
	return &Install{
		statuses: map[string]engineStatus{},
		results:  map[string]xexec.Result{},
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

func (i *Install) SetDryRun(d bool) { i.DryRun = d }

func (i *Install) SetAgplAcknowledged(ack bool) { i.AgplAcknowledged = ack }

func (i *Install) Init() tea.Cmd {
	if len(i.Engines) == 0 {
		i.finished = true
		return nil
	}
	return tea.Batch(i.spinner.Tick, i.startEngine(0))
}

// startEngine prepares the engine at idx (prereq check + status), then returns
// the Cmd that runs the install (or completes immediately for prereq misses
// and dry-run).
func (i *Install) startEngine(idx int) tea.Cmd {
	id := i.Engines[idx]
	eng, err := stack.Resolve(id)
	if err != nil {
		i.statuses[id] = stError
		return func() tea.Msg {
			return installResultMsg{id: id, status: stError,
				result: xexec.Result{Err: err}}
		}
	}

	if id == "mirofish" && !i.AgplAcknowledged {
		i.statuses[id] = stError
		return func() tea.Msg {
			return installResultMsg{
				id:     id,
				status: stError,
				result: xexec.Result{Err: stack.ErrAgplConsentRequired},
			}
		}
	}

	// Prereq check (e.g. `go`, `pipx`, `npm`, `docker` on PATH).
	if pre := eng.Prereq(); pre != "" && !xexec.Check(pre) {
		i.statuses[id] = stPrereqMissing
		return func() tea.Msg {
			return installResultMsg{
				id:     id,
				status: stPrereqMissing,
				result: xexec.Result{
					Cmd: prereqMissingMsg(pre),
					Err: fmt.Errorf("%q not found on PATH", pre),
				},
			}
		}
	}

	i.statuses[id] = stRunning
	cmd := eng.Command("latest")
	if len(cmd) == 0 {
		i.statuses[id] = stError
		return func() tea.Msg {
			return installResultMsg{
				id:     id,
				status: stError,
				result: xexec.Result{Err: fmt.Errorf("empty install command for %q", id)},
			}
		}
	}
	return i.runCommand(id, cmd)
}

// runCommand returns the Cmd that actually executes (or pretends to, when
// DryRun is set).
func (i *Install) runCommand(id string, cmd []string) tea.Cmd {
	if i.DryRun {
		return func() tea.Msg {
			return installResultMsg{
				id:     id,
				status: stSkipped,
				result: xexec.Result{Cmd: strings.Join(cmd, " ")},
			}
		}
	}
	return func() tea.Msg {
		r := xexec.Run(cmd[0], cmd[1:]...)
		st := stDone
		if r.Err != nil {
			st = stError
		}
		return installResultMsg{id: id, status: st, result: r}
	}
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

	case installResultMsg:
		i.statuses[m.id] = m.status
		i.results[m.id] = m.result
		i.current++
		if i.current < len(i.Engines) {
			return i, i.startEngine(i.current)
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

	mode := theme.Accent.Render("REAL · v0.2")
	if i.DryRun {
		mode = theme.Warn.Render("DRY-RUN · sólo previsualización")
	}

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

	summary := ""
	if i.finished {
		summary = i.renderSummary()
	}

	creditsText := ""
	if i.finished {
		var b strings.Builder
		credits.Print(&b)
		creditsText = b.String()
	}

	hint := theme.Dim.Render("Instalando...  (ctrl-c para cancelar)")
	if i.finished {
		hint = theme.Dim.Render("[enter] terminar")
	}

	parts := []string{title, mode, "", theme.Label10("Motores")}
	parts = append(parts, rows...)
	parts = append(parts, "", backendLine)
	if summary != "" {
		parts = append(parts, "", summary)
	}
	if creditsText != "" {
		parts = append(parts, "", creditsText)
	}
	parts = append(parts, "", hint)

	return theme.Frame(i.width, lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (i *Install) renderRow(id string) string {
	st := i.statuses[id]
	var icon, suffix string
	switch st {
	case stPending:
		icon = theme.Dim.Render("○ ")
		suffix = theme.Dim.Render("(pendiente)")
	case stRunning:
		icon = i.spinner.View() + " "
		eng, _ := stack.Resolve(id)
		if eng != nil {
			suffix = theme.Dim.Render("$ " + strings.Join(eng.Command("latest"), " "))
		}
	case stDone:
		icon = theme.Accent.Render("✓ ")
		r := i.results[id]
		suffix = theme.Dim.Render(fmt.Sprintf("ok · %s", r.Duration.Round(100_000_000)))
	case stError:
		icon = theme.Warn.Render("✗ ")
		r := i.results[id]
		msg := r.LastLine()
		if msg == "" && r.Err != nil {
			msg = r.Err.Error()
		}
		suffix = theme.Warn.Render(truncate(msg, 60))
	case stPrereqMissing:
		icon = theme.Warn.Render("⚠ ")
		r := i.results[id]
		suffix = theme.Warn.Render(r.Cmd)
	case stSkipped:
		icon = theme.Dim.Render("· ")
		r := i.results[id]
		suffix = theme.Dim.Render("$ " + r.Cmd + "  (dry-run)")
	}
	name := theme.Body.Render(pad(id, 12))
	return icon + name + suffix
}

func (i *Install) renderSummary() string {
	var ok, fail, skip, warn int
	for _, st := range i.statuses {
		switch st {
		case stDone:
			ok++
		case stError:
			fail++
		case stSkipped:
			skip++
		case stPrereqMissing:
			warn++
		}
	}
	parts := []string{}
	if ok > 0 {
		parts = append(parts, theme.Accent.Render(fmt.Sprintf("✓ %d instalado", ok)))
	}
	if fail > 0 {
		parts = append(parts, theme.Warn.Render(fmt.Sprintf("✗ %d falló", fail)))
	}
	if warn > 0 {
		parts = append(parts, theme.Warn.Render(fmt.Sprintf("⚠ %d sin prereq", warn)))
	}
	if skip > 0 {
		parts = append(parts, theme.Dim.Render(fmt.Sprintf("· %d dry-run", skip)))
	}
	return theme.Divider + "\n" + strings.Join(parts, "   ")
}

func prereqMissingMsg(tool string) string {
	switch tool {
	case "brew":
		return "install Homebrew — see https://brew.sh"
	case "go":
		return "install Go from https://go.dev/dl/"
	case "pipx":
		return "install pipx — see https://pipx.pypa.io"
	case "npm":
		return "install Node.js + npm from https://nodejs.org"
	case "docker":
		return "install Docker from https://docs.docker.com/get-docker/"
	default:
		return tool + " not found on PATH"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
