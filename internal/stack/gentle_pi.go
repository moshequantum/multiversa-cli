package stack

import (
	"strings"

	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

type GentlePi struct{}

func (GentlePi) ID() string          { return "gentle-pi" }
func (GentlePi) DisplayName() string { return "gentle-pi" }
func (GentlePi) Author() string      { return "Gentleman-Programming" }
func (GentlePi) Repo() string        { return "https://github.com/Gentleman-Programming/gentle-pi" }
func (GentlePi) License() string     { return "MIT" }
func (GentlePi) OptIn() bool         { return false }

// Strategies: pnpm only. npm is banned across the Multiversa stack — see
// docs and the project-rules-pnpm-only memory note.
func (g GentlePi) Strategies(version string) []Strategy {
	pkg := "gentle-pi"
	if version != "" && version != "latest" {
		pkg = "gentle-pi@" + version
	}
	return []Strategy{{
		Prereq: "pnpm",
		Cmd:    []string{"pnpm", "add", "-g", pkg},
		Note:   "paquete global con pnpm (npm prohibido por política)",
	}}
}

func (g GentlePi) Install(version string) error { return runInstall(g, version) }

// Status cannot probe the PATH: the gentle-pi npm package declares no `bin`
// at all — it is a development harness (SDD/OpenSpec, subagents, TDD
// evidence), not a CLI. Looking for a `gentle-pi` executable therefore always
// failed, reporting a correctly installed engine as missing. Ask the package
// manager that owns it instead.
func (g GentlePi) Status() (Status, error) {
	if !xexec.Check("pnpm") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("pnpm", "list", "-g", "--depth", "0")
	if r.Err != nil {
		return Status{Installed: false}, nil
	}
	for _, line := range r.Output {
		// pnpm renders entries as `gentle-pi@1.1.0`, possibly inside a
		// tree-drawing prefix such as "└── ".
		idx := strings.Index(line, "gentle-pi@")
		if idx < 0 {
			continue
		}
		version := strings.Fields(line[idx+len("gentle-pi@"):])
		if len(version) == 0 {
			return Status{Installed: true}, nil
		}
		return Status{Installed: true, Version: version[0]}, nil
	}
	return Status{Installed: false}, nil
}

func (g GentlePi) Uninstall() error { return ErrNotImplemented }
