package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/detect"
)

// TestWorkspaceShowPrintsScript verifies that --show dumps the embedded
// bash script to the writer. We look for 'ssh-keygen' as a marker.
func TestWorkspaceShowPrintsScript(t *testing.T) {
	var buf bytes.Buffer
	err := runWorkspace(workspaceOpts{showOnly: true, out: &buf})
	if err != nil {
		t.Fatalf("runWorkspace(--show) returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ssh-keygen") {
		t.Fatalf("expected --show output to contain 'ssh-keygen', got:\n%s", out)
	}
}

// TestWorkspacePrereqMissingNonInteractive checks that missing
// prerequisites cause a non-nil error and a diagnostic message.
func TestWorkspacePrereqMissingNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	report := detect.Report{Tools: []detect.Tool{
		{Name: "go", Installed: true},
	}}
	missing := detect.RequiredMissing(report, []string{"git", "ssh"})
	if len(missing) != 2 {
		t.Fatalf("setup error: expected 2 missing tools, got %d", len(missing))
	}
	err := runWorkspaceNonInteractive(&buf, report, missing)
	if err == nil {
		t.Fatalf("expected prereq-missing error, got nil")
	}
	if !strings.Contains(buf.String(), "Prerrequisitos faltantes") {
		t.Fatalf("expected 'Prerrequisitos faltantes' in output, got:\n%s", buf.String())
	}
}
