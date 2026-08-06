package tenant

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SecretsFileName is the env-style file inside a tenant's vault that holds its
// connection keys (ElevenLabs voice, its own backend, …). It lives inside the
// 0700 vault and is itself 0600. It is a real .env file so a local runtime can
// source it; it is never serialized into the manifest, synced, or shown to an
// agent — that is the vault invariant, enforced by the vault-leak test.
const SecretsFileName = "secrets.env"

// envKeyRe constrains secret names to POSIX-ish env var identifiers, so a name
// can never inject a newline or shell metacharacter into the file.
var envKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// SecretsPath returns the path to a tenant's vault secrets file. It does not
// require the file to exist.
func SecretsPath(slug string) (string, error) {
	dir, err := Dir(slug)
	if err != nil {
		return "", err
	}
	// The vault path comes from the manifest, but every template uses "vault"
	// and New creates exactly that; fall back to it without loading the whole
	// manifest just to read one field.
	return filepath.Join(dir, "vault", SecretsFileName), nil
}

// SetSecret writes or updates one KEY=value entry in the tenant's vault
// secrets file, creating it 0600 inside the 0700 vault. Existing entries are
// preserved; a repeated key is updated in place. Returns the number of keys
// stored afterward.
//
// The value is treated as opaque and is never logged by this package. Callers
// (the CLI command, the installer) must not print it either.
func SetSecret(slug, key, value string) (int, error) {
	key = strings.TrimSpace(key)
	if !envKeyRe.MatchString(key) {
		return 0, fmt.Errorf("nombre de secreto inválido %q — usa MAYÚSCULAS, números y guion bajo (ej. ELEVENLABS_API_KEY)", key)
	}
	if strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("el secreto %q está vacío — no se guarda una clave en blanco", key)
	}
	if len(value) > 16*1024 {
		return 0, fmt.Errorf("el secreto %q excede el límite seguro de 16 KiB", key)
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return 0, fmt.Errorf("el secreto %q contiene caracteres de control no permitidos", key)
	}

	dir, err := Dir(slug)
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("el tenant %q no existe — créalo antes de guardarle secretos", slug)
		}
		return 0, err
	}

	vaultDir := filepath.Join(dir, "vault")
	// Defensive: New already makes this 0700, but a hand-moved profile might
	// not have it. Never widen the perms if it is already there.
	if err := os.MkdirAll(vaultDir, 0o700); err != nil {
		return 0, err
	}

	path := filepath.Join(vaultDir, SecretsFileName)
	entries, order, err := readSecrets(path)
	if err != nil {
		return 0, err
	}
	if _, exists := entries[key]; !exists {
		order = append(order, key)
	}
	entries[key] = value

	if err := writeSecrets(path, entries, order); err != nil {
		return 0, err
	}
	return len(order), nil
}

// SecretNames returns the names — never the values — of the secrets stored in a
// tenant's vault, in the order they were added. This is what any surface that
// needs to *show* the vault uses: the installer's review step, `tenant show`,
// an agent asking "is this profile connected?". A missing vault is not an
// error; it is an empty list.
func SecretNames(slug string) ([]string, error) {
	path, err := SecretsPath(slug)
	if err != nil {
		return nil, err
	}
	_, order, err := readSecrets(path)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// Environ renders a tenant's vault as KEY=value pairs ready to hand to a child
// process, plus the list of names it injected (for a human-readable note; the
// values are never returned to a caller that only wants to report).
//
// aliases maps a vault key to the name the child expects — e.g. the InsForge
// MCP server reads API_KEY, while the vault stores INSFORGE_API_KEY so several
// backends can coexist in one profile. An alias adds a name, it does not remove
// the original.
//
// The returned slice contains secrets. Callers must pass it straight to
// exec.Cmd.Env and never log, print, or serialize it.
func Environ(slug string, aliases map[string]string) (env []string, names []string, err error) {
	path, err := SecretsPath(slug)
	if err != nil {
		return nil, nil, err
	}
	entries, order, err := readSecrets(path)
	if err != nil {
		return nil, nil, err
	}
	for src, dst := range aliases {
		if _, ok := entries[src]; !ok {
			return nil, nil, fmt.Errorf("el vault de %q no tiene %q — guárdalo con: printf %%s \"$VALOR\" | multiversa tenant set-secret %s %s", slug, src, slug, src)
		}
		if !envKeyRe.MatchString(dst) {
			return nil, nil, fmt.Errorf("alias inválido %q — el destino debe ser un nombre de variable de entorno", dst)
		}
	}
	for _, k := range order {
		env = append(env, k+"="+entries[k])
		names = append(names, k)
	}
	// Aliases are applied after the originals so a rename always wins over an
	// identically-named vault entry.
	for src, dst := range aliases {
		env = append(env, dst+"="+entries[src])
		names = append(names, dst)
	}
	sort.Strings(names)
	return env, names, nil
}

// readSecrets parses an existing secrets file into a map plus the key order, so
// rewriting preserves the order the user added things in. A missing file is not
// an error — it just means no secrets yet.
func readSecrets(path string) (map[string]string, []string, error) {
	entries := map[string]string{}
	var order []string

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return entries, order, nil
		}
		return nil, nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue // not a KEY=value line; skip rather than choke
		}
		key := strings.TrimSpace(line[:eq])
		if !envKeyRe.MatchString(key) {
			continue
		}
		if _, seen := entries[key]; !seen {
			order = append(order, key)
		}
		entries[key] = unquote(strings.TrimSpace(line[eq+1:]))
	}
	return entries, order, sc.Err()
}

// writeSecrets rewrites the file atomically (temp + rename) so a crash mid-write
// never leaves a half-written secrets file, and forces 0600 on the result.
func writeSecrets(path string, entries map[string]string, order []string) error {
	var b strings.Builder
	b.WriteString("# Secretos de este tenant — generados por MultiversaOS.\n")
	b.WriteString("# Formato .env: se pueden `source`ear. Nunca se sincroniza ni se expone.\n\n")
	for _, k := range order {
		fmt.Fprintf(&b, "%s=%s\n", k, singleQuote(entries[k]))
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".secrets-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if the rename succeeded

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(b.String()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// singleQuote wraps a value so it survives shell sourcing intact, escaping any
// embedded single quote the POSIX way ('\” ). API keys rarely contain quotes,
// but a secret store must not corrupt a value that does.
func singleQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}

// unquote reverses singleQuote for reading back. It also tolerates unquoted and
// double-quoted values in case a file was hand-edited.
func unquote(v string) string {
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		return strings.ReplaceAll(v[1:len(v)-1], `'\''`, "'")
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return v
}
