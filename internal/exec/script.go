package exec

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
)

// RunEmbeddedScript materializes script bytes to a private temp dir,
// executes it via bash with stdin/stdout/stderr passed through, then
// removes the temp dir. We write to disk rather than using bash -s
// so scripts can call `read -r` for interactive prompts.
func RunEmbeddedScript(data []byte) error {
	dir, err := os.MkdirTemp("", "multiversa-script-*")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		return fmt.Errorf("write temp script: %w", err)
	}

	c := osexec.Command("bash", path)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
