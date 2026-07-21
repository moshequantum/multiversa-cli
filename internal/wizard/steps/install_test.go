// Tests for the wizard's install step — specifically that engine
// prerequisites are resolved per-strategy rather than assuming a single
// package manager. This is the T9 regression: before multi-strategy support,
// a Linux box without Homebrew could not install Engram at all, which
// blocked the whole "mi propio OS desde la TUI" flow.
package steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// Compile-time assertion that Install still satisfies the Step contract the
// wizard chain depends on.
func TestInstallSatisfiesStep(t *testing.T) {
	var _ Step = NewInstall()
}
