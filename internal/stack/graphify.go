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
func (Graphify) Prereq() string      { return "pipx" }

func (g Graphify) Command(version string) []string {
	pkg := "graphifyy"
	if version != "" && version != "latest" {
		pkg = "graphifyy==" + version
	}
	return []string{"pipx", "install", pkg}
}

func (g Graphify) Install(version string) error {
	cmd := g.Command(version)
	return xexec.Run(cmd[0], cmd[1:]...).Err
}

func (g Graphify) Status() (Status, error) {
	if !xexec.Check("graphify") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("graphify", "--version")
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (g Graphify) Uninstall() error { return ErrNotImplemented }
