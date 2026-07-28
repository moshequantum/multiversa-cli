// Package graphify provides a small, testable process adapter for the
// Graphify CLI. It deliberately never invokes a shell: every argument is
// passed directly to os/exec so URLs, paths, and user-supplied metadata cannot
// be interpreted as shell syntax.
package graphify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/moshequantum/multiversa-cli/internal/ingest"
)

const defaultBinary = "graphify"

// Command is one direct process invocation.
type Command struct {
	Name string
	Args []string
	Dir  string
	Env  []string
	// Timeout is an optional whole-process deadline. A caller context with an
	// earlier deadline still wins.
	Timeout time.Duration
}

// Result captures a completed process invocation.
type Result = ingest.Result

// Runner makes process execution replaceable in unit tests.
type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, command Command) Result
}

// OSRunner executes commands on the host without a shell.
type OSRunner struct{}

func (OSRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (OSRunner) Run(ctx context.Context, command Command) Result {
	start := time.Now()
	if command.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, command.Timeout)
		defer cancel()
	}
	c := exec.CommandContext(ctx, command.Name, command.Args...)
	c.Dir = command.Dir
	if command.Env != nil {
		c.Env = mergedEnvironment(os.Environ(), command.Env)
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	code := 0
	if err != nil {
		code = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		}
	}
	return Result{
		Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code,
		Duration: time.Since(start), Err: err,
	}
}

// Client invokes one Graphify binary through a Runner.
type Client struct {
	Runner Runner
	Binary string
	Env    []string
}

func New(r Runner) *Client {
	if r == nil {
		r = OSRunner{}
	}
	return &Client{Runner: r, Binary: defaultBinary}
}

func (c *Client) binary() string {
	if strings.TrimSpace(c.Binary) == "" {
		return defaultBinary
	}
	return c.Binary
}

// Preflight verifies that Graphify is on PATH and responds to --version.
func (c *Client) Preflight(ctx context.Context) (string, error) {
	path, err := c.Runner.LookPath(c.binary())
	if err != nil {
		return "", fmt.Errorf("graphify no está instalado o no está en PATH: %w", err)
	}
	res := c.Runner.Run(ctx, Command{Name: path, Args: []string{"--version"}, Env: c.Env})
	if res.Err != nil || res.ExitCode != 0 {
		return "", commandError("graphify --version", res)
	}
	version := strings.TrimSpace(res.Stdout)
	if version == "" {
		version = strings.TrimSpace(res.Stderr)
	}
	if version == "" {
		return "", errors.New("graphify --version no devolvió una versión")
	}
	return version, nil
}

// AddOptions controls one URL ingestion.
type AddOptions = ingest.AddOptions

// AddResult identifies the artifact Graphify created.
type AddResult = ingest.AddResult

