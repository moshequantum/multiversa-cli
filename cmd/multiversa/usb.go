// Multiversa usb — encrypted bootable USB lab with dual rendering
// surface and a two-gate confirmation pattern.
//
// Gate 1 (this file, in the Go TUI): user types EXACTLY "i understand".
// Gate 2 (the embedded bash script):  user types the device path TWICE.
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

const usbConfirmPhrase = "i understand"

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

func runUSB(stdout io.Writer, showOnly bool) error {
	report := detect.Run()

	scriptName, err := usbScriptFor(report.OS.Kind)
	if err != nil {
		if report.OS.Kind == "windows" {
			fmt.Fprintln(stdout, theme.Accent.Render("multiversa usb"))
			fmt.Fprintln(stdout, theme.Warn.Render("Crear el USB cifrado desde Windows aún no es soportado."))
			fmt.Fprintln(stdout, theme.Dim.Render("Arranca desde un Linux live ISO y vuelve a correr `multiversa usb`."))
			return nil
		}
		return err
	}

	if showOnly {
		data, err := readEmbeddedScript(scriptName)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(data))
		return nil
	}

	required := requiredForUSB(report.OS.Kind)
	missing := detect.RequiredMissing(report, required)
	if len(missing) > 0 {
		fmt.Fprintln(stdout, theme.Accent.Render("multiversa usb"))
		fmt.Fprintln(stdout, theme.Warn.Render("Faltan prerequisitos: "+strings.Join(missing, ", ")))
		switch report.OS.Kind {
		case "linux":
			fmt.Fprintln(stdout, theme.Dim.Render("Instala con: sudo "+report.OS.PkgMgr+" install cryptsetup"))
		}
		return fmt.Errorf("prerequisites missing")
	}

	if isTTY(stdout) {
		return runUSBTUI(report, scriptName)
	}
	return runUSBPlain(stdout, report, scriptName)
}

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

func requiredForUSB(osKind string) []string {
	switch osKind {
	case "linux":
		return []string{"cryptsetup"}
	case "darwin":
		return nil
	}
	return nil
}

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
		os.Exit(2)
	case usbOutcomeConfirm:
		return runEmbeddedScript(fm.scriptName)
	}
	return nil
}

type usbOutcome int

const (
	usbOutcomePending usbOutcome = iota
	usbOutcomeConfirm
	usbOutcomeCancel
)

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

func NewUSBModel(report detect.Report, scriptName string, v tui.Verbosity) USBModel {
	return USBModel{
		report:     report,
		scriptName: scriptName,
		verbosity:  v,
	}
}

func (m USBModel) Init() tea.Cmd { return nil }

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

func (m USBModel) View() string {
	if m.outcome != usbOutcomePending {
		return ""
	}

	var b strings.Builder
	b.WriteString(tui.Header("multiversa usb",
		theme.Warn.Render("destructivo — borra el dispositivo destino"),
		0, 0))
	b.WriteByte('\n')

	b.WriteString(theme.Label.Render("host") + "     " +
		theme.Body.Render(fmt.Sprintf("%s/%s · %s",
			m.report.OS.Kind, m.report.OS.Arch, m.report.OS.Distro)))
	b.WriteByte('\n')
	b.WriteString(theme.Label.Render("script") + "   " + m.scriptName + theme.Dim.Render(" (embebido)"))
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

func confirmUSBPhrase(input string) bool {
	return strings.ToLower(strings.TrimSpace(input)) == usbConfirmPhrase
}
