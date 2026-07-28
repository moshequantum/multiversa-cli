package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/tenant"
)

func createProviderTestTenant(t *testing.T, slug string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, _, err := tenant.New(slug, "Provider Test", "project-os"); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return home
}

func TestTenantConnectProviderStoresSecretAndPublicConfigSeparately(t *testing.T) {
	home := createProviderTestTenant(t, "provider-os")
	const secret = "gemini-secret-that-must-not-leak"
	var out bytes.Buffer
	cmd := newTenantConnectProviderCmd()
	cmd.SetIn(strings.NewReader(secret + "\n"))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"provider-os", "GeMiNi", "--model", "gemini-2.5-flash", "--priority", "2"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out.String(), secret) {
		t.Fatal("secret leaked to command output")
	}

	vaultData, err := os.ReadFile(filepath.Join(home, ".multiversa", "tenants", "provider-os", "vault", tenant.SecretsFileName))
	if err != nil {
		t.Fatalf("read vault: %v", err)
	}
	if !strings.Contains(string(vaultData), "GEMINI_API_KEY=") || !strings.Contains(string(vaultData), secret) {
		t.Fatalf("secret not stored under expected key: %s", vaultData)
	}
	configs, err := tenant.LoadProviderConfigs("provider-os")
	if err != nil {
		t.Fatalf("LoadProviderConfigs: %v", err)
	}
	if len(configs) != 1 || configs[0].Provider != "gemini" ||
		configs[0].Model != "gemini-2.5-flash" || configs[0].Priority != 2 || !configs[0].Enabled {
		t.Fatalf("unexpected configs: %+v", configs)
	}
	providerJSON, err := os.ReadFile(filepath.Join(home, ".multiversa", "tenants", "provider-os", "graph", "providers.json"))
	if err != nil {
		t.Fatalf("read providers.json: %v", err)
	}
	if strings.Contains(string(providerJSON), secret) {
		t.Fatal("secret leaked to providers.json")
	}
}

func TestTenantConnectProviderJSONNeverIncludesSecretValue(t *testing.T) {
	createProviderTestTenant(t, "provider-json")
	const secret = "groq-private-value"
	var out bytes.Buffer
	cmd := newTenantConnectProviderCmd()
	cmd.SetIn(strings.NewReader(secret))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"provider-json", "groq", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out.String(), secret) {
		t.Fatal("secret leaked to JSON")
	}
	var envelope agentout.Envelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if !envelope.OK || envelope.Schema != "multiversa.tenant-provider/v1" ||
		envelope.Command != "tenant-provider" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestTenantConnectProviderRejectsUnknownBeforeReadingStdin(t *testing.T) {
	createProviderTestTenant(t, "provider-invalid")
	reader := &countingReader{Reader: strings.NewReader("must-not-be-consumed")}
	cmd := newTenantConnectProviderCmd()
	cmd.SetIn(reader)
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{"provider-invalid", "unknown"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if reader.reads != 0 {
		t.Fatalf("stdin was consumed before provider validation: %d reads", reader.reads)
	}
}

type countingReader struct {
	*strings.Reader
	reads int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.Reader.Read(p)
}

func TestTenantConnectProviderRejectsEmptySecretWithoutConfig(t *testing.T) {
	home := createProviderTestTenant(t, "provider-empty")
	cmd := newTenantConnectProviderCmd()
	cmd.SetIn(strings.NewReader("\n"))
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{"provider-empty", "mistral"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected empty secret error")
	}
	if _, err := os.Stat(filepath.Join(home, ".multiversa", "tenants", "provider-empty", "graph", "providers.json")); !os.IsNotExist(err) {
		t.Fatalf("config written after empty secret: %v", err)
	}
}
