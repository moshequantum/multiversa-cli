package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
	"github.com/moshequantum/multiversa-cli/internal/lang"
	"github.com/moshequantum/multiversa-cli/internal/profile"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// newStackCmd installs the OS-level developer toolchain — Go, Rust,
// Python, Node, pnpm, Docker. v0.4.0 unifies the UX behind the shared
// internal/tui primitives: when stdout is a TTY and --yes is NOT set,
// the command launches a Bubble Tea program (Selector → ProgressList).
// Otherwise it falls back to the non-interactive plan/print path so
// CI, pipes, and --yes scripted runs keep working unchanged.
func newStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Instala la cadena de herramientas (Go, Rust, Python, Node, pnpm, Docker).",
		Long: "Instala o actualiza la cadena de herramientas a nivel de sistema\n" +
			"que el laboratorio Multiversa necesita. Es distinto de `multiversa\n" +
			"init`, que instala los engines agénticos (Engram, Graphify, …)\n" +
			"sobre esta base.\n\n" +
			"Por defecto abre una TUI interactiva (selector + progreso). Usa\n" +
			"--yes para correr sin prompts, --dry-run para imprimir comandos\n" +
			"sin ejecutarlos, --only=a,b para operar sobre un subconjunto.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			yes, _ := cmd.Flags().GetBool("yes")
			only, _ := cmd.Flags().GetStringSlice("only")
			return runStack(stackOpts{dryRun: dryRun, yes: yes, only: only, out: os.Stdout})
		},
	}
	cmd.Flags().Bool("dry-run", false, "Imprime los planes de instalación sin ejecutarlos.")
	cmd.Flags().Bool("yes", false, "Instala todo lo faltante sin confirmar paso a paso.")
	cmd.Flags().StringSlice("only", nil, "IDs separados por coma (ej. --only=rust,pnpm).")
	return cmd
}

type stackOpts struct {
	dryRun bool
	yes    bool
	only   []string
	out    io.Writer
}

// runStack is the entry point. It decides between the TUI flow and the
// non-interactive flow based on stdout-is-tty + --yes.
func runStack(opts stackOpts) error {
	if opts.out == nil {
		opts.out = os.Stdout
	}
	planned, report := planStack(opts)

	if shouldRunTUI(opts) {
		return runStackTUI(opts, report, planned)
	}
	return runStackNonInteractive(opts, report, planned)
}

