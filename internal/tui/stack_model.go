package tui

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
	"github.com/moshequantum/multiversa-cli/internal/profile"
	"github.com/moshequantum/multiversa-cli/internal/stack"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type stackPhase int

const (
	phaseSelect stackPhase = iota
	phaseInstall
	phaseDone
)

type stackInstallStepMsg struct{ index int }

type stackInstallResultMsg struct {
	index int
	err   error
}

// StackModel is the Bubble Tea model that drives the `stack` TUI.
// Phase 1 (select): Selector with one row per planned tool.
// Phase 2 (install): ProgressList; each tool transitions Pending -> Done/Failed.
// Phase 3 (done): final summary.
type StackModel struct {
	report   detect.Report
	planned  []stack.ToolPlan
	selector Selector
	progress ProgressList
	selected []bool
	phase    stackPhase
	width    int
	height   int

	queue        []int
	cursor       int
	prof         profile.Profile
	profErr      error
	cancelled    bool
	done         int
	failed       int
	skipped      int
	installCount int
}

// NewStackModel builds the initial selector view from a plan.
func NewStackModel(report detect.Report, planned []stack.ToolPlan) *StackModel {
	prof, profErr := profile.Load()
	items := make([]SelectorItem, 0, len(planned))
	selected := make([]bool, len(planned))
	cursor := -1
	for i, tp := range planned {
		var (
			marker   string
			hint     string
			disabled bool
		)
		switch {
		case tp.Installed:
			marker = theme.Accent.Render("✓")
			hint = "ya instalado"
			disabled = true
		case tp.Err != nil:
			marker = theme.Warn.Render("⚠")
			hint = tp.Err.Error()
			disabled = true
		default:
			marker = theme.Dim.Render("·")
			hint = stack.PlanSummary(tp.Plan)
			selected[i] = true
			if cursor < 0 {
				cursor = i
			}
		}
		items = append(items, SelectorItem{
			Label:    tp.Tool.DisplayName(),
			Hint:     hint,
			Marker:   marker,
			Disabled: disabled,
		})
	}
	if cursor < 0 {
		cursor = 0
	}
	return &StackModel{
		report:   report,
		planned:  planned,
		selector: Selector{Items: items, Cursor: cursor},
		selected: selected,
		phase:    phaseSelect,
		prof:     prof,
		profErr:  profErr,
	}
}

func (m *StackModel) Init() tea.Cmd { return nil }

func (m *StackModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, Cancel
		}
		if m.phase == phaseSelect {
			return m.updateSelect(msg)
		}
		return m, nil
	case CancelMsg:
		m.cancelled = true
		if m.phase == phaseInstall {
			_ = m.prof.Save()
		}
		return m, tea.Quit
	case stackInstallStepMsg:
		return m.advanceInstall(msg.index)
	case stackInstallResultMsg:
		return m.recordResult(msg)
	}
	return m, nil
}

func (m *StackModel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.selector.MoveUp()
	case "down", "j":
		m.selector.MoveDown()
	case " ":
		i := m.selector.Cursor
		if i >= 0 && i < len(m.planned) && !m.selector.Items[i].Disabled {
			m.selected[i] = !m.selected[i]
			if m.selected[i] {
				m.selector.Items[i].Marker = theme.Accent.Render("•")
			} else {
				m.selector.Items[i].Marker = theme.Dim.Render("·")
			}
		}
	case "enter":
		return m.startInstall()
	}
	return m, nil
}

func (m *StackModel) startInstall() (tea.Model, tea.Cmd) {
	m.queue = nil
	progItems := make([]ProgressItem, 0, len(m.planned))
	for i, tp := range m.planned {
		state := Skipped
		note := ""
		switch {
		case tp.Installed:
			state = Skipped
			note = "ya instalado"
		case tp.Err != nil:
			state = Failed
			note = tp.Err.Error()
		case !m.selected[i]:
			state = Skipped
			note = "omitido"
		default:
			state = Pending
			note = stack.PlanSummary(tp.Plan)
			m.queue = append(m.queue, i)
		}
		progItems = append(progItems, ProgressItem{
			Label: tp.Tool.DisplayName(),
			Note:  note,
			State: state,
		})
	}
	m.progress = ProgressList{Items: progItems}
	m.phase = phaseInstall
	if len(m.queue) == 0 {
		m.phase = phaseDone
		return m, tea.Quit
	}
	m.cursor = 0
	return m, m.runNext()
}

func (m *StackModel) runNext() tea.Cmd {
	if m.cursor >= len(m.queue) {
		return func() tea.Msg { return stackInstallStepMsg{index: -1} }
	}
	idx := m.queue[m.cursor]
	m.progress.Items[idx].State = Running
	plan := m.planned[idx].Plan
	return func() tea.Msg {
		err := xexec.RunPlan(plan)
		return stackInstallResultMsg{index: idx, err: err}
	}
}

func (m *StackModel) recordResult(msg stackInstallResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.progress.Items[msg.index].State = Failed
		m.progress.Items[msg.index].Note = msg.err.Error()
		m.failed++
	} else {
		m.progress.Items[msg.index].State = Done
		m.progress.Items[msg.index].Note = "instalado"
		m.prof.MarkInstalled(m.planned[msg.index].Tool.ID())
		m.installCount++
		m.done++
	}
	m.cursor++
	return m, func() tea.Msg { return stackInstallStepMsg{index: m.cursor} }
}

func (m *StackModel) advanceInstall(_ int) (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.queue) {
		_ = m.prof.Save()
		m.phase = phaseDone
		return m, tea.Quit
	}
	return m, m.runNext()
}

func (m *StackModel) View() string {
	pkgMgr := m.report.OS.PkgMgr
	if pkgMgr == "" {
		pkgMgr = "(ninguno)"
	}
	header := Header(
		"multiversa stack",
		fmt.Sprintf("host: %s/%s · pkg-mgr: %s",
			m.report.OS.Kind, m.report.OS.Arch, pkgMgr),
		0, 0,
	)
	switch m.phase {
	case phaseSelect:
		hint := theme.Dim.Render("[↑↓] mover  ·  [espacio] alternar  ·  [enter] instalar  ·  [esc] cancelar")
		return header + "\n" + m.selector.Render() + "\n" + hint + "\n"
	case phaseInstall:
		return header + "\n" + m.progress.Render() + "\n" +
			theme.Dim.Render("[esc] cancelar (conserva lo ya instalado)") + "\n"
	default:
		done, skipped, failed, _ := m.progress.Counts()
		summary := theme.Dim.Render(fmt.Sprintf(
			"Listo: %d instalados · %d omitidos · %d fallidos", done, skipped, failed))
		return header + "\n" + m.progress.Render() + "\n" + summary + "\n"
	}
}

// RunStackTUI drives the Bubble Tea program and translates cancellation
// into exit code 2. Failed installs return a non-nil error.
func RunStackTUI(out io.Writer, report detect.Report, planned []stack.ToolPlan) error {
	if len(planned) == 0 {
		fmt.Fprintln(out, theme.Warn.Render("Sin coincidencias para el filtro --only."))
		return nil
	}
	m := NewStackModel(report, planned)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	if m.cancelled {
		os.Exit(2)
	}
	if m.failed > 0 {
		return fmt.Errorf("%d herramienta(s) fallaron", m.failed)
	}
	return nil
}
