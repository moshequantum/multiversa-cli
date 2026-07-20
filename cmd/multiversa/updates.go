// multiversa updates — the release watcher for the curated ecosystem.
//
// Checks the latest upstream release of every curated engine plus the
// CLI itself and reports what has a newer version. It NEVER updates
// anything: the report is the proposal, applying it is the human's
// decision. `updates cron` manages an optional crontab entry so the
// check runs daily and the user finds the report in ~/.multiversa.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/credits"
	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/upstream"
	"github.com/moshequantum/multiversa-cli/internal/version"
)

// cronMarker tags the crontab line we own so install/remove/status are
// idempotent and auditable — we never touch any other line.
const cronMarker = "# multiversa-updates"

// updatesJSON is the payload of `multiversa updates --json`
// (schema multiversa.updates/v1).
type updatesJSON struct {
	CheckedAt        string                 `json:"checked_at"`
	UpdatesAvailable int                    `json:"updates_available"`
	Components       []upstream.ReleaseInfo `json:"components"`
}

func newUpdatesCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "updates",
		Short: "Consulta los últimos releases del stack curado y reporta qué actualizar.",
		Long: "Consulta el último release publicado de cada motor curado y del propio\n" +
			"CLI contra las versiones instaladas localmente. Solo lectura: reporta,\n" +
			"nunca actualiza. Aplicar la actualización es siempre decisión humana.\n\n" +
			"Usa `multiversa updates cron` para programar el chequeo diario.",
		RunE: func(cmd *cobra.Command, args []string) error {
			results := upstream.Check(watchedComponents(), 15*time.Second)

			available := 0
			for _, r := range results {
				if r.UpdateAvailable {
					available++
				}
			}

			if jsonOut {
				return agentout.Emit(os.Stdout, "updates", updatesJSON{
					CheckedAt:        time.Now().UTC().Format(time.RFC3339),
					UpdatesAvailable: available,
					Components:       results,
				})
			}

			fmt.Println(theme.Accent.Render("multiversa updates") + theme.Dim.Render(" · vigilante de releases"))
			for _, r := range results {
				switch {
				case r.Error != "":
					fmt.Printf("  %-12s %s\n", r.ID, theme.Dim.Render("· "+r.Error))
				case r.UpdateAvailable:
					fmt.Printf("  %-12s %s\n", r.ID,
						theme.Warn.Render("⬆ "+orDash(r.Installed)+" → "+r.Latest)+"  "+theme.Dim.Render(r.ReleaseURL))
				case r.Installed == "":
					fmt.Printf("  %-12s %s\n", r.ID, theme.Dim.Render("· no instalado · último: "+orDash(r.Latest)))
				default:
					fmt.Printf("  %-12s %s\n", r.ID, theme.Accent.Render("✓ al día ")+theme.Body.Render(r.Latest))
				}
			}
			if available > 0 {
				fmt.Println(theme.Body.Render("\n" + fmt.Sprintf("%d actualización(es) disponible(s). La IA propone, tú decides.", available)))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit the release report as a JSON envelope (schema multiversa.updates/v1).")
	cmd.AddCommand(newUpdatesCronCmd())
	return cmd
}

// watchedComponents builds the watch list: the CLI itself first, then
// every curated engine from the attribution registry, wired to the
// locally detected version when we have one.
func watchedComponents() []upstream.Component {
	report := detect.Run()
	installedByID := map[string]string{}
	for _, e := range report.Multiversa.Engines {
		if e.Installed {
			v := e.Version
			if v == "" {
				v = "instalado"
			}
			installedByID[e.ID] = v
		}
	}

	comps := []upstream.Component{{
		ID:        "multiversa-cli",
		Name:      "Multiversa CLI",
		Repo:      "https://github.com/moshequantum/multiversa-cli",
		Installed: version.Version,
	}}
	for _, s := range credits.Sources {
		comps = append(comps, upstream.Component{
			ID:        s.ID,
			Name:      s.Name,
			Repo:      s.Repo,
			Installed: installedByID[s.ID],
		})
	}
	return comps
}

// cronJSON is the payload of `multiversa updates cron --json`
// (schema multiversa.updates-cron/v1).
type cronJSON struct {
	Supported bool   `json:"supported"`
	Installed bool   `json:"installed"`
	Entry     string `json:"entry"`
	LogPath   string `json:"log_path"`
	Action    string `json:"action"` // "status" | "installed" | "removed" | "proposed"
}

func newUpdatesCronCmd() *cobra.Command {
	var jsonOut, apply, remove bool
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Propone (o instala con --apply) el chequeo diario de releases vía crontab.",
		Long: "Gestiona la entrada de crontab que ejecuta `multiversa updates --json`\n" +
			"una vez al día y guarda el reporte en ~/.multiversa/updates.log.\n\n" +
			"Sin flags: muestra el estado y la línea propuesta (no modifica nada).\n" +
			"--apply instala la entrada. --remove la elimina. Solo se toca la línea\n" +
			"marcada con \"" + cronMarker + "\" — el resto del crontab es intocable.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS == "windows" {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "updates-cron", "unsupported_platform",
						"crontab no existe en Windows",
						"Usa el Programador de tareas de Windows con: multiversa updates --json")
					os.Exit(1)
				}
				return fmt.Errorf("crontab no existe en Windows — usa el Programador de tareas con `multiversa updates --json`")
			}

			entry := cronEntry()
			logPath := updatesLogPath()

			switch {
			case apply && remove:
				return fmt.Errorf("--apply y --remove son excluyentes")
			case apply:
				if err := installCronEntry(entry); err != nil {
					if jsonOut {
						_ = agentout.EmitError(os.Stdout, "updates-cron", "crontab_failed", err.Error(), "")
						os.Exit(1)
					}
					return err
				}
				return emitCron(jsonOut, cronJSON{Supported: true, Installed: true, Entry: entry, LogPath: logPath, Action: "installed"},
					theme.Accent.Render("✓ cron instalado")+" — chequeo diario 09:00, reporte en "+logPath)
			case remove:
				if err := removeCronEntry(); err != nil {
					if jsonOut {
						_ = agentout.EmitError(os.Stdout, "updates-cron", "crontab_failed", err.Error(), "")
						os.Exit(1)
					}
					return err
				}
				return emitCron(jsonOut, cronJSON{Supported: true, Installed: false, Entry: entry, LogPath: logPath, Action: "removed"},
					theme.Accent.Render("✓ cron eliminado")+" — el chequeo diario queda desactivado")
			default:
				installed := cronEntryInstalled()
				status := "no instalado — propuesta:\n  " + entry + "\n\nInstalar: multiversa updates cron --apply"
				if installed {
					status = theme.Accent.Render("✓ instalado") + " — reporte diario en " + logPath
				}
				return emitCron(jsonOut, cronJSON{Supported: true, Installed: installed, Entry: entry, LogPath: logPath, Action: "status"},
					theme.Accent.Render("multiversa updates cron")+"\n"+status)
			}
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.updates-cron/v1).")
	cmd.Flags().BoolVar(&apply, "apply", false, "Instala la entrada de crontab (decisión humana explícita).")
	cmd.Flags().BoolVar(&remove, "remove", false, "Elimina la entrada de crontab gestionada por Multiversa.")
	return cmd
}

