package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadinessDoesNotCountBlockedArtifacts(t *testing.T) {
	r := Report{
		Tools: []Tool{
			{Name: "go", Installed: true, State: "blocked"},
			{Name: "git", Installed: true, State: "installed"},
		},
		Multiversa: MultiversaState{Engines: []EngineState{
			{ID: "gentle-pi", Installed: true, State: "blocked"},
			{ID: "engram", Installed: true, State: "configured"},
		}},
	}
	if ready, total := r.ReadyTools(); ready != 1 || total != 2 {
		t.Fatalf("tools ready = %d/%d", ready, total)
	}
	if ready, total := r.ReadyEngines(); ready != 1 || total != 2 {
		t.Fatalf("engines ready = %d/%d", ready, total)
	}
}

func TestYAMLHasNestedKeyReadsNamesWithoutValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "model:\n  provider: secret-provider\nmcp_servers:\n  other:\n    url: https://example.test\n  multiversa:\n    command: /safe/bin/multiversa\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if !yamlHasNestedKey(path, "mcp_servers", "multiversa") {
		t.Fatal("expected mcp_servers.multiversa")
	}
	if yamlHasNestedKey(path, "mcp_servers", "missing") {
		t.Fatal("unexpected missing key")
	}
}
