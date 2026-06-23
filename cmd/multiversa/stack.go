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
	istack "github.com/moshequantum/multiversa-cli/internal/stack"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/tui"
)

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

func runStack(opts stackOpts) error {
	if opts.out == nil {
		opts.out = os.Stdout
	}
	planned, report := planStack(opts)

	if shouldRunTUI(opts) {
		return tui.RunStackTUI(opts.out, report, planned)
	}
	return runStackNonInteractive(opts, report, planned)
}

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

func planStack(opts stackOpts) ([]istack.ToolPlan, detect.Report) {
	report := detect.Run()
	onlySet := toSet(opts.only)
	tools := lang.Registry()

	var planned []istack.ToolPlan
	for _, t := range tools {
		if len(onlySet) > 0 && !onlySet[t.ID()] {
			continue
		}
		tp := istack.ToolPlan{Tool: t, Installed: t.Installed()}
		if !tp.Installed {
			plan, err := t.PlanFor(report.OS.Kind, report.OS.PkgMgr)
			tp.Plan = plan
			tp.Err = err
		}
		planned = append(planned, tp)
	}
	return planned, report
}

func runStackNonInteractive(opts stackOpts, report detect.Report, planned []istack.ToolPlan) error {
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
		if tp.Installed {
			skipped++
			continue
		}
		if tp.Err != nil {
			fmt.Fprintf(opts.out, "%s %s: %v\n", theme.Warn.Render("⚠"), tp.Tool.DisplayName(), tp.Err)
			failed++
			continue
		}
		if !opts.yes && !confirmInstall(opts.out, tp) {
			fmt.Fprintf(opts.out, "%s %s omitido\n", theme.Dim.Render("·"), tp.Tool.DisplayName())
			skipped++
			continue
		}
		if err := xexec.RunPlan(tp.Plan); err != nil {
			fmt.Fprintf(opts.out, "%s %s falló: %v\n", theme.Warn.Render("✗"), tp.Tool.DisplayName(), err)
			failed++
			continue
		}
		fmt.Fprintf(opts.out, "%s %s instalado\n", theme.Accent.Render("✓"), tp.Tool.DisplayName())
		prof.MarkInstalled(tp.Tool.ID())
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

func printToolRow(w io.Writer, tp istack.ToolPlan) {
	id := lipglossPad(tp.Tool.ID(), 10)
	switch {
	case tp.Installed:
		fmt.Fprintf(w, "  %s %s %s\n", theme.Accent.Render("✓"), id, theme.Dim.Render("ya instalado"))
	case tp.Err != nil:
		fmt.Fprintf(w, "  %s %s %s\n", theme.Warn.Render("⚠"), id, theme.Warn.Render(tp.Err.Error()))
	default:
		fmt.Fprintf(w, "  %s %s %s\n", theme.Dim.Render("·"), id, theme.Body.Render(istack.PlanSummary(tp.Plan)))
	}
}

func confirmInstall(w io.Writer, tp istack.ToolPlan) bool {
	fmt.Fprintf(w, "\n%s %s\n", theme.Accent.Render("instalar"), tp.Tool.DisplayName())
	if tp.Plan.Shell != "" {
		fmt.Fprintln(w, "  "+theme.Body.Render("$ "+tp.Plan.Shell))
	} else {
		fmt.Fprintln(w, "  "+theme.Body.Render("$ "+tp.Plan.Program+" "+strings.Join(tp.Plan.Args, " ")))
	}
	if tp.Plan.Notes != "" {
		fmt.Fprintln(w, "  "+theme.Dim.Render(tp.Plan.Notes))
	}
	fmt.Fprint(w, theme.Label.Render("¿proceder? [y/N] "))
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil {
		return false
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes"
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

func lipglossPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
