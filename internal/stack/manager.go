// Package stack defines the Engine interface and a registry of curated engines.
// Multiversa orchestrates these engines; it does not author them. See
// internal/credits for canonical attribution.
package stack

import (
	"errors"
	"fmt"
)

var ErrNotImplemented = errors.New("not implemented yet")

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
	Install(version string) error
	Update() error
	Status() (Status, error)
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
