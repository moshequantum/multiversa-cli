package adapters

import (
	"os/exec"
)

type GeminiCLI struct{}

func (GeminiCLI) ID() string          { return "gemini-cli" }
func (GeminiCLI) DisplayName() string { return "Gemini CLI" }

func (GeminiCLI) Detect() bool {
	_, err := exec.LookPath("gemini")
	return err == nil
}

func (g GeminiCLI) Connect(opts ConnectOptions) error { return ErrNotImplemented }
func (g GeminiCLI) Disconnect() error                 { return ErrNotImplemented }
