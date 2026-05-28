package tui

// Verbosity is the explanation density a wizard renders. It maps
// one-to-one to profile.Level (newcomer → Verbose, enthusiast →
// Standard, expert → Condensed) but stays here as a UI concern so
// the tui package does not depend on internal/profile.
type Verbosity int

const (
	// Verbose shows the full explanation, hints, and rationale.
	Verbose Verbosity = iota
	// Standard hides only the most expository hints.
	Standard
	// Condensed strips everything except the action — one line per
	// item, keyboard hints kept.
	Condensed
)

// String returns the human label for a verbosity level.
func (v Verbosity) String() string {
	switch v {
	case Verbose:
		return "verbose"
	case Standard:
		return "standard"
	case Condensed:
		return "condensed"
	default:
		return "unknown"
	}
}

// VerbosityForLevel maps a profile level name to the matching
// Verbosity. Unknown levels default to Standard so missing config
// never breaks the UI.
func VerbosityForLevel(level string) Verbosity {
	switch level {
	case "newcomer":
		return Verbose
	case "expert":
		return Condensed
	case "enthusiast":
		return Standard
	default:
		return Standard
	}
}

// Choose returns one of the three strings based on the current
// verbosity. Useful for inline conditionals in View() methods:
//
//	hint := tui.Choose(v, "Press y/n to confirm", "y/n", "")
func Choose[T any](v Verbosity, verbose, standard, condensed T) T {
	switch v {
	case Verbose:
		return verbose
	case Condensed:
		return condensed
	default:
		return standard
	}
}
