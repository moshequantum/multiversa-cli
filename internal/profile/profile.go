// Package profile persists the per-user wizard profile across
// Multiversa CLI invocations. The TOML file at
// `~/.multiversa/profile.toml` is the primary store; a best-effort
// mirror to Engram (`multiversa/profile` topic key) lets other
// agents on the user's machine surface the same context.
//
// The profile is intentionally minimal: who the user is (level),
// what language to speak (locale), when we last saw them, and what
// engines they already have. Everything else stays in the engines'
// own state.
package profile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Level expresses how much explanation the user wants per step.
// Maps to tui.Verbosity at render time.
type Level string

const (
	Newcomer   Level = "newcomer"
	Enthusiast Level = "enthusiast"
	Expert     Level = "expert"
)

// IsValid reports whether the level is one of the canonical values.
func (l Level) IsValid() bool {
	switch l {
	case Newcomer, Enthusiast, Expert:
		return true
	}
	return false
}

// Profile is the on-disk shape persisted as TOML.
type Profile struct {
	Level            Level     `toml:"level"`
	Locale           string    `toml:"locale"`
	LastRun          time.Time `toml:"last_run"`
	InstalledEngines []string  `toml:"installed_engines"`
}

// Default returns a freshly-initialized profile with the locale
// detected from the host environment. Callers should ask the user
// for their preferred Level on first run and overwrite the zero
// value before saving.
func Default() Profile {
	return Profile{
		Locale:           DetectLocale(os.Getenv("LANG")),
		InstalledEngines: []string{},
	}
}

// Path returns the canonical filesystem path of the profile.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".multiversa", "profile.toml"), nil
}

// Load reads the profile from ~/.multiversa/profile.toml. If the
// file does not exist, Load returns (Default(), os.ErrNotExist) so
// callers can distinguish "first run" from a real error.
func Load() (Profile, error) {
	path, err := Path()
	if err != nil {
		return Profile{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), os.ErrNotExist
	}
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := toml.Unmarshal(data, &p); err != nil {
		return Profile{}, err
	}
	if p.Locale == "" {
		p.Locale = DetectLocale(os.Getenv("LANG"))
	}
	return p, nil
}

// Save writes the profile to disk and best-effort mirrors to
// Engram. A failure to mirror is logged (in caller code via the
// returned bool) but never blocks the disk write.
func (p *Profile) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	p.LastRun = time.Now().UTC()

	buf, err := marshalTOML(*p)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return err
	}
	// Mirror to Engram if available — never block on this.
	_ = MirrorEngram(*p)
	return nil
}

// MarkInstalled appends an engine id to InstalledEngines if it is
// not already present. The list stays sorted-by-insertion-order so
// the user can see what they installed in what order.
func (p *Profile) MarkInstalled(engineID string) {
	for _, e := range p.InstalledEngines {
		if e == engineID {
			return
		}
	}
	p.InstalledEngines = append(p.InstalledEngines, engineID)
}

// HasEngine reports whether the named engine appears in the
// installed-engines list.
func (p Profile) HasEngine(engineID string) bool {
	for _, e := range p.InstalledEngines {
		if e == engineID {
			return true
		}
	}
	return false
}

// DetectLocale extracts the BCP-47 family from a POSIX LANG value.
// Mapping rules:
//   - "es_*"      → "es-LA" (Spanish Latin neutral, the Multiversa default)
//   - "en_*"      → "en"
//   - empty / "C" → "en" (safe English fallback)
//   - anything else → "en" (Multiversa only ships EN/ES today)
func DetectLocale(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" || lang == "c" || lang == "posix" {
		return "en"
	}
	switch {
	case strings.HasPrefix(lang, "es"):
		return "es-LA"
	case strings.HasPrefix(lang, "en"):
		return "en"
	}
	return "en"
}

// marshalTOML wraps toml.Marshal so the test file can stub the
// encoder if it ever needs to. Today it just delegates.
func marshalTOML(p Profile) ([]byte, error) {
	return toml.Marshal(p)
}
