package tenant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/manifest"
)

// withTempHome redirects HOME so tests never touch the real
// ~/.multiversa. Mirrors the repo rule: never read real profiles.
func withTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestNewScaffoldsIsolatedProfile(t *testing.T) {
	home := withTempHome(t)

	m, dir, err := New("pulseos-cintia", "PulseOS — Cintia", "personal-brand")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if want := filepath.Join(home, ".multiversa", "tenants", "pulseos-cintia"); dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}

	// Vault must exist with 0700 — the isolation contract.
	fi, err := os.Stat(filepath.Join(dir, "vault"))
	if err != nil || !fi.IsDir() {
		t.Fatalf("vault dir missing: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Fatalf("vault perm = %o, want 0700", perm)
	}

	// Template shape: personal-brand = PulseOS pillars + insforge sync.
	if m.Tenant.Kind != "personal-brand" || len(m.Pillars) != 3 {
		t.Fatalf("unexpected template: kind=%q pillars=%d", m.Tenant.Kind, len(m.Pillars))
	}
	if m.Sync.Auto {
		t.Fatal("sync.auto must default to false — la IA propone, el humano decide")
	}

	// Manifest round-trips through TOML.
	loaded, err := manifest.Load(filepath.Join(dir, manifest.DefaultPath))
	if err != nil {
		t.Fatalf("reload manifest: %v", err)
	}
	if loaded.Tenant.Slug != "pulseos-cintia" || loaded.Graph.Anchor != "identity" {
		t.Fatalf("round-trip lost data: %+v", loaded.Tenant)
	}

	// Never overwrite an existing profile.
	if _, _, err := New("pulseos-cintia", "x", "agency"); err == nil {
		t.Fatal("expected error on duplicate tenant, got nil")
	}
}

func TestUseAndActive(t *testing.T) {
	withTempHome(t)
	if _, _, err := New("elevatos-andrea", "ElevatOS — Andrea", "agency"); err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := Use("elevatos-andrea"); err != nil {
		t.Fatalf("Use: %v", err)
	}
	if got := Active(); got != "elevatos-andrea" {
		t.Fatalf("Active = %q", got)
	}
	if err := Use("no-existe"); err == nil {
		t.Fatal("expected error activating missing tenant")
	}
}

func TestSlugValidation(t *testing.T) {
	withTempHome(t)
	for _, bad := range []string{"", "UPPER", "con espacios", "../escape", "a"} {
		if _, _, err := New(bad, "", ""); err == nil {
			t.Fatalf("slug %q should be rejected", bad)
		}
	}
}

func TestVaultNeverSerialized(t *testing.T) {
	withTempHome(t)
	m, dir, err := New("seguro", "Seguro", "personal-os")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Plant a fake secret in the vault, then confirm no JSON view of the
	// tenant ever contains it.
	secret := "TOKEN-SUPER-SECRETO-123"
	if err := os.WriteFile(filepath.Join(dir, "vault", "api.key"), []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secret) {
		t.Fatal("vault content leaked into manifest JSON")
	}
	info := Inspect("seguro", "")
	ib, _ := json.Marshal(info)
	if strings.Contains(string(ib), secret) {
		t.Fatal("vault content leaked into tenant Info")
	}
	if !info.VaultOK {
		t.Fatal("expected VaultOK=true for 0700 vault")
	}
}

// Pillars passed to New replace the template's, so the Tauri installer can
// delegate tenant creation here instead of writing its own multiversa.toml —
// which is how the Go and Rust implementations drifted apart in the first place.
func TestNewAcceptsPillarsAndKeepsTemplateWhenNoneGiven(t *testing.T) {
	withTempHome(t)

	m, _, err := New("con-pilares", "Con pilares", "agency",
		manifest.Pillar{Name: "Contenido", Metric: "engagement semanal"},
		manifest.Pillar{Name: "Ofertas", Metric: "conversiones", Weight: 0.7},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(m.Pillars) != 2 {
		t.Fatalf("got %d pillars, want 2", len(m.Pillars))
	}
	if m.Pillars[0].ID != "contenido" {
		t.Errorf("id = %q, want derived from the name", m.Pillars[0].ID)
	}
	if m.Pillars[0].Weight != 1.0 {
		t.Errorf("weight = %v, want the 1.0 default", m.Pillars[0].Weight)
	}
	if m.Pillars[1].Weight != 0.7 {
		t.Errorf("weight = %v, want the explicit 0.7", m.Pillars[1].Weight)
	}

	tpl, _, err := New("sin-pilares", "Sin pilares", "agency")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(tpl.Pillars) == 0 {
		t.Error("passing no pillars must keep the template's, not blank them")
	}
}

// This product is Spanish-first: an accent must fold to its ASCII letter, not
// become a word break. "Operación" slugging to "operaci-n" is a real defect.
func TestSlugifyPillarFoldsSpanishAccents(t *testing.T) {
	cases := map[string]string{
		"Operación diaria":    "operacion-diaria",
		"Diseño & Marca":      "diseno-marca",
		"Atención al cliente": "atencion-al-cliente",
		"  Márgenes  ":        "margenes",
		"Añadir/Quitar":       "anadir-quitar",
	}
	for in, want := range cases {
		if got := slugifyPillar(in); got != want {
			t.Errorf("slugifyPillar(%q) = %q, want %q", in, got, want)
		}
	}
}

// Unnamed pillars cannot be scored against anything, and duplicate ids would
// collide in the manifest — both are dropped rather than written out broken.
func TestNormalizePillarsDropsUnusableEntries(t *testing.T) {
	got := normalizePillars([]manifest.Pillar{
		{Name: "Contenido"},
		{Name: "   "},                  // blank
		{Name: "contenido"},            // duplicate id
		{Name: "Ofertas", Weight: 2.5}, // explicit weight preserved
	})
	if len(got) != 2 {
		t.Fatalf("got %d pillars, want 2: %+v", len(got), got)
	}
	if got[1].Weight != 2.5 {
		t.Errorf("explicit weight lost: %v", got[1].Weight)
	}
}

// A slug rejected only for its length used to be reported as bad kebab-case,
// sending the user to fix characters that were already correct. Each rejection
// reason must name itself, and the echoed slug must stay readable.
func TestSlugErrorsExplainTheActualProblem(t *testing.T) {
	withTempHome(t)

	long := strings.Repeat("operacion-", 18) // 180 chars, valid kebab-case
	_, _, err := New(long, "X", "")
	if err == nil {
		t.Fatal("expected a slug over 63 characters to be rejected")
	}
	msg := err.Error()
	if !strings.Contains(msg, "63") {
		t.Errorf("length rejection must mention the limit, got: %s", msg)
	}
	if strings.Contains(msg, "kebab-case") {
		t.Errorf("a valid kebab-case slug must not be blamed on its characters: %s", msg)
	}
	if len(msg) > 160 {
		t.Errorf("error echoes the whole slug and is unreadable (%d chars)", len(msg))
	}

	if _, _, err := New("a", "X", ""); err == nil {
		t.Fatal("expected a one-character slug to be rejected")
	} else if !strings.Contains(err.Error(), "2 caracteres") {
		t.Errorf("short rejection must state the minimum, got: %s", err.Error())
	}

	if _, _, err := New("Mi_OS", "X", ""); err == nil {
		t.Fatal("expected an invalid-character slug to be rejected")
	} else if !strings.Contains(err.Error(), "kebab-case") {
		t.Errorf("character rejection should mention kebab-case, got: %s", err.Error())
	}
}
