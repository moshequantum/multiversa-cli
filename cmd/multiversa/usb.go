// Multiversa usb — encrypted bootable USB lab with dual rendering
// surface and a two-gate confirmation pattern.
//
// This is the MOST DESTRUCTIVE command in the CLI: the embedded script
// wipes the target device and writes a LUKS container (Linux) or
// drives a guided VeraCrypt + balenaEtcher flow (macOS). The user
// experience is structured around two independent gates that BOTH
// must pass before any byte is written:
//
//	Gate 1 (this file, in the Go TUI): user types EXACTLY "i understand".
//	                                   Anything else cancels with exit 2.
//	Gate 2 (the embedded bash script):  user types the device path TWICE.
//
// Both gates are intentional and complementary. Gate 1 catches the
// "I hit enter on the wrong terminal" mistake before the script even
// loads. Gate 2 catches the "I typed the wrong device" mistake the
// only way that is robust — by making the user re-type the path. The
// bash scripts already implement gate 2; do not remove or weaken it.
//
// When stdout is a TTY and --show is NOT passed → run the Bubble Tea
// program. When --show is passed → print the embedded script body.
// When stdout is not a TTY → fall back to the v0.3.0 stdin Y/N flow.
// On Windows → print a friendly notice and exit 0.
//
// Exit codes:
//
//	0 success (script ran, or --show, or Windows notice)
//	1 prereq missing, unsupported OS, or script failure
//	2 user cancel (gate 1 declined, esc, ctrl+c, or stdin no)
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/profile"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// usbConfirmPhrase is the EXACT sentence the user must type at gate 1.
// Comparison is case-insensitive and trimmed but the phrase itself is
// fixed: a single typo cancels. Keep this string a const so the tests
// and the View() share one source of truth.
const usbConfirmPhrase = "i understand"

// newUSBCmd registers the `multiversa usb` subcommand.
func newUSBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usb",
		Short: "Crea un USB cifrado y arrancable (LUKS en Linux, guiado VeraCrypt/balenaEtcher en macOS).",
		Long: "Crea un USB cifrado y arrancable: LUKS de disco completo en\n" +
			"Linux, flujo guiado VeraCrypt + balenaEtcher en macOS.\n\n" +
			"Esta operación es DESTRUCTIVA: el dispositivo destino se borra.\n" +
			"El asistente te pide escribir exactamente \"i understand\" antes\n" +
			"de cargar el script, y el script te pide la ruta del dispositivo\n" +
			"DOS VECES antes de cualquier escritura. Ambas barreras existen\n" +
			"a propósito.\n\n" +
			"El script específico de cada plataforma está embebido en el\n" +
			"binario. Usa --show para imprimir el cuerpo del script sin\n" +
			"ejecutarlo.",
		RunE: func(cmd *cobra.Command, args []string) error {
			showOnly, _ := cmd.Flags().GetBool("show")
			return runUSB(os.Stdout, showOnly)
		},
	}
	cmd.Flags().Bool("show", false, "Imprime el cuerpo del script embebido sin ejecutarlo.")
	return cmd
}

// runUSB is the shared entry point. It picks the right rendering
// surface (TUI / plain / --show / Windows notice) and orchestrates
// gate 1 before delegating to the embedded script that owns gate 2.
func runUSB(stdout io.Writer, showOnly bool) error {
	report := detect.Run()

	scriptName, err := usbScriptFor(report.OS.Kind)
	if err != nil {
		// Windows is the special case: there is nothing to install yet
		// from Windows itself; tell the user how to boot a Linux ISO
		// and exit 0 — no damage done.
		if report.OS.Kind == "windows" {
			fmt.Fprintln(stdout, theme.Accent.Render("multiversa usb"))
			fmt.Fprintln(stdout, theme.Warn.Render("Crear el USB cifrado desde Windows aún no es soportado."))
			fmt.Fprintln(stdout, theme.Dim.Render("Arranca desde un Linux live ISO y vuelve a correr `multiversa usb`."))
			return nil
		}
		return err
	}

	// --show short-circuits everything: it never invokes the script
	// and never asks for confirmation. The bytes are the script body.
	if showOnly {
		data, err := readEmbeddedScript(scriptName)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(data))
		return nil
	}

	// Prerequisite check: on Linux we need cryptsetup before going
	// any further. macOS uses the GUI tools, so the bash script does
	// the check itself.
	required := requiredForUSB(report.OS.Kind)
	missing := requiredMissing(report, required)
	if len(missing) > 0 {
		fmt.Fprintln(stdout, theme.Accent.Render("multiversa usb"))
		fmt.Fprintln(stdout, theme.Warn.Render("Faltan prerequisitos: "+strings.Join(missing, ", ")))
		switch report.OS.Kind {
		case "linux":
			fmt.Fprintln(stdout, theme.Dim.Render("Instala con: sudo "+report.OS.PkgMgr+" install cryptsetup"))
		}
		return fmt.Errorf("prerequisites missing")
	}

	// Pick the rendering surface. The TUI path runs the Bubble Tea
	// program for gate 1; the plain path falls back to a stdin prompt
	// so CI, pipes, and redirected runs keep working unchanged.
	if isTTY(stdout) {
		return runUSBTUI(report, scriptName)
	}
	return runUSBPlain(stdout, report, scriptName)
}

