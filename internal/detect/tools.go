package detect

import (
	"strings"

	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

// Tool describes one developer-stack binary being probed.
// Advisory=true means presence is informational only (e.g. npm — banned by
// policy, we flag it but don't fail).
type Tool struct {
	Name      string
	Installed bool
	Version   string
	Advisory  bool // true = informational only, not a "missing" failure
	Warn      bool // true = present but flagged (e.g. npm policy violation)
	Category  string
	Path      string
}

// toolProbe describes one item we want to check on the host.
type toolProbe struct {
	name     string
	category string
	versionArgs []string
	advisory bool
	warnIfPresent bool
}

// canonicalProbes is the curated list checked by `multiversa detect`.
// Order is the rendering order. pnpm comes before npm intentionally:
// the report should reinforce the policy ("pnpm yes, npm no").
var canonicalProbes = []toolProbe{
	// Languages & runtimes
	{name: "go", category: "language", versionArgs: []string{"version"}},
	{name: "rustc", category: "language", versionArgs: []string{"--version"}},
	{name: "python3", category: "language", versionArgs: []string{"--version"}},
	{name: "node", category: "language", versionArgs: []string{"--version"}},

	// Package managers
	{name: "pnpm", category: "pkgmgr", versionArgs: []string{"--version"}},
	{name: "pipx", category: "pkgmgr", versionArgs: []string{"--version"}},
	{name: "cargo", category: "pkgmgr", versionArgs: []string{"--version"}},
	{name: "npm", category: "pkgmgr", versionArgs: []string{"--version"}, advisory: true, warnIfPresent: true},

	// Containers
	{name: "docker", category: "container", versionArgs: []string{"--version"}},
	{name: "podman", category: "container", versionArgs: []string{"--version"}, advisory: true},

	// VCS & security
	{name: "git", category: "vcs", versionArgs: []string{"--version"}},
	{name: "gpg", category: "security", versionArgs: []string{"--version"}},
	{name: "ssh", category: "security", versionArgs: []string{"-V"}},
	{name: "age", category: "security", versionArgs: []string{"--version"}, advisory: true},

	// Encryption (Linux only; harmless to probe on others — just shows missing)
	{name: "cryptsetup", category: "encryption", versionArgs: []string{"--version"}, advisory: true},

	// Editors
	{name: "nvim", category: "editor", versionArgs: []string{"--version"}, advisory: true},
	{name: "code", category: "editor", versionArgs: []string{"--version"}, advisory: true},

	// Shell
	{name: "zsh", category: "shell", versionArgs: []string{"--version"}, advisory: true},
}

func detectTools() []Tool {
	out := make([]Tool, 0, len(canonicalProbes))
	for _, p := range canonicalProbes {
		t := Tool{
			Name:     p.name,
			Category: p.category,
			Advisory: p.advisory,
		}
		if !xexec.Check(p.name) {
			out = append(out, t)
			continue
		}
		t.Installed = true
		if p.warnIfPresent {
			t.Warn = true
		}
		if len(p.versionArgs) > 0 {
			r := xexec.Run(p.name, p.versionArgs...)
			if r.Err == nil || len(r.Output) > 0 {
				t.Version = compactVersion(r.LastLine())
			}
		}
		out = append(out, t)
	}
	return out
}

// compactVersion squeezes long version strings into a single short line.
// Example: "go version go1.22.0 darwin/arm64" → "go1.22.0 darwin/arm64".
func compactVersion(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Drop common leading noise like "Python ", "rustc ", "go version ".
	for _, prefix := range []string{"go version ", "rustc ", "Python ", "git version ", "gpg (GnuPG) ", "OpenSSH_"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			if prefix == "OpenSSH_" {
				s = "OpenSSH_" + s
			}
			break
		}
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	const maxLen = 48
	if len(s) > maxLen {
		s = s[:maxLen] + "…"
	}
	return s
}
