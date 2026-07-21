package tenant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTenantForSecrets(t *testing.T, slug string) {
	t.Helper()
	withTempHome(t)
	if _, _, err := New(slug, "Prueba", "personal-brand"); err != nil {
		t.Fatalf("New: %v", err)
	}
}

// The core promise: a secret lands in the vault, the file is 0600, and the
// vault directory stays 0700. If either perm regresses, a secret is world- or
// group-readable — the whole point of the vault is defeated.
func TestSetSecretWritesInsideVaultWithTightPerms(t *testing.T) {
	newTenantForSecrets(t, "mi-os")

	n, err := SetSecret("mi-os", "ELEVENLABS_API_KEY", "xi-abc-SECRETO")
	if err != nil {
		t.Fatalf("SetSecret: %v", err)
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}

	path, _ := SecretsPath("mi-os")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("secrets file not created: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("secrets.env perms = %o, want 600", perm)
	}
	vi, _ := os.Stat(filepath.Dir(path))
	if perm := vi.Mode().Perm(); perm != 0o700 {
		t.Errorf("vault perms = %o, want 700", perm)
	}
}

// Round-trip: what goes in comes back out verbatim, including values with the
// shell-hostile characters a naive .env writer would corrupt.
func TestSetSecretRoundTripsAwkwardValues(t *testing.T) {
	newTenantForSecrets(t, "mi-os")

	cases := map[string]string{
		"ELEVENLABS_API_KEY": "xi-normal-key-123",
		"WITH_QUOTE":         "abc'def'ghi",
		"WITH_SPACE":         "value with spaces",
		"WITH_DOLLAR":        "a$b`c\\d",
	}
	for k, v := range cases {
		if _, err := SetSecret("mi-os", k, v); err != nil {
			t.Fatalf("SetSecret(%s): %v", k, err)
		}
	}

	path, _ := SecretsPath("mi-os")
	got, _, err := readSecrets(path)
	if err != nil {
		t.Fatalf("readSecrets: %v", err)
	}
	for k, want := range cases {
		if got[k] != want {
			t.Errorf("%s round-trip = %q, want %q", k, got[k], want)
		}
	}
}

// Updating a key changes only that key and does not grow the file; order of the
// other keys is preserved so the file stays diff-stable.
func TestSetSecretUpdatesInPlace(t *testing.T) {
	newTenantForSecrets(t, "mi-os")

	SetSecret("mi-os", "A_KEY", "first")
	SetSecret("mi-os", "B_KEY", "second")
	n, err := SetSecret("mi-os", "A_KEY", "updated")
	if err != nil {
		t.Fatalf("SetSecret: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2 (update must not add a key)", n)
	}

	path, _ := SecretsPath("mi-os")
	entries, order, _ := readSecrets(path)
	if entries["A_KEY"] != "updated" {
		t.Errorf("A_KEY = %q, want updated", entries["A_KEY"])
	}
	if entries["B_KEY"] != "second" {
		t.Errorf("B_KEY = %q, want second (untouched)", entries["B_KEY"])
	}
	if len(order) != 2 || order[0] != "A_KEY" || order[1] != "B_KEY" {
		t.Errorf("order = %v, want [A_KEY B_KEY] preserved", order)
	}
}

// A secret for a tenant that does not exist is refused — we never create a
// stray vault outside the tenant lifecycle.
func TestSetSecretRejectsUnknownTenant(t *testing.T) {
	withTempHome(t)
	if _, err := SetSecret("no-existe", "K", "v"); err == nil {
		t.Fatal("expected an error for a nonexistent tenant")
	}
}

// Junk key names must be refused: a name with a newline or shell metachar could
// otherwise inject a second line into the env file.
func TestSetSecretRejectsBadKeyNames(t *testing.T) {
	newTenantForSecrets(t, "mi-os")
	for _, bad := range []string{"has space", "has-dash", "1LEADING", "with=eq", "inject\nEVIL", ""} {
		if _, err := SetSecret("mi-os", bad, "v"); err == nil {
			t.Errorf("expected rejection for key %q", bad)
		}
	}
}

// A blank value is refused: storing an empty string looks like a configured
// connection when it is not.
func TestSetSecretRejectsBlankValue(t *testing.T) {
	newTenantForSecrets(t, "mi-os")
	if _, err := SetSecret("mi-os", "ELEVENLABS_API_KEY", "   "); err == nil {
		t.Fatal("expected rejection for a blank value")
	}
}

// The written file must be shell-sourceable, because a local runtime sources it
// to use the key. Simulate `sh -c 'set -a; . file; echo $VAR'` semantics by
// checking the quoting a POSIX shell would need.
func TestSecretsFileIsSourceable(t *testing.T) {
	newTenantForSecrets(t, "mi-os")
	SetSecret("mi-os", "TRICKY", "a'b c$d")

	path, _ := SecretsPath("mi-os")
	data, _ := os.ReadFile(path)
	// The value line must single-quote and escape the embedded quote as '\'' .
	if !strings.Contains(string(data), `TRICKY='a'\''b c$d'`) {
		t.Errorf("value not shell-escaped correctly:\n%s", data)
	}
}

// The vault-leak guarantee still holds with secrets present: a tenant's Info
// (what gets serialized to agents/JSON) must never contain a secret value.
func TestSecretsNeverLeakIntoTenantInfo(t *testing.T) {
	newTenantForSecrets(t, "mi-os")
	SetSecret("mi-os", "ELEVENLABS_API_KEY", "xi-TOP-SECRET-VALUE")

	info := Inspect("mi-os", "")
	blob := info.Dir + info.Name + info.Slug + info.Kind + info.Owner
	if strings.Contains(blob, "TOP-SECRET") {
		t.Fatal("secret value leaked into tenant Info")
	}
}
