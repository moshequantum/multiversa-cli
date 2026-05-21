package adapters

import (
	"os/exec"
)

type Cursor struct{}

func (Cursor) ID() string          { return "cursor" }
func (Cursor) DisplayName() string { return "Cursor" }

func (Cursor) Detect() bool {
	_, err := exec.LookPath("cursor")
	return err == nil
}

func (c Cursor) Connect(opts ConnectOptions) error {
	// TODO: write .cursorrules and ~/.cursor/mcp.json
	return ErrNotImplemented
}

func (c Cursor) Disconnect() error { return ErrNotImplemented }
