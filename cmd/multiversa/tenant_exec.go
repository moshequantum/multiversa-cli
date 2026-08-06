// multiversa tenant exec — run a process with one tenant's vault as its
// environment, and with the operator's ambient credentials stripped out.
//
// This is the piece that makes "un vault por perfil" true for everything that
// is not the LLM router: an MCP server, a deploy script, a migration, a curl.
// Before this, a backend key lived in one global place (~/.multiversa/secrets.env
// or a repo file) and every profile shared it. Now each profile carries its own,
// and a process launched for Cintia cannot see Andrea's key — not because it is
// asked not to look, but because it is not in its environment.
//
// Secrets go from the vault straight into the child's environment. They never
// touch argv, stdout, a log line, or the manifest.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/tenant"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// credentialShaped matches variable names that look like a credential. Anything
// matching is dropped from the inherited environment before the tenant's own
// vault is layered on, so an operator shell that exports INSFORGE_API_KEY for
// one project cannot silently leak it into another profile's process.
var credentialShaped = regexp.MustCompile(`(?i)(_API_KEY|_APIKEY|_TOKEN|_SECRET|_PASSWORD|_CREDENTIALS)$|^(API_KEY|APPKEY|HOTTOK)$`)

func newTenantExecCmd() *cobra.Command {
	var aliasSpecs []string
	var keep []string
	var quiet bool

	cmd := &cobra.Command{
		Use:   "exec <slug|.> -- <comando> [args...]",
		Short: "Ejecuta un comando con el vault de ese tenant como entorno.",
		Long: "Carga ~/.multiversa/tenants/<slug>/vault/secrets.env en el entorno del\n" +
			"proceso hijo y le quita al entorno heredado todo lo que parece una\n" +
			"credencial (*_API_KEY, *_TOKEN, *_SECRET, *_PASSWORD, API_KEY, HOTTOK…),\n" +
			"para que la clave global de otro perfil no se cuele.\n\n" +
			"Usa `.` como slug para el tenant activo (`multiversa tenant use`).\n\n" +
			"Los valores nunca se imprimen: van del vault al entorno del hijo y nada más.\n\n" +
			"Ejemplos:\n" +
			"  multiversa tenant exec cintia-larizzati -- ./scripts/deploy-functions.sh\n" +
			"  multiversa tenant exec . --alias INSFORGE_API_KEY=API_KEY -- bunx @insforge/mcp@latest\n\n" +
			"--alias VAULT_KEY=NOMBRE_ESPERADO adapta el nombre cuando el programa hijo\n" +
			"espera una variable genérica (API_KEY) y el vault guarda una específica\n" +
			"(INSFORGE_API_KEY) para que varios backends convivan en un mismo perfil.",
		Args: cobra.MinimumNArgs(2),
		// The child owns its own error reporting; a non-zero exit from it is not
		// a usage error of ours.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if slug == "." || slug == "-" {
				slug = tenant.Active()
				if slug == "" {
					return errors.New("no hay tenant activo — elige uno con `multiversa tenant use <slug>` o pasa el slug")
				}
			}

			argv := args[1:]
			if cmd.ArgsLenAtDash() > 0 {
				argv = args[cmd.ArgsLenAtDash():]
			}
			if len(argv) == 0 {
				return errors.New("falta el comando a ejecutar — escríbelo después de `--`")
			}

			aliases, err := parseAliases(aliasSpecs)
			if err != nil {
				return err
			}

			vaultEnv, names, err := tenant.Environ(slug, aliases)
			if err != nil {
				return err
			}
			if len(vaultEnv) == 0 {
				return fmt.Errorf("el vault de %q está vacío — guarda sus claves con `multiversa tenant set-secret %s <KEY>`", slug, slug)
			}

			bin, err := exec.LookPath(argv[0])
			if err != nil {
				return fmt.Errorf("no encuentro el ejecutable %q: %w", argv[0], err)
			}

			child := exec.Command(bin, argv[1:]...)
			child.Env = append(sanitizedEnviron(names, keep), vaultEnv...)
			child.Env = append(child.Env, "MULTIVERSA_TENANT="+slug)
			child.Stdin = cmd.InOrStdin()
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr

			if !quiet {
				// stderr, never stdout: this command wraps stdio protocols (MCP)
				// where a stray line on stdout corrupts the stream. Names only.
				fmt.Fprintln(os.Stderr, theme.Dim.Render(
					fmt.Sprintf("· vault %s → %s (%s)", slug, argv[0], strings.Join(names, ", "))))
			}

			if err := child.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&aliasSpecs, "alias", nil, "Renombra una clave del vault para el hijo: VAULT_KEY=NOMBRE_ESPERADO. Repetible.")
	cmd.Flags().StringArrayVar(&keep, "keep", nil, "Conserva una variable heredada aunque parezca credencial (ej. GITHUB_TOKEN). Repetible.")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "No escribir la nota de qué claves se inyectaron.")
	return cmd
}

// parseAliases turns --alias VAULT_KEY=NOMBRE into a map, rejecting anything
// ambiguous rather than guessing.
func parseAliases(specs []string) (map[string]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(specs))
	for _, spec := range specs {
		src, dst, ok := strings.Cut(spec, "=")
		src, dst = strings.TrimSpace(src), strings.TrimSpace(dst)
		if !ok || src == "" || dst == "" {
			return nil, fmt.Errorf("--alias %q inválido — usa VAULT_KEY=NOMBRE_ESPERADO", spec)
		}
		if prev, dup := out[src]; dup {
			return nil, fmt.Errorf("--alias repetido para %q (%s y %s)", src, prev, dst)
		}
		out[src] = dst
	}
	return out, nil
}

// sanitizedEnviron copies the operator's environment minus anything that looks
// like a credential, minus the names the vault is about to define. `keep` is the
// escape hatch for a variable the child genuinely needs from the shell.
func sanitizedEnviron(vaultNames, keep []string) []string {
	drop := make(map[string]bool, len(vaultNames))
	for _, n := range vaultNames {
		drop[n] = true
	}
	keepSet := make(map[string]bool, len(keep))
	for _, k := range keep {
		keepSet[strings.TrimSpace(k)] = true
	}

	var out []string
	for _, kv := range os.Environ() {
		name, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if keepSet[name] {
			out = append(out, kv)
			continue
		}
		if drop[name] || credentialShaped.MatchString(name) {
			continue
		}
		out = append(out, kv)
	}
	return out
}
