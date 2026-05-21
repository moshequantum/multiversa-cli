package adapters

import (
	"os"
	"path/filepath"
)

type ClaudeCode struct{}

func (ClaudeCode) ID() string          { return "claude-code" }
func (ClaudeCode) DisplayName() string { return "Claude Code" }

func (ClaudeCode) Detect() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".claude"))
	return err == nil
}

func (c ClaudeCode) Connect(opts ConnectOptions) error {
	// TODO: write/merge ~/.claude/settings.json with MCP server entries pointing
	// to engram, graphify, gentle-ai. Append a CLAUDE.md note explaining the stack.
	return ErrNotImplemented
}

func (c ClaudeCode) Disconnect() error { return ErrNotImplemented }
