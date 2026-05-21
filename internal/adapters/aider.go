package adapters

import "os/exec"

type Aider struct{}

func (Aider) ID() string                          { return "aider" }
func (Aider) DisplayName() string                 { return "Aider" }
func (Aider) Detect() bool                        { _, err := exec.LookPath("aider"); return err == nil }
func (Aider) Connect(opts ConnectOptions) error   { return ErrNotImplemented }
func (Aider) Disconnect() error                   { return ErrNotImplemented }
