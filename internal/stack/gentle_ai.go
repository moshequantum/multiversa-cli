package stack

import (
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

type GentleAI struct{}

func (GentleAI) ID() string          { return "gentle-ai" }
func (GentleAI) DisplayName() string { return "gentle-ai" }
func (GentleAI) Author() string      { return "Gentleman-Programming" }
func (GentleAI) Repo() string        { return "https://github.com/Gentleman-Programming/gentle-ai" }
func (GentleAI) License() string     { return "MIT" }
func (GentleAI) OptIn() bool         { return false }
func (GentleAI) Prereq() string      { return "brew" }

func (g GentleAI) Command(version string) []string {
	return []string{"brew", "install", "Gentleman-Programming/homebrew-tap/gentle-ai"}
}

func (g GentleAI) Install(version string) error {
	cmd := g.Command(version)
	return xexec.Run(cmd[0], cmd[1:]...).Err
}

func (g GentleAI) Status() (Status, error) {
	if !xexec.Check("gentle") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("gentle", "--version")
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (g GentleAI) Uninstall() error { return ErrNotImplemented }