// shouldRunTUI gates the Bubble Tea path. The non-interactive path is
// used whenever stdout is not a TTY (pipes, CI) or the caller asked for
// --yes (scripted), preserving v0.3.0 behavior for those callers.
func shouldRunTUI(opts stackOpts) bool {
	if opts.yes || opts.dryRun {
		return false
	}
	f, ok := opts.out.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// planStack runs detection and builds the working set once. Used by
// both the TUI and the non-interactive paths.
func planStack(opts stackOpts) ([]toolPlan, detect.Report) {
	report := detect.Run()
	onlySet := toSet(opts.only)
	tools := lang.Registry()

	var planned []toolPlan
	for _, t := range tools {
		if len(onlySet) > 0 && !onlySet[t.ID()] {
			continue
		}
		tp := toolPlan{tool: t, installed: t.Installed()}
		if !tp.installed {
			plan, err := t.PlanFor(report.OS.Kind, report.OS.PkgMgr)
			tp.plan = plan
			tp.err = err
		}
		planned = append(planned, tp)
	}
	return planned, report
}

// runStackNonInteractive preserves the v0.3.0 behavior: print summary,
// either dry-run or honor --yes (no per-step prompt when --yes is set;
// otherwise a basic Y/N prompt). Used in CI/pipes/scripted runs.
func runStackNonInteractive(opts stackOpts, report detect.Report, planned []toolPlan) error {
	fmt.Fprintln(opts.out, theme.Accent.Render("multiversa stack"))
	fmt.Fprintln(opts.out, theme.Dim.Render(fmt.Sprintf("host: %s/%s · %s · pkg-mgr: %s",
		report.OS.Kind, report.OS.Arch, report.OS.Distro, displayPkgMgr(report.OS.PkgMgr))))
	fmt.Fprintln(opts.out)

	if len(planned) == 0 {
		fmt.Fprintln(opts.out, theme.Warn.Render("Sin coincidencias para --only="+strings.Join(opts.only, ",")))
		return nil
	}

	for _, tp := range planned {
		printToolRow(opts.out, tp)
	}
	fmt.Fprintln(opts.out)

	if opts.dryRun {
		fmt.Fprintln(opts.out, theme.Dim.Render("Dry run — nada se instaló. Re-ejecuta sin --dry-run para aplicar."))
		return nil
	}

	prof, _ := profile.Load()
	var installed, skipped, failed int
	for _, tp := range planned {
		if tp.installed {
			skipped++
			continue
		}
		if tp.err != nil {
			fmt.Fprintf(opts.out, "%s %s: %v\n", theme.Warn.Render("⚠"), tp.tool.DisplayName(), tp.err)
			failed++
			continue
		}
		if !opts.yes && !confirmInstall(opts.out, tp) {
			fmt.Fprintf(opts.out, "%s %s omitido\n", theme.Dim.Render("·"), tp.tool.DisplayName())
			skipped++
			continue
		}
		if err := executePlan(tp.plan); err != nil {
			fmt.Fprintf(opts.out, "%s %s falló: %v\n", theme.Warn.Render("✗"), tp.tool.DisplayName(), err)
			failed++
			continue
		}
		fmt.Fprintf(opts.out, "%s %s instalado\n", theme.Accent.Render("✓"), tp.tool.DisplayName())
		prof.MarkInstalled(tp.tool.ID())
		installed++
	}
	_ = prof.Save()

	fmt.Fprintln(opts.out)
	fmt.Fprintln(opts.out, theme.Dim.Render(fmt.Sprintf(
		"Listo: %d instalados · %d omitidos · %d fallidos", installed, skipped, failed)))
	if failed > 0 {
		return fmt.Errorf("%d herramienta(s) fallaron", failed)
	}
	return nil
}

// toolPlan bundles a Tool with its current state and Plan for output.
type toolPlan struct {
	tool      lang.Tool
	installed bool
	plan      lang.Plan
	err       error
}

func printToolRow(w io.Writer, tp toolPlan) {
	id := lipglossPad(tp.tool.ID(), 10)
	switch {
	case tp.installed:
		fmt.Fprintf(w, "  %s %s %s\n", theme.Accent.Render("✓"), id, theme.Dim.Render("ya instalado"))
	case tp.err != nil:
		fmt.Fprintf(w, "  %s %s %s\n", theme.Warn.Render("⚠"), id, theme.Warn.Render(tp.err.Error()))
	default:
		fmt.Fprintf(w, "  %s %s %s\n", theme.Dim.Render("·"), id, theme.Body.Render(planSummary(tp.plan)))
	}
}

func planSummary(p lang.Plan) string {
	switch {
	case p.Shell != "":
		return truncate(p.Shell, 70)
	case p.Program != "":
		return truncate(p.Program+" "+strings.Join(p.Args, " "), 70)
	}
	return "(sin plan)"
}

func confirmInstall(w io.Writer, tp toolPlan) bool {
	fmt.Fprintf(w, "\n%s %s\n", theme.Accent.Render("instalar"), tp.tool.DisplayName())
	if tp.plan.Shell != "" {
		fmt.Fprintln(w, "  "+theme.Body.Render("$ "+tp.plan.Shell))
	} else {
		fmt.Fprintln(w, "  "+theme.Body.Render("$ "+tp.plan.Program+" "+strings.Join(tp.plan.Args, " ")))
	}
	if tp.plan.Notes != "" {
		fmt.Fprintln(w, "  "+theme.Dim.Render(tp.plan.Notes))
	}
	fmt.Fprint(w, theme.Label.Render("¿proceder? [y/N] "))
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil {
		return false
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes"
}

func executePlan(p lang.Plan) error {
	if p.Shell != "" {
		// `sh -c` is deliberate: rustup, pnpm, nvm, pyenv all ship as
		// curl-pipe-shell installers and the Plan.Shell field already
		// encodes that exact pipeline.
		r := xexec.Run("sh", "-c", p.Shell)
		if r.Err != nil {
			return r.Err
		}
		return nil
	}
	r := xexec.Run(p.Program, p.Args...)
	return r.Err
}

func displayPkgMgr(p string) string {
	if p == "" {
		return "(ninguno)"
	}
	return p
}

func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	s := make(map[string]bool, len(items))
	for _, it := range items {
		s[strings.TrimSpace(it)] = true
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// lipglossPad pads a string with spaces to a fixed width. Plain ASCII —
// rendering is applied separately by the caller.
func lipglossPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
