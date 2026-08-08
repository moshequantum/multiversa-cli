package steps

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moshequantum/multiversa-cli/internal/profile"
)

func isolateNamingProfile(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Profile.Save mirrors to Engram when it is on PATH. Naming tests only
	// exercise the canonical TOML write, so keep that optional integration out.
	t.Setenv("PATH", t.TempDir())
	return home
}

func TestNamingStartsWithSlugSafeSuggestion(t *testing.T) {
	step := newNamingWithSuggestion("demo-host", "fallback-user")

	if got := step.Name(); got != "demo-hostOS" {
		t.Fatalf("suggested name = %q, want demo-hostOS", got)
	}
	if err := validateProjectOSName(step.Name()); err != nil {
		t.Fatalf("suggested name is invalid: %v", err)
	}
	if view := step.View(); !strings.Contains(view, "¿Cómo se llamará tu ProjectOS?") || !strings.Contains(view, step.Name()) {
		t.Fatalf("naming view does not show the prompt and suggestion:\n%s", view)
	}
}

func TestNamingSuggestionFallsBackToUser(t *testing.T) {
	if got := suggestProjectOSName("áé", "fallback-user"); got != "fallback-userOS" {
		t.Fatalf("fallback suggestion = %q, want fallback-userOS", got)
	}
}

func TestNamingRejectsEmptyAndUnsafeNames(t *testing.T) {
	for _, name := range []string{"", "Mini Universo OS", "MíOS", "-demo", "demo/../../os"} {
		t.Run(name, func(t *testing.T) {
			step := newNamingWithSuggestion("demo-host", "fallback-user")
			step.SetName(name)

			_, cmd := step.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if cmd != nil {
				t.Fatalf("invalid name %q advanced or persisted", name)
			}
			if step.validationErr == "" {
				t.Fatalf("invalid name %q did not explain how to fix it", name)
			}
		})
	}
}

func TestNamingPersistsBeforeAdvancing(t *testing.T) {
	isolateNamingProfile(t)
	step := newNamingWithSuggestion("demo-host", "fallback-user")
	step.SetName("MiniUniversoOS")

	_, cmd := step.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("real naming step did not return a persistence command")
	}
	msg, ok := cmd().(namingSavedMsg)
	if !ok {
		t.Fatalf("persistence command returned %T", cmd())
	}
	if msg.err != nil {
		t.Fatalf("persist name: %v", msg.err)
	}
	_, next := step.Update(msg)
	if next == nil {
		t.Fatal("successful persistence did not advance the wizard")
	}
	if _, ok := next().(NextMsg); !ok {
		t.Fatalf("successful persistence returned %T, want NextMsg", next())
	}

	p, err := profile.Load()
	if err != nil {
		t.Fatalf("load persisted profile: %v", err)
	}
	if p.ProjectOSName != "MiniUniversoOS" {
		t.Fatalf("project OS name = %q, want MiniUniversoOS", p.ProjectOSName)
	}
}

func TestNamingDryRunShowsNameWithoutWriting(t *testing.T) {
	isolateNamingProfile(t)
	step := newNamingWithSuggestion("demo-host", "fallback-user")
	step.SetDryRun(true)
	step.SetName("PreviewOS")

	_, cmd := step.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("dry-run did not advance")
	}
	if _, ok := cmd().(NextMsg); !ok {
		t.Fatalf("dry-run returned %T, want NextMsg", cmd())
	}
	if _, err := profile.Load(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run wrote profile.toml: %v", err)
	}
	if view := step.View(); !strings.Contains(view, "PreviewOS") || !strings.Contains(view, "dry-run") {
		t.Fatalf("dry-run view does not show the chosen name and mode:\n%s", view)
	}
}

func TestNamingDoesNotOverwriteCorruptProfile(t *testing.T) {
	home := isolateNamingProfile(t)
	path := filepath.Join(home, ".multiversa", "profile.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const corrupt = "this is not = valid [toml"
	if err := os.WriteFile(path, []byte(corrupt), 0o600); err != nil {
		t.Fatal(err)
	}

	step := newNamingWithSuggestion("demo-host", "fallback-user")
	step.SetName("SafeOS")
	_, cmd := step.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd().(namingSavedMsg)
	if msg.err == nil {
		t.Fatal("corrupt profile should block the write")
	}
	_, next := step.Update(msg)
	if next != nil {
		t.Fatal("wizard advanced despite profile persistence failure")
	}
	if step.saveErr == "" || !strings.Contains(step.View(), "Corrige") {
		t.Fatalf("save failure does not tell the person what to do:\n%s", step.View())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != corrupt {
		t.Fatalf("corrupt profile was overwritten:\n%s", data)
	}
}

func TestNamingSatisfiesStep(t *testing.T) {
	var _ Step = NewNaming()
}
