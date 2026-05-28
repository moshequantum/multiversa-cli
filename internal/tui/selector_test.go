package tui

import (
	"strings"
	"testing"
)

func TestSelector_RenderHasOneRowPerItem(t *testing.T) {
	s := Selector{
		Items: []SelectorItem{
			{Label: "go"},
			{Label: "rust"},
			{Label: "python"},
		},
		Cursor: 0,
	}
	lines := strings.Split(strings.TrimRight(s.Render(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 rows, got %d:\n%s", len(lines), s.Render())
	}
}

func TestSelector_MoveDownSkipsDisabled(t *testing.T) {
	s := Selector{
		Items: []SelectorItem{
			{Label: "go"},
			{Label: "rust", Disabled: true},
			{Label: "python"},
		},
		Cursor: 0,
	}
	s.MoveDown()
	if s.Cursor != 2 {
		t.Errorf("MoveDown should skip disabled row, cursor at %d", s.Cursor)
	}
}

func TestSelector_MoveUpSkipsDisabled(t *testing.T) {
	s := Selector{
		Items: []SelectorItem{
			{Label: "go"},
			{Label: "rust", Disabled: true},
			{Label: "python"},
		},
		Cursor: 2,
	}
	s.MoveUp()
	if s.Cursor != 0 {
		t.Errorf("MoveUp should skip disabled row, cursor at %d", s.Cursor)
	}
}

func TestSelector_MoveDoesNotEscapeBounds(t *testing.T) {
	s := Selector{Items: []SelectorItem{{Label: "only"}}, Cursor: 0}
	s.MoveDown()
	if s.Cursor != 0 {
		t.Errorf("MoveDown past end should stay put, got %d", s.Cursor)
	}
	s.MoveUp()
	if s.Cursor != 0 {
		t.Errorf("MoveUp past start should stay put, got %d", s.Cursor)
	}
}

func TestSelector_SelectedReturnsCurrent(t *testing.T) {
	s := Selector{
		Items:  []SelectorItem{{Label: "a"}, {Label: "b"}},
		Cursor: 1,
	}
	sel := s.Selected()
	if sel == nil || sel.Label != "b" {
		t.Errorf("Selected at cursor 1 = %v, want label 'b'", sel)
	}
}

func TestSelector_SelectedNilOnEmpty(t *testing.T) {
	if (Selector{}).Selected() != nil {
		t.Error("Selected on empty selector should return nil")
	}
}
