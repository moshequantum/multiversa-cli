package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// WorkspaceModel is the Bubble Tea model behind `multiversa workspace`.
// Single-screen: info pane + prereq status + confirm prompt.
type WorkspaceModel struct {
	report    detect.Report
	missing   []string
	width     int
	height    int
	confirmed bool
	cancelled bool
	input     string
}

// NewWorkspaceModel builds the model used by the TUI and tests.
func NewWorkspaceModel(r detect.Report, missing []string) WorkspaceModel {
	return WorkspaceModel{report: r, missing: missing}
}

// MissingTools returns the prerequisite tools that are absent.
func (m WorkspaceModel) MissingTools() []string { return m.missing }

// Confirmed reports whether the user confirmed the workspace setup.
func (m WorkspaceModel) Confirmed() bool { return m.confirmed }

// Cancelled reports whether the user cancelled.
func (m WorkspaceModel) Cancelled() bool { return m.cancelled }

// Init satisfies tea.Model.
func (m WorkspaceModel) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m WorkspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case CancelMsg:
		m.cancelled = true
		return m, tea.Quit
	case tea.KeyMsg:
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
			return m, Cancel
		case "enter":
			if ConfirmDecision(m.input) {
				m.confirmed = true
				return m, tea.Quit
			}
			m.cancelled = true
			return m, Cancel
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
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
	b.WriteString(Header("multiversa workspace", "MultiversaGroup — configuración del workspace privado", 0, 0))
	b.WriteByte('\n')

	b.WriteString(theme.Label.Render("script"))
	b.WriteString("  ")
	b.WriteString(theme.Body.Render("setup_multiversa.sh"))
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
