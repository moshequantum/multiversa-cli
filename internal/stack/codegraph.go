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

// Strategies: pnpm only. npm is banned across the Multiversa stack — see
// docs and the project-rules-pnpm-only memory note.
func (c CodeGraph) Strategies(version string) []Strategy {
	pkg := "@colbymchenry/codegraph"
	if version != "" && version != "latest" {
		pkg = "@colbymchenry/codegraph@" + version
	}
	return []Strategy{{
		Prereq: "pnpm",
		Cmd:    []string{"pnpm", "add", "-g", pkg},
		Note:   "paquete global con pnpm (npm prohibido por política)",
	}}
}

func (c CodeGraph) Install(version string) error { return runInstall(c, version) }

func (c CodeGraph) Status() (Status, error) {
	if !xexec.Check("codegraph") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("codegraph", "--version")
	if r.Err != nil {
		return Status{Installed: true, Path: binaryPath("codegraph"), Blocked: true, Missing: []string{"working-version-probe"}}, nil
	}
	return Status{Installed: true, Version: r.LastLine(), Path: binaryPath("codegraph")}, nil
}

func (c CodeGraph) Uninstall() error { return ErrNotImplemented }
