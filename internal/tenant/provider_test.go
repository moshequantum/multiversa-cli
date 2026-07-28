package tenant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderRegistry(t *testing.T) {
	cases := map[string]struct{ key, backend string }{
		"gemini":  {"GEMINI_API_KEY", "gemini"},
		"mistral": {"MISTRAL_API_KEY", "openai"},
		"groq":    {"GROQ_API_KEY", "openai"},
	}
	for name, want := range cases {
		spec, ok := LookupProvider(name)
		if !ok || spec.SecretKey != want.key || spec.Backend != want.backend {
			t.Fatalf("LookupProvider(%q) = %#v, %v", name, spec, ok)
		}
	}
	if _, ok := LookupProvider("unknown"); ok {
		t.Fatal("unknown provider accepted")
	}
}

func TestProviderConfigContainsNoSecretAndRuntimeMapsCompat(t *testing.T) {
	home := withTempHome(t)
	if _, _, err := New("client", "Client", "project-os"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetSecret("client", "MISTRAL_API_KEY", "mistral-secret"); err != nil {
		t.Fatal(err)
	}
	if err := ConnectProvider("client", ProviderConfig{Provider: "mistral", Model: "mistral-large-latest", Enabled: true, Priority: 1}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(home, ".multiversa", "tenants", "client", "graph", ProvidersFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "mistral-secret") {
		t.Fatal("provider config leaked secret")
	}
	plans, err := BuildProviderFallback("client", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Backend != "openai" || plans[0].Model != "mistral-large-latest" {
		t.Fatalf("plans=%#v", plans)
	}
	env := strings.Join(plans[0].Env, "\n")
	for _, want := range []string{"OPENAI_API_KEY=mistral-secret", "OPENAI_BASE_URL=https://api.mistral.ai/v1", "OPENAI_MODEL=mistral-large-latest"} {
		if !strings.Contains(env, want) {
			t.Fatalf("env missing %q: %q", want, env)
		}
	}
}

func TestProviderFallbackSkipsMissingKeysAndHonorsPreference(t *testing.T) {
	withTempHome(t)
	if _, _, err := New("client", "Client", "project-os"); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"gemini", "mistral", "groq"} {
		if err := ConnectProvider("client", ProviderConfig{Provider: name, Enabled: true, Priority: i}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := SetSecret("client", "GEMINI_API_KEY", "g"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetSecret("client", "GROQ_API_KEY", "q"); err != nil {
		t.Fatal(err)
	}
	plans, err := BuildProviderFallback("client", []string{"groq", "mistral", "gemini"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 2 || plans[0].Provider != "groq" || plans[1].Provider != "gemini" {
		t.Fatalf("plans=%#v", plans)
	}
	if !strings.Contains(strings.Join(plans[1].Env, "\n"), "GEMINI_API_KEY=g") {
		t.Fatalf("gemini env=%#v", plans[1].Env)
	}
}

func TestProviderFallbackRejectsReadableSecretFile(t *testing.T) {
	withTempHome(t)
	if _, _, err := New("client", "Client", "project-os"); err != nil {
		t.Fatal(err)
	}
	if err := ConnectProvider("client", ProviderConfig{Provider: "gemini", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := SetSecret("client", "GEMINI_API_KEY", "g"); err != nil {
		t.Fatal(err)
	}
	path, _ := SecretsPath("client")
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildProviderFallback("client", nil); err == nil || !strings.Contains(err.Error(), "0600") {
		t.Fatalf("expected permissions error, got %v", err)
	}
}
