package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/tenant"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

const maxProviderSecretBytes = 16 * 1024

// newTenantConnectProviderCmd connects one supported LLM provider without ever
// accepting its API key as an argument. The secret travels only from stdin to
// the tenant vault; stdout contains provider metadata, never credential data.
func newTenantConnectProviderCmd() *cobra.Command {
	var jsonOut bool
	var model string
	var priority int

	cmd := &cobra.Command{
		Use:     "connect <slug> <provider>",
		Aliases: []string{"connect-provider"},
		Short:   "Conecta un proveedor de IA; lee su clave por STDIN y la guarda en el vault.",
		Long: "Conecta Gemini, Mistral o Groq a un tenant. La API key se lee de STDIN,\n" +
			"nunca de argumentos, y se guarda únicamente en vault/secrets.env. La\n" +
			"configuración no secreta se registra en graph/providers.json.\n\n" +
			"Ejemplo:\n" +
			"  printf %s \"$GEMINI_API_KEY\" | multiversa tenant connect mi-os gemini",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			provider := strings.ToLower(strings.TrimSpace(args[1]))

			// Validate all non-secret input before consuming stdin or touching
			// the vault. ConnectProvider performs the definitive validation too.
			if _, ok := tenant.LookupProvider(provider); !ok {
				return fmt.Errorf("proveedor %q no soportado — usa gemini, mistral o groq", provider)
			}
			if priority < 0 {
				return fmt.Errorf("priority inválida %d: debe ser cero o mayor", priority)
			}
			if _, _, err := tenant.Load(slug); err != nil {
				return fmt.Errorf("no se pudo conectar el proveedor: %w", err)
			}
			secretKey, err := tenant.ProviderSecretKey(provider)
			if err != nil {
				return err
			}
			var raw []byte
			in := cmd.InOrStdin()
			if file, ok := in.(*os.File); ok && term.IsTerminal(file.Fd()) {
				fmt.Fprintf(cmd.ErrOrStderr(), "Clave %s (entrada oculta): ", secretKey)
				raw, err = term.ReadPassword(file.Fd())
				fmt.Fprintln(cmd.ErrOrStderr())
			} else {
				raw, err = io.ReadAll(io.LimitReader(in, maxProviderSecretBytes+1))
			}
			if err != nil {
				return fmt.Errorf("no se pudo leer la clave de STDIN: %w", err)
			}
			if len(raw) > maxProviderSecretBytes {
				return fmt.Errorf("la clave de %s excede el límite de 16 KiB", provider)
			}
			secret := strings.TrimRight(string(raw), "\r\n")

			count, err := tenant.SetSecret(slug, secretKey, secret)
			if err != nil {
				return fmt.Errorf("no se pudo guardar la clave de %s: %w", provider, err)
			}
			cfg := tenant.ProviderConfig{
				Provider: provider,
				Model:    strings.TrimSpace(model),
				Priority: priority,
				Enabled:  true,
			}
			if err := tenant.ConnectProvider(slug, cfg); err != nil {
				return fmt.Errorf("la clave quedó segura en el vault, pero no se pudo guardar la configuración de %s: %w", provider, err)
			}

			if jsonOut {
				return agentout.Emit(cmd.OutOrStdout(), "tenant-provider", struct {
					Action   string                `json:"action"`
					Slug     string                `json:"slug"`
					Provider tenant.ProviderConfig `json:"provider"`
					Secret   string                `json:"secret_key"`
					Count    int                   `json:"vault_key_count"`
				}{
					Action: "provider_connected", Slug: slug, Provider: cfg,
					Secret: secretKey, Count: count,
				})
			}
			fmt.Fprintln(cmd.OutOrStdout(), theme.Accent.Render("✓ proveedor conectado · "+provider))
			fmt.Fprintln(cmd.OutOrStdout(), theme.Dim.Render("  tenant: "+slug+" · secreto: "+secretKey+" · valor nunca expuesto"))
			return nil
		},
	}
	cmd.Flags().StringVar(&model, "model", "", "Modelo preferido del proveedor (opcional).")
	cmd.Flags().IntVar(&priority, "priority", 0, "Prioridad de fallback; menor se intenta primero.")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.tenant-provider/v1).")
	return cmd
}
