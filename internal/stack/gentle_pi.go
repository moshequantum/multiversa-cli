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
func (GentlePi) Prereq() string      { return "pnpm" }

func (g GentlePi) Command(version string) []string {
	// pnpm-only by Multiversa policy. npm is banned across the stack — see
	// docs and project-rules-pnpm-only memory note.
	pkg := "gentle-pi"
	if version != "" && version != "latest" {
		pkg = "gentle-pi@" + version
	}
	return []string{"pnpm", "add", "-g", pkg}
}

func (g GentlePi) Install(version string) error {
	cmd := g.Command(version)
	return xexec.Run(cmd[0], cmd[1:]...).Err
}

func (g GentlePi) Status() (Status, error) {
	if !xexec.Check("gentle-pi") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("gentle-pi", "--version")
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (g GentlePi) Uninstall() error { return ErrNotImplemented }
