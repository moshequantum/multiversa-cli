package tenant

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/graphify"
)

type fakeBootstrapRunner struct {
	calls      []graphify.Command
	fail       string
	failModels map[string]bool
}

func (f *fakeBootstrapRunner) LookPath(name string) (string, error) { return name, nil }

func (f *fakeBootstrapRunner) Run(_ context.Context, cmd graphify.Command) graphify.Result {
	f.calls = append(f.calls, cmd)
	if len(cmd.Args) > 0 && cmd.Args[0] == f.fail {
		return graphify.Result{Stderr: "simulated failure", ExitCode: 1, Err: os.ErrInvalid}
	}
	if len(cmd.Args) > 0 && cmd.Args[0] == "--version" {
		return graphify.Result{Stdout: "graphify 0.9.21\n"}
	}
	if len(cmd.Args) > 0 && cmd.Args[0] == "add" {
		var target string
		for i := range cmd.Args {
			if cmd.Args[i] == "--dir" && i+1 < len(cmd.Args) {
				target = cmd.Args[i+1]
			}
		}
		_ = os.MkdirAll(target, 0o700)
		name := filepath.Join(target, strings.ReplaceAll(cmd.Args[1], "/", "_")+".md")
		_ = os.WriteFile(name, []byte("source"), 0o600)
	}
	if len(cmd.Args) > 0 && cmd.Args[0] == "extract" {
		for i := range cmd.Args {
			if cmd.Args[i] == "--model" && i+1 < len(cmd.Args) && f.failModels[cmd.Args[i+1]] {
				return graphify.Result{Stderr: "provider unavailable", ExitCode: 1, Err: os.ErrInvalid}
			}
		}
		graphDir := cmd.Args[3]
		out := filepath.Join(graphDir, "graphify-out")
		_ = os.MkdirAll(out, 0o700)
		_ = os.WriteFile(filepath.Join(out, "graph.json"), []byte(`{"nodes":[{"id":"identity"}],"links":[]}`), 0o600)
	}
	return graphify.Result{Stdout: "ok"}
}

