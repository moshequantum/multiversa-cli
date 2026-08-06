package detect

import (
	"bufio"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/capability"
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

// AgentState is the evidence-only view of an optional agent runtime. Agent
// runtimes are deliberately separate from stack engines and skills.
type AgentState struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Author           string           `json:"author"`
	Repo             string           `json:"repo"`
	License          string           `json:"license"`
	OptIn            bool             `json:"opt_in"`
	State            capability.State `json:"state"`
	Installed        bool             `json:"installed"`
	Configured       bool             `json:"configured"`
	Connected        bool             `json:"connected"`
	Version          string           `json:"version,omitempty"`
	Path             string           `json:"path,omitempty"`
	Evidence         []string         `json:"evidence,omitempty"`
	Missing          []string         `json:"missing,omitempty"`
	NextActions      []string         `json:"next_actions,omitempty"`
	RequiresApproval bool             `json:"requires_approval"`
}

func detectAgents() []AgentState {
	return []AgentState{detectHermes()}
}

func detectHermes() AgentState {
	a := AgentState{
		ID:      "hermes",
		Name:    "Hermes Agent",
		Author:  "Nous Research",
		Repo:    "https://github.com/NousResearch/hermes-agent",
		License: "MIT",
		OptIn:   true,
		State:   capability.Absent,
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return a
	}
	hermesHome := filepath.Join(home, ".hermes")
	installDir := filepath.Join(hermesHome, "hermes-agent")
	configPath := filepath.Join(hermesHome, "config.yaml")

	if p, err := osexec.LookPath("hermes"); err == nil {
		a.Installed = true
		a.Path = p
		a.State = capability.Installed
		a.Evidence = append(a.Evidence, "binary:"+p)
		if r := xexec.Run("hermes", "--version"); len(r.Output) > 0 {
			a.Version = compactVersion(r.Output[0])
		}
	} else if dirExists(installDir) {
		a.State = capability.Blocked
		a.Evidence = append(a.Evidence, "source:"+installDir)
		a.Missing = append(a.Missing, "binary:hermes-in-path")
		a.NextActions = append(a.NextActions, "repair-hermes-path")
		a.RequiresApproval = true
	}

	if fileExists(configPath) {
		a.Configured = true
		a.Evidence = append(a.Evidence, "config:"+configPath)
		if a.Installed {
			a.State = capability.Configured
		}
	}
	if yamlHasNestedKey(configPath, "mcp_servers", "multiversa") {
		a.Connected = true
		a.Evidence = append(a.Evidence, "mcp:multiversa")
		if a.Installed {
			a.State = capability.Connected
		}
	}

	bridgeSkill := filepath.Join(hermesHome, "skills", "autonomous-ai-agents", "multiversa-dispatch", "SKILL.md")
	if fileExists(bridgeSkill) {
		a.Evidence = append(a.Evidence, "skill:multiversa-dispatch")
	}
	return a
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// yamlHasNestedKey inspects key names only; values (which may contain secrets)
// are never returned or serialized. It handles the mapping form Hermes writes.
func yamlHasNestedKey(path, parent, child string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	inParent := false
	parentIndent := -1
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		raw := sc.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		key := strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[0])
		if !inParent {
			if key == parent && strings.Contains(trimmed, ":") {
				inParent = true
				parentIndent = indent
			}
			continue
		}
		if indent <= parentIndent {
			return false
		}
		if key == child && strings.Contains(trimmed, ":") {
			return true
		}
	}
	return false
}
