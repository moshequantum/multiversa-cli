package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseAliases(t *testing.T) {
	ok, err := parseAliases([]string{"INSFORGE_API_KEY=API_KEY", " MANYCHAT_API_KEY = MC_TOKEN "})
	if err != nil {
		t.Fatalf("alias válido rechazado: %v", err)
	}
	if ok["INSFORGE_API_KEY"] != "API_KEY" || ok["MANYCHAT_API_KEY"] != "MC_TOKEN" {
		t.Fatalf("alias mal parseado: %#v", ok)
	}

	for _, bad := range []string{"SIN_IGUAL", "=API_KEY", "VAULT_KEY="} {
		if _, err := parseAliases([]string{bad}); err == nil {
			t.Errorf("se aceptó un alias inválido: %q", bad)
		}
	}
	if _, err := parseAliases([]string{"K=A", "K=B"}); err == nil {
		t.Error("se aceptó un alias duplicado para la misma clave del vault")
	}
}

// The point of the whole command: a credential exported in the operator's shell
// must not reach a tenant's child process, whether or not that tenant's vault
// happens to define the same name.
func TestSanitizedEnvironDropsAmbientCredentials(t *testing.T) {
	t.Setenv("INSFORGE_API_KEY", "clave-global-de-otro-perfil")
	t.Setenv("API_KEY", "clave-generica")
	t.Setenv("HOTMART_WEBHOOK_SECRET", "hottok-viejo")
	t.Setenv("GITHUB_TOKEN", "token-de-la-terminal")
	t.Setenv("PATH", os.Getenv("PATH"))
	t.Setenv("MULTIVERSA_HARMLESS", "sí")

	env := sanitizedEnviron([]string{"GEMINI_API_KEY"}, []string{"GITHUB_TOKEN"})

	names := map[string]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		names[k] = v
	}

	for _, leaked := range []string{"INSFORGE_API_KEY", "API_KEY", "HOTMART_WEBHOOK_SECRET", "GEMINI_API_KEY"} {
		if _, present := names[leaked]; present {
			t.Errorf("%s se filtró al entorno del hijo", leaked)
		}
	}
	if names["GITHUB_TOKEN"] != "token-de-la-terminal" {
		t.Error("--keep no conservó GITHUB_TOKEN")
	}
	if names["MULTIVERSA_HARMLESS"] != "sí" {
		t.Error("se descartó una variable que no es credencial")
	}
	if names["PATH"] == "" {
		t.Error("se descartó PATH: el hijo no podría ni arrancar")
	}
}

func TestCredentialShapedMatchesTheUsualSuspects(t *testing.T) {
	shaped := []string{"INSFORGE_API_KEY", "RESEND_TOKEN", "HOTMART_WEBHOOK_SECRET", "DB_PASSWORD", "GOOGLE_APPLICATION_CREDENTIALS", "API_KEY", "HOTTOK", "manychat_api_key"}
	for _, n := range shaped {
		if !credentialShaped.MatchString(n) {
			t.Errorf("%s debería tratarse como credencial", n)
		}
	}
	plain := []string{"PATH", "HOME", "INSFORGE_BASE_URL", "MULTIVERSA_TENANT", "KEYBOARD"}
	for _, n := range plain {
		if credentialShaped.MatchString(n) {
			t.Errorf("%s NO es una credencial y se estaría descartando", n)
		}
	}
}
