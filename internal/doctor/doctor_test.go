package doctor

import (
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/capability"
	"github.com/moshequantum/multiversa-cli/internal/detect"
)

func TestAnalyzeFindsPriorityDriftWithoutApplyingIt(t *testing.T) {
	report := detect.Report{
		Tools: []detect.Tool{{Name: "go", Installed: true, State: capability.Blocked, Path: "/snap/bin/go", ProbeError: "permission denied"}},
		Multiversa: detect.MultiversaState{
			CLIPath: "/home/test/.local/bin/multiversa",
			CLIBinaries: []detect.BinaryState{
				{Path: "/home/test/.local/bin/multiversa", Version: "v0.8.0 dev", Active: true},
				{Path: "/usr/local/bin/multiversa", Version: "v0.8.0 release"},
			},
			Engines: []detect.EngineState{{
				ID: "gentle-pi", State: capability.Blocked, Installed: true,
				Missing: []string{"runtime:pi"}, Evidence: []string{"legacy-global-package:gentle-pi"},
			}},
		},
		Agents: []detect.AgentState{{
			ID: "hermes", Installed: true, Configured: true, State: capability.Configured,
			Evidence: []string{"binary:/home/test/.local/bin/hermes"},
		}},
	}
	tenants := []TenantState{{Slug: "rootfounder", VaultOK: true, GraphEngine: "graphify", GraphIndexed: false}}
	cron := CronState{Supported: true, Readable: true, Installed: true, Binary: "/usr/local/bin/multiversa", HasPATH: false}

	got := Analyze(report, tenants, cron)
	if got.State != capability.Blocked {
		t.Fatalf("state = %q, want blocked", got.State)
	}
	for _, id := range []string{
		"cli.duplicate-binaries", "tool.go.probe-blocked", "engine.gentle-pi.blocked",
		"agent.hermes.mcp-not-connected", "tenant.rootfounder.graph-not-indexed",
		"updates.cron-binary-drift", "updates.cron-path-incomplete",
	} {
		if !hasFinding(got.Findings, id) {
			t.Errorf("missing finding %q in %+v", id, got.Findings)
		}
	}
	if got.Summary.P1 != 1 {
		t.Fatalf("P1 count = %d, want 1", got.Summary.P1)
	}
}

func TestAnalyzeHealthyWhenNoEvidenceOfDrift(t *testing.T) {
	got := Analyze(detect.Report{}, nil, CronState{})
	if got.State != capability.Healthy || got.Summary.Open != 0 || len(got.Findings) != 0 {
		t.Fatalf("unexpected report: %+v", got)
	}
}

func TestParseCronEntryFindsExplicitPathAndBinary(t *testing.T) {
	line := "0 9 * * * PATH=/home/test/.local/bin:/usr/bin /home/test/.local/bin/multiversa updates --json # multiversa-updates"
	bin, hasPath := parseCronEntry(line)
	if bin != "/home/test/.local/bin/multiversa" || !hasPath {
		t.Fatalf("parseCronEntry = %q, %v", bin, hasPath)
	}
}

func hasFinding(findings []Finding, id string) bool {
	for _, f := range findings {
		if f.ID == id {
			return true
		}
	}
	return false
}