// usbScriptFor returns the embedded script name for the host OS, or
// an error for unsupported OSes. Windows is handled by the caller.
func usbScriptFor(osKind string) (string, error) {
	switch osKind {
	case "linux":
		return "encrypted_usb_linux.sh", nil
	case "darwin":
		return "encrypted_usb_macos.sh", nil
	case "windows":
		return "", fmt.Errorf("windows is handled separately")
	default:
		return "", fmt.Errorf("unsupported OS for usb command: %s", osKind)
	}
}

// requiredForUSB lists the binaries that must be on PATH before gate 1
// even renders. We keep the list narrow on purpose: false positives
// here are more annoying than a missing tool surfacing inside the
// bash script.
func requiredForUSB(osKind string) []string {
	switch osKind {
	case "linux":
		return []string{"cryptsetup"}
	case "darwin":
		// VeraCrypt and balenaEtcher may live in /Applications, so we
		// let the script probe for them — PATH-only detection here
		// would produce false negatives for GUI installs.
		return nil
	}
	return nil
}

// runUSBPlain is the non-TTY fallback. It mirrors the v0.3.0 UX so
// scripts and `/lab-setup` keep working. Gate 1 here is the typed
// phrase via stdin (same string contract as the TUI).
func runUSBPlain(stdout io.Writer, report detect.Report, scriptName string) error {
	fmt.Fprintln(stdout, theme.Accent.Render("multiversa usb"))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, theme.Label.Render("host")+"   "+theme.Body.Render(fmt.Sprintf("%s/%s · %s", report.OS.Kind, report.OS.Arch, report.OS.Distro)))
	fmt.Fprintln(stdout, theme.Label.Render("script")+" "+scriptName+theme.Dim.Render(" (embebido)"))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, theme.Warn.Render("⚠ destructivo — borra el dispositivo destino. Ten la ruta lista (ej. /dev/sdb en Linux, disk4 en macOS)."))
	fmt.Fprintln(stdout, theme.Dim.Render("El script te pedirá la ruta DOS veces antes de cualquier escritura."))
	fmt.Fprintln(stdout)
	fmt.Fprint(stdout, theme.Label.Render("escribe \"i understand\" para continuar: "))

	var line string
	if _, err := fmt.Fscanln(os.Stdin, &line); err != nil {
		fmt.Fprintln(stdout, theme.Dim.Render("cancelado"))
		os.Exit(2)
	}
	if !confirmUSBPhrase(line) {
		fmt.Fprintln(stdout, theme.Dim.Render("cancelado"))
		os.Exit(2)
	}
	return runEmbeddedScript(scriptName)
}

// runUSBTUI launches the Bubble Tea program for gate 1. On a clean
// confirmation it exits the alt-screen and shells out to the embedded
// script so the script can drive its own gate 2 prompts directly
// against the real terminal.
func runUSBTUI(report detect.Report, scriptName string) error {
	verbosity := tui.Standard
	if p, err := profile.Load(); err == nil {
		verbosity = tui.VerbosityForLevel(string(p.Level))
	}

	m := NewUSBModel(report, scriptName, verbosity)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}
	fm, ok := final.(USBModel)
	if !ok {
		return fmt.Errorf("usb tui returned unexpected model type %T", final)
	}
	switch fm.outcome {
	case usbOutcomeCancel:
		// Exit 2 = user cancel. The bash script never ran, so there
		// is nothing to roll back.
		os.Exit(2)
	case usbOutcomeConfirm:
		// Gate 1 passed. Hand off to gate 2 in the bash script with
		// raw stdin/stdout/stderr so its `read` prompts work.
		return runEmbeddedScript(fm.scriptName)
	}
	return nil
}

// usbOutcome captures the final state of the TUI so the host can
// translate it into the right exit code and follow-up action.
type usbOutcome int

const (
	usbOutcomePending usbOutcome = iota
	usbOutcomeConfirm
	usbOutcomeCancel
)

// USBModel is the Bubble Tea model behind `multiversa usb`. It is a
// single-screen program: header + info pane + typed-phrase input.
// Anything other than the exact phrase (case-insensitive, trimmed)
// cancels the flow.
type USBModel struct {
	report     detect.Report
	scriptName string
	verbosity  tui.Verbosity

	input   string
	outcome usbOutcome
	err     string

	width  int
	height int
}

