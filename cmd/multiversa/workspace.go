// Multiversa workspace — MultiversaGroup private workspace setup.
//
// v0.4.0 unifies the UX behind the shared internal/tui primitives.
// When stdout is a TTY and --show is not passed, the command launches
// a Bubble Tea program that explains what will happen, checks
// prerequisites, and asks for a strict confirmation. Once the user
// confirms, the TUI exits cleanly and we hand control over to the
// embedded bash script — which needs raw stdin/stdout for its own
// interactive prompts (passphrases, etc.). Running a long interactive
// shell script inside a Bubble Tea AltScreen corrupts the terminal,
// so the consultive front and the executor live in separate phases.
//
// Exit codes:
//   0  success (or --show, or non-TTY abort)
//   1  prerequisites missing OR script failure
//   2  user cancel (Esc or n at the confirm prompt)
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

const workspaceScript = "setup_multiversa.sh"

// userCancelErr is a sentinel returned by the TUI flow when the user
// dismisses the confirmation. main.go's top-level error printer would
// render it as an "error: …" line, so we intercept it at the cobra
// RunE boundary and translate to os.Exit(2) cleanly.
var userCancelErr = errors.New("user cancelled workspace setup")

// newWorkspaceCmd configures the MultiversaGroup private workspace:
// SSH key for GitHub, GPG signing key, git identity, private repo
// clone, ~/.multiversa/ scaffolding, encrypted secrets vault.
//
// The destructive parts (key generation, repo clone, vault create)
// all live in the embedded bash script. This command is the
// consultive entry point: it explains what will happen, asks for
// confirmation, then hands off.
func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Configura el workspace privado MultiversaGroup (SSH, GPG, repos, bóveda).",
		Long: "Configura el workspace privado MultiversaGroup: llave SSH para\n" +
			"GitHub, llave GPG de firma, identidad git, clon del monorepo\n" +
			"privado, scaffolding de ~/.multiversa/, bóveda de secretos\n" +
			"cifrada.\n\n" +
			"El script de instalación viene embebido dentro del binario, así\n" +
			"que funciona en una máquina recién instalada sin necesidad de un\n" +
			"checkout de skills de Claude Code. Usa --show para imprimir el\n" +
			"cuerpo del script y salir.",
		RunE: func(cmd *cobra.Command, args []string) error {
			showOnly, _ := cmd.Flags().GetBool("show")
			err := runWorkspace(workspaceOpts{showOnly: showOnly, out: os.Stdout})
			// Translate the cancel sentinel into exit code 2 without
			// dragging an "error:" line through the user's terminal.
			if errors.Is(err, userCancelErr) {
				os.Exit(2)
			}
			return err
		},
	}
	cmd.Flags().Bool("show", false, "Imprime el cuerpo del script embebido y sale sin ejecutar.")
	return cmd
}

// workspaceOpts captures the cobra flags + the writer used for output.
// Threading the writer through lets tests inspect non-TTY behavior.
type workspaceOpts struct {
	showOnly bool
	out      io.Writer
}

// runWorkspace is the entry point shared by RunE and tests. It picks
// between three branches: --show, TUI, or plain stdin fallback.
func runWorkspace(opts workspaceOpts) error {
	if opts.out == nil {
		opts.out = os.Stdout
	}

	// --show always takes precedence: dump the script and exit.
	if opts.showOnly {
		data, err := readEmbeddedScript(workspaceScript)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(opts.out, string(data))
		return nil
	}

	report := detect.Run()
	missing := requiredMissing(report, []string{"git", "ssh"})

	if shouldRunWorkspaceTUI(opts) {
		return runWorkspaceTUI(report, missing)
	}
	return runWorkspaceNonInteractive(opts.out, report, missing)
}

// shouldRunWorkspaceTUI gates the Bubble Tea path. We require a real
// TTY on the configured writer; pipes, redirects, and the test buffer
// all fall through to the plain stdin fallback.
func shouldRunWorkspaceTUI(opts workspaceOpts) bool {
	f, ok := opts.out.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// runWorkspaceNonInteractive preserves the v0.3.0 stdin Y/N flow for
// CI, pipes, and non-TTY runs. The shape is intentionally identical
// to the previous implementation so existing scripts keep working.
func runWorkspaceNonInteractive(out io.Writer, report detect.Report, missing []string) error {
	fmt.Fprintln(out, theme.Accent.Render("multiversa workspace"))
	fmt.Fprintln(out, theme.Dim.Render("MultiversaGroup — configuración del workspace privado"))
	fmt.Fprintln(out)
	fmt.Fprintln(out, theme.Label.Render("script")+" "+workspaceScript+theme.Dim.Render(" (embebido)"))
	fmt.Fprintln(out, theme.Label.Render("hace")+"   "+theme.Body.Render("ssh-keygen · gpg --gen-key · git config · clone monorepo · ~/.multiversa init · bóveda"))
	fmt.Fprintln(out, theme.Label.Render("seguro")+" "+theme.Dim.Render("idempotente — al re-ejecutar se omiten pasos ya completados"))
	fmt.Fprintln(out)

	if len(missing) > 0 {
		fmt.Fprintln(out, theme.Warn.Render("Prerrequisitos faltantes: "+strings.Join(missing, ", ")))
		fmt.Fprintln(out, theme.Dim.Render("Ejecuta `multiversa stack --only=git` o tu gestor de paquetes primero."))
		return fmt.Errorf("prerrequisitos faltantes")
	}

	fmt.Fprint(out, theme.Label.Render("¿continuar? [y/N] "))
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil {
		fmt.Fprintln(out, theme.Dim.Render("cancelado"))
		return nil
	}
	if !tui.ConfirmDecision(ans) {
		fmt.Fprintln(out, theme.Dim.Render("cancelado"))
		return nil
	}

	return runEmbeddedScript(workspaceScript)
}

// runWorkspaceTUI runs the Bubble Tea consultive front. On confirm
// it exits cleanly and the caller runs the embedded script with raw
// stdio passthrough. On cancel it returns userCancelErr (→ exit 2);
// on prereq-missing it returns a plain error (→ exit 1).
func runWorkspaceTUI(report detect.Report, missing []string) error {
	m := NewWorkspaceModel(report, missing)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}
	fm, ok := final.(WorkspaceModel)
	if !ok {
		return fmt.Errorf("workspace: unexpected model type %T", final)
	}
	if len(fm.missing) > 0 {
		// The model already showed the missing-prereqs warning in the
		// TUI; print a plain reminder so the shell still records why
		// the command exited 1 in its scrollback.
		fmt.Fprintln(os.Stderr, theme.Warn.Render("Prerrequisitos faltantes: "+strings.Join(fm.missing, ", ")))
		return fmt.Errorf("prerrequisitos faltantes")
	}
	if fm.cancelled {
		return userCancelErr
	}
	if !fm.confirmed {
		// Reached when the program exits via something other than the
		// confirm or cancel paths (e.g. ctrl+c). Treat as a cancel.
		return userCancelErr
	}

	// TUI is closed: the bash script owns stdin/stdout/stderr from
	// here. Errors propagate as exit 1 via cobra's SilenceUsage.
	return runEmbeddedScript(workspaceScript)
}

