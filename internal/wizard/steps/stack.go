package steps

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moshequantum/multiversa-cli/internal/stack"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type Stack struct {
	cursor   int
	selected map[string]bool
	engines  []stack.Engine
}

func NewStack() Step {
	engines := stack.List()
	selected := make(map[string]bool, len(engines))
	for _, e := range engines {
		if !e.OptIn() {
			selected[e.ID()] = true
		}
	}
	return &Stack{engines: engines, selected: selected}
}

func (*Stack) Title() string { return "Stack" }
func (*Stack) Init() tea.Cmd { return nil }

func (s *Stack) Update(msg tea.Msg) (Step, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch k.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(s.engines)-1 {
			s.cursor++
		}
	case " ":
		id := s.engines[s.cursor].ID()
		s.selected[id] = !s.selected[id]
	case "enter":
		return s, Next
	case "b":
		return s, Back
	}
	return s, nil
}

func (s *Stack) Selected() []string {
	out := []string{}
	for _, e := range s.engines {
		if s.selected[e.ID()] {
			out = append(out, e.ID())
		}
	}
	return out
}

func (s *Stack) View() string {
	title := theme.Display.Render("Selecciona los motores")
	subtitle := theme.Dim.Render("Multiversa orquesta — los autores son ajenos. Crédito visible.")

	var rows []string
	for i, e := range s.engines {
		mark := "  "
		if s.selected[e.ID()] {
			mark = theme.Accent.Render("● ")
		} else {
			mark = theme.Dim.Render("○ ")
		}
		cursor := "  "
		if i == s.cursor {
			cursor = theme.Accent.Render("▸ ")
		}
		tag := ""
		switch {
		case e.License() == "AGPL-3.0":
			tag = " " + theme.Warn.Render("[AGPL · external-only]")
		case e.OptIn():
			tag = " " + theme.Dim.Render("[opt-in]")
		}
		row := cursor + mark + theme.Body.Render(pad(e.DisplayName(), 14)) +
			theme.Dim.Render(pad(e.Author(), 26)) + tag
		rows = append(rows, row)
	}

	hint := theme.Dim.Render("[↑↓] mover  ·  [space] alternar  ·  [enter] continuar  ·  [b] atrás")

	return theme.Box.Render(lipgloss.JoinVertical(lipgloss.Left,
		title, subtitle, "",
		strings.Join(rows, "\n"),
		"", hint,
	))
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
