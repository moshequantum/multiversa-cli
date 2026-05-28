// Multiversa detect — host scanner with dual rendering surface.
//
// When stdout is an interactive TTY this command runs a Bubble Tea
// program built on top of the shared `internal/tui` primitives
// (Header, Selector, Verbosity). When stdout is piped/redirected/CI,
// it falls back to the plain v0.3.0 renderer so existing scripts and
// the `/lab-setup` Claude Code skill keep working unchanged.
//
// The scan itself is read-only — see internal/detect. Cancelling the
// TUI (q/Esc) exits with status 0 because there is nothing to roll
// back: the user just dismissed a report.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/profile"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// detectLong is exported as a var so newDoctorCmd can reuse it.
const detectLong = "Ejecuta un escaneo de solo lectura del entorno local. Reporta\n" +
	"el SO y el gestor de paquetes, el toolchain de desarrollo (Go,\n" +
	"Rust, Python, Node, pnpm, Docker, …), el estado del CLI\n" +
	"Multiversa y los motores curados.\n\n" +
	"Este comando es seguro en cualquier entorno: nunca instala,\n" +
	"descarga ni modifica nada. Usa `multiversa init` después para\n" +
	"actuar sobre los hallazgos."

// runDetect is the shared entry point for both `detect` and `doctor`.
// It picks the TUI vs. plain branch based on whether stdout is a TTY.
func runDetect(stdout io.Writer) error {
	report := detect.Run()

	if !isTTY(stdout) {
		// CI / pipe / redirect: keep v0.3.0 plain output byte-for-byte.
		report.Render(stdout)
		return nil
	}

	// Pick verbosity from the persisted profile. A missing profile is
	// the most common first-run case — fall back to Standard.
	verbosity := tui.Standard
	if p, err := profile.Load(); err == nil {
		verbosity = tui.VerbosityForLevel(string(p.Level))
	} else if !errors.Is(err, os.ErrNotExist) {
		// A real load error (malformed TOML, permission denied, …) is
		// not fatal: we just keep the default verbosity and continue.
		verbosity = tui.Standard
	}

	m := NewDetectModel(report, verbosity)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// isTTY checks whether w is an *os.File pointing at a terminal. Any
// other writer (bytes.Buffer in tests, pipes, …) is treated as
// non-interactive.
func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

// DetectModel is the Bubble Tea model behind `multiversa detect`. It
// is a single-step program: report on top, navigable category list
// below, detail pane for the focused category. q/Esc exits cleanly.
type DetectModel struct {
	report    detect.Report
	verbosity tui.Verbosity
	selector  tui.Selector
	width     int
	height    int
	quitting  bool
}

// NewDetectModel builds the model used by both the TUI and tests. It
// is intentionally exported so detect_test.go can satisfy itself that
// the Model honors the tea.Model contract without spinning up a real
// program.
func NewDetectModel(r detect.Report, v tui.Verbosity) DetectModel {
	items := []tui.SelectorItem{
		{Label: "Sistema operativo", Hint: osHint(r)},
		{Label: "Dev stack", Hint: devHint(r)},
		{Label: "Multiversa", Hint: mvHint(r)},
	}
	return DetectModel{
		report:    r,
		verbosity: v,
		selector:  tui.Selector{Items: items},
	}
}

// Init satisfies tea.Model. No async work to kick off.
func (m DetectModel) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m DetectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			m.selector.MoveUp()
		case "down", "j":
			m.selector.MoveDown()
		}
	}
	return m, nil
}

// View satisfies tea.Model.
func (m DetectModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	// Single-shot command: step=0/total=0 omits the progress crumb.
	b.WriteString(tui.Header("multiversa detect", "host scan", 0, 0))
	b.WriteByte('\n')

	// Summary line — one tight row of totals.
	ti, tt := m.report.ReadyTools()
	ei, et := m.report.ReadyEngines()
	summary := fmt.Sprintf("listos: %d/%d herramientas · %d/%d motores", ti, tt, ei, et)
	b.WriteString(theme.Dim.Render(summary))
	b.WriteString("\n\n")

	b.WriteString(theme.Label.Render("Categorías"))
	b.WriteByte('\n')
	b.WriteString(m.selector.Render())
	b.WriteByte('\n')

	b.WriteString(theme.Label.Render("Detalle"))
	b.WriteByte('\n')
	b.WriteString(m.detailPane())
	b.WriteByte('\n')

	hint := tui.Choose(m.verbosity,
		"[↑/↓] navegar  ·  [q]/[esc] salir",
		"[↑/↓] · [q] salir",
		"q salir",
	)
	b.WriteString(theme.Dim.Render(hint))

	body := b.String()
	if m.width > 0 {
		return theme.Frame(m.width, body)
	}
	// No size info yet — render raw so tests can still inspect View().
	return lipgloss.NewStyle().Render(body)
}

