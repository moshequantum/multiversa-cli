// Tests for `multiversa lab` — the consultive meta-wizard that composes
// detect · stack · init · workspace · usb · credits into a sidebar-driven
// Bubble Tea flow organized by layer (Técnica / Identitaria / Operacional).
//
// We exercise the LabModel contract — Init/Update/View, navigation, layer
// progression, key handling, and the profile-driven completed state — without
// spinning up a real tea.Program (which would block on a real TTY).
package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/profile"
)

// TestLabModelSatisfiesTeaModel is a compile-time assertion: LabModel
// honours the tea.Model interface. If this stops compiling, the lab
// is decoupled from Bubble Tea and the whole wizard chain breaks.
func TestLabModelSatisfiesTeaModel(t *testing.T) {
	var _ tea.Model = NewLabModel(profile.Profile{}, false)
}

// TestLabInitNilCmd confirms Init() is a no-op — the lab model needs
// no async start-up work; the real terminal size comes from
// WindowSizeMsg.
func TestLabInitNilCmd(t *testing.T) {
	m := NewLabModel(profile.Profile{}, false)
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil Init cmd, got %T", cmd)
	}
}

// TestLabViewRendersHeader confirms that View() contains the canonical
// "multiversa lab" header text. Tests that scrape the TUI output (e.g.
// CI snapshot diffs, the /lab-setup skill) depend on this string being
// present.
func TestLabViewRendersHeader(t *testing.T) {
	m := NewLabModel(profile.Profile{}, false)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := updated.View()
	if !strings.Contains(view, "multiversa lab") {
		t.Fatalf("View missing header, got:\n%s", view)
	}
}

// TestLabQuitKeys verifies that q, esc, and ctrl+c all set
// outcomeExit and return a tea.Quit command. This is the escape hatch
// that every wizard must honour so the user is never trapped.
func TestLabQuitKeys(t *testing.T) {
	for _, key := range []string{"q", "esc", "ctrl+c"} {
		m := NewLabModel(profile.Profile{}, false)
		var msg tea.KeyMsg
		switch key {
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "ctrl+c":
			msg = tea.KeyMsg{Type: tea.KeyCtrlC}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}
		updated, cmd := m.Update(msg)
		final, ok := updated.(LabModel)
		if !ok {
			t.Fatalf("key %q: expected LabModel, got %T", key, updated)
		}
		if final.outcome != outcomeExit {
			t.Fatalf("key %q: expected outcomeExit, got %v", key, final.outcome)
		}
		if cmd == nil {
			t.Fatalf("key %q: expected tea.Quit cmd, got nil", key)
		}
		msg2 := cmd()
		if _, ok := msg2.(tea.QuitMsg); !ok {
			t.Fatalf("key %q: expected tea.QuitMsg, got %T", key, msg2)
		}
	}
}

