package tui

import (
	"strings"
	"testing"
)

func TestProgressList_RendersAllItems(t *testing.T) {
	p := ProgressList{Items: []ProgressItem{
		{Label: "go", State: Done},
		{Label: "rust", State: Running},
		{Label: "pnpm", State: Pending},
	}}
	out := p.Render()
	for _, want := range []string{"go", "rust", "pnpm"} {
		if !strings.Contains(out, want) {
			t.Errorf("Render() missing label %q:\n%s", want, out)
		}
	}
}

func TestProgressList_Counts(t *testing.T) {
	p := ProgressList{Items: []ProgressItem{
		{State: Done},
		{State: Done},
		{State: Skipped},
		{State: Failed},
		{State: Pending},
		{State: Running},
	}}
	done, skipped, failed, pending := p.Counts()
	if done != 2 || skipped != 1 || failed != 1 || pending != 2 {
		t.Errorf("Counts() = (%d,%d,%d,%d), want (2,1,1,2)", done, skipped, failed, pending)
	}
}

func TestStateGlyph_ChecksAllStates(t *testing.T) {
	// Smoke test — every state must yield non-empty glyph + style
	// without panicking.
	for _, s := range []ProgressState{Pending, Running, Done, Skipped, Failed} {
		glyph, _ := stateGlyph(s)
		if glyph == "" {
			t.Errorf("stateGlyph(%v) returned empty glyph", s)
		}
	}
}
