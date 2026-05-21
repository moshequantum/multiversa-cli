// Package adapters wires the installed engine stack to a specific agent's
// configuration files (settings.json, .cursorrules, mcp.json, etc.).
// Each adapter is isolated — adding a new agent means adding one file.
package adapters

import (
	"errors"
	"fmt"
)

var ErrNotImplemented = errors.New("adapter not implemented yet")

// ConnectOptions tells the adapter what to wire into the agent's config.
type ConnectOptions struct {
	EnabledEngines []string
	Backend        string
	Manifest       string // path to multiversa.toml
}

type Adapter interface {
	ID() string
	DisplayName() string
	Detect() bool
	Connect(opts ConnectOptions) error
	Disconnect() error
}

func Registry() map[string]Adapter {
	return map[string]Adapter{
		"claude-code":  &ClaudeCode{},
		"cursor":       &Cursor{},
		"codex":        &Codex{},
		"gemini-cli":   &GeminiCLI{},
		"opencode":     &OpenCode{},
		"aider":        &Aider{},
		"cline":        &Cline{},
		"continue":     &Continue{},
		"roo-code":     &RooCode{},
		"generic-mcp":  &GenericMCP{},
	}
}

func List() []Adapter {
	order := []string{"claude-code", "cursor", "codex", "gemini-cli", "opencode", "aider", "cline", "continue", "roo-code", "generic-mcp"}
	reg := Registry()
	out := make([]Adapter, 0, len(order))
	for _, id := range order {
		out = append(out, reg[id])
	}
	return out
}

func Resolve(id string) (Adapter, error) {
	if a, ok := Registry()[id]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("unknown agent %q", id)
}

// DetectInstalled returns the IDs of agents that look installed on this machine.
func DetectInstalled() []string {
	var out []string
	for _, a := range List() {
		if a.Detect() {
			out = append(out, a.ID())
		}
	}
	return out
}
