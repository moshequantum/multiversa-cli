package adapters

import (
	"bytes"
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Hermes connects Nous Research's optional Hermes Agent runtime to the
// read-only Multiversa MCP server. The user must explicitly run
// `multiversa connect hermes`; detection alone never changes Hermes config.
type Hermes struct{}

func (Hermes) ID() string          { return "hermes" }
func (Hermes) DisplayName() string { return "Hermes Agent" }

func (Hermes) Detect() bool {
	if _, err := osexec.LookPath("hermes"); err == nil {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	fi, err := os.Stat(filepath.Join(home, ".hermes", "hermes-agent"))
	return err == nil && fi.IsDir()
}

func (Hermes) Connect(opts ConnectOptions) error {
	if _, err := osexec.LookPath("hermes"); err != nil {
		return fmt.Errorf("Hermes Agent no está disponible en PATH")
	}
	exe, err := os.Executable()
	if err != nil || exe == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("resolver binario Multiversa: %w", err)
		}
		exe = filepath.Join(home, ".local", "bin", "multiversa")
	}
	if err := runHermes("mcp", "add", "multiversa", "--command", exe, "--args", "mcp", "serve"); err != nil {
		return fmt.Errorf("hermes mcp add: %w", err)
	}
	return nil
}

func (Hermes) Disconnect() error {
	if _, err := osexec.LookPath("hermes"); err != nil {
		return nil
	}
	if err := runHermes("mcp", "remove", "multiversa"); err != nil {
		return fmt.Errorf("hermes mcp remove: %w", err)
	}
	return nil
}

func runHermes(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := osexec.CommandContext(ctx, "hermes", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		last := ""
		if len(lines) > 0 {
			last = lines[len(lines)-1]
		}
		return fmt.Errorf("%w: %s", err, last)
	}
	return nil
}
