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

// Strategies: Homebrew first (the upstream-blessed route on macOS), then
// `go install`. Engram is a Go binary — see the module path below — so the
// second route works anywhere the Go toolchain is present, which is what
// makes Engram installable on Linux without Homebrew.
func (e Engram) Strategies(version string) []Strategy {
	return []Strategy{
		{
			Prereq: "brew",
			Cmd:    []string{"brew", "install", "gentleman-programming/tap/engram"},
			Note:   "vía Homebrew (ruta recomendada por upstream)",
		},
		{
			Prereq: "go",
			// Module path is capitalised upstream; gentle-ai's is not.
			Cmd:  []string{"go", "install", "github.com/Gentleman-Programming/engram/cmd/engram" + goModuleVersion(version)},
			Note: "compilado con el toolchain de Go (sin Homebrew)",
		},
	}
}

func (e Engram) Install(version string) error { return runInstall(e, version) }

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
