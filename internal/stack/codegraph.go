package stack

import (
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

type CodeGraph struct{}

func (CodeGraph) ID() string          { return "codegraph" }
func (CodeGraph) DisplayName() string { return "codegraph" }
func (CodeGraph) Author() string      { return "Colby McHenry" }
func (CodeGraph) Repo() string        { return "https://github.com/colbymchenry/codegraph" }
func (CodeGraph) License() string     { return "MIT" }
func (CodeGraph) OptIn() bool         { return true }
func (CodeGraph) Prereq() string      { return "pnpm" }

func (c CodeGraph) Command(version string) []string {
	// pnpm-only by Multiversa policy. npm is banned across the stack — see
	// docs and project-rules-pnpm-only memory note.
	pkg := "@colbymchenry/codegraph"
	if version != "" && version != "latest" {
		pkg = "@colbymchenry/codegraph@" + version
	}
	return []string{"pnpm", "add", "-g", pkg}
}

func (c CodeGraph) Install(version string) error {
	cmd := c.Command(version)
	return xexec.Run(cmd[0], cmd[1:]...).Err
}

func (c CodeGraph) Status() (Status, error) {
	if !xexec.Check("codegraph") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("codegraph", "--version")
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (c CodeGraph) Uninstall() error { return ErrNotImplemented }
