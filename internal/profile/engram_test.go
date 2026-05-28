package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// withEmptyPATH replaces PATH with the test temp dir so the engram
// binary cannot be found regardless of the host. Used to assert the
// "engram missing → no-op" contract.
func withEmptyPATH(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

func TestMirrorEngram_NoOpWhenEngramMissing(t *testing.T) {
	withEmptyPATH(t)
	err := MirrorEngram(Profile{Level: Expert, Locale: "en"})
	if err != nil {
		t.Errorf("MirrorEngram should be a no-op without engram on PATH, got %v", err)
	}
}

func TestEngramAvailable_FalseOnEmptyPATH(t *testing.T) {
	withEmptyPATH(t)
	if engramAvailable() {
		t.Error("engramAvailable() should be false when PATH lacks engram")
	}
}

func TestEngramAvailable_TrueWhenFakeBinaryOnPATH(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "engram")
	// A zero-length executable is enough — LookPath only checks
	// existence + the execute bit.
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("create fake engram: %v", err)
	}
	t.Setenv("PATH", dir)
	if !engramAvailable() {
		t.Error("engramAvailable() should detect fake engram on PATH")
	}
}
