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
			//
			// CAVEAT: upstream tags v2.x but its go.mod still declares the
			// v1 module path, breaking Go's major-version rule. So
			// `@latest` resolves to v1.49.0 and `/v2` does not exist. This
			// route therefore installs a major version behind Homebrew —
			// which is why the Note says so out loud rather than letting
			// the user believe they got the current release.
			Cmd:  []string{"go", "install", "github.com/gentleman-programming/gentle-ai/cmd/gentle-ai" + goModuleVersion(version)},
			Note: "toolchain de Go — ⚠️ queda en v1.x: upstream publica v2 sin ruta /v2",
		},
	}
}

func (g GentleAI) Install(version string) error { return runInstall(g, version) }

// Status probes both binary names: Homebrew installs `gentle`, while
// `go install` names the binary after its cmd/ directory, `gentle-ai`.
// Checking only one made a correctly installed engine report as missing.
func (g GentleAI) Status() (Status, error) {
	for _, bin := range []string{"gentle-ai", "gentle"} {
		if !xexec.Check(bin) {
			continue
		}
		r := xexec.Run(bin, "--version")
		if r.Err != nil {
			return Status{Installed: true, Path: bin}, nil
		}
		return Status{Installed: true, Version: r.LastLine(), Path: bin}, nil
	}
	return Status{Installed: false}, nil
}

func (g GentleAI) Uninstall() error { return ErrNotImplemented }
