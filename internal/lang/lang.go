// Package lang declares the OS-level developer toolchain that
// `multiversa stack` installs. These are distinct from the curated
// agentic engines in internal/stack (Engram, Graphify, …): tools here
// are foundational languages and runtimes (Go, Rust, Python, Node, pnpm).
//
// Every Tool installs into the user's home directory by default so a
// portable USB lab can carry its own toolchain without sudo.
package lang

import (
	"errors"

	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

// ErrUnsupportedOS is returned by a Tool when the current host has no
// install path defined (e.g. asking for pyenv on Windows).
var ErrUnsupportedOS = errors.New("not supported on this OS — use the bash fallback or platform-native installer")

// Plan describes how a Tool will be installed on the current host.
// The shell field, when non-empty, is a single command piped into sh
// (used by rustup, pnpm install, pyenv, nvm). When empty, Args is run
// directly via internal/exec.
type Plan struct {
	Program string   // executable to run ("sh", "brew", "apt", …)
	Args    []string // arguments
	Shell   string   // optional: full shell pipeline (e.g. "curl … | sh")
	Notes   string   // human-readable note about what happens
}

// Tool is one OS-level developer dependency.
type Tool interface {
	ID() string           // stable kebab-case identifier
	DisplayName() string  // human label
	Description() string  // one-line purpose
	Probe() string        // binary name to check on PATH (matches detect)

	// PlanFor returns the install Plan for the given OS kind ("darwin",
	// "linux", "windows") and pkgMgr ("brew", "apt", "dnf", …). It returns
	// (Plan{}, ErrUnsupportedOS) when no path is defined.
	PlanFor(osKind, pkgMgr string) (Plan, error)

	// Installed reports whether the tool is already on PATH.
	Installed() bool
}

// Registry returns the curated list in canonical install order.
// Order matters: Rust before pnpm before Node before … nothing depends
// on this strictly today, but a stable order makes runs reproducible.
func Registry() []Tool {
	return []Tool{
		&Go{},
		&Rust{},
		&Python{},
		&Node{},
		&Pnpm{},
		&Docker{},
	}
}

// installed is the shared default implementation of Tool.Installed.
func installed(probe string) bool { return xexec.Check(probe) }