// Add downloads one public HTTP(S) source into TargetDir.
func (c *Client) Add(ctx context.Context, opts AddOptions) (AddResult, error) {
	if err := validateHTTPURL(opts.URL); err != nil {
		return AddResult{}, err
	}
	if strings.TrimSpace(opts.TargetDir) == "" {
		return AddResult{}, errors.New("graphify add requiere TargetDir")
	}
	target, err := filepath.Abs(opts.TargetDir)
	if err != nil {
		return AddResult{}, fmt.Errorf("resolver TargetDir: %w", err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return AddResult{}, fmt.Errorf("crear corpus Graphify: %w", err)
	}
	before, err := directorySnapshot(target)
	if err != nil {
		return AddResult{}, err
	}
	args := []string{"add", opts.URL, "--dir", target}
	if opts.Author != "" {
		args = append(args, "--author", opts.Author)
	}
	if opts.Contributor != "" {
		args = append(args, "--contributor", opts.Contributor)
	}
	res := c.Runner.Run(ctx, Command{
		Name: c.binary(), Args: args, Dir: opts.WorkingDir, Env: c.Env, Timeout: opts.Timeout,
	})
	if res.Err != nil || res.ExitCode != 0 {
		return AddResult{Result: res}, commandError("graphify add", res)
	}
	after, err := directorySnapshot(target)
	if err != nil {
		return AddResult{Result: res}, err
	}
	var created []string
	for path := range after {
		if _, existed := before[path]; !existed {
			created = append(created, path)
		}
	}
	sort.Strings(created)
	if len(created) != 1 {
		return AddResult{Result: res}, fmt.Errorf(
			"graphify add terminó correctamente pero creó %d artefactos (se esperaba 1)", len(created),
		)
	}
	return AddResult{Artifact: created[0], Result: res}, nil
}

// ExtractOptions controls a headless Graphify extraction.
type ExtractOptions = ingest.ExtractOptions

// Extract runs Graphify and validates the resulting graph.
func (c *Client) Extract(ctx context.Context, opts ExtractOptions) (GraphStats, Result, error) {
	command, graphPath, err := c.ExtractCommand(opts)
	if err != nil {
		return GraphStats{}, Result{}, err
	}
	res := c.Runner.Run(ctx, command)
	if res.Err != nil || res.ExitCode != 0 {
		return GraphStats{}, res, commandError("graphify extract", res)
	}
	stats, err := Validate(graphPath)
	if err != nil {
		return GraphStats{}, res, fmt.Errorf("validar salida Graphify: %w", err)
	}
	return stats, res, nil
}

// ExtractCommand builds a direct, shell-free extraction command. It never adds
// --allow-partial; preserving the last complete graph is the safe default.
func (c *Client) ExtractCommand(opts ExtractOptions) (Command, string, error) {
	if opts.CorpusDir == "" || opts.OutputDir == "" {
		return Command{}, "", errors.New("graphify extract requiere CorpusDir y OutputDir")
	}
	corpus, err := filepath.Abs(opts.CorpusDir)
	if err != nil {
		return Command{}, "", fmt.Errorf("resolver CorpusDir: %w", err)
	}
	out, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return Command{}, "", fmt.Errorf("resolver OutputDir: %w", err)
	}
	if fi, statErr := os.Stat(corpus); statErr != nil || !fi.IsDir() {
		if statErr == nil {
			statErr = errors.New("no es un directorio")
		}
		return Command{}, "", fmt.Errorf("corpus Graphify inválido %q: %w", corpus, statErr)
	}
	args := []string{"extract", corpus, "--out", out}
	if opts.Backend != "" {
		args = append(args, "--backend", opts.Backend)
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Mode != "" {
		if opts.Mode != "deep" {
			return Command{}, "", fmt.Errorf("modo Graphify inválido %q", opts.Mode)
		}
		args = append(args, "--mode", opts.Mode)
	}
	if opts.MaxWorkers < 0 || opts.TokenBudget < 0 || opts.MaxConcurrency < 0 || opts.APITimeout < 0 {
		return Command{}, "", errors.New("los límites de Graphify no pueden ser negativos")
	}
	if opts.MaxWorkers > 0 {
		args = append(args, "--max-workers", fmt.Sprint(opts.MaxWorkers))
	}
	if opts.TokenBudget > 0 {
		args = append(args, "--token-budget", fmt.Sprint(opts.TokenBudget))
	}
	if opts.MaxConcurrency > 0 {
		args = append(args, "--max-concurrency", fmt.Sprint(opts.MaxConcurrency))
	}
	if opts.APITimeout > 0 {
		seconds := int(opts.APITimeout.Round(time.Second) / time.Second)
		if seconds < 1 {
			seconds = 1
		}
		args = append(args, "--api-timeout", fmt.Sprint(seconds))
	}
	if opts.Force {
		args = append(args, "--force")
	}
	if opts.NoCluster {
		args = append(args, "--no-cluster")
	}
	env := c.Env
	if opts.Env != nil {
		env = mergedEnvironment(c.Env, opts.Env)
	}
	return Command{Name: c.binary(), Args: args, Dir: opts.WorkingDir, Env: env, Timeout: opts.Timeout},
		filepath.Join(out, "graphify-out", "graph.json"), nil
}

// GraphStats is the minimal machine-readable health view of graph.json.
type GraphStats = ingest.GraphStats

// Validate ensures graph.json exists, is valid JSON, and contains node and
// edge arrays. Both "edges" and NetworkX's "links" representation are accepted.
func Validate(path string) (GraphStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return GraphStats{}, fmt.Errorf("abrir graph.json: %w", err)
	}
	defer f.Close()
	var graph struct {
		Nodes json.RawMessage `json:"nodes"`
		Edges json.RawMessage `json:"edges"`
		Links json.RawMessage `json:"links"`
	}
	dec := json.NewDecoder(f)
	if err := dec.Decode(&graph); err != nil {
		return GraphStats{}, fmt.Errorf("graph.json inválido: %w", err)
	}
	nodes, err := arrayLength(graph.Nodes, "nodes")
	if err != nil {
		return GraphStats{}, err
	}
	edgeRaw := graph.Edges
	if len(edgeRaw) == 0 || string(edgeRaw) == "null" {
		edgeRaw = graph.Links
	}
	edges, err := arrayLength(edgeRaw, "edges/links")
	if err != nil {
		return GraphStats{}, err
	}
	absolute, _ := filepath.Abs(path)
	return GraphStats{Path: absolute, Nodes: nodes, Edges: edges}, nil
}

func arrayLength(raw json.RawMessage, name string) (int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, fmt.Errorf("graph.json no contiene el arreglo %s", name)
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return 0, fmt.Errorf("graph.json: %s no es un arreglo", name)
	}
	return len(values), nil
}

func validateHTTPURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("URL inválida %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL inválida %q: solo se permite http(s)", raw)
	}
	if u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("URL inválida %q: requiere host público y no admite credenciales", raw)
	}
	return nil
}

func directorySnapshot(dir string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("leer corpus Graphify: %w", err)
	}
	out := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			out[filepath.Join(dir, entry.Name())] = struct{}{}
		}
	}
	return out, nil
}

func commandError(name string, res Result) error {
	detail := strings.TrimSpace(res.Stderr)
	if detail == "" {
		detail = strings.TrimSpace(res.Stdout)
	}
	if detail == "" && res.Err != nil {
		detail = res.Err.Error()
	}
	if len(detail) > 500 {
		detail = detail[:500] + "…"
	}
	return fmt.Errorf("%s falló (exit=%d): %s", name, res.ExitCode, detail)
}

func mergedEnvironment(base, overrides []string) []string {
	values := make(map[string]string, len(base)+len(overrides))
	order := make([]string, 0, len(base)+len(overrides))
	put := func(entry string) {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return
		}
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = entry
	}
	for _, entry := range base {
		put(entry)
	}
	for _, entry := range overrides {
		put(entry)
	}
	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, values[key])
	}
	return out
}
