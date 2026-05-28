package tui

import "testing"

func TestVerbosityForLevel(t *testing.T) {
	cases := map[string]Verbosity{
		"newcomer":   Verbose,
		"enthusiast": Standard,
		"expert":     Condensed,
		"":           Standard, // unknown defaults to Standard
		"sysadmin":   Standard, // any non-canonical → Standard
	}
	for level, want := range cases {
		if got := VerbosityForLevel(level); got != want {
			t.Errorf("VerbosityForLevel(%q) = %v, want %v", level, got, want)
		}
	}
}

func TestVerbosity_String(t *testing.T) {
	cases := map[Verbosity]string{
		Verbose:   "verbose",
		Standard:  "standard",
		Condensed: "condensed",
	}
	for v, want := range cases {
		if got := v.String(); got != want {
			t.Errorf("Verbosity(%d).String() = %q, want %q", v, got, want)
		}
	}
}

func TestChoose_PicksCorrectBranch(t *testing.T) {
	if got := Choose(Verbose, "v", "s", "c"); got != "v" {
		t.Errorf("Choose(Verbose) = %q, want %q", got, "v")
	}
	if got := Choose(Standard, "v", "s", "c"); got != "s" {
		t.Errorf("Choose(Standard) = %q, want %q", got, "s")
	}
	if got := Choose(Condensed, "v", "s", "c"); got != "c" {
		t.Errorf("Choose(Condensed) = %q, want %q", got, "c")
	}
}
