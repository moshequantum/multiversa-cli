// Package ingest declares the core port used to turn a source corpus into a
// validated knowledge graph. Implementations live in driven adapters (for
// example internal/graphify); the tenant domain depends only on this contract.
package ingest

import (
	"context"
	"time"
)

// Result is the implementation-neutral outcome of an engine invocation.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Err      error
}

// AddOptions describes one public source to materialize in a corpus.
type AddOptions struct {
	URL         string
	TargetDir   string
	Author      string
	Contributor string
	WorkingDir  string
	Timeout     time.Duration
}

// AddResult identifies the artifact created by an ingest engine.
type AddResult struct {
	Artifact string
	Result   Result
}

// ExtractOptions controls a headless semantic extraction.
type ExtractOptions struct {
	CorpusDir      string
	OutputDir      string
	WorkingDir     string
	Backend        string
	Model          string
	Mode           string
	MaxWorkers     int
	TokenBudget    int
	MaxConcurrency int
	APITimeout     time.Duration
	Timeout        time.Duration
	Force          bool
	NoCluster      bool
	// Env contains per-attempt credentials and routing overrides. Adapters must
	// pass it only to the child engine process and must not persist or log it.
	Env []string
}

// GraphStats is the minimal machine-readable health view of a graph.
type GraphStats struct {
	Path  string `json:"path"`
	Nodes int    `json:"nodes"`
	Edges int    `json:"edges"`
}

// Engine is the port driven by tenant bootstrap.
type Engine interface {
	Preflight(context.Context) (string, error)
	Add(context.Context, AddOptions) (AddResult, error)
	Extract(context.Context, ExtractOptions) (GraphStats, Result, error)
}
