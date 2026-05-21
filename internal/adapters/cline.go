package adapters

import "os/exec"

type Cline struct{}

func (Cline) ID() string                          { return "cline" }
func (Cline) DisplayName() string                 { return "Cline" }
func (Cline) Detect() bool                        { _, err := exec.LookPath("cline"); return err == nil }
func (Cline) Connect(opts ConnectOptions) error   { return ErrNotImplemented }
func (Cline) Disconnect() error                   { return ErrNotImplemented }
