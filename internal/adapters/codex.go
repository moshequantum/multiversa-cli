package adapters

import (
	"os"
	"path/filepath"
)

type Codex struct{}

func (Codex) ID() string          { return "codex" }
func (Codex) DisplayName() string { return "Codex CLI" }

func (Codex) Detect() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, ".codex"))
	return err == nil
}

func (c Codex) Connect(opts ConnectOptions) error { return ErrNotImplemented }
func (c Codex) Disconnect() error                 { return ErrNotImplemented }
