// Package wizard hosts the Bubble Tea program that drives the interactive
// installer. Each screen is a Step (see internal/wizard/steps).
package wizard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/moshequantum/multiversa-cli/internal/wizard/steps"
)

type Model struct {
	current int
	steps   []steps.Step
	width   int
	height  int
}

func New() Model {
	return Model{
		steps: []steps.Step{
			steps.NewWelcome(),
			steps.NewConsent(),
			steps.NewStack(),
			steps.NewBackend(),
			steps.NewReview(),
			steps.NewInstall(),
		},
	}
}

// Options configures a wizard run. Currently controls dry-run mode.
type Options struct {
	DryRun bool
}

func Run() error {
	return RunWith(Options{})
}

func RunWith(opts Options) error {
	m := New()
	for _, s := range m.steps {
		if inst, ok := s.(*steps.Install); ok {
			inst.SetDryRun(opts.DryRun)
			// The wizard always shows the consent screen before stack selection.
			// If the user declines, the program exits before Install can run.
			inst.SetAgplAcknowledged(true)
		}
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return m.steps[m.current].Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate size to every step so any of them can re-render correctly
		// when activated later.
		var cmds []tea.Cmd
		for i := range m.steps {
			var cmd tea.Cmd
			m.steps[i], cmd = m.steps[i].Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case steps.NextMsg:
		m.propagate()
		if m.current+1 < len(m.steps) {
			m.current++
			return m, m.steps[m.current].Init()
		}
		return m, tea.Quit
	case steps.BackMsg:
		if m.current > 0 {
			m.current--
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.steps[m.current], cmd = m.steps[m.current].Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.steps[m.current].View()
}

// propagate hands user choices from earlier steps into later ones (Review,
// Install) when the wizard advances.
func (m *Model) propagate() {
	var engines []string
	var backend string
	for _, s := range m.steps {
		switch v := s.(type) {
		case *steps.Stack:
			engines = v.Selected()
		case *steps.Backend:
			backend = v.Choice()
		}
	}
	for _, s := range m.steps {
		switch v := s.(type) {
		case *steps.Review:
			v.Set(engines, backend)
		case *steps.Install:
			v.Set(engines, backend)
		}
	}
}
