package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// withTempHome redirects HOME to a temp dir so Load/Save touch a
// throwaway profile.toml. Returns the temp directory path.
func withTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func TestLoad_MissingFileReturnsDefaultAndErrNotExist(t *testing.T) {
	withTempHome(t)
	p, err := Load()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
	if p.Locale == "" {
		t.Errorf("default profile should have a non-empty Locale")
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	withTempHome(t)
	in := Profile{
		Level:            Enthusiast,
		Locale:           "es-LA",
		InstalledEngines: []string{"engram", "graphify"},
	}
	if err := in.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	out, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if out.Level != in.Level {
		t.Errorf("Level: got %q, want %q", out.Level, in.Level)
	}
	if out.Locale != in.Locale {
		t.Errorf("Locale: got %q, want %q", out.Locale, in.Locale)
	}
	if len(out.InstalledEngines) != 2 {
		t.Errorf("InstalledEngines: got %v, want 2 entries", out.InstalledEngines)
	}
	if out.LastRun.IsZero() {
		t.Errorf("Save() must stamp LastRun")
	}
}

func TestSave_CreatesParentDirectory(t *testing.T) {
	dir := withTempHome(t)
	p := Default()
	p.Level = Newcomer
	if err := p.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".multiversa", "profile.toml")); err != nil {
		t.Errorf("profile file not created: %v", err)
	}
}

func TestDetectLocale(t *testing.T) {
	cases := map[string]string{
		"":              "en",
		"C":             "en",
		"POSIX":         "en",
		"en_US.UTF-8":   "en",
		"en_GB":         "en",
		"es_VE.UTF-8":   "es-LA",
		"es_MX":         "es-LA",
		"es_ES.UTF-8":   "es-LA", // Spain still maps to es-LA — we only ship neutral Latam.
		"fr_FR.UTF-8":   "en",    // Unsupported language falls back to English.
		"  en_US ":      "en",    // Whitespace tolerance.
	}
	for in, want := range cases {
		if got := DetectLocale(in); got != want {
			t.Errorf("DetectLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMarkInstalled_Idempotent(t *testing.T) {
	p := Default()
	p.MarkInstalled("engram")
	p.MarkInstalled("engram") // second call should be a no-op
	p.MarkInstalled("graphify")
	if len(p.InstalledEngines) != 2 {
		t.Errorf("MarkInstalled deduplication failed: %v", p.InstalledEngines)
	}
}

func TestHasEngine(t *testing.T) {
	p := Profile{InstalledEngines: []string{"engram", "graphify"}}
	if !p.HasEngine("engram") {
		t.Error("HasEngine(engram) should be true")
	}
	if p.HasEngine("mirofish") {
		t.Error("HasEngine(mirofish) should be false")
	}
}

func TestLevel_IsValid(t *testing.T) {
	for _, lv := range []Level{Newcomer, Enthusiast, Expert} {
		if !lv.IsValid() {
			t.Errorf("%q should be valid", lv)
		}
	}
	if Level("sysadmin").IsValid() {
		t.Error("non-canonical level should be invalid")
	}
}
