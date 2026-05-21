// Package stack defines the Engine interface and a registry of curated engines.
// Multiversa orchestrates these engines; it does not author them. See
// internal/credits for canonical attribution.
package stack

import (
	"errors"
	"fmt"
)

var (
	ErrNotImplemented      = errors.New("not implemented yet")
	ErrAgplConsentRequired = errors.New("AGPL-3.0 disclaimer not acknowledged — MiroFish must be invoked external-only")
)

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

	// Prereq returns the name of the external tool required to install this
	// engine (e.g. "go", "pipx", "npm", "docker"). Empty string if no
	// prerequisite is needed.
	Prereq() string

	// Command returns the install command as a slice (program + args), to be
	// executed via internal/exec.Run. Multiversa prints this command to the
	// user before running it.
	Command(version string) []string

	// Install is a convenience wrapper around Command. Most callers should
	// use Command + exec.Run directly so they can stream progress.
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