func emitCron(jsonOut bool, payload cronJSON, human string) error {
	if jsonOut {
		return agentout.Emit(os.Stdout, "updates-cron", payload)
	}
	fmt.Println(human)
	return nil
}

func updatesLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.multiversa/updates.log"
	}
	return filepath.Join(home, ".multiversa", "updates.log")
}

// cronEntry renders the exact crontab line we manage. Daily at 09:00
// local: frequent enough to catch releases, quiet enough to not spam.
func cronEntry() string {
	bin, err := os.Executable()
	if err != nil || bin == "" {
		bin = "multiversa"
	}
	return fmt.Sprintf("0 9 * * * %s updates --json >> %s 2>&1 %s", bin, updatesLogPath(), cronMarker)
}

// currentCrontab returns the user's crontab lines. A missing crontab
// ("no crontab for user") is an empty list, not an error.
func currentCrontab() ([]string, error) {
	out, err := exec.Command("crontab", "-l").CombinedOutput()
	if err != nil {
		if strings.Contains(strings.ToLower(string(out)), "no crontab") {
			return nil, nil
		}
		return nil, fmt.Errorf("crontab -l: %v: %s", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}

func cronEntryInstalled() bool {
	lines, err := currentCrontab()
	if err != nil {
		return false
	}
	for _, l := range lines {
		if strings.Contains(l, cronMarker) {
			return true
		}
	}
	return false
}

// writeCrontab replaces the user's crontab with the given lines.
func writeCrontab(lines []string) error {
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("crontab -: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// installCronEntry adds (or refreshes) our marked line, preserving
// every other crontab line byte-for-byte.
func installCronEntry(entry string) error {
	lines, err := currentCrontab()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(updatesLogPath()), 0o755); err != nil {
		return err
	}
	kept := lines[:0]
	for _, l := range lines {
		if !strings.Contains(l, cronMarker) {
			kept = append(kept, l)
		}
	}
	return writeCrontab(append(kept, entry))
}

// removeCronEntry drops our marked line and nothing else.
func removeCronEntry() error {
	lines, err := currentCrontab()
	if err != nil {
		return err
	}
	kept := lines[:0]
	for _, l := range lines {
		if !strings.Contains(l, cronMarker) {
			kept = append(kept, l)
		}
	}
	return writeCrontab(kept)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
