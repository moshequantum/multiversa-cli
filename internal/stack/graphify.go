package stack

import (
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

type Graphify struct{}

func (Graphify) ID() string          { return "graphify" }
func (Graphify) DisplayName() string { return "Graphify" }
func (Graphify) Author() string      { return "Safi Shamsi" }
func (Graphify) Repo() string        { return "https://github.com/safishamsi/graphify" }
func (Graphify) License() string     { return "MIT" }
func (Graphify) OptIn() bool         { return false }

// Strategies: pipx only. Graphify is a Python package and pipx is the one
// route upstream documents; a bare `pip install` would leak into the system
// interpreter, so it is deliberately not offered.
func (g Graphify) Strategies(version string) []Strategy {
	pkg := "graphifyy"
	if version != "" && version != "latest" {
		pkg = "graphifyy==" + version
	}
	return []Strategy{{
		Prereq: "pipx",
		Cmd:    []string{"pipx", "install", pkg},
		Note:   "paquete Python aislado con pipx",
	}}
}

func (g Graphify) Install(version string) error { return runInstall(g, version) }

func (g Graphify) Status() (Status, error) {
	if !xexec.Check("graphify") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("graphify", "--version")
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (g Graphify) Uninstall() error { return ErrNotImplemented }
