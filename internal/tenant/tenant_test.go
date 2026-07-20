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
