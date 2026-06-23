package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/detect"
)

func TestStackDryRunPrintsPlans(t *testing.T) {
	var buf bytes.Buffer
	err := runStack(stackOpts{dryRun: true, out: &buf})
	if err != nil {
		t.Fatalf("runStack(--dry-run) returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "multiversa stack") {
		t.Errorf("expected header in dry-run output; got:\n%s", out)
	}
	if !strings.Contains(out, "Dry run") {
		t.Errorf("expected dry-run sentinel in output; got:\n%s", out)
	}
}

func TestStackOnlyFiltersInNonTTY(t *testing.T) {
	var buf bytes.Buffer
	err := runStack(stackOpts{dryRun: true, only: []string{"docker"}, out: &buf})
	if err != nil {
		t.Fatalf("runStack(--only=docker) returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(strings.ToLower(out), "docker") {
		t.Errorf("expected docker row in --only=docker output; got:\n%s", out)
	}
	for _, id := range []string{"rust", "python", "node", "pnpm"} {
		padded := lipglossPad(id, 10)
		if strings.Contains(out, padded) {
			t.Errorf("expected %q to be filtered out by --only=docker; got:\n%s", id, out)
		}
	}
}

func TestPlanStackBaseline(t *testing.T) {
	planned, report := planStack(stackOpts{})
	if len(planned) == 0 {
		t.Error("expected planStack to return some tools from the registry")
	}
	if report.OS.Kind == "" {
		t.Error("expected detect.Report to have a non-empty OS kind")
	}
	_ = detect.Report{}
}
