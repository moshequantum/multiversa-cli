package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/graphify"
	"github.com/moshequantum/multiversa-cli/internal/ingest"
	"github.com/moshequantum/multiversa-cli/internal/tenant"
)

type inertBootstrapRunner struct{}

func (inertBootstrapRunner) LookPath(string) (string, error) { return "graphify", nil }
func (inertBootstrapRunner) Run(context.Context, graphify.Command) graphify.Result {
	return graphify.Result{}
}

func stubBootstrap(t *testing.T, fn func(context.Context, string, tenant.BootstrapOptions, ingest.Engine) (*tenant.BootstrapResult, error)) {
	t.Helper()
	oldRun, oldClient := runTenantBootstrap, newBootstrapClient
	runTenantBootstrap = fn
	newBootstrapClient = func() *graphify.Client { return graphify.New(inertBootstrapRunner{}) }
	t.Cleanup(func() {
		runTenantBootstrap, newBootstrapClient = oldRun, oldClient
	})
}

func TestTenantBootstrapMapsFlagsToOptions(t *testing.T) {
	var gotSlug string
	var got tenant.BootstrapOptions
	var gotDeadline time.Duration
	stubBootstrap(t, func(ctx context.Context, slug string, opts tenant.BootstrapOptions, _ ingest.Engine) (*tenant.BootstrapResult, error) {
		gotSlug, got = slug, opts
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("bootstrap context has no deadline")
		}
		gotDeadline = time.Until(deadline)
		return &tenant.BootstrapResult{Slug: slug, DryRun: true, Plan: []string{"extract_graph"}}, nil
	})

	cmd := newTenantBootstrapCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{
		"cintia-larizzati",
		"--name", "Cintia Larizzati",
		"--os-name", "CintiaOS",
		"--owner", "Cintia Larizzati",
		"--route", "group",
		"--engagement", "consulting",
		"--pillar", "Autoridad=fuentes verificadas=0.8",
		"--source", "https://example.com/cintia",
		"--source", "https://example.org/profile",
		"--author", "Cintia Larizzati",
		"--contributor", "Multiversa",
		"--activate",
		"--dry-run",
		"--timeout", "45m",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotSlug != "cintia-larizzati" {
		t.Fatalf("slug = %q", gotSlug)
	}
	if got.Name != "Cintia Larizzati" || got.OSName != "CintiaOS" ||
		got.Owner != "Cintia Larizzati" || got.Route != "group" ||
		got.Engagement != "consulting" || !got.Activate || !got.DryRun {
		t.Fatalf("options not mapped: %+v", got)
	}
	if gotDeadline < 44*time.Minute || gotDeadline > 45*time.Minute {
		t.Errorf("deadline = %v", gotDeadline)
	}
	if len(got.Sources) != 2 {
		t.Fatalf("sources = %d", len(got.Sources))
	}
	for _, source := range got.Sources {
		if source.Author != "Cintia Larizzati" || source.Contributor != "Multiversa" {
			t.Errorf("source attribution lost: %+v", source)
		}
	}
	if len(got.Pillars) != 1 || got.Pillars[0].Metric != "fuentes verificadas" || got.Pillars[0].Weight != 0.8 {
		t.Errorf("pillar not parsed: %+v", got.Pillars)
	}
}

func TestTenantBootstrapJSONOutputUsesStableEnvelope(t *testing.T) {
	stubBootstrap(t, func(_ context.Context, slug string, _ tenant.BootstrapOptions, _ ingest.Engine) (*tenant.BootstrapResult, error) {
		return &tenant.BootstrapResult{
			Slug: slug, Dir: "/tmp/tenant", Created: true, Added: 2,
			GraphPath: "/tmp/tenant/graph/graphify-out/graph.json",
		}, nil
	})
	var out bytes.Buffer
	cmd := newTenantBootstrapCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"cintia-larizzati", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var envelope agentout.Envelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if !envelope.OK || envelope.Schema != "multiversa.tenant-bootstrap/v1" || envelope.Command != "tenant-bootstrap" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	data, ok := envelope.Data.(map[string]any)
	if !ok || data["action"] != "bootstrap" {
		t.Fatalf("unexpected data: %#v", envelope.Data)
	}
}

func TestTenantBootstrapJSONErrorIsEmittedWithoutProcessExit(t *testing.T) {
	stubBootstrap(t, func(context.Context, string, tenant.BootstrapOptions, ingest.Engine) (*tenant.BootstrapResult, error) {
		return nil, errors.New("graphify no está instalado")
	})
	var out bytes.Buffer
	cmd := newTenantBootstrapCmd()
	cmd.SetOut(&out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"cintia-larizzati", "--json"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "graphify") {
		t.Fatalf("error = %v", err)
	}
	var envelope agentout.Envelope
	if jsonErr := json.Unmarshal(out.Bytes(), &envelope); jsonErr != nil {
		t.Fatalf("invalid error JSON: %v\n%s", jsonErr, out.String())
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "bootstrap_failed" {
		t.Fatalf("unexpected error envelope: %+v", envelope)
	}
}

func TestTenantBootstrapRejectsInvalidRoutingBeforeRunning(t *testing.T) {
	called := false
	stubBootstrap(t, func(context.Context, string, tenant.BootstrapOptions, ingest.Engine) (*tenant.BootstrapResult, error) {
		called = true
		return nil, nil
	})
	cmd := newTenantBootstrapCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"cintia-larizzati", "--route", "elsewhere"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "route inválido") {
		t.Fatalf("error = %v", err)
	}
	if called {
		t.Fatal("bootstrap ran despite invalid routing")
	}
}
