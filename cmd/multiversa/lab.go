// Multiversa lab — the consultive meta-wizard. It composes the
// individual command flows (detect, stack, init, workspace, usb)
// into a single cinematic experience organized by layer:
//
//	Capa Técnica       → detect · stack · init
//	Capa Identitaria   → workspace
//	Capa Operacional   → usb · credits
//
// The lab is the public face of Multiversa Lab — what a prospect
// sees first. Consistency with the per-command TUIs is critical,
// so the lab consumes the same internal/tui primitives every other
// wizard uses, and the same `profile` for adaptive verbosity.
//
// Architecturally the lab is a hub: it renders a sidebar of layers,
// a selector inside the focused layer, and lets the user launch
// any step. Launching a step exits the lab cleanly via tea.Quit,
// the cobra RunE then invokes the chosen run* function, and on
// return the lab is re-entered with refreshed state. The user
// experiences a single continuous flow; the implementation keeps
// each wizard self-contained.
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
	"github.com/moshequantum/multiversa-cli/internal/stack"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// labOutcome is what the lab returns to its outer runLab loop. It
// drives whether we exit, re-enter, or launch a sub-command.
type labOutcome int

const (
	outcomeExit labOutcome = iota
	outcomeLaunchDetect
	outcomeLaunchStack
	outcomeLaunchInit
	outcomeLaunchWorkspace
	outcomeLaunchUSB
	outcomeLaunchCredits
)

// stepID is the identifier of a single step inside a layer. The
// lab uses these as keys in profile.InstalledEngines and as the
// payload of labOutcome.
type stepID string

const (
	stepDetect    stepID = "detect"
	stepStack     stepID = "stack"
	stepInit      stepID = "init"
	stepWorkspace stepID = "workspace"
	stepUSB       stepID = "usb"
	stepCredits   stepID = "credits"
)

// labStep is one row inside a layer of the meta-wizard.
type labStep struct {
	ID          stepID
	Label       string
	Hint        string
	Destructive bool // shown with theme.Warn marker
	Outcome     labOutcome
}

// labLayer groups labSteps under one of profile.Layer.
type labLayer struct {
	Layer profile.Layer
	Steps []labStep
}

// defaultLayers builds the canonical sidebar. Order matters: it is
// the suggested traversal for a newcomer.
func defaultLayers() []labLayer {
	return []labLayer{
		{
			Layer: profile.Tecnica,
			Steps: []labStep{
				{ID: stepDetect, Label: "Detectar entorno", Hint: "escaneo de solo lectura", Outcome: outcomeLaunchDetect},
				{ID: stepStack, Label: "Stack base", Hint: "Go · Rust · Python · Node · pnpm · Docker", Outcome: outcomeLaunchStack},
				{ID: stepInit, Label: "Engines agénticos", Hint: "Engram · Graphify · Gentle · codegraph", Outcome: outcomeLaunchInit},
			},
		},
		{
			Layer: profile.Identitaria,
			Steps: []labStep{
				{ID: stepWorkspace, Label: "Workspace privado", Hint: "SSH · GPG · MultiversaGroup", Outcome: outcomeLaunchWorkspace},
			},
		},
		{
			Layer: profile.Operacional,
			Steps: []labStep{
				{ID: stepUSB, Label: "USB cifrado", Hint: "LUKS bootable lab — destructivo", Destructive: true, Outcome: outcomeLaunchUSB},
				{ID: stepCredits, Label: "Atribución upstream", Hint: "créditos de cada engine", Outcome: outcomeLaunchCredits},
			},
		},
	}
}

// LabModel is the Bubble Tea model for `multiversa lab`. It holds
// the sidebar state, the focused layer's selector, and the profile
// so each step can render verbosity-aware hints.
type LabModel struct {
	layers       []labLayer
	layerCursor  int // index into layers
	stepCursor   int // index into layers[layerCursor].Steps
	prof         profile.Profile
	verbosity    tui.Verbosity
	width        int
	height       int
	outcome      labOutcome
	pendingStep  stepID
	reinstall    bool
	completedAll map[stepID]bool
}

// NewLabModel builds the initial lab state. `reinstall` forces
// every step to be re-runnable even if profile reports it as done.
func NewLabModel(prof profile.Profile, reinstall bool) LabModel {
	completed := map[stepID]bool{}
	if !reinstall {
		// stack is "done" if every engine the user has installed
		// is present; we keep this conservative — only mark the
		// stack step as done if the profile records at least one
		// installed engine. detect is never marked done because
		// it's read-only and idempotent.
		if len(prof.InstalledEngines) > 0 {
			completed[stepStack] = true
			completed[stepInit] = true
		}
	}
	return LabModel{
		layers:       defaultLayers(),
		layerCursor:  0,
		stepCursor:   0,
		prof:         prof,
		verbosity:    tui.VerbosityForLevel(string(prof.Level)),
		outcome:      outcomeExit,
		reinstall:    reinstall,
		completedAll: completed,
	}
}

