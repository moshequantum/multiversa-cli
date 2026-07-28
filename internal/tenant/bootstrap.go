package tenant

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/ingest"
	"github.com/moshequantum/multiversa-cli/internal/manifest"
)

const (
	corpusDirName     = "corpus"
	rawDirName        = "raw"
	provenanceDirName = "provenance"
	sourcesFileName   = "sources.jsonl"
	identityFileName  = "identity.md"
)

// BootstrapSource is one public source approved for ingestion. Credentials do
// not belong here: Graphify receives API keys only through its process
// environment.
type BootstrapSource struct {
	URL         string `json:"url"`
	Author      string `json:"author,omitempty"`
	Contributor string `json:"contributor,omitempty"`
}

// BootstrapOptions describes a resumable tenant bootstrap. Creation fields are
// used only when the tenant does not exist; an existing manifest is never
// overwritten.
type BootstrapOptions struct {
	Name        string
	Kind        string
	OSName      string
	Owner       string
	Brand       string
	Voice       string
	Language    string
	Taboos      []string
	Route       string
	Engagement  string
	Pillars     []manifest.Pillar
	Sources     []BootstrapSource
	Backend     string
	Model       string
	Mode        string
	NoCluster   bool
	Force       bool
	SkipExtract bool
	DryRun      bool
	Activate    bool
}

type BootstrapResult struct {
	Slug             string            `json:"slug"`
	Dir              string            `json:"dir"`
	Created          bool              `json:"created"`
	Resumed          bool              `json:"resumed"`
	DryRun           bool              `json:"dry_run"`
	Activated        bool              `json:"activated"`
	Added            int               `json:"added"`
	Skipped          int               `json:"skipped"`
	GraphPath        string            `json:"graph_path,omitempty"`
	SourceFile       string            `json:"source_file"`
	Plan             []string          `json:"plan,omitempty"`
	ProviderAttempts []ProviderAttempt `json:"provider_attempts,omitempty"`
}

// ProviderAttempt is deliberately redacted: it records routing and outcome,
// never environment variables, credentials, or provider response bodies.
type ProviderAttempt struct {
	Provider  string `json:"provider"`
	Backend   string `json:"backend"`
	Model     string `json:"model"`
	Success   bool   `json:"success"`
	ErrorCode string `json:"error_code,omitempty"`
}

// sourceRecord is the stable, append-free provenance representation. The file
// is rewritten sorted after each successful add, making retries deterministic.
type sourceRecord struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Author      string `json:"author,omitempty"`
	Contributor string `json:"contributor,omitempty"`
}

