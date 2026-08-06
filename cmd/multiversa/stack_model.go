package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/profile"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// stackPhase describes which screen the stackModel is showing.
type stackPhase int

const (
	phaseSelect stackPhase = iota
	phaseInstall
	phaseDone
)

// stackInstallStepMsg signals that the install queue should advance one
// step. The model uses index-driven progression: each Done step emits a
// fresh stackInstallStepMsg via a tea.Cmd, so the View() can repaint
// between installs.
type stackInstallStepMsg struct{ index int }

// stackInstallResultMsg carries the outcome of one install operation.
type stackInstallResultMsg struct {
	index int
	err   error
}

// stackModel is the Bubble Tea model that drives the `stack` TUI.
// Phase 1 (select): renders a Selector with one row per planned tool.
//
//	Installed → "✓" marker + disabled; missing-with-plan → "·";
//	unsupported-OS → "⚠" + disabled.
//
// Phase 2 (install): renders a ProgressList; each tool transitions
//
//	Pending → Running → Done/Failed/Skipped.
//
// Phase 3 (done): renders the final summary.
//
// Cancellation: Esc/q/ctrl+c. Before installs start → cancelled flag
// set, no profile mutation. Mid-install → stop the queue, save what
// actually completed.
type stackModel struct {
	report   detect.Report
	planned  []toolPlan
	selector tui.Selector
	progress tui.ProgressList
	// selected[i] mirrors a per-row toggle. Defaults to true for every
	// tool that has a real Plan (not installed, not unsupported).
	selected []bool
	phase    stackPhase
	width    int
	height   int

	// install-time state
	queue   []int // indices into planned of items to install, in order
	cursor  int   // position in queue
	prof    profile.Profile
	profErr error
	// profSavable is false when profile.Load() failed for a reason
	// other than "file does not exist" — i.e. the file is present but
	// corrupt/unparseable. In that case m.prof is a zero-value Profile
	// and must never be written to disk, or level/locale/
	// installed_engines on the real file would be silently wiped.
	profSavable bool

	cancelled    bool
	done         int
	failed       int
	skipped      int
	installCount int
}

// newStackModel builds the initial selector view from a plan.
func newStackModel(report detect.Report, planned []toolPlan) *stackModel {
	prof, profErr := profile.Load()
	profSavable := profErr == nil || errors.Is(profErr, os.ErrNotExist)
	items := make([]tui.SelectorItem, 0, len(planned))
	selected := make([]bool, len(planned))
	cursor := -1
	for i, tp := range planned {
		var (
			marker   string
			hint     string
			disabled bool
		)
		switch {
		case tp.installed:
			marker = theme.Accent.Render("✓")
			hint = "ya instalado"
			disabled = true
		case tp.err != nil:
			marker = theme.Warn.Render("⚠")
			hint = tp.err.Error()
			disabled = true
		default:
			marker = theme.Dim.Render("·")
			hint = planSummary(tp.plan)
			selected[i] = true
			if cursor < 0 {
				cursor = i
			}
		}
		items = append(items, tui.SelectorItem{
			Label:    tp.tool.DisplayName(),
			Hint:     hint,
			Marker:   marker,
			Disabled: disabled,
		})
	}
	if cursor < 0 {
		cursor = 0
	}
	return &stackModel{
		report:      report,
		planned:     planned,
		selector:    tui.Selector{Items: items, Cursor: cursor},
		selected:    selected,
		phase:       phaseSelect,
		prof:        prof,
		profErr:     profErr,
		profSavable: profSavable,
	}
}

func (m *stackModel) Init() tea.Cmd { return nil }

