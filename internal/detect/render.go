package detect

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/moshequantum/multiversa-cli/internal/theme"
)

const (
	glyphOK   = "✓"
	glyphMiss = "·"
	glyphWarn = "⚠"
)

// Render writes the report to w using the Multiversa theme.
// The output is meant to be copy-pasteable; no ANSI hyperlinks, only colors.
func (r Report) Render(w io.Writer) {
	var b strings.Builder

	// Header
	b.WriteString(theme.Accent.Render("multiversa detect"))
	b.WriteString("\n")
	b.WriteString(theme.Dim.Render(`"La IA propone, tú decides."`))
	b.WriteString("\n\n")

	// OS block
	b.WriteString(theme.Label.Render("OS"))
	b.WriteString("\n")
	b.WriteString(kv("kind", fmt.Sprintf("%s/%s", r.OS.Kind, r.OS.Arch)))
	if r.OS.Distro != "" {
		b.WriteString(kv("distro", r.OS.Distro))
	}
	if r.OS.Version != "" {
		b.WriteString(kv("version", r.OS.Version))
	}
	pkgMgr := r.OS.PkgMgr
	if pkgMgr == "" {
		pkgMgr = theme.Warn.Render("none detected")
	}
	b.WriteString(kv("pkg mgr", pkgMgr))
	b.WriteString("\n")

	// Tools, grouped by category in declaration order.
	b.WriteString(theme.Label.Render("Dev stack"))
	b.WriteString("\n")
	for _, t := range r.Tools {
		b.WriteString(renderTool(t))
	}
	b.WriteString("\n")

	// Multiversa block.
	b.WriteString(theme.Label.Render("Multiversa"))
	b.WriteString("\n")

	cliStatus := theme.Dim.Render(glyphMiss + " not in PATH")
	if r.Multiversa.CLIInstalled {
		v := r.Multiversa.CLIVersion
		if v == "" {
			v = "installed"
		}
		cliStatus = theme.Accent.Render(glyphOK+" ") + theme.Body.Render(v)
	}
	b.WriteString(kv("cli", cliStatus))

	if r.Multiversa.HomeDir != "" {
		b.WriteString(kv("home", theme.Body.Render(r.Multiversa.HomeDir)))
	} else {
		b.WriteString(kv("home", theme.Dim.Render(glyphMiss+" ~/.multiversa not present")))
	}

	for _, e := range r.Multiversa.Engines {
		var status string
		switch {
		case e.Installed && e.Version != "":
			status = theme.Accent.Render(glyphOK+" ") + theme.Body.Render(e.Version)
		case e.Installed:
			status = theme.Accent.Render(glyphOK+" ") + theme.Body.Render("installed")
		case e.OptIn:
			status = theme.Dim.Render(glyphMiss + " opt-in, not installed")
		default:
			status = theme.Dim.Render(glyphMiss + " not installed")
		}
		b.WriteString(kv(e.ID, status))
	}

	if len(r.Multiversa.Repos) > 0 {
		b.WriteString("\n")
		b.WriteString(theme.Label.Render("Repos"))
		b.WriteString("\n")
		for _, p := range r.Multiversa.Repos {
			b.WriteString(kv("·", theme.Body.Render(p)))
		}
	}

	// Summary footer.
	b.WriteString("\n")
	ti, tt := r.ReadyTools()
	ei, et := r.ReadyEngines()
	summary := fmt.Sprintf("Ready: %d/%d tools · %d/%d engines", ti, tt, ei, et)
	b.WriteString(theme.Dim.Render(summary))
	b.WriteString("\n")

	// Policy nudge.
	if hasNpmWarning(r.Tools) {
		b.WriteString("\n")
		b.WriteString(theme.Warn.Render(glyphWarn + " npm present — Multiversa policy is pnpm-only."))
		b.WriteString("\n")
	}

	fmt.Fprint(w, b.String())
}

func renderTool(t Tool) string {
	var glyph, ver string
	switch {
	case t.Warn && t.Installed:
		glyph = theme.Warn.Render(glyphWarn)
		ver = theme.Warn.Render(t.Version)
	case t.Installed:
		glyph = theme.Accent.Render(glyphOK)
		ver = theme.Body.Render(t.Version)
	case t.Advisory:
		glyph = theme.Dim.Render(glyphMiss)
		ver = theme.Dim.Render("optional")
	default:
		glyph = theme.Dim.Render(glyphMiss)
		ver = theme.Dim.Render("missing")
	}
	return kv(t.Name, glyph+" "+ver)
}

// kv renders one indented key/value line with a 14-col left pad on the key.
func kv(key, value string) string {
	keyStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7A7A82")).
		Width(14).
		Render(key)
	return "  " + keyStyled + value + "\n"
}

func hasNpmWarning(tools []Tool) bool {
	for _, t := range tools {
		if t.Name == "npm" && t.Installed {
			return true
		}
	}
	return false
}
