// Package credits is the single source of truth for upstream attribution.
// CREDITS.md, the wizard's install step, and `multiversa credits` all read
// from Sources here.
package credits

import (
	"fmt"
	"io"
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type Source struct {
	ID       string
	Name     string
	Author   string
	Repo     string
	License  string
	Role     string
	OptIn    bool
	AgplGate bool // true if license is AGPL and must never be embedded
}

var Sources = []Source{
	{
		ID:      "engram",
		Name:    "Engram",
		Author:  "Gentleman-Programming",
		Repo:    "https://github.com/Gentleman-Programming/engram",
		License: "MIT",
		Role:    "persistent agent memory",
	},
	{
		ID:      "graphify",
		Name:    "Graphify",
		Author:  "Safi Shamsi",
		Repo:    "https://github.com/safishamsi/graphify",
		License: "MIT",
		Role:    "content → knowledge graph",
	},
	{
		ID:      "gentle-ai",
		Name:    "gentle-ai",
		Author:  "Gentleman-Programming",
		Repo:    "https://github.com/Gentleman-Programming/gentle-ai",
		License: "MIT",
		Role:    "agentic ecosystem framework (memory + SDD + skills + MCP)",
	},
	{
		ID:      "gentle-pi",
		Name:    "gentle-pi",
		Author:  "Gentleman-Programming",
		Repo:    "https://github.com/Gentleman-Programming/gentle-pi",
		License: "MIT",
		Role:    "SDD harness for the Pi agent",
	},
	{
		ID:      "codegraph",
		Name:    "codegraph",
		Author:  "Colby McHenry",
		Repo:    "https://github.com/colbymchenry/codegraph",
		License: "MIT",
		Role:    "semantic code knowledge graph",
		OptIn:   true,
	},
	{
		ID:       "mirofish",
		Name:     "MiroFish",
		Author:   "666ghj",
		Repo:     "https://github.com/666ghj/MiroFish",
		License:  "AGPL-3.0",
		Role:     "swarm-intelligence simulation engine",
		OptIn:    true,
		AgplGate: true,
	},
}

func ByID(id string) (Source, bool) {
	for _, s := range Sources {
		if s.ID == id {
			return s, true
		}
	}
	return Source{}, false
}

// Print writes the canonical attribution footer to w.
func Print(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, theme.Accent.Render("Multiversa Lab orchestrates the following open-source stack:"))
	fmt.Fprintln(w)
	for _, s := range Sources {
		tag := ""
		switch {
		case s.AgplGate:
			tag = "  " + theme.Warn.Render("[AGPL-3.0 · external only]")
		case s.OptIn:
			tag = "  " + theme.Dim.Render("[opt-in]")
		}
		fmt.Fprintf(w, "  %s  %s  %s%s\n",
			theme.Accent.Render(pad(s.Name, 14)),
			theme.Body.Render(pad(s.Author, 26)),
			theme.Dim.Render(s.Repo),
			tag,
		)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, theme.Dim.Render("  Curation, design system, and ethics: Moshe — Multiversa Lab / Group."))
	fmt.Fprintln(w, theme.Dim.Render("  Full attribution: https://github.com/moshequantum/multiversa-cli/blob/main/CREDITS.md"))
	fmt.Fprintln(w)
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
