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

// slugProblem explains why a slug was rejected, in the user's terms. The
// single "usa kebab-case" message used to cover every case, so a perfectly
// kebab-case slug that was merely too long sent the user off to fix
// characters that were already correct.
func slugProblem(slug string) string {
	shown := slug
	if len(shown) > 40 {
		shown = shown[:40] + "…"
	}
	switch {
	case len(slug) < 2:
		return fmt.Sprintf("slug inválido %q — necesita al menos 2 caracteres", shown)
	case len(slug) > 63:
		return fmt.Sprintf("slug inválido %q — tiene %d caracteres y el máximo son 63", shown, len(slug))
	default:
		return fmt.Sprintf("slug inválido %q — usa kebab-case: minúsculas, números y guiones, empezando por letra o número", shown)
	}
}

// Dir returns the directory for a slug after validating it.
func Dir(slug string) (string, error) {
	if !slugRe.MatchString(slug) {
		return "", fmt.Errorf("%s", slugProblem(slug))
	}
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, slug), nil
}

// New scaffolds a unique project OS from neutral technical defaults. It never overwrites:
// an existing tenant dir is an error, keeping profiles append-only.
//
// Layout created:
//
//	~/.multiversa/tenants/<slug>/
//	  multiversa.toml   the tenant's DNA (template pre-filled)
//	  vault/            0700 — secrets live here, never serialized
//	  graph/            knowledge graph home (anchored to identity)
//	  memory/           Engram home for this tenant
//
// Pillars given here replace the template's defaults; passing none keeps them.
// It is variadic so every existing three-argument call still compiles — and so
// that the Tauri installer can delegate tenant creation here instead of writing
// its own multiversa.toml, which is how the two implementations drifted apart.
func New(slug, name, kind string, pillars ...manifest.Pillar) (*manifest.Manifest, string, error) {
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
	if len(pillars) > 0 {
		m.Pillars = normalizePillars(pillars)
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
		"# Multiversa NUNCA serializa, sincroniza ni expone a un agente lo que hay\n" +
		"# aquí. Solo procesos locales que tú ejecutas (p. ej. la voz) leen un\n" +
		"# secreto para usarlo; nunca sale de esta máquina.\n"
	if err := os.WriteFile(filepath.Join(dir, m.Vault.Path, "README"), []byte(keep), 0o600); err != nil {
		return nil, "", err
	}

	if err := manifest.Save(filepath.Join(dir, manifest.DefaultPath), m); err != nil {
		return nil, "", err
	}
	return m, dir, nil
}

// normalizePillars fills in what the caller left blank: an id derived from the
// name (the manifest keys pillars by id, so an empty one would collide) and the
// default weight of 1.0. Pillars without a name are dropped — an unnamed pillar
// cannot be scored against anything.
func normalizePillars(in []manifest.Pillar) []manifest.Pillar {
	out := make([]manifest.Pillar, 0, len(in))
	seen := map[string]bool{}
	for _, p := range in {
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			continue
		}
		if p.ID == "" {
			p.ID = slugifyPillar(p.Name)
		}
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		if p.Weight == 0 {
			p.Weight = 1.0
		}
		out = append(out, p)
	}
	return out
}

// asciiFold maps the accented Latin characters Spanish and Portuguese actually
// use down to ASCII. Without it "Operación" would slug to "operaci-n" — the
// accent silently becoming a word break — and this product is Spanish-first.
var asciiFold = map[rune]rune{
	'á': 'a', 'à': 'a', 'ä': 'a', 'â': 'a', 'ã': 'a', 'å': 'a',
	'é': 'e', 'è': 'e', 'ë': 'e', 'ê': 'e',
	'í': 'i', 'ì': 'i', 'ï': 'i', 'î': 'i',
	'ó': 'o', 'ò': 'o', 'ö': 'o', 'ô': 'o', 'õ': 'o',
	'ú': 'u', 'ù': 'u', 'ü': 'u', 'û': 'u',
	'ñ': 'n', 'ç': 'c', 'ý': 'y', 'ÿ': 'y',
}

// slugifyPillar reduces a display name to a manifest-safe id: accents folded to
// ASCII, lowercase, remaining non-alphanumerics collapsed to single hyphens.
func slugifyPillar(name string) string {
	var b strings.Builder
	lastHyphen := true // leading hyphens are suppressed
	for _, r := range strings.ToLower(name) {
		if folded, ok := asciiFold[r]; ok {
			r = folded
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
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

// Template returns neutral technical defaults for a unique project OS. The
// legacy kind argument is preserved for source compatibility but never selects
// a customer-shaped profile or changes the OS identity.
func Template(kind string) *manifest.Manifest {
	m := manifest.Default()
	m.Multiversa.Version = "0.3"
	m.Tenant.Kind = "project-os"
	m.Routing = manifest.Routing{Primary: "lab", GroupTriggers: []string{"custom-implementation", "managed-operation", "private-data"}}
	m.Engagement = manifest.Engagement{Mode: "self-service", Phase: "foundation"}
	m.Vault = manifest.Vault{Path: "vault"}
	m.Graph = manifest.Graph{Engine: "graphify", Anchor: "identity", Ingest: []string{"docs", "decisions"}}
	m.Sync = manifest.Sync{Providers: []string{"insforge", "gdrive"}, Mode: "backup", Plan: "nano", Auto: false}
	m.Scoring = manifest.Scoring{
		Alignment: 1.0, Urgency: 0.8, Impact: 1.0, Effort: 0.6, Confidence: 0.7,
		ExplorationQuota: 0.10, DecayHalfLifeDays: 45,
	}

	m.Pillars = []manifest.Pillar{
		{ID: "outcomes", Name: "Resultados", Metric: "resultados verificados", Weight: 1.0},
		{ID: "knowledge", Name: "Conocimiento", Metric: "decisiones recuperables", Weight: 0.8},
	}
	m.Deploy = manifest.Deploy{Targets: []string{"github"}}
	return m
}
