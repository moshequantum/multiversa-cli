package graphify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	path     string
	lookErr  error
	commands []Command
	run      func(Command) Result
}

func (f *fakeRunner) LookPath(string) (string, error) { return f.path, f.lookErr }
func (f *fakeRunner) Run(_ context.Context, c Command) Result {
	f.commands = append(f.commands, c)
	if f.run != nil {
		return f.run(c)
	}
	return Result{}
}

func TestPreflight(t *testing.T) {
	f := &fakeRunner{path: "/bin/graphify", run: func(c Command) Result {
		return Result{Stdout: "graphify 0.9.21\n"}
	}}
	got, err := New(f).Preflight(context.Background())
	if err != nil || got != "graphify 0.9.21" {
		t.Fatalf("Preflight() = %q, %v", got, err)
	}
	if f.commands[0].Name != "/bin/graphify" {
		t.Fatalf("preflight did not use resolved binary: %#v", f.commands[0])
	}
}

func TestPreflightMissing(t *testing.T) {
	f := &fakeRunner{lookErr: errors.New("missing")}
	if _, err := New(f).Preflight(context.Background()); err == nil {
		t.Fatal("expected missing binary error")
	}
}

func TestAddBuildsSafeCommandAndFindsArtifact(t *testing.T) {
	dir := t.TempDir()
	f := &fakeRunner{run: func(c Command) Result {
		if err := os.WriteFile(filepath.Join(dir, "source.md"), []byte("ok"), 0o644); err != nil {
			t.Fatal(err)
		}
		return Result{}
	}}
	got, err := New(f).Add(context.Background(), AddOptions{
		URL: "https://example.com/a?x=$(bad)", TargetDir: dir,
		Author: "Cintia; echo bad", Contributor: "Multiversa", WorkingDir: "/tmp",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"add", "https://example.com/a?x=$(bad)", "--dir", dir,
		"--author", "Cintia; echo bad", "--contributor", "Multiversa"}
	if !reflect.DeepEqual(f.commands[0].Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", f.commands[0].Args, wantArgs)
	}
	if got.Artifact != filepath.Join(dir, "source.md") {
		t.Fatalf("artifact = %q", got.Artifact)
	}
}

func TestAddRejectsUnsafeURLsWithoutRunning(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "ftp://example.com/a", "https://u:p@example.com/a", "not a url"} {
		f := &fakeRunner{}
		_, err := New(f).Add(context.Background(), AddOptions{URL: raw, TargetDir: t.TempDir()})
		if err == nil {
			t.Errorf("%q: expected error", raw)
		}
		if len(f.commands) != 0 {
			t.Errorf("%q: runner was called", raw)
		}
	}
}

func TestAddRequiresExactlyOneNewArtifact(t *testing.T) {
	dir := t.TempDir()
	f := &fakeRunner{}
	_, err := New(f).Add(context.Background(), AddOptions{URL: "https://example.com", TargetDir: dir})
	if err == nil || !strings.Contains(err.Error(), "creó 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractCommand(t *testing.T) {
	corpus := t.TempDir()
	out := t.TempDir()
	c := New(&fakeRunner{})
	cmd, graph, err := c.ExtractCommand(ExtractOptions{
		CorpusDir: corpus, OutputDir: out, WorkingDir: "/tmp",
		Backend: "openai", Model: "model;bad", Mode: "deep",
		MaxWorkers: 2, TokenBudget: 4000, MaxConcurrency: 1,
		APITimeout: 90 * time.Second, Force: true, NoCluster: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Dir != "/tmp" || cmd.Name != "graphify" {
		t.Fatalf("command = %#v", cmd)
	}
	joined := strings.Join(cmd.Args, " ")
	for _, want := range []string{"extract", "--backend openai", "--model model;bad", "--mode deep",
		"--max-workers 2", "--token-budget 4000", "--max-concurrency 1",
		"--api-timeout 90", "--force", "--no-cluster"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q missing %q", joined, want)
		}
	}
	if strings.Contains(joined, "--allow-partial") {
		t.Fatal("unsafe --allow-partial added")
	}
	if graph != filepath.Join(out, "graphify-out", "graph.json") {
		t.Fatalf("graph path = %q", graph)
	}
}

func TestExtractValidatesOutput(t *testing.T) {
	corpus := t.TempDir()
	out := t.TempDir()
	f := &fakeRunner{run: func(Command) Result {
		dir := filepath.Join(out, "graphify-out")
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "graph.json"),
			[]byte(`{"nodes":[{"id":"a"},{"id":"b"}],"links":[{"source":"a","target":"b"}]}`), 0o644)
		return Result{}
	}}
	stats, _, err := New(f).Extract(context.Background(), ExtractOptions{CorpusDir: corpus, OutputDir: out})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Nodes != 2 || stats.Edges != 1 {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestExtractFailureDoesNotValidate(t *testing.T) {
	f := &fakeRunner{run: func(Command) Result {
		return Result{ExitCode: 1, Stderr: "backend failed", Err: errors.New("exit")}
	}}
	_, _, err := New(f).Extract(context.Background(), ExtractOptions{
		CorpusDir: t.TempDir(), OutputDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "backend failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEdgesAndInvalidGraphs(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "ok.json")
	if err := os.WriteFile(ok, []byte(`{"nodes":[],"edges":[{},{}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stats, err := Validate(ok)
	if err != nil || stats.Nodes != 0 || stats.Edges != 2 {
		t.Fatalf("Validate() = %#v, %v", stats, err)
	}
	bad := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(bad, []byte(`{"nodes":{},"edges":[]}`), 0o644)
	if _, err := Validate(bad); err == nil {
		t.Fatal("expected invalid nodes error")
	}
}

func TestMergedEnvironmentOverridesHost(t *testing.T) {
	got := mergedEnvironment(
		[]string{"PATH=/bin", "OPENAI_API_KEY=host"},
		[]string{"OPENAI_API_KEY=tenant", "MODEL=x"},
	)
	want := []string{"PATH=/bin", "OPENAI_API_KEY=tenant", "MODEL=x"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergedEnvironment() = %#v, want %#v", got, want)
	}
}
