// Tests for the wizard's install step — specifically that engine
// prerequisites are resolved per-strategy rather than assuming a single
// package manager. This is the T9 regression: before multi-strategy support,
// a Linux box without Homebrew could not install Engram at all, which
// blocked the whole "mi propio OS desde la TUI" flow.
package steps

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/profile"
)

// onlyOnPath rewrites PATH to a temp dir holding just the named fake
// executables, so the step can be tested against a machine that
// deliberately lacks Homebrew.
func onlyOnPath(t *testing.T, tools ...string) {
	t.Helper()
	dir := t.TempDir()
	for _, tool := range tools {
		p := filepath.Join(dir, tool)
		if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("writing fake %q: %v", tool, err)
		}
	}
	t.Setenv("PATH", dir)
}

// runEngineOnce drives startEngine for a single engine and returns the
// message it produced, without spinning up a real tea.Program.
func runEngineOnce(t *testing.T, id string) installResultMsg {
	t.Helper()
	step := NewInstall().(*Install)
	step.SetDryRun(true)
	step.Set([]string{id}, "local")

	cmd := step.startEngine(0)
	if cmd == nil {
		t.Fatalf("%s: startEngine returned no command", id)
	}
	msg, ok := cmd().(installResultMsg)
	if !ok {
		t.Fatalf("%s: expected installResultMsg, got %T", id, cmd())
	}
	return msg
}

// The T9 regression itself: Engram must install on a machine with the Go
// toolchain and no Homebrew.
func TestInstallStepUsesGoRouteWithoutHomebrew(t *testing.T) {
	onlyOnPath(t, "go")

	msg := runEngineOnce(t, "engram")

	if msg.status == stPrereqMissing {
		t.Fatalf("engram reported a missing prerequisite despite Go being available: %s", msg.result.Cmd)
	}
	if !strings.Contains(msg.result.Cmd, "go install") {
		t.Errorf("expected the Go route, got %q", msg.result.Cmd)
	}
	if strings.Contains(msg.result.Cmd, "brew") {
		t.Errorf("brew should not appear when it is absent from PATH: %q", msg.result.Cmd)
	}
}

// When every route is blocked the user must see all of them, not just the
// first — otherwise a Linux user is told to install Homebrew when installing
// Go would do.
func TestInstallStepNamesEveryRouteWhenAllBlocked(t *testing.T) {
	onlyOnPath(t) // empty PATH

	msg := runEngineOnce(t, "engram")

	if msg.status != stPrereqMissing {
		t.Fatalf("expected stPrereqMissing on an empty PATH, got %v", msg.status)
	}
	for _, want := range []string{"Homebrew", "Go"} {
		if !strings.Contains(msg.result.Cmd, want) {
			t.Errorf("hint %q does not mention %q", msg.result.Cmd, want)
		}
	}
}

// Single-route engines still report their one prerequisite plainly.
func TestInstallStepSingleRouteHintIsUnchanged(t *testing.T) {
	onlyOnPath(t) // empty PATH

	msg := runEngineOnce(t, "graphify")

	if msg.status != stPrereqMissing {
		t.Fatalf("expected stPrereqMissing, got %v", msg.status)
	}
	if !strings.Contains(msg.result.Cmd, "pipx") {
		t.Errorf("expected the pipx hint, got %q", msg.result.Cmd)
	}
	if strings.Contains(msg.result.Cmd, "o bien") {
		t.Errorf("single-route engine should not offer alternatives: %q", msg.result.Cmd)
	}
}

// MiroFish stays gated on the AGPL acknowledgement regardless of PATH.
func TestInstallStepMiroFishStillRequiresAgplConsent(t *testing.T) {
	onlyOnPath(t, "docker")

	step := NewInstall().(*Install)
	step.SetDryRun(true)
	step.SetAgplAcknowledged(false)
	step.Set([]string{"mirofish"}, "local")

	msg, ok := step.startEngine(0)().(installResultMsg)
	if !ok {
		t.Fatal("expected installResultMsg")
	}
	if msg.status != stError {
		t.Fatalf("expected stError without AGPL consent, got %v", msg.status)
	}
}

// The other half of the T9 flow: after a real install, the profile must be on
// disk so the next launch — and Engram — start from the machine's true state.
// Before this wiring the wizard installed engines and then forgot everything.
func TestInstallPersistsProfileAfterRealRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	step := NewInstall().(*Install)
	step.Set([]string{"engram", "graphify"}, "local")
	// Simulate the end state of a run where only engram succeeded.
	step.statuses["engram"] = stDone
	step.statuses["graphify"] = stError
	step.finished = true

	msg, ok := step.persistProfile()().(profileSavedMsg)
	if !ok {
		t.Fatalf("expected profileSavedMsg, got %T", step.persistProfile()())
	}
	if msg.err != nil {
		t.Fatalf("persisting profile failed: %v", msg.err)
	}

	p, err := profile.Load()
	if err != nil {
		t.Fatalf("profile not readable after save: %v", err)
	}
	if !p.HasEngine("engram") {
		t.Error("engram installed but not recorded in the profile")
	}
	if p.HasEngine("graphify") {
		t.Error("graphify failed to install but was recorded as present")
	}
	if !p.Level.IsValid() {
		t.Errorf("profile persisted with an invalid level %q", p.Level)
	}
}

// A dry-run previews; it must not write a profile, and must not claim to.
func TestInstallDryRunPersistsNothing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	step := NewInstall().(*Install)
	step.SetDryRun(true)
	step.Set([]string{"engram"}, "local")
	step.statuses["engram"] = stSkipped
	step.finished = true

	if _, ok := step.persistProfile()().(profileSavedMsg); !ok {
		t.Fatal("expected profileSavedMsg from dry-run")
	}
	if _, err := profile.Load(); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dry-run wrote a profile it should not have (Load err = %v)", err)
	}

	// And the surface must say so, never implying a write happened.
	step.saved = true
	if got := step.renderProfileLine(); !strings.Contains(got, "dry-run") {
		t.Errorf("dry-run profile line should mark itself a preview, got %q", got)
	}
}

// Compile-time assertion that Install still satisfies the Step contract the
// wizard chain depends on.
func TestInstallSatisfiesStep(t *testing.T) {
	var _ Step = NewInstall()
}
