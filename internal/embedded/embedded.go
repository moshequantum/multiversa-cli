// Package embedded ships the platform-specific bash scripts inside the
// `multiversa` binary. They are sourced from the lab-setup Claude Code
// skill (~/.claude/skills/lab-setup/scripts/) at build time and stay
// in lock-step with that skill — see Makefile target `sync-scripts`
// for the refresh procedure.
//
// Shipping the scripts inside the binary lets `multiversa workspace`
// and `multiversa usb` work on a freshly installed machine that has
// only the brew/apt/winget binary and no Claude Code skill checkout.
package embedded

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed scripts/*.sh
var scripts embed.FS

// Script returns the raw bytes of the named embedded script.
// `name` is the filename without any path (e.g. "setup_multiversa.sh").
func Script(name string) ([]byte, error) {
	data, err := scripts.ReadFile("scripts/" + name)
	if err != nil {
		return nil, fmt.Errorf("embedded script %q not found: %w", name, err)
	}
	return data, nil
}

// List returns every embedded script filename, for debug/credit output.
func List() ([]string, error) {
	entries, err := fs.ReadDir(scripts, "scripts")
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}