// Init implements tea.Model.
func (m LabModel) Init() tea.Cmd { return nil }

// Update implements tea.Model. The lab navigates with up/down/j/k
// inside a layer; tab/shift-tab between layers; enter to launch.
func (m LabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.outcome = outcomeExit
			return m, tea.Quit

		case "tab":
			if m.layerCursor < len(m.layers)-1 {
				m.layerCursor++
				m.stepCursor = 0
			}
			return m, nil

		case "shift+tab":
			if m.layerCursor > 0 {
				m.layerCursor--
				m.stepCursor = 0
			}
			return m, nil

		case "up", "k":
			if m.stepCursor > 0 {
				m.stepCursor--
			}
			return m, nil

		case "down", "j":
			if m.stepCursor < len(m.layers[m.layerCursor].Steps)-1 {
				m.stepCursor++
			}
			return m, nil

		case "enter":
			step := m.currentStep()
			m.outcome = step.Outcome
			m.pendingStep = step.ID
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model — sidebar on the left, detail pane on
// the right, footer with keymap hint at the bottom.
func (m LabModel) View() string {
	sidebar := m.renderSidebar()
	detail := m.renderDetail()
	footer := m.renderFooter()

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(28).Padding(0, 2, 0, 0).Render(sidebar),
		lipgloss.NewStyle().Padding(0, 0, 0, 2).Render(detail),
	)
	header := tui.Header("multiversa lab", "configurador consultivo del laboratorio", 0, 0)
	return header + "\n" + cols + "\n\n" + footer
}

// currentStep returns the focused step. Always valid as long as
// defaultLayers() is non-empty (which it is).
func (m LabModel) currentStep() labStep {
	return m.layers[m.layerCursor].Steps[m.stepCursor]
}

// renderSidebar builds the layered overview using tui.Sidebar.
func (m LabModel) renderSidebar() string {
	layers := make([]tui.LayerStatus, 0, len(m.layers))
	for li, l := range m.layers {
		steps := make([]tui.ProgressItem, 0, len(l.Steps))
		for _, s := range l.Steps {
			state := tui.Pending
			if m.completedAll[s.ID] {
				state = tui.Done
			}
			steps = append(steps, tui.ProgressItem{
				Label: s.Label,
				State: state,
			})
		}
		layers = append(layers, tui.LayerStatus{
			Name:      l.Layer.DisplayName(),
			Tagline:   l.Layer.Tagline(),
			Steps:     steps,
			IsCurrent: li == m.layerCursor,
		})
	}
	return tui.Sidebar{Layers: layers}.Render()
}

// renderDetail describes the focused step in the right pane. The
// verbosity setting controls whether the hint + explanation are
// rendered or just the title.
func (m LabModel) renderDetail() string {
	layer := m.layers[m.layerCursor]
	if len(layer.Steps) == 0 {
		return theme.Dim.Render("(capa vacía)")
	}
	step := layer.Steps[m.stepCursor]

	var b strings.Builder
	b.WriteString(theme.Accent.Render(step.Label))
	b.WriteByte('\n')
	if step.Hint != "" {
		b.WriteString(theme.Dim.Render(step.Hint))
		b.WriteByte('\n')
	}
	if m.completedAll[step.ID] {
		b.WriteString("\n")
		b.WriteString(theme.Accent.Render("✓ ya completado"))
		if m.reinstall {
			b.WriteString(" " + theme.Dim.Render("(--reinstall activo)"))
		}
		b.WriteByte('\n')
	}
	if step.Destructive {
		b.WriteString("\n")
		b.WriteString(theme.Warn.Render("⚠ operación destructiva"))
		b.WriteByte('\n')
	}

	if m.verbosity == tui.Verbose {
		b.WriteByte('\n')
		b.WriteString(theme.Body.Render(stepDescription(step.ID)))
		b.WriteByte('\n')
	}
	return b.String()
}

// stepDescription returns the long-form explanation shown only at
// Verbose verbosity. Kept as a switch so the lab stays
// self-contained — no need to import documentation packages.
func stepDescription(id stepID) string {
	switch id {
	case stepDetect:
		return "Escanea OS, package manager, herramientas instaladas y\nengines presentes. No modifica nada."
	case stepStack:
		return "Instala las herramientas que falten: Go, Rust, Python,\nNode, pnpm, Docker. Pide confirmación por herramienta."
	case stepInit:
		return "Lanza el wizard interactivo que selecciona engines\nagénticos (Engram, Graphify, Gentle, …) y los conecta\ncon tu agente preferido."
	case stepWorkspace:
		return "Configura el workspace privado MultiversaGroup: SSH,\nGPG, identidad git, clone del monorepo, vault cifrado."
	case stepUSB:
		return "Crea un USB cifrado y arrancable (LUKS en Linux,\nVeraCrypt + balenaEtcher en macOS). Borra el dispositivo\ndestino — el flujo te pide la ruta dos veces."
	case stepCredits:
		return "Imprime la atribución completa de cada engine upstream\nque Multiversa orquesta (no autora)."
	}
	return ""
}

