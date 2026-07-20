// Package manifest parses and persists the multiversa.toml declarative manifest.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const DefaultPath = "multiversa.toml"

type Manifest struct {
	Multiversa Meta              `toml:"multiversa" json:"multiversa"`
	Tenant     Tenant            `toml:"tenant,omitempty" json:"tenant,omitempty"`
	Identity   Identity          `toml:"identity,omitempty" json:"identity,omitempty"`
	Pillars    []Pillar          `toml:"pillars,omitempty" json:"pillars,omitempty"`
	Scoring    Scoring           `toml:"scoring,omitempty" json:"scoring,omitempty"`
	Vault      Vault             `toml:"vault,omitempty" json:"vault,omitempty"`
	Graph      Graph             `toml:"graph,omitempty" json:"graph,omitempty"`
	Sync       Sync              `toml:"sync,omitempty" json:"sync,omitempty"`
	Deploy     Deploy            `toml:"deploy,omitempty" json:"deploy,omitempty"`
	Stack      map[string]string `toml:"stack" json:"stack"`
	Agents     Agents            `toml:"agents" json:"agents"`
	Backend    Backend           `toml:"backend" json:"backend"`
	Credits    Credits           `toml:"credits" json:"credits"`
}

// Tenant identifies whose OS this manifest instantiates. One manifest =
// one tenant = one isolated installation (ElevatOS, PulseOS, …).
type Tenant struct {
	Slug  string `toml:"slug,omitempty" json:"slug,omitempty"`   // kebab-case id, dir name under ~/.multiversa/tenants/
	Name  string `toml:"name,omitempty" json:"name,omitempty"`   // display name, e.g. "PulseOS — Cintia Larizatti"
	Kind  string `toml:"kind,omitempty" json:"kind,omitempty"`   // personal-os | personal-brand | agency | team
	Owner string `toml:"owner,omitempty" json:"owner,omitempty"` // contact of the human who decides
}

// Identity is the brand DNA — the lightweight manifest-level slice of
// the L4 identity layer. The knowledge graph and every skill run anchor
// to this section, so ingestion inherits brand context from setup.
type Identity struct {
	Brand    string   `toml:"brand,omitempty" json:"brand,omitempty"`
	Voice    string   `toml:"voice,omitempty" json:"voice,omitempty"`       // tone in the tenant's own words
	Language string   `toml:"language,omitempty" json:"language,omitempty"` // primary output language
	Avatar   string   `toml:"avatar,omitempty" json:"avatar,omitempty"`     // avatar personality id
	Taboos   []string `toml:"taboos,omitempty" json:"taboos,omitempty"`     // hard boundaries — runner blocks these
}

// Pillar is one strategic pillar of the tenant's business. The scoring
// engine classifies every signal against these.
type Pillar struct {
	ID     string  `toml:"id" json:"id"`
	Name   string  `toml:"name" json:"name"`
	Metric string  `toml:"metric,omitempty" json:"metric,omitempty"` // what "impact" means for this pillar
	Weight float64 `toml:"weight,omitempty" json:"weight,omitempty"` // relative priority, default 1.0
}

// Scoring holds the tenant's weights for the five scoring dimensions
// plus the anti-negative-loop parameters (see motor-scoring-spec).
type Scoring struct {
	Alignment         float64 `toml:"alignment,omitempty" json:"alignment,omitempty"`
	Urgency           float64 `toml:"urgency,omitempty" json:"urgency,omitempty"`
	Impact            float64 `toml:"impact,omitempty" json:"impact,omitempty"`
	Effort            float64 `toml:"effort,omitempty" json:"effort,omitempty"`
	Confidence        float64 `toml:"confidence,omitempty" json:"confidence,omitempty"`
	ExplorationQuota  float64 `toml:"exploration_quota,omitempty" json:"exploration_quota,omitempty"`
	DecayHalfLifeDays int     `toml:"decay_half_life_days,omitempty" json:"decay_half_life_days,omitempty"`
}

// Vault declares where the tenant's secrets live. The manifest only
// ever stores the path — vault CONTENTS are never read, serialized, or
// synced by any Multiversa surface. The directory is created 0700.
type Vault struct {
	Path string `toml:"path,omitempty" json:"path,omitempty"` // relative to the tenant dir
}

// Graph configures the knowledge graph. Anchored to the identity
// section: everything ingested inherits the brand DNA as context.
type Graph struct {
	Engine string   `toml:"engine,omitempty" json:"engine,omitempty"` // "graphify"
	Anchor string   `toml:"anchor,omitempty" json:"anchor,omitempty"` // "identity" — graph roots at the brand DNA
	Ingest []string `toml:"ingest,omitempty" json:"ingest,omitempty"` // sources: docs, posts, decisions, …
}

// Sync declares backup/sync providers for the tenant's data (memory,
// graph, manifest — NEVER the vault). auto=false means every sync is
// proposed and human-approved; there is no silent replication.
type Sync struct {
	Providers []string `toml:"providers,omitempty" json:"providers,omitempty"` // "insforge", "gdrive" — order = priority
	Mode      string   `toml:"mode,omitempty" json:"mode,omitempty"`           // backup | sync
	Plan      string   `toml:"plan,omitempty" json:"plan,omitempty"`           // e.g. "nano" (InsForge free tier)
	Auto      bool     `toml:"auto" json:"auto"`                               // default false — la IA propone, tú decides
}

// Deploy lists where this tenant's software ships. The manifest is the
// unit of replication; the deploy target is the per-client difference.
type Deploy struct {
	Targets []string `toml:"targets,omitempty" json:"targets,omitempty"` // cloudflare | vercel | github | vps | insforge
}

type Meta struct {
	Profile string `toml:"profile" json:"profile"`
	Ethic   string `toml:"ethic" json:"ethic"`
	Version string `toml:"version" json:"version"`
}

type Agents struct {
	Enabled []string `toml:"enabled" json:"enabled"`
}

type Backend struct {
	Provider string `toml:"provider" json:"provider"`
	URL      string `toml:"url,omitempty" json:"url,omitempty"`
	AnonKey  string `toml:"anon_key,omitempty" json:"-"`
}

type Credits struct {
	AutoPrint bool `toml:"auto_print" json:"auto_print"`
}

func Default() *Manifest {
	return &Manifest{
		Multiversa: Meta{
			Profile: "personal-os",
			Ethic:   "human-in-the-loop",
			Version: "0.1",
		},
		Stack:   map[string]string{},
		Agents:  Agents{Enabled: []string{}},
		Backend: Backend{Provider: "local"},
		Credits: Credits{AutoPrint: true},
	}
}

func Load(path string) (*Manifest, error) {
	m := &Manifest{}
	if _, err := toml.DecodeFile(path, m); err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	return m, nil
}

func Save(path string, m *Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString("# multiversa.toml — generated by `multiversa init`\n"); err != nil {
		return err
	}
	enc := toml.NewEncoder(f)
	enc.Indent = ""
	return enc.Encode(m)
}

func (m *Manifest) Validate() error {
	if m.Multiversa.Ethic != "" && m.Multiversa.Ethic != "human-in-the-loop" {
		return fmt.Errorf("ethic must be 'human-in-the-loop' — Multiversa is built on that contract")
	}
	switch m.Backend.Provider {
	case "", "local", "supabase", "firebase", "insforge":
	default:
		return fmt.Errorf("unknown backend provider %q (allowed: local, supabase, firebase, insforge)", m.Backend.Provider)
	}
	return nil
}
