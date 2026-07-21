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

// Strategies: Homebrew first, then `go install`. Like Engram, gentle-ai is a
// Go binary, so the toolchain route unblocks Linux machines without brew.
func (g GentleAI) Strategies(version string) []Strategy {
	return []Strategy{
		{
			Prereq: "brew",
			Cmd:    []string{"brew", "install", "Gentleman-Programming/homebrew-tap/gentle-ai"},
			Note:   "vía Homebrew (ruta recomendada por upstream)",
		},
		{
			Prereq: "go",
			// Module path is lowercase upstream, unlike Engram's. Do not
			// "normalise" this — `go install` is case-sensitive.
			Cmd:  []string{"go", "install", "github.com/gentleman-programming/gentle-ai/cmd/gentle-ai" + goModuleVersion(version)},
			Note: "compilado con el toolchain de Go (sin Homebrew)",
		},
	}
}

func (g GentleAI) Install(version string) error { return runInstall(g, version) }

func (g GentleAI) Status() (Status, error) {
	if !xexec.Check("gentle") {
		return Status{Installed: false}, nil
	}
	r := xexec.Run("gentle", "--version")
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (g GentleAI) Uninstall() error { return ErrNotImplemented }