func TestBootstrapUsesProviderFallbackAndRestoresClientEnv(t *testing.T) {
	withTempHome(t)
	if _, _, err := New("fallback-os", "Fallback OS", "project-os"); err != nil {
		t.Fatal(err)
	}
	for i, cfg := range []ProviderConfig{
		{Provider: "mistral", Model: "fail-model", Enabled: true, Priority: 1},
		{Provider: "groq", Model: "working-model", Enabled: true, Priority: 2},
	} {
		cfg.Priority = i + 1
		if err := ConnectProvider("fallback-os", cfg); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := SetSecret("fallback-os", "MISTRAL_API_KEY", "m-secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetSecret("fallback-os", "GROQ_API_KEY", "g-secret"); err != nil {
		t.Fatal(err)
	}

	runner := &fakeBootstrapRunner{failModels: map[string]bool{"fail-model": true}}
	client := graphify.New(runner)
	client.Env = []string{"OPENAI_API_KEY=host-must-not-win", "ANTHROPIC_API_KEY=host-too", "KEEP=yes"}
	got, err := Bootstrap(context.Background(), "fallback-os", BootstrapOptions{}, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ProviderAttempts) != 2 || got.ProviderAttempts[0].Success ||
		got.ProviderAttempts[0].ErrorCode != "extract_failed" || !got.ProviderAttempts[1].Success {
		t.Fatalf("attempts not redacted/ordered: %#v", got.ProviderAttempts)
	}
	if !reflect.DeepEqual(client.Env, []string{"OPENAI_API_KEY=host-must-not-win", "ANTHROPIC_API_KEY=host-too", "KEEP=yes"}) {
		t.Fatalf("client env not restored: %#v", client.Env)
	}
	var extracts []graphify.Command
	for _, call := range runner.calls {
		if len(call.Args) > 0 && call.Args[0] == "extract" {
			extracts = append(extracts, call)
		}
	}
	if len(extracts) != 2 {
		t.Fatalf("extract calls=%d: %#v", len(extracts), runner.calls)
	}
	firstEnv := strings.Join(extracts[0].Env, "\n")
	secondEnv := strings.Join(extracts[1].Env, "\n")
	if !strings.Contains(firstEnv, "OPENAI_API_KEY=m-secret") || !strings.Contains(firstEnv, "OPENAI_BASE_URL=https://api.mistral.ai/v1") {
		t.Fatalf("mistral env=%q", firstEnv)
	}
	if !strings.Contains(firstEnv, "ANTHROPIC_API_KEY=") {
		t.Fatalf("unrelated host LLM key was not explicitly cleared: %q", firstEnv)
	}
	if !strings.Contains(secondEnv, "OPENAI_API_KEY=g-secret") || !strings.Contains(secondEnv, "OPENAI_BASE_URL=https://api.groq.com/openai/v1") {
		t.Fatalf("groq env=%q", secondEnv)
	}
}

func TestBootstrapCreatesPrivateCorpusAndRunsGraphify(t *testing.T) {
	home := withTempHome(t)
	runner := &fakeBootstrapRunner{}
	got, err := Bootstrap(context.Background(), "cintia-os", BootstrapOptions{
		Name: "Cintia OS", Owner: "Cintia Larizzati", OSName: "CintiaOS",
		Route: "group", Engagement: "consulting", Activate: true,
		Sources: []BootstrapSource{
			{URL: "HTTPS://Example.com/b#bio", Author: "Cintia"},
			{URL: "https://example.com/a", Contributor: "Moshe"},
		},
	}, graphify.New(runner))
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if !got.Created || got.Resumed || !got.Activated || got.Added != 2 {
		t.Fatalf("unexpected result: %+v", got)
	}
	dir := filepath.Join(home, ".multiversa", "tenants", "cintia-os")
	for _, rel := range []string{"", "graph", "memory", "graph/corpus", "graph/corpus/raw", "graph/corpus/provenance"} {
		fi, err := os.Stat(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("stat %s: %v", rel, err)
		}
		if fi.Mode().Perm() != 0o700 {
			t.Errorf("%s perm = %o, want 0700", rel, fi.Mode().Perm())
		}
	}
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		want := os.FileMode(0o600)
		if entry.IsDir() {
			want = 0o700
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s perm = %o, want %o", path, got, want)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	identity, err := os.ReadFile(filepath.Join(dir, "graph/corpus/raw/identity.md"))
	if err != nil || !strings.Contains(string(identity), "Cintia Larizzati") {
		t.Fatalf("identity missing owner: %v %s", err, identity)
	}
	if len(runner.calls) != 4 || runner.calls[0].Args[0] != "--version" ||
		runner.calls[1].Args[0] != "add" || runner.calls[2].Args[0] != "add" ||
		runner.calls[3].Args[0] != "extract" {
		t.Fatalf("unexpected calls: %+v", runner.calls)
	}
	if runner.calls[1].Args[1] != "https://example.com/a" ||
		runner.calls[2].Args[1] != "https://example.com/b" {
		t.Fatalf("sources not canonical and sorted: %+v", runner.calls)
	}
	if Active() != "cintia-os" {
		t.Fatalf("active = %q", Active())
	}
}

func TestBootstrapResumeIsIdempotentAndDoesNotOverwriteManifest(t *testing.T) {
	withTempHome(t)
	first := &fakeBootstrapRunner{}
	opts := BootstrapOptions{
		Name: "Original", Owner: "Owner",
		Sources: []BootstrapSource{{URL: "https://example.com/profile"}},
	}
	if _, err := Bootstrap(context.Background(), "resume-os", opts, graphify.New(first)); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(first.calls[1].Dir, "graph/corpus/provenance/sources.jsonl")
	before, _ := os.ReadFile(sourcePath)

	second := &fakeBootstrapRunner{}
	opts.Name = "Must Not Replace"
	got, err := Bootstrap(context.Background(), "resume-os", opts, graphify.New(second))
	if err != nil {
		t.Fatal(err)
	}
	if got.Created || !got.Resumed || got.Added != 0 || got.Skipped != 1 {
		t.Fatalf("unexpected resumed result: %+v", got)
	}
	if len(second.calls) != 2 || second.calls[0].Args[0] != "--version" || second.calls[1].Args[0] != "extract" {
		t.Fatalf("resume should only extract: %+v", second.calls)
	}
	after, _ := os.ReadFile(sourcePath)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("deterministic ledger changed:\nbefore %s\nafter %s", before, after)
	}
	m, _, err := Load("resume-os")
	if err != nil || m.Tenant.Name != "Original" {
		t.Fatalf("manifest overwritten: %v %+v", err, m)
	}
}

func TestBootstrapDryRunMakesNoChanges(t *testing.T) {
	home := withTempHome(t)
	got, err := Bootstrap(context.Background(), "planned-os", BootstrapOptions{
		DryRun: true, Activate: true, Sources: []BootstrapSource{{URL: "https://example.com"}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.DryRun || len(got.Plan) != 6 {
		t.Fatalf("unexpected plan: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".multiversa")); !os.IsNotExist(err) {
		t.Fatalf("dry run changed filesystem: %v", err)
	}
}

func TestBootstrapPersistsSuccessfulAddsForResume(t *testing.T) {
	withTempHome(t)
	runner := &fakeBootstrapRunner{fail: "add"}
	_, err := Bootstrap(context.Background(), "partial-os", BootstrapOptions{
		Sources: []BootstrapSource{{URL: "https://example.com/a"}},
	}, graphify.New(runner))
	if err == nil {
		t.Fatal("expected add failure")
	}
	ledger := filepath.Join(runner.calls[1].Dir, "graph/corpus/provenance/sources.jsonl")
	data, readErr := os.ReadFile(ledger)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		t.Fatalf("failed source must not be marked successful: %s", data)
	}
}

func TestNormalizeBootstrapSourcesRejectsUnsafeAndConflictingInput(t *testing.T) {
	if _, err := normalizeBootstrapSources([]BootstrapSource{{URL: "file:///etc/passwd"}}); err == nil {
		t.Fatal("expected non-http URL rejection")
	}
	if _, err := normalizeBootstrapSources([]BootstrapSource{
		{URL: "https://example.com/x#one", Author: "A"},
		{URL: "https://EXAMPLE.com/x#two", Author: "B"},
	}); err == nil {
		t.Fatal("expected conflicting metadata rejection")
	}
}

func TestValidateGraphRejectsMalformedGraph(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")
	for _, content := range []string{"", `{}`, `{"nodes":{}}`, `not-json`} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := graphify.Validate(path); err == nil {
			t.Fatalf("expected validation error for %q", content)
		}
	}
	if err := os.WriteFile(path, []byte(`{"nodes":[],"links":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := graphify.Validate(path); err != nil {
		t.Fatalf("valid graph rejected: %v", err)
	}
}

func TestSourceLedgerIsJSONLines(t *testing.T) {
	withTempHome(t)
	runner := &fakeBootstrapRunner{}
	got, err := Bootstrap(context.Background(), "jsonl-os", BootstrapOptions{
		Sources: []BootstrapSource{{URL: "https://example.com/?a=1&b=2"}},
	}, graphify.New(runner))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(got.SourceFile)
	if err != nil {
		t.Fatal(err)
	}
	var record sourceRecord
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &record); err != nil {
		t.Fatalf("invalid JSONL: %v", err)
	}
	if record.URL != "https://example.com/?a=1&b=2" {
		t.Fatalf("query changed: %q", record.URL)
	}
}