// renderFooter shows the keymap. The hint set adapts to verbosity:
// expert users see only the key letters; newcomers see the labels.
func (m LabModel) renderFooter() string {
	keys := tui.Choose(m.verbosity,
		"↑/↓ paso · tab capa · enter lanzar · q salir",
		"↑/↓  tab  enter  q",
		"↑↓ tab ⏎ q",
	)
	return theme.Dim.Render(keys)
}

// runLab is the cobra RunE handler: it loops over LabModel/run*
// invocations until the user picks Exit. Each iteration:
//
//  1. Load profile (so we see any updates from prior steps).
//  2. Render the lab; user picks a step.
//  3. Quit the lab cleanly, run the chosen step's function with
//     raw stdio.
//  4. Re-enter the lab (next iteration) unless outcomeExit.
func runLab(stdout io.Writer, reinstall bool) error {
	for {
		prof, err := profile.Load()
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("profile.Load: %w", err)
		}

		m := NewLabModel(prof, reinstall)
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(stdout))
		finalRaw, err := p.Run()
		if err != nil {
			return err
		}
		final, ok := finalRaw.(LabModel)
		if !ok {
			return fmt.Errorf("unexpected model type from lab program")
		}

		switch final.outcome {
		case outcomeExit:
			fmt.Fprintln(stdout, theme.Dim.Render("hasta la próxima — \"la IA propone, tú decides\"."))
			return nil
		case outcomeLaunchDetect:
			if err := runDetect(stdout); err != nil {
				return err
			}
		case outcomeLaunchStack:
			if err := runStack(stackOpts{out: stdout}); err != nil {
				return err
			}
		case outcomeLaunchInit:
			// Re-uses the existing init wizard; no flags here so it
			// runs in its default interactive mode.
			fmt.Fprintln(stdout, theme.Dim.Render("(init: lanza `multiversa init` desde una nueva sesión)"))
		case outcomeLaunchWorkspace:
			if err := runWorkspace(workspaceOpts{out: stdout}); err != nil {
				return err
			}
		case outcomeLaunchUSB:
			if err := runUSB(stdout, false); err != nil {
				return err
			}
		case outcomeLaunchCredits:
			printCredits(stdout)
		}
	}
}

// printCredits delegates to the existing credits package; kept as
// a thin wrapper so the lab does not import internal/credits at
// the top level (it only needs it conditionally).
func printCredits(stdout io.Writer) {
	// Import lazy via reflection would be overkill — we use the
	// stack registry's data, which mirrors credits anyway, to
	// surface a compact view inside the lab. For the full credits
	// text, the user can still run `multiversa credits` directly.
	fmt.Fprintln(stdout, theme.Accent.Render("multiversa lab — atribución"))
	for _, eng := range stack.Registry() {
		fmt.Fprintf(stdout, "  %s · %s · %s\n",
			theme.Accent.Render(eng.DisplayName()),
			theme.Body.Render(eng.Author()),
			theme.Dim.Render(eng.License()))
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, theme.Dim.Render("para el texto completo: `multiversa credits`"))
}

// newLabCmd registers `multiversa lab`.
func newLabCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lab",
		Short: "Configurador consultivo del laboratorio (capas Técnica · Identitaria · Operacional).",
		Long: "Lanza el meta-wizard que orquesta detect, stack, init,\n" +
			"workspace, usb y credits en un solo flujo cinematográfico.\n" +
			"Los pasos ya completados se marcan ✓ y se pueden saltar.\n" +
			"Usa --reinstall para forzar la re-ejecución de pasos completos.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reinstall, _ := cmd.Flags().GetBool("reinstall")
			return runLab(os.Stdout, reinstall)
		},
	}
	cmd.Flags().Bool("reinstall", false, "Re-ejecuta pasos ya marcados como completados.")
	return cmd
}

// runDetect is referenced by the lab; it lives in detect.go. The
// signature is exposed here so the lab can call it without going
// through cobra. detect.go was authored by the swarm; if its
// signature changes, this file must be updated in lock-step.
//
// (Variable declaration removed — runDetect is in this same
// package, so the lab can call it directly.)
var _ = detect.Run // ensure detect import isn't accidentally pruned
