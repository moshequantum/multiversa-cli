package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/moshequantum/multiversa-cli/internal/embedded"
)

// readEmbeddedScript is a thin pass-through used by --show flags.
func readEmbeddedScript(name string) ([]byte, error) {
	return embedded.Script(name)
}

// runEmbeddedScript materializes one of the bash scripts shipped
// inside the binary to a private temp file, executes it via bash,
// and streams stdin/stdout/stderr through. The temp file is removed
// after the script exits.
//
// We write to disk (instead of `bash -s` via stdin) because the
// scripts themselves use `read -r` to prompt the user — if stdin
// were the script source, those prompts would hang.
func runEmbeddedScript(name string) error {
	data, err := embedded.Script(name)
	if err != nil {
		return err
	}

	dir, err := os.MkdirTemp("", "multiversa-script-*")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o700); err != nil {
		return fmt.Errorf("write temp script: %w", err)
	}

	c := exec.Command("bash", path)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