// WorkspaceModel is the Bubble Tea model behind `multiversa workspace`.
// It is a single-screen program: info pane + prereq status + confirm
// prompt. Exit paths set the boolean flags so the host can act on the
// final state after p.Run returns.
type WorkspaceModel struct {
	report    detect.Report
	missing   []string
	width     int
	height    int
	confirmed bool
	cancelled bool
	// input is the buffered confirm answer. Strict ConfirmDecision is
	// the only path to confirmed=true.
	input string
}

// NewWorkspaceModel builds the model used by both the TUI and tests.
// It is exported so workspace_test.go can satisfy itself that the
// Model honors the tea.Model contract without spinning up a program.
func NewWorkspaceModel(r detect.Report, missing []string) WorkspaceModel {
	return WorkspaceModel{report: r, missing: missing}
}

// Init satisfies tea.Model. No async work to kick off.
func (m WorkspaceModel) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m WorkspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tui.CancelMsg:
		m.cancelled = true
		return m, tea.Quit
	case tea.KeyMsg:
		// If prerequisites are missing, the only valid action is to
		// dismiss the screen. We do NOT mark as cancelled because the
		// host translates the missing-prereqs case to exit 1.
		if len(m.missing) > 0 {
			switch msg.String() {
			case "esc", "q", "enter", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			m.cancelled = true
			return m, tui.Cancel
		case "enter":
			if tui.ConfirmDecision(m.input) {
				m.confirmed = true
				return m, tea.Quit
			}
			// Anything other than a strict yes is a cancel.
			m.cancelled = true
			return m, tui.Cancel
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			// Accept printable input only; ignore special keys.
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
		}
	}
	return m, nil
}

// View satisfies tea.Model.
func (m WorkspaceModel) View() string {
	var b strings.Builder
	b.WriteString(tui.Header("multiversa workspace", "MultiversaGroup — configuración del workspace privado", 0, 0))
	b.WriteByte('\n')

	b.WriteString(theme.Label.Render("script"))
	b.WriteString("  ")
	b.WriteString(theme.Body.Render(workspaceScript))
	b.WriteString(theme.Dim.Render(" (embebido)"))
	b.WriteByte('\n')

	b.WriteString(theme.Label.Render("hace"))
	b.WriteString("    ")
	b.WriteString(theme.Body.Render("ssh-keygen · gpg --gen-key · git config · clone monorepo · ~/.multiversa init · bóveda"))
	b.WriteByte('\n')

	b.WriteString(theme.Label.Render("seguro"))
	b.WriteString("  ")
	b.WriteString(theme.Dim.Render("idempotente — al re-ejecutar se omiten pasos ya completados"))
	b.WriteString("\n\n")

	if len(m.missing) > 0 {
		b.WriteString(theme.Warn.Render("Prerrequisitos faltantes: " + strings.Join(m.missing, ", ")))
		b.WriteByte('\n')
		b.WriteString(theme.Dim.Render("Ejecuta `multiversa stack --only=git` o tu gestor de paquetes primero."))
		b.WriteString("\n\n")
		b.WriteString(theme.Dim.Render("[enter] / [esc] cerrar"))
	} else {
		b.WriteString(theme.Accent.Render("Listo para configurar el workspace."))
		b.WriteString("\n\n")
		b.WriteString(theme.Label.Render("¿continuar? [y/N] "))
		b.WriteString(theme.Body.Render(m.input))
		b.WriteString(theme.Dim.Render("_"))
		b.WriteString("\n\n")
		b.WriteString(theme.Dim.Render("[enter] confirmar  ·  [esc] cancelar"))
	}

	body := b.String()
	if m.width > 0 {
		return theme.Frame(m.width, body)
	}
	return lipgloss.NewStyle().Render(body)
}

// requiredMissing returns the subset of `required` tools that are not
// installed according to the detect report. It is shared with usb.go.
func requiredMissing(r detect.Report, required []string) []string {
	have := map[string]bool{}
	for _, t := range r.Tools {
		if t.Installed {
			have[t.Name] = true
		}
	}
	var missing []string
	for _, req := range required {
		if !have[req] {
			missing = append(missing, req)
		}
	}
	return missing
}
