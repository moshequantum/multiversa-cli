package stack

import (
	"strings"

	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

type GentlePi struct{}

func (GentlePi) ID() string          { return "gentle-pi" }
func (GentlePi) DisplayName() string { return "gentle-pi" }
func (GentlePi) Author() string      { return "Gentleman-Programming" }
func (GentlePi) Repo() string        { return "https://github.com/Gentleman-Programming/gentle-pi" }
func (GentlePi) License() string     { return "MIT" }
func (GentlePi) OptIn() bool         { return false }

// gentle-pi is a Pi package, not a standalone global package. Upstream's
// supported route is `pi install npm:gentle-pi`; the npm: prefix identifies
// the registry package and does not invoke the banned npm CLI.
func (g GentlePi) Strategies(version string) []Strategy {
	pkg := "npm:gentle-pi"
	if version != "" && version != "latest" {
		pkg = "npm:gentle-pi@" + version
	}
	return []Strategy{{
		Prereq: "pi",
		Cmd:    []string{"pi", "install", pkg},
		Note:   "paquete del runtime Pi (requiere Pi instalado)",
	}}
}

func (g GentlePi) Install(version string) error { return runInstall(g, version) }

// Status separates the package artifact from the runtime that can execute it.
// Older Multiversa builds installed gentle-pi globally with pnpm; that leaves
// a detectable package but no usable integration when `pi` is absent.
func (g GentlePi) Status() (Status, error) {
	legacyPresent, legacyVersion := legacyGentlePiPackage()
	if !xexec.Check("pi") {
		if legacyPresent {
			return Status{
				Installed: true,
				Version:   legacyVersion,
				Blocked:   true,
				Missing:   []string{"runtime:pi"},
				Evidence:  []string{"legacy-global-package:gentle-pi"},
			}, nil
		}
		return Status{Installed: false}, nil
	}

	// Pi owns its package inventory. If its list surface changes, preserve the
	// runtime evidence but do not claim gentle-pi is installed.
	r := xexec.Run("pi", "list")
	for _, line := range r.Output {
		if strings.Contains(line, "gentle-pi") {
			return Status{
				Installed: true,
				Version:   piPackageVersion(line),
				Path:      binaryPath("pi"),
				Evidence:  []string{"runtime:pi", "pi-package:gentle-pi"},
			}, nil
		}
	}
	if legacyPresent {
		return Status{
			Installed: true,
			Version:   legacyVersion,
			Path:      binaryPath("pi"),
			Blocked:   true,
			Missing:   []string{"pi-package:gentle-pi"},
			Evidence:  []string{"legacy-global-package:gentle-pi", "runtime:pi"},
		}, nil
	}
	return Status{Installed: false, Path: binaryPath("pi"), Evidence: []string{"runtime:pi"}}, nil
}

// piPackageVersion extracts a pinned version from Pi's package inventory.
// An unpinned source is still a valid installed package, but has no reliable
// version in `pi list`, so it intentionally returns an empty string.
func piPackageVersion(line string) string {
	const marker = "npm:gentle-pi@"
	idx := strings.Index(line, marker)
	if idx < 0 {
		return ""
	}
	version := strings.Fields(line[idx+len(marker):])
	if len(version) == 0 {
		return ""
	}
	return version[0]
}

func legacyGentlePiPackage() (bool, string) {
	if !xexec.Check("pnpm") {
		return false, ""
	}
	r := xexec.Run("pnpm", "list", "-g", "--depth", "0")
	if r.Err != nil {
		return false, ""
	}
	for _, line := range r.Output {
		idx := strings.Index(line, "gentle-pi@")
		if idx < 0 {
			continue
		}
		version := strings.Fields(line[idx+len("gentle-pi@"):])
		if len(version) == 0 {
			return true, ""
		}
		return true, version[0]
	}
	return false, ""
}

func (g GentlePi) Uninstall() error { return ErrNotImplemented }
