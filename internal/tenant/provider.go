package tenant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ProvidersFileName = "providers.json"

type ProviderSpec struct {
	Name         string
	SecretKey    string
	Backend      string
	DefaultModel string
	BaseURL      string
}

type ProviderConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Enabled  bool   `json:"enabled"`
}

type ProviderPlan struct {
	Provider string   `json:"provider"`
	Backend  string   `json:"backend"`
	Model    string   `json:"model"`
	Env      []string `json:"-"`
}

var providerRegistry = map[string]ProviderSpec{
	"gemini":  {Name: "gemini", SecretKey: "GEMINI_API_KEY", Backend: "gemini", DefaultModel: "gemini-2.5-flash"},
	"mistral": {Name: "mistral", SecretKey: "MISTRAL_API_KEY", Backend: "openai", DefaultModel: "mistral-small-latest", BaseURL: "https://api.mistral.ai/v1"},
	"groq":    {Name: "groq", SecretKey: "GROQ_API_KEY", Backend: "openai", DefaultModel: "llama-3.3-70b-versatile", BaseURL: "https://api.groq.com/openai/v1"},
}

// llmEnvironmentKeys is cleared for every provider attempt so credentials or
// routing inherited from the operator shell cannot shadow the tenant plan.
var llmEnvironmentKeys = []string{
	"GEMINI_API_KEY", "GOOGLE_API_KEY", "MISTRAL_API_KEY", "GROQ_API_KEY",
	"OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL", "GRAPHIFY_OPENAI_MODEL",
	"GRAPHIFY_GEMINI_MODEL",
	"ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "DEEPSEEK_API_KEY", "DEEPSEEK_BASE_URL",
	"MOONSHOT_API_KEY", "KIMI_BASE_URL", "AZURE_OPENAI_API_KEY", "AZURE_OPENAI_ENDPOINT",
	"OLLAMA_API_KEY", "OLLAMA_BASE_URL",
}

func LookupProvider(name string) (ProviderSpec, bool) {
	spec, ok := providerRegistry[strings.ToLower(strings.TrimSpace(name))]
	return spec, ok
}

func ProviderSecretKey(name string) (string, error) {
	spec, ok := LookupProvider(name)
	if !ok {
		return "", fmt.Errorf("proveedor LLM inválido %q: usa gemini, mistral o groq", name)
	}
	return spec.SecretKey, nil
}

func ProviderNames() []string { return []string{"gemini", "mistral", "groq"} }

func providersPath(slug string) (string, error) {
	dir, err := Dir(slug)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "graph", ProvidersFileName), nil
}

// ConnectProvider stores only routing metadata. API keys always remain in vault/secrets.env.
func ConnectProvider(slug string, cfg ProviderConfig) error {
	spec, ok := LookupProvider(cfg.Provider)
	if !ok {
		return fmt.Errorf("proveedor LLM inválido %q: usa gemini, mistral o groq", cfg.Provider)
	}
	cfg.Provider = spec.Name
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = spec.DefaultModel
	}
	configs, err := LoadProviderConfigs(slug)
	if err != nil {
		return err
	}
	found := false
	for i := range configs {
		if configs[i].Provider == cfg.Provider {
			configs[i] = cfg
			found = true
			break
		}
	}
	if !found {
		configs = append(configs, cfg)
	}
	sort.Slice(configs, func(i, j int) bool {
		if configs[i].Priority == configs[j].Priority {
			return configs[i].Provider < configs[j].Provider
		}
		return configs[i].Priority < configs[j].Priority
	})
	path, err := providersPath(slug)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return atomicWrite(path, b, 0600)
}

func LoadProviderConfigs(slug string) ([]ProviderConfig, error) {
	path, err := providersPath(slug)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []ProviderConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	var configs []ProviderConfig
	if err := json.Unmarshal(b, &configs); err != nil {
		return nil, fmt.Errorf("leer configuración de proveedores: %w", err)
	}
	seen := map[string]bool{}
	for i := range configs {
		spec, ok := LookupProvider(configs[i].Provider)
		if !ok {
			return nil, fmt.Errorf("proveedor LLM inválido en configuración: %q", configs[i].Provider)
		}
		configs[i].Provider = spec.Name
		if seen[spec.Name] {
			return nil, fmt.Errorf("proveedor duplicado en configuración: %s", spec.Name)
		}
		seen[spec.Name] = true
	}
	sort.SliceStable(configs, func(i, j int) bool { return configs[i].Priority < configs[j].Priority })
	return configs, nil
}

// BuildProviderFallback resolves enabled providers in requested/configured order and
// injects tenant-local keys only into the child process environment.
func BuildProviderFallback(slug string, preferred []string) ([]ProviderPlan, error) {
	configs, err := LoadProviderConfigs(slug)
	if err != nil {
		return nil, err
	}
	byName := map[string]ProviderConfig{}
	for _, c := range configs {
		byName[c.Provider] = c
	}
	var order []string
	if len(preferred) > 0 {
		order = preferred
	} else {
		for _, c := range configs {
			if c.Enabled {
				order = append(order, c.Provider)
			}
		}
	}
	path, err := SecretsPath(slug)
	if err != nil {
		return nil, err
	}
	secrets, err := readProviderSecretsSecure(path)
	if err != nil {
		return nil, fmt.Errorf("leer vault local: %w", err)
	}
	plans := make([]ProviderPlan, 0, len(order))
	used := map[string]bool{}
	for _, raw := range order {
		spec, ok := LookupProvider(raw)
		if !ok {
			return nil, fmt.Errorf("proveedor LLM inválido %q: usa gemini, mistral o groq", raw)
		}
		if used[spec.Name] {
			continue
		}
		used[spec.Name] = true
		cfg, configured := byName[spec.Name]
		if configured && !cfg.Enabled {
			continue
		}
		key := strings.TrimSpace(secrets[spec.SecretKey])
		if key == "" {
			continue
		}
		model := cfg.Model
		if model == "" {
			model = spec.DefaultModel
		}
		env := make([]string, 0, len(llmEnvironmentKeys)+4)
		for _, envKey := range llmEnvironmentKeys {
			env = append(env, envKey+"=")
		}
		env = append(env, spec.SecretKey+"="+key)
		if spec.Backend == "openai" {
			env = append(env, "OPENAI_API_KEY="+key, "OPENAI_BASE_URL="+spec.BaseURL, "OPENAI_MODEL="+model)
		}
		plans = append(plans, ProviderPlan{Provider: spec.Name, Backend: spec.Backend, Model: model, Env: env})
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("ningún proveedor configurado tiene una clave local disponible en %s", path)
	}
	return plans, nil
}

// readProviderSecretsSecure refuses links and group/world-readable vault data.
// Provider credentials are deliberately never inherited from the host env.
func readProviderSecretsSecure(path string) (map[string]string, error) {
	vault := filepath.Dir(path)
	fi, err := os.Lstat(vault)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() || fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("vault inseguro %s: requiere directorio privado 0700", vault)
	}
	fi, err = os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("archivo de secretos inseguro %s: debe ser un archivo regular", path)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("archivo de secretos inseguro %s: requiere permisos 0600", path)
	}
	entries, _, err := readSecrets(path)
	return entries, err
}
