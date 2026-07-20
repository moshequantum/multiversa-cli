// Package tenant manages isolated tenant profiles under
// ~/.multiversa/tenants/<slug>/. One tenant = one manifest (the brand
// DNA) + one vault (0700, never read by any Multiversa surface) + one
// memory/graph home. Switching tenants swaps the whole context safely:
// nothing from one tenant's dir is ever read while another is active.
package tenant

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/manifest"
)

// slugRe validates tenant slugs: kebab-case, filesystem-safe.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

// Info is the JSON-safe view of a tenant profile. Vault contents are
// never included — only the path and whether the layout is intact.
type Info struct {
	Slug     string `json:"slug"`
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Owner    string `json:"owner,omitempty"`
	Dir      string `json:"dir"`
	Active   bool   `json:"active"`
	VaultOK  bool   `json:"vault_ok"` // vault dir exists with 0700
	Manifest bool   `json:"manifest"` // multiversa.toml present and parseable
}

// Root returns ~/.multiversa/tenants.
func Root() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".multiversa", "tenants"), nil
}

// activePath returns the file holding the active tenant slug.
func activePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".multiversa", "active-tenant"), nil
}

// Active returns the currently active tenant slug, or "" if none.
func Active() string {
	p, err := activePath()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Use marks slug as the active tenant. The tenant must exist.
func Use(slug string) error {
	dir, err := Dir(slug)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, manifest.DefaultPath)); err != nil {
		return fmt.Errorf("tenant %q no existe o no tiene manifiesto — crea uno con `multiversa tenant new %s`", slug, slug)
	}
	p, err := activePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(slug+"\n"), 0o644)
}

// Dir returns the directory for a slug after validating it.
func Dir(slug string) (string, error) {
	if !slugRe.MatchString(slug) {
		return "", fmt.Errorf("slug inválido %q — usa kebab-case: letras minúsculas, números y guiones", slug)
	}
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, slug), nil
}

// New scaffolds a tenant profile from a template. It never overwrites:
// an existing tenant dir is an error, keeping profiles append-only.
//
// Layout created:
//
//	~/.multiversa/tenants/<slug>/
//	  multiversa.toml   the tenant's DNA (template pre-filled)
//	  vault/            0700 — secrets live here, never serialized
//	  graph/            knowledge graph home (anchored to identity)
//	  memory/           Engram home for this tenant
func New(slug, name, kind string) (*manifest.Manifest, string, error) {
	dir, err := Dir(slug)
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Stat(dir); err == nil {
		return nil, "", fmt.Errorf("el tenant %q ya existe en %s — los perfiles no se sobreescriben", slug, dir)
	}

	m := Template(kind)
	m.Tenant.Slug = slug
	m.Tenant.Name = name
	if m.Tenant.Name == "" {
		m.Tenant.Name = slug
	}

	for _, sub := range []string{"graph", "memory"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return nil, "", err
		}
	}
	// The vault is the one directory with tightened permissions.
	if err := os.MkdirAll(filepath.Join(dir, m.Vault.Path), 0o700); err != nil {
		return nil, "", err
	}
	keep := "# Vault de " + m.Tenant.Name + " — secretos de este tenant.\n" +
		"# Multiversa NUNCA lee, serializa ni sincroniza el contenido de este directorio.\n"
	if err := os.WriteFile(filepath.Join(dir, m.Vault.Path, "README"), []byte(keep), 0o600); err != nil {
		return nil, "", err
	}

	if err := manifest.Save(filepath.Join(dir, manifest.DefaultPath), m); err != nil {
		return nil, "", err
	}
	return m, dir, nil
}

// List returns every tenant profile found under Root.
func List() ([]Info, error) {
	root, err := Root()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	active := Active()
	var out []Info
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, Inspect(e.Name(), active))
	}
	return out, nil
}

// Inspect builds the Info view for one slug. It reads the manifest but
// never enters the vault directory.
func Inspect(slug, active string) Info {
	info := Info{Slug: slug, Active: slug == active}
	dir, err := Dir(slug)
	if err != nil {
		return info
	}
	info.Dir = dir

	m, err := manifest.Load(filepath.Join(dir, manifest.DefaultPath))
	if err == nil {
		info.Manifest = true
		info.Name = m.Tenant.Name
		info.Kind = m.Tenant.Kind
		info.Owner = m.Tenant.Owner
		vaultPath := m.Vault.Path
		if vaultPath == "" {
			vaultPath = "vault"
		}
		if fi, err := os.Stat(filepath.Join(dir, vaultPath)); err == nil && fi.IsDir() {
			info.VaultOK = fi.Mode().Perm() == 0o700
		}
	}
	return info
}

// Load reads a tenant's manifest by slug.
func Load(slug string) (*manifest.Manifest, string, error) {
	dir, err := Dir(slug)
	if err != nil {
		return nil, "", err
	}
	path := filepath.Join(dir, manifest.DefaultPath)
	m, err := manifest.Load(path)
	if err != nil {
		return nil, path, err
	}
	return m, path, nil
}

// Template returns the pre-filled DNA for a tenant kind. These are the
// replicable profiles: "agency" is the ElevatOS shape, "personal-brand"
// is the PulseOS shape, anything else gets the personal-os default.
func Template(kind string) *manifest.Manifest {
	m := manifest.Default()
	m.Multiversa.Version = "0.2"
	m.Tenant.Kind = kind
	m.Vault = manifest.Vault{Path: "vault"}
	m.Graph = manifest.Graph{Engine: "graphify", Anchor: "identity", Ingest: []string{"docs", "decisions"}}
	m.Sync = manifest.Sync{Providers: []string{"insforge", "gdrive"}, Mode: "backup", Plan: "nano", Auto: false}
	m.Scoring = manifest.Scoring{
		Alignment: 1.0, Urgency: 0.8, Impact: 1.0, Effort: 0.6, Confidence: 0.7,
		ExplorationQuota: 0.10, DecayHalfLifeDays: 45,
	}

	switch kind {
	case "agency":
		m.Multiversa.Profile = "ecosystem"
		m.Pillars = []manifest.Pillar{
			{ID: "clientes", Name: "Cartera de clientes", Metric: "cuentas activas", Weight: 1.0},
			{ID: "entrega", Name: "Entrega y calidad", Metric: "proyectos a tiempo", Weight: 1.0},
			{ID: "marca", Name: "Marca y comunidad", Metric: "alcance mensual", Weight: 0.8},
		}
		m.Deploy = manifest.Deploy{Targets: []string{"vercel", "cloudflare", "github"}}
	case "personal-brand":
		m.Multiversa.Profile = "personal-os"
		m.Pillars = []manifest.Pillar{
			{ID: "contenido", Name: "Contenido y comunidad", Metric: "engagement semanal", Weight: 1.0},
			{ID: "ofertas", Name: "Ofertas y lanzamientos", Metric: "conversiones", Weight: 1.0},
			{ID: "operacion", Name: "Operación diaria", Metric: "horas recuperadas", Weight: 0.7},
		}
		m.Deploy = manifest.Deploy{Targets: []string{"vercel", "insforge"}}
	default:
		m.Tenant.Kind = "personal-os"
		m.Pillars = []manifest.Pillar{
			{ID: "productividad", Name: "Productividad", Metric: "tareas cerradas", Weight: 1.0},
			{ID: "conocimiento", Name: "Base de conocimiento", Metric: "memorias útiles", Weight: 0.8},
		}
		m.Deploy = manifest.Deploy{Targets: []string{"github"}}
	}
	return m
}
