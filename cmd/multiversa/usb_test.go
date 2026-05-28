// Tests for `multiversa usb`. The contract is destructive, so the
// suite focuses on the gate-1 phrase check (any deviation cancels)
// and the tea.Model wiring (esc emits Quit + Cancel outcome). The
// embedded-script-runner path is intentionally NOT exercised here
// — those scripts run real `cryptsetup`/`diskutil` commands.
package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

// TestUSBPhraseStrict locks the gate-1 contract: only the exact
// phrase passes. Anything else — empty, "yes", "y", partial, extra
// chars — must cancel. This protects the most destructive command in
// the CLI from a one-keystroke mistake.
func TestUSBPhraseStrict(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// The exact phrase, case-insensitive, with surrounding space.
		{"i understand", true},
		{"I Understand", true},
		{"  i understand  ", true},
		{"I UNDERSTAND", true},

		// Everything else MUST cancel — including the legacy single
		// "y" that was good enough in v0.3.0.
		{"", false},
		{" ", false},
		{"y", false},
		{"Y", false},
		{"yes", false},
		{"YES", false},
		{"i understand!", false},
		{"i understand.", false},
		{"i  understand", false}, // double space
		{"understand", false},
		{"i", false},
		{"sí entiendo", false}, // Spanish phrasing is intentionally not honored.
	}
	for _, c := range cases {
		if got := confirmUSBPhrase(c.in); got != c.want {
			t.Errorf("confirmUSBPhrase(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestUSBScriptFor verifies the platform → script mapping. We avoid
// touching --show end-to-end (that would read the real embed) and
// instead pin the contract at the helper that picks the script name.
func TestUSBScriptFor(t *testing.T) {
	cases := []struct {
		kind, want string
		wantErr    bool
	}{
		{"linux", "encrypted_usb_linux.sh", false},
		{"darwin", "encrypted_usb_macos.sh", false},
		{"windows", "", true},
		{"plan9", "", true},
	}
	for _, c := range cases {
		got, err := usbScriptFor(c.kind)
		if (err != nil) != c.wantErr {
			t.Errorf("usbScriptFor(%q) err=%v wantErr=%v", c.kind, err, c.wantErr)
		}
		if got != c.want {
			t.Errorf("usbScriptFor(%q) = %q, want %q", c.kind, got, c.want)
		}
	}
}

// TestUSBShowMacOSScript exercises the --show short-circuit on a
// fabricated macOS report. We read the embedded script via the same
// helper the production path uses; the test passes when it returns
// non-empty bytes that look like a bash script.
func TestUSBShowMacOSScript(t *testing.T) {
	name, err := usbScriptFor("darwin")
	if err != nil {
		t.Fatalf("usbScriptFor(darwin): %v", err)
	}
	data, err := readEmbeddedScript(name)
	if err != nil {
		t.Fatalf("readEmbeddedScript(%q): %v", name, err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty script body for %s", name)
	}
	if !strings.HasPrefix(string(data), "#!") {
		t.Fatalf("expected shebang at start of %s, got: %q", name, string(data[:min(len(data), 40)]))
	}
}

// TestUSBModelSatisfiesTeaModel is a compile-time assertion: the
// model honors the tea.Model contract. If this stops compiling, the
// dual-mode TUI/plain contract is broken at the API surface.
func TestUSBModelSatisfiesTeaModel(t *testing.T) {
	var _ tea.Model = NewUSBModel(detect.Report{}, "encrypted_usb_linux.sh", tui.Standard)
}

// TestUSBModelEscCancels confirms that esc transitions the outcome to
// cancel AND returns a Quit command. The combined ExitAltScreen+Quit
// batch matters for the handoff to gate 2 (the bash script): without
// the alt-screen exit, the script's prompts would render on a clean
// terminal and the user would lose the scrollback.
func TestUSBModelEscCancels(t *testing.T) {
	m := NewUSBModel(detect.Report{}, "encrypted_usb_linux.sh", tui.Standard)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	final, ok := updated.(USBModel)
	if !ok {
		t.Fatalf("expected USBModel, got %T", updated)
	}
	if final.outcome != usbOutcomeCancel {
		t.Fatalf("expected outcome=cancel after esc, got %v", final.outcome)
	}
	if cmd == nil {
		t.Fatalf("expected tea.Quit-bearing cmd after esc, got nil")
	}
}

// TestUSBModelWrongPhraseCancels feeds the model a phrase that isn't
// the exact magic words, then presses enter. The model MUST cancel.
// This is the keystroke-level proof of the strict no-default-yes rule.
func TestUSBModelWrongPhraseCancels(t *testing.T) {
	cases := []string{"yes", "y", "i agree", "ok"}
	for _, phrase := range cases {
		m := NewUSBModel(detect.Report{}, "encrypted_usb_linux.sh", tui.Standard)
		// Type the phrase one rune at a time, the way the user would.
		for _, r := range phrase {
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(USBModel)
		}
		// Press enter.
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(USBModel)
		if m.outcome != usbOutcomeCancel {
			t.Errorf("phrase %q: expected outcome=cancel, got %v", phrase, m.outcome)
		}
	}
}

// TestUSBModelRightPhraseConfirms feeds the exact phrase and presses
// enter. The model MUST transition to confirm. This is the other half
// of the strict contract: gate 1 IS passable, just only by typing the
// real phrase.
func TestUSBModelRightPhraseConfirms(t *testing.T) {
	m := NewUSBModel(detect.Report{}, "encrypted_usb_linux.sh", tui.Standard)
	for _, r := range "i understand" {
		// Space needs the KeySpace type to round-trip through Bubble Tea
		// the way real terminals deliver it.
		var msg tea.KeyMsg
		if r == ' ' {
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		}
		updated, _ := m.Update(msg)
		m = updated.(USBModel)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(USBModel)
	if m.outcome != usbOutcomeConfirm {
		t.Fatalf("expected outcome=confirm after typing phrase + enter, got %v (input=%q)",
			m.outcome, m.input)
	}
	if cmd == nil {
		t.Fatalf("expected tea.Quit-bearing cmd after enter, got nil")
	}
}

// TestUSBModelInitViewSize covers the WindowSizeMsg → View path: the
// header text and the warn banner MUST appear once we know the
// terminal width. This is the smoke test that the screen renders.
func TestUSBModelInitViewSize(t *testing.T) {
	m := NewUSBModel(detect.Report{OS: detect.OSInfo{Kind: "linux", Arch: "amd64", Distro: "test"}},
		"encrypted_usb_linux.sh", tui.Standard)
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil Init cmd, got %T", cmd)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	view := updated.View()
	if !strings.Contains(view, "multiversa usb") {
		t.Fatalf("View missing header, got:\n%s", view)
	}
	if !strings.Contains(view, "i understand") {
		t.Fatalf("View missing gate-1 phrase reminder, got:\n%s", view)
	}
	if !strings.Contains(view, "encrypted_usb_linux.sh") {
		t.Fatalf("View missing script name, got:\n%s", view)
	}
}

// min is a tiny helper for the script-bytes preview in error
// messages. Go 1.21+ has a builtin min(), but we stay portable.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