// Bootstrap creates or resumes a tenant, materializes an identity-rooted
// corpus, ingests each new URL, extracts the graph, and optionally activates
// the tenant only after graph validation.
func Bootstrap(ctx context.Context, slug string, opts BootstrapOptions, client ingest.Engine) (*BootstrapResult, error) {
	if client == nil && !opts.DryRun {
		return nil, fmt.Errorf("bootstrap: Graphify client is required")
	}
	dir, err := Dir(slug)
	if err != nil {
		return nil, err
	}

	m, _, loadErr := Load(slug)
	exists := loadErr == nil
	if loadErr != nil {
		if _, statErr := os.Stat(dir); statErr == nil {
			return nil, fmt.Errorf("bootstrap: tenant %q exists but cannot be loaded: %w", slug, loadErr)
		} else if !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("bootstrap: inspect tenant %q: %w", slug, statErr)
		}
	}

	normalized, err := normalizeBootstrapSources(opts.Sources)
	if err != nil {
		return nil, err
	}
	graphDir := filepath.Join(dir, "graph")
	corpusDir := filepath.Join(graphDir, corpusDirName)
	rawDir := filepath.Join(corpusDir, rawDirName)
	provenanceDir := filepath.Join(corpusDir, provenanceDirName)
	sourceFile := filepath.Join(provenanceDir, sourcesFileName)
	result := &BootstrapResult{
		Slug: slug, Dir: dir, Created: !exists, Resumed: exists, DryRun: opts.DryRun,
		SourceFile: sourceFile,
		Plan:       []string{"reconcile_tenant", "materialize_corpus", "ingest_sources", "extract_graph", "validate_graph"},
	}
	if opts.Activate {
		if opts.SkipExtract {
			return nil, errors.New("bootstrap: --activate requiere extracción y validación del grafo")
		}
		result.Plan = append(result.Plan, "activate_tenant")
	}
	if opts.SkipExtract {
		result.Plan = []string{"reconcile_tenant", "materialize_corpus", "ingest_sources"}
	}
	if opts.DryRun {
		return result, nil
	}
	if _, err := client.Preflight(ctx); err != nil {
		return result, fmt.Errorf("bootstrap: Graphify preflight: %w", err)
	}

	if !exists {
		m, _, err = New(slug, opts.Name, opts.Kind, opts.Pillars...)
		if err != nil {
			return nil, fmt.Errorf("bootstrap: create tenant: %w", err)
		}
		m.Tenant.OSName = opts.OSName
		if m.Tenant.OSName == "" {
			m.Tenant.OSName = m.Tenant.Name
		}
		m.Tenant.Owner = opts.Owner
		m.Identity.Brand = opts.Brand
		m.Identity.Voice = opts.Voice
		m.Identity.Language = opts.Language
		m.Identity.Taboos = append([]string(nil), opts.Taboos...)
		if opts.Route != "" {
			m.Routing.Primary = opts.Route
		}
		if opts.Engagement != "" {
			m.Engagement.Mode = opts.Engagement
		}
		if err := manifest.Save(filepath.Join(dir, manifest.DefaultPath), m); err != nil {
			return result, fmt.Errorf("bootstrap: save manifest: %w", err)
		}
	}

	// A tenant corpus can contain a sensitive client dossier. Tighten the whole
	// tenant tree before materializing it; files are independently written 0600.
	for _, p := range []string{dir, graphDir, filepath.Join(dir, "memory"), corpusDir, rawDir, provenanceDir} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			return result, fmt.Errorf("bootstrap: create private directory %s: %w", p, err)
		}
		if err := os.Chmod(p, 0o700); err != nil {
			return result, fmt.Errorf("bootstrap: secure directory %s: %w", p, err)
		}
	}
	if err := atomicWrite(filepath.Join(rawDir, identityFileName), []byte(identityDocument(m)), 0o600); err != nil {
		return result, fmt.Errorf("bootstrap: write identity: %w", err)
	}
	if err := secureTenantTree(dir); err != nil {
		return result, fmt.Errorf("bootstrap: secure tenant scaffold: %w", err)
	}

	records, err := readSourceRecords(sourceFile)
	if err != nil {
		return result, fmt.Errorf("bootstrap: read provenance: %w", err)
	}
	known := make(map[string]sourceRecord, len(records))
	for _, record := range records {
		known[record.ID] = record
	}
	// Materialize provenance before the first network operation so a failed
	// fetch still leaves a valid, resumable tenant layout.
	if err := writeSourceRecords(sourceFile, records); err != nil {
		return result, fmt.Errorf("bootstrap: write provenance: %w", err)
	}
	for _, record := range normalized {
		if _, ok := known[record.ID]; ok {
			result.Skipped++
			continue
		}
		if _, err := client.Add(ctx, ingest.AddOptions{
			URL: record.URL, TargetDir: rawDir, Author: record.Author,
			Contributor: record.Contributor, WorkingDir: dir,
		}); err != nil {
			return result, fmt.Errorf("bootstrap: graphify add %q failed: %w", record.URL, err)
		}
		if err := secureTenantTree(dir); err != nil {
			return result, fmt.Errorf("bootstrap: secure downloaded source: %w", err)
		}
		known[record.ID] = record
		records = append(records, record)
		if err := writeSourceRecords(sourceFile, records); err != nil {
			return result, fmt.Errorf("bootstrap: write provenance: %w", err)
		}
		result.Added++
	}
	// Ensure an empty, but valid, ledger also exists.
	if err := writeSourceRecords(sourceFile, records); err != nil {
		return result, fmt.Errorf("bootstrap: write provenance: %w", err)
	}

	if opts.SkipExtract {
		return result, nil
	}
	extract := func(backend, model string, env []string) (ingest.GraphStats, error) {
		stats, _, extractErr := client.Extract(ctx, ingest.ExtractOptions{
			CorpusDir: rawDir, OutputDir: graphDir, WorkingDir: dir,
			Backend: backend, Model: model, Mode: opts.Mode,
			NoCluster: opts.NoCluster, Force: opts.Force, Env: env,
		})
		return stats, extractErr
	}
	var stats ingest.GraphStats
	if opts.Backend == "" {
		providerPath, pathErr := providersPath(slug)
		if pathErr != nil {
			return result, pathErr
		}
		if _, statErr := os.Stat(providerPath); statErr == nil {
			plans, planErr := BuildProviderFallback(slug, nil)
			if planErr != nil {
				return result, fmt.Errorf("bootstrap: resolver proveedores LLM: %w", planErr)
			}
			var lastErr error
			for _, plan := range plans {
				stats, lastErr = extract(plan.Backend, plan.Model, plan.Env)
				attempt := ProviderAttempt{Provider: plan.Provider, Backend: plan.Backend, Model: plan.Model, Success: lastErr == nil}
				if lastErr != nil {
					attempt.ErrorCode = "extract_failed"
				}
				result.ProviderAttempts = append(result.ProviderAttempts, attempt)
				if lastErr == nil {
					break
				}
			}
			err = lastErr
		} else if !os.IsNotExist(statErr) {
			return result, fmt.Errorf("bootstrap: inspeccionar proveedores: %w", statErr)
		} else {
			stats, err = extract(opts.Backend, opts.Model, nil)
		}
	} else {
		stats, err = extract(opts.Backend, opts.Model, nil)
	}
	if err != nil {
		_ = secureTenantTree(dir)
		return result, fmt.Errorf("bootstrap: graphify extract failed: %w", err)
	}
	if err := secureTenantTree(dir); err != nil {
		return result, fmt.Errorf("bootstrap: secure graph output: %w", err)
	}
	result.GraphPath = stats.Path

	if opts.Activate {
		if err := Use(slug); err != nil {
			return result, fmt.Errorf("bootstrap: activate tenant: %w", err)
		}
		result.Activated = true
	}
	return result, nil
}