// TestLabTabAdvancesLayer confirms that pressing tab moves the focus to
// the next layer and resets the step cursor to zero. The order is
// Técnica → Identitaria → Operacional; a tab on the last layer is a
// no-op (not a wrap-around).
func TestLabTabAdvancesLayer(t *testing.T) {
	m := NewLabModel(profile.Profile{}, false)
	if m.layerCursor != 0 {
		t.Fatalf("expected layerCursor=0, got %d", m.layerCursor)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(LabModel)
	if m.layerCursor != 1 {
		t.Fatalf("after tab: expected layerCursor=1, got %d", m.layerCursor)
	}
	if m.stepCursor != 0 {
		t.Fatalf("after tab: expected stepCursor reset to 0, got %d", m.stepCursor)
	}

	// Tab again → layer 2.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(LabModel)
	if m.layerCursor != 2 {
		t.Fatalf("after second tab: expected layerCursor=2, got %d", m.layerCursor)
	}

	// Tab on last layer → no-op (stays at 2).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(LabModel)
	if m.layerCursor != 2 {
		t.Fatalf("tab past last layer: expected layerCursor=2, got %d", m.layerCursor)
	}
}

// TestLabShiftTabGoesBack confirms shift+tab goes to the previous layer
// and that pressing it on layer 0 is a no-op.
func TestLabShiftTabGoesBack(t *testing.T) {
	m := NewLabModel(profile.Profile{}, false)
	// Move to layer 2 first.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(LabModel)
	if m.layerCursor != 2 {
		t.Fatalf("setup: expected layerCursor=2, got %d", m.layerCursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(LabModel)
	if m.layerCursor != 1 {
		t.Fatalf("shift+tab: expected layerCursor=1, got %d", m.layerCursor)
	}

	// Back to 0.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(LabModel)
	if m.layerCursor != 0 {
		t.Fatalf("shift+tab: expected layerCursor=0, got %d", m.layerCursor)
	}

	// shift+tab on layer 0 → no-op.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(LabModel)
	if m.layerCursor != 0 {
		t.Fatalf("shift+tab past first layer: expected layerCursor=0, got %d", m.layerCursor)
	}
}

// TestLabDownMovesStep confirms the down arrow (and j) increments
// stepCursor within the current layer without going out of bounds.
func TestLabDownMovesStep(t *testing.T) {
	m := NewLabModel(profile.Profile{}, false)
	// Layer 0 (Técnica) has 3 steps.
	if len(m.layers[0].Steps) < 2 {
		t.Fatalf("expected at least 2 steps in Técnica layer, got %d", len(m.layers[0].Steps))
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(LabModel)
	if m.stepCursor != 1 {
		t.Fatalf("after j: expected stepCursor=1, got %d", m.stepCursor)
	}

	// "down" key.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(LabModel)
	if m.stepCursor != 2 {
		t.Fatalf("after down: expected stepCursor=2, got %d", m.stepCursor)
	}

	// Going past the last step is a no-op.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(LabModel)
	if m.stepCursor != 2 {
		t.Fatalf("down past last step: expected stepCursor=2, got %d", m.stepCursor)
	}
}

// TestLabUpMovesStep confirms up / k decrements stepCursor and that
// going past step 0 is a no-op.
func TestLabUpMovesStep(t *testing.T) {
	m := NewLabModel(profile.Profile{}, false)
	// Move down to step 2 first.
	for range 2 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(LabModel)
	}
	if m.stepCursor != 2 {
		t.Fatalf("setup: expected stepCursor=2, got %d", m.stepCursor)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = updated.(LabModel)
	if m.stepCursor != 1 {
		t.Fatalf("after k: expected stepCursor=1, got %d", m.stepCursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(LabModel)
	if m.stepCursor != 0 {
		t.Fatalf("after up: expected stepCursor=0, got %d", m.stepCursor)
	}

	// No-op past step 0.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(LabModel)
	if m.stepCursor != 0 {
		t.Fatalf("up past step 0: expected stepCursor=0, got %d", m.stepCursor)
	}
}

// TestLabEnterFirstStepEmitsDetectOutcome confirms that pressing enter
// on the very first step (Detect) sets outcome=outcomeLaunchDetect and
// returns a tea.Quit command. This is the hot path that runLab() uses
// to decide which run* function to call.
func TestLabEnterFirstStepEmitsDetectOutcome(t *testing.T) {
	m := NewLabModel(profile.Profile{}, false)
	// Layer 0, step 0 must be the Detect step.
	if m.layers[0].Steps[0].ID != stepDetect {
		t.Fatalf("expected first step to be stepDetect, got %q", m.layers[0].Steps[0].ID)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final, ok := updated.(LabModel)
	if !ok {
		t.Fatalf("expected LabModel, got %T", updated)
	}
	if final.outcome != outcomeLaunchDetect {
		t.Fatalf("expected outcomeLaunchDetect, got %v", final.outcome)
	}
	if final.pendingStep != stepDetect {
		t.Fatalf("expected pendingStep=stepDetect, got %q", final.pendingStep)
	}
	if cmd == nil {
		t.Fatalf("expected tea.Quit cmd, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

// TestLabEnterUSBStepEmitsUSBOutcome navigates to the USB step and
// confirms enter produces outcomeLaunchUSB. USB is the destructive step
// that needs careful outcome routing in runLab.
func TestLabEnterUSBStepEmitsUSBOutcome(t *testing.T) {
	m := NewLabModel(profile.Profile{}, false)
	// Navigate to Operacional layer (index 2) via two tabs.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(LabModel)

	// First step of Operacional is USB.
	if m.layers[2].Steps[0].ID != stepUSB {
		t.Fatalf("expected first Operacional step to be stepUSB, got %q", m.layers[2].Steps[0].ID)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(LabModel)
	if final.outcome != outcomeLaunchUSB {
		t.Fatalf("expected outcomeLaunchUSB, got %v", final.outcome)
	}
	if cmd == nil {
		t.Fatalf("expected tea.Quit cmd, got nil")
	}
}

// TestLabCompletedStepsFromProfile confirms that a profile with at
// least one installed engine causes the stack and init steps to appear
// as completed in the model, while the detect step remains pending
// (detect is idempotent and never permanently marked done).
func TestLabCompletedStepsFromProfile(t *testing.T) {
	prof := profile.Profile{}
	prof.MarkInstalled("engram")

	m := NewLabModel(prof, false)

	if m.completedAll[stepDetect] {
		t.Error("detect should never be marked done (idempotent, always re-runnable)")
	}
	if !m.completedAll[stepStack] {
		t.Error("expected stack to be marked done when profile has installed engines")
	}
	if !m.completedAll[stepInit] {
		t.Error("expected init to be marked done when profile has installed engines")
	}
}

// TestLabReinstallClearsCompleted confirms that --reinstall causes
// the model to ignore the profile's installed state, so every step
// appears as pending regardless of prior installs.
func TestLabReinstallClearsCompleted(t *testing.T) {
	prof := profile.Profile{}
	prof.MarkInstalled("engram")

	m := NewLabModel(prof, true /*reinstall*/)

	if m.completedAll[stepStack] {
		t.Error("reinstall=true: stack should not be marked done")
	}
	if m.completedAll[stepInit] {
		t.Error("reinstall=true: init should not be marked done")
	}
}

// TestDefaultLayersStructure pins the canonical three-layer layout.
// Layer names and step IDs are part of the external contract (the
// /lab-setup skill, README, and docs reference them by name).
func TestDefaultLayersStructure(t *testing.T) {
	layers := defaultLayers()
	if len(layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(layers))
	}

	// Técnica: detect · stack · init
	if layers[0].Layer != profile.Tecnica {
		t.Errorf("layer[0]: expected Tecnica, got %q", layers[0].Layer)
	}
	tecnicaIDs := []stepID{stepDetect, stepStack, stepInit}
	for i, want := range tecnicaIDs {
		if layers[0].Steps[i].ID != want {
			t.Errorf("Técnica step[%d]: expected %q, got %q", i, want, layers[0].Steps[i].ID)
		}
	}

	// Identitaria: workspace
	if layers[1].Layer != profile.Identitaria {
		t.Errorf("layer[1]: expected Identitaria, got %q", layers[1].Layer)
	}
	if layers[1].Steps[0].ID != stepWorkspace {
		t.Errorf("Identitaria step[0]: expected stepWorkspace, got %q", layers[1].Steps[0].ID)
	}

	// Operacional: usb · credits
	if layers[2].Layer != profile.Operacional {
		t.Errorf("layer[2]: expected Operacional, got %q", layers[2].Layer)
	}
	opIDs := []stepID{stepUSB, stepCredits}
	for i, want := range opIDs {
		if layers[2].Steps[i].ID != want {
			t.Errorf("Operacional step[%d]: expected %q, got %q", i, want, layers[2].Steps[i].ID)
		}
	}
}

// TestLabUSBStepIsDestructive guards the Destructive marker on the USB
// step — the View renders a ⚠ warning for any step with Destructive=true,
// and removing that flag would silently drop the user-facing warning.
func TestLabUSBStepIsDestructive(t *testing.T) {
	layers := defaultLayers()
	for _, l := range layers {
		for _, s := range l.Steps {
			if s.ID == stepUSB && !s.Destructive {
				t.Fatal("USB step must be marked Destructive=true")
			}
		}
	}
}
