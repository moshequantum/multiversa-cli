package stack

import (
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

type Engram struct{}

func (Engram) ID() string          { return "engram" }
func (Engram) DisplayName() string { return "Engram" }
func (Engram) Author() string      { return "Gentleman-Programming" }
func (Engram) Repo() string        { return "https://github.com/Gentleman-Programming/engram" }
func (Engram) License() string     { return "MIT" }
func (Engram) OptIn() bool         { return false }
func (Engram) Prereq() string      { return "brew" }

func (e Engram) Command(version string) []string {
	return []string{"brew", "install", "gentleman-programming/tap/engram"}
}

func (e Engram) Install(version string) error {
	cmd := e.Command(version)
	return xexec.Run(cmd[0], cmd[1:]...).Err
}

func (e Engram) Status() (Status, error) {
	if !xexec.Check("engram") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("engram", "--version")
	if r.Err != nil {
		return Status{Installed: true}, nil
	}
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (e Engram) Uninstall() error { return ErrNotImplemented }