// secureTenantTree enforces the dossier privacy boundary even for files and
// directories created by third-party tools such as Graphify, which otherwise
// inherit the host umask (commonly producing world-readable 0644/0755 data).
func secureTenantTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink inside tenant: %s", path)
		}
		mode := os.FileMode(0o600)
		if entry.IsDir() {
			mode = 0o700
		}
		return os.Chmod(path, mode)
	})
}

func normalizeBootstrapSources(in []BootstrapSource) ([]sourceRecord, error) {
	byID := make(map[string]sourceRecord, len(in))
	for _, source := range in {
		u, err := url.Parse(strings.TrimSpace(source.URL))
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("bootstrap: invalid source URL %q", source.URL)
		}
		u.Scheme = strings.ToLower(u.Scheme)
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("bootstrap: source URL %q must use http or https", source.URL)
		}
		u.Host = strings.ToLower(u.Host)
		u.Fragment = ""
		canonical := u.String()
		sum := sha256.Sum256([]byte(canonical))
		record := sourceRecord{
			ID: hex.EncodeToString(sum[:]), URL: canonical,
			Author: strings.TrimSpace(source.Author), Contributor: strings.TrimSpace(source.Contributor),
		}
		if previous, exists := byID[record.ID]; exists &&
			(previous.Author != record.Author || previous.Contributor != record.Contributor) {
			return nil, fmt.Errorf("bootstrap: conflicting metadata for source %q", canonical)
		}
		byID[record.ID] = record
	}
	out := make([]sourceRecord, 0, len(byID))
	for _, record := range byID {
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
	return out, nil
}

func identityDocument(m *manifest.Manifest) string {
	var b strings.Builder
	b.WriteString("# Tenant identity\n\n")
	fmt.Fprintf(&b, "- Slug: %s\n- Name: %s\n- OS name: %s\n- Owner: %s\n",
		m.Tenant.Slug, m.Tenant.Name, m.Tenant.OSName, m.Tenant.Owner)
	fmt.Fprintf(&b, "- Brand: %s\n- Voice: %s\n- Language: %s\n",
		m.Identity.Brand, m.Identity.Voice, m.Identity.Language)
	if len(m.Identity.Taboos) > 0 {
		b.WriteString("- Taboos:\n")
		for _, taboo := range m.Identity.Taboos {
			fmt.Fprintf(&b, "  - %s\n", taboo)
		}
	}
	b.WriteString("\nThis document is generated from multiversa.toml. Public-source claims belong in provenance, not in identity.\n")
	return b.String()
}

func readSourceRecords(path string) ([]sourceRecord, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []sourceRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		var record sourceRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("invalid sources.jsonl: %w", err)
		}
		if record.ID == "" || record.URL == "" {
			return nil, fmt.Errorf("invalid sources.jsonl: record is missing id or url")
		}
		out = append(out, record)
	}
	return out, scanner.Err()
}

func writeSourceRecords(path string, records []sourceRecord) error {
	records = append([]sourceRecord(nil), records...)
	sort.Slice(records, func(i, j int) bool { return records[i].URL < records[j].URL })
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			return err
		}
	}
	return atomicWrite(path, b.Bytes(), 0o600)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