// detailPane renders the body for the currently-selected category.
func (m DetectModel) detailPane() string {
	switch m.selector.Cursor {
	case 0:
		return m.renderOS()
	case 1:
		return m.renderTools()
	case 2:
		return m.renderMultiversa()
	default:
		return ""
	}
}

func (m DetectModel) renderOS() string {
	r := m.report
	var b strings.Builder
	b.WriteString(detailKV("kind", fmt.Sprintf("%s/%s", r.OS.Kind, r.OS.Arch)))
	if r.OS.Distro != "" {
		b.WriteString(detailKV("distro", r.OS.Distro))
	}
	if r.OS.Version != "" {
		b.WriteString(detailKV("version", r.OS.Version))
	}
	pkg := r.OS.PkgMgr
	if pkg == "" {
		pkg = theme.Warn.Render("sin gestor de paquetes")
	}
	b.WriteString(detailKV("pkg mgr", pkg))
	return b.String()
}

func (m DetectModel) renderTools() string {
	var b strings.Builder
	for _, t := range m.report.Tools {
		var status string
		switch {
		case t.Warn && t.Installed:
			status = theme.Warn.Render("⚠ " + t.Version)
		case t.Installed:
			status = theme.Accent.Render("✓ ") + theme.Body.Render(t.Version)
		case t.Advisory:
			status = theme.Dim.Render("· opcional")
		default:
			status = theme.Dim.Render("· no instalado")
		}
		b.WriteString(detailKV(t.Name, status))
	}
	// Policy nudge — pnpm-only across the ecosystem.
	for _, t := range m.report.Tools {
		if t.Name == "npm" && t.Installed {
			b.WriteString("\n")
			b.WriteString(theme.Warn.Render("⚠ npm presente — la política Multiversa es solo pnpm."))
			b.WriteByte('\n')
			break
		}
	}
	return b.String()
}

func (m DetectModel) renderMultiversa() string {
	r := m.report
	var b strings.Builder
	cli := theme.Dim.Render("· no está en PATH")
	if r.Multiversa.CLIInstalled {
		v := r.Multiversa.CLIVersion
		if v == "" {
			v = "instalado"
		}
		cli = theme.Accent.Render("✓ ") + theme.Body.Render(v)
	}
	b.WriteString(detailKV("cli", cli))

	if r.Multiversa.HomeDir != "" {
		b.WriteString(detailKV("home", theme.Body.Render(r.Multiversa.HomeDir)))
	} else {
		b.WriteString(detailKV("home", theme.Dim.Render("· ~/.multiversa ausente")))
	}

	for _, e := range r.Multiversa.Engines {
		var status string
		switch {
		case e.Installed && e.Version != "":
			status = theme.Accent.Render("✓ ") + theme.Body.Render(e.Version)
		case e.Installed:
			status = theme.Accent.Render("✓ ") + theme.Body.Render("instalado")
		case e.OptIn:
			status = theme.Dim.Render("· opt-in, no instalado")
		default:
			status = theme.Dim.Render("· no instalado")
		}
		b.WriteString(detailKV(e.ID, status))
	}
	return b.String()
}

// detailKV mirrors the layout of detect/render.go without importing
// from that package (kv is unexported there) so the TUI keeps the
// same visual alignment as the plain renderer.
func detailKV(key, value string) string {
	keyStyled := lipgloss.NewStyle().
		Foreground(theme.Muted).
		Width(14).
		Render(key)
	return "  " + keyStyled + value + "\n"
}

// osHint returns the one-line hint shown next to the OS category.
func osHint(r detect.Report) string {
	pkg := r.OS.PkgMgr
	if pkg == "" {
		pkg = "sin pkg mgr"
	}
	return fmt.Sprintf("%s · %s", r.OS.Kind, pkg)
}

// devHint summarizes the dev-stack readiness for the category list.
func devHint(r detect.Report) string {
	ti, tt := r.ReadyTools()
	return fmt.Sprintf("%d/%d listos", ti, tt)
}

// mvHint summarizes the Multiversa engine readiness.
func mvHint(r detect.Report) string {
	ei, et := r.ReadyEngines()
	return fmt.Sprintf("%d/%d motores", ei, et)
}

// newDetectCmd is the canonical environment scanner. Read-only.
func newDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "detect",
		Aliases: []string{"scan"},
		Short:   "Escanea el host: SO, gestor de paquetes, dev stack, estado Multiversa.",
		Long:    detectLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDetect(os.Stdout)
		},
	}
}

// newDoctorCmd keeps the npm/brew-style `doctor` alias alive. It
// delegates to runDetect so the report shape stays single-source.
func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "doctor",
		Short:  "Alias de `multiversa detect`.",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDetect(os.Stdout)
		},
	}
}
