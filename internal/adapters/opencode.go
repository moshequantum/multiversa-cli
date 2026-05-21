package adapters

import (
	"os/exec"
)

type OpenCode struct{}

func (OpenCode) ID() string          { return "opencode" }
func (OpenCode) DisplayName() string { return "OpenCode" }

func (OpenCode) Detect() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

func (o OpenCode) Connect(opts ConnectOptions) error { return ErrNotImplemented }
func (o OpenCode) Disconnect() error                 { return ErrNotImplemented }