// NewUSBModel builds the model used by the TUI and by tests. The
// constructor is exported so usb_test.go can drive the model without
// spinning up a real tea.Program.
func NewUSBModel(report detect.Report, scriptName string, v tui.Verbosity) USBModel {
	return USBModel{
		report:     report,
		scriptName: scriptName,
		verbosity:  v,
	}
}

// Init satisfies tea.Model.
func (m USBModel) Init() tea.Cmd { return nil }

// Update satisfies tea.Model. Only one screen, so the routing is
// simple: characters append to the buffer, enter confirms, esc/ctrl+c
// cancel.
func (m USBModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.outcome = usbOutcomeCancel
			return m, tea.Batch(tea.ExitAltScreen, tea.Quit)
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
			m.err = ""
		case tea.KeyEnter:
			if confirmUSBPhrase(m.input) {
				m.outcome = usbOutcomeConfirm
				return m, tea.Batch(tea.ExitAltScreen, tea.Quit)
			}
			m.outcome = usbOutcomeCancel
			return m, tea.Batch(tea.ExitAltScreen, tea.Quit)
		case tea.KeyRunes, tea.KeySpace:
			m.input += string(msg.Runes)
			if msg.Type == tea.KeySpace && len(msg.Runes) == 0 {
				m.input += " "
			}
			m.err = ""
		}
	}
	return m, nil
}

// View satisfies tea.Model.
func (m USBModel) View() string {
	if m.outcome != usbOutcomePending {
		// Quit screens stay empty so the parent terminal regains the
		// scrollback cleanly before the bash script (gate 2) prints.
		return ""
	}

	var b strings.Builder
	b.WriteString(tui.Header("multiversa usb",
		theme.Warn.Render("destructivo — borra el dispositivo destino"),
		0, 0))
	b.WriteByte('\n')

	// Info pane: host, script, prereq summary, device-path hint.
	b.WriteString(theme.Label.Render("host")+"     "+
		theme.Body.Render(fmt.Sprintf("%s/%s · %s",
			m.report.OS.Kind, m.report.OS.Arch, m.report.OS.Distro)), )
	b.WriteByte('\n')
	b.WriteString(theme.Label.Render("script")+"   "+m.scriptName+theme.Dim.Render(" (embebido)"))
	b.WriteByte('\n')

	prereq := tui.Choose(m.verbosity,
		"ninguno — el script revisa VeraCrypt/balenaEtcher",
		"sin prereqs",
		"")
	if m.report.OS.Kind == "linux" {
		prereq = tui.Choose(m.verbosity,
			"cryptsetup (presente)",
			"cryptsetup ok",
			"cryptsetup")
	}
	if prereq != "" {
		b.WriteString(theme.Label.Render("prereq") + "   " + theme.Body.Render(prereq))
		b.WriteByte('\n')
	}
	b.WriteString(theme.Label.Render("device") + "   " +
		theme.Dim.Render("ej. /dev/sdb en Linux · disk4 en macOS"))
	b.WriteString("\n\n")

	// Warn banner — repeats the destructive line in big letters.
	b.WriteString(theme.Warn.Render(
		"⚠ ESTA OPERACIÓN BORRA EL DISPOSITIVO DESTINO. NO HAY DESHACER."))
	b.WriteString("\n\n")

	hint := tui.Choose(m.verbosity,
		"Escribe exactamente  \"i understand\"  y pulsa enter para continuar.\n"+
			"Cualquier otra cosa cancela. El script te pedirá la ruta del dispositivo dos veces.",
		"Escribe \"i understand\" y enter. Cualquier otra cosa cancela.",
		"escribe \"i understand\" + enter",
	)
	b.WriteString(theme.Body.Render(hint))
	b.WriteString("\n\n")

	prompt := theme.Accent.Render("> ") + theme.Body.Render(m.input) + theme.Dim.Render("_")
	b.WriteString(prompt)
	b.WriteString("\n\n")

	keyhint := tui.Choose(m.verbosity,
		"[enter] confirmar  ·  [esc] cancelar  ·  [ctrl+c] cancelar",
		"[enter] · [esc] cancelar",
		"enter/esc",
	)
	b.WriteString(theme.Dim.Render(keyhint))

	body := b.String()
	if m.width > 0 {
		return theme.Frame(m.width, body)
	}
	return lipgloss.NewStyle().Render(body)
}

// confirmUSBPhrase implements the strict gate-1 contract: only the
// exact phrase (case-insensitive, trimmed) passes. Everything else —
// including blanks, "yes", "y", or near-misses — fails. The strictness
// is the entire point: if the user is not present and engaged enough
// to type the phrase, they should not be wiping a disk.
func confirmUSBPhrase(input string) bool {
	return strings.ToLower(strings.TrimSpace(input)) == usbConfirmPhrase
}
