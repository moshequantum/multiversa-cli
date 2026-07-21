package stack

import (
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

func (g GentlePi) Status() (Status, error) {
	if !xexec.Check("gentle-pi") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("gentle-pi", "--version")
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (g GentlePi) Uninstall() error { return ErrNotImplemented }
