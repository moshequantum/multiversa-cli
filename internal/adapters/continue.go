package adapters

import (
	"os"
	"path/filepath"
)

type Continue struct{}

func (Continue) ID() string          { return "continue" }
func (Continue) DisplayName() string { return "Continue.dev" }

func (Continue) Detect() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, ".continue"))
	return err == nil
}

func (c Continue) Connect(opts ConnectOptions) error { return ErrNotImplemented }
func (c Continue) Disconnect() error                 { return ErrNotImplemented }