func (m *stackModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tui.Cancel
		}
		if m.phase == phaseSelect {
			return m.updateSelect(msg)
		}
		return m, nil

	case tui.CancelMsg:
		m.cancelled = true
		// Mid-install cancellation persists the partial result — but only
		// when the on-disk profile was actually readable (or absent). If
		// it was corrupt, m.prof is a zero-value stand-in and writing it
		// would destroy the real level/locale/installed_engines.
		if m.phase == phaseInstall && m.profSavable {
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

// updateSelect handles keys while the selector is visible.
func (m *stackModel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.selector.MoveUp()
	case "down", "j":
		m.selector.MoveDown()
	case " ":
		i := m.selector.Cursor
		if i >= 0 && i < len(m.planned) && !m.selector.Items[i].Disabled {
			m.selected[i] = !m.selected[i]
			// Re-render the row marker to reflect the new selection.
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

// startInstall switches to phaseInstall and kicks off the queue.
func (m *stackModel) startInstall() (tea.Model, tea.Cmd) {
	m.queue = nil
	progItems := make([]tui.ProgressItem, 0, len(m.planned))
	for i, tp := range m.planned {
		state := tui.Skipped
		note := ""
		switch {
		case tp.installed:
			state = tui.Skipped
			note = "ya instalado"
		case tp.err != nil:
			state = tui.Failed
			note = tp.err.Error()
		case !m.selected[i]:
			state = tui.Skipped
			note = "omitido"
		default:
			state = tui.Pending
			note = planSummary(tp.plan)
			m.queue = append(m.queue, i)
		}
		progItems = append(progItems, tui.ProgressItem{
			Label: tp.tool.DisplayName(),
			Note:  note,
			State: state,
		})
	}
	m.progress = tui.ProgressList{Items: progItems}
	m.phase = phaseInstall
	if len(m.queue) == 0 {
		m.phase = phaseDone
		return m, tea.Quit
	}
	m.cursor = 0
	return m, m.runNext()
}

// runNext returns a tea.Cmd that executes the install at queue[cursor]
// and reports the result back as a stackInstallResultMsg.
func (m *stackModel) runNext() tea.Cmd {
	if m.cursor >= len(m.queue) {
		return func() tea.Msg { return stackInstallStepMsg{index: -1} }
	}
	idx := m.queue[m.cursor]
	// Flip to Running before the command starts so the user sees motion.
	m.progress.Items[idx].State = tui.Running
	plan := m.planned[idx].plan
	return func() tea.Msg {
		err := executePlan(plan)
		return stackInstallResultMsg{index: idx, err: err}
	}
}

// recordResult ingests the outcome of one install, updates the
// ProgressList row, and either schedules the next install or finishes.
func (m *stackModel) recordResult(msg stackInstallResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.progress.Items[msg.index].State = tui.Failed
		m.progress.Items[msg.index].Note = msg.err.Error()
		m.failed++
	} else {
		m.progress.Items[msg.index].State = tui.Done
		m.progress.Items[msg.index].Note = "instalado"
		m.prof.MarkInstalled(m.planned[msg.index].tool.ID())
		m.installCount++
		m.done++
	}
	m.cursor++
	return m, func() tea.Msg { return stackInstallStepMsg{index: m.cursor} }
}

// advanceInstall reacts to a stackInstallStepMsg by either starting the
// next install or transitioning to phaseDone.
func (m *stackModel) advanceInstall(_ int) (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.queue) {
		// Done — persist installed marks, unless the profile on disk was
		// corrupt (see profSavable's doc comment).
		if m.profSavable {
			_ = m.prof.Save()
		}
		m.phase = phaseDone
		return m, tea.Quit
	}
	return m, m.runNext()
}

func (m *stackModel) View() string {
	header := tui.Header(
		"multiversa stack",
		fmt.Sprintf("host: %s/%s · pkg-mgr: %s",
			m.report.OS.Kind, m.report.OS.Arch, displayPkgMgr(m.report.OS.PkgMgr)),
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
		out := header + "\n" + m.progress.Render() + "\n" + summary + "\n"
		if !m.profSavable && m.installCount > 0 {
			out += theme.Warn.Render(fmt.Sprintf(
				"⚠ perfil existente ilegible (%v); installed_engines no se actualizó para no perder level/locale", m.profErr)) + "\n"
		}
		return out
	}
}

// runStackTUI drives the Bubble Tea program and translates cancellation
// into exit code 2. Failed installs return a non-nil error so cobra
// prints "error: …" and exits 1.
func runStackTUI(opts stackOpts, report detect.Report, planned []toolPlan) error {
	if len(planned) == 0 {
		fmt.Fprintln(opts.out, theme.Warn.Render("Sin coincidencias para el filtro --only."))
		return nil
	}
	m := newStackModel(report, planned)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	if m.cancelled {
		// Exit code 2 = user cancel. cobra would print "error:" for a
		// returned error, which is the wrong tone here, so we exit
		// directly.
		os.Exit(2)
	}
	if m.failed > 0 {
		return fmt.Errorf("%d herramienta(s) fallaron", m.failed)
	}
	return nil
}
