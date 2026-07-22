// Package stack defines the Engine interface and a registry of curated engines.
// Multiversa orchestrates these engines; it does not author them. See
// internal/credits for canonical attribution.
package stack

import (
	"errors"
	"fmt"
	"strings"

	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

var (
	ErrNotImplemented      = errors.New("not implemented yet")
	ErrAgplConsentRequired = errors.New("AGPL-3.0 disclaimer not acknowledged — MiroFish must be invoked external-only")
)

// Strategy is one way to install an engine: the external tool it needs on
// PATH plus the command that uses it. Engines declare several in preference
// order, so a machine without Homebrew can still install a Go binary. An
// engine with a single strategy behaves exactly as it did before.
type Strategy struct {
	// Prereq is the external tool required on PATH ("brew", "pipx", "pnpm",
	// "go", "docker"). Empty means the command needs nothing extra.
	// NOTE: Multiversa policy bans npm — pnpm only.
	Prereq string

	// Cmd is the install command (program + args), run via internal/exec.Run.
	Cmd []string

	// Note is a short human explanation of when this route applies. It is
	// shown when this is the strategy that got picked.
	Note string
}

// PrereqError reports that no strategy is viable because none of their
// prerequisites are on PATH. It names every option so the user can pick.
type PrereqError struct {
	Engine  string
	Options []string
}

func (e *PrereqError) Error() string {
	if len(e.Options) == 1 {
		return fmt.Sprintf("%s necesita %q en el PATH", e.Engine, e.Options[0])
	}
	return fmt.Sprintf("%s necesita alguno de estos en el PATH: %s",
		e.Engine, strings.Join(e.Options, ", "))
}

// PickStrategy returns the first strategy whose prerequisite is available.
// When none is viable it returns the preferred (first) strategy and false,
// so callers can still show the user what would have run.
func PickStrategy(e Engine, version string) (Strategy, bool) {
	all := e.Strategies(version)
	if len(all) == 0 {
		return Strategy{}, false
	}
	for _, s := range all {
		if s.Prereq == "" || xexec.Check(s.Prereq) {
			return s, true
		}
	}
	return all[0], false
}

// Prereqs lists the prerequisite of every strategy, in preference order —
// i.e. what the user could install to unblock this engine. Deduplicated.
func Prereqs(e Engine, version string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range e.Strategies(version) {
		if s.Prereq != "" && !seen[s.Prereq] {
			seen[s.Prereq] = true
			out = append(out, s.Prereq)
		}
	}
	return out
}

// runInstall is the shared body of every Engine.Install: pick a viable
// strategy, or report every prerequisite that would have made one viable.
func runInstall(e Engine, version string) error {
	s, ok := PickStrategy(e, version)
	if !ok {
		return &PrereqError{Engine: e.ID(), Options: Prereqs(e, version)}
	}
	return xexec.Run(s.Cmd[0], s.Cmd[1:]...).Err
}

// goModuleVersion turns a Multiversa version string into the suffix `go
// install` expects. Empty and "latest" both mean @latest; anything else is
// pinned, with the `v` prefix added when missing.
func goModuleVersion(version string) string {
	if version == "" || version == "latest" {
		return "@latest"
	}
	if strings.HasPrefix(version, "v") {
		return "@" + version
	}
	return "@v" + version
}

// Status is the local installation state of an engine.
type Status struct {
	Installed bool
	Version   string
	Path      string
}

// Engine is the contract every curated engine must satisfy.
type Engine interface {
	ID() string
	DisplayName() string
	Author() string
	Repo() string
	License() string
	OptIn() bool

	// Strategies returns every way this engine can be installed, in
	// preference order. The first whose Prereq is on PATH wins — that is
	// what lets Engram install on a machine without Homebrew. Never empty.
	// Use PickStrategy to resolve one against the current machine.
	Strategies(version string) []Strategy

	// Install is a convenience wrapper around PickStrategy. Most callers
	// should use PickStrategy + exec.Run directly so they can stream
	// progress and show which route was taken.
	Install(version string) error

	// Status checks whether the engine is already present locally.
	Status() (Status, error)

	// Uninstall removes the engine (best-effort).
	Uninstall() error
}

// Registry returns the canonical map of supported engines.
func Registry() map[string]Engine {
	return map[string]Engine{
		"engram":    &Engram{},
		"graphify":  &Graphify{},
		"gentle-ai": &GentleAI{},
		"gentle-pi": &GentlePi{},
		"codegraph": &CodeGraph{},
		"mirofish":  &MiroFish{},
	}
}

// List returns the engines in canonical display order.
func List() []Engine {
	reg := Registry()
	order := []string{"engram", "graphify", "gentle-ai", "gentle-pi", "codegraph", "mirofish"}
	out := make([]Engine, 0, len(order))
	for _, id := range order {
		out = append(out, reg[id])
	}
	return out
}

// Resolve returns the Engine for the given id, or an error.
func Resolve(id string) (Engine, error) {
	if e, ok := Registry()[id]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("unknown engine %q", id)
}
