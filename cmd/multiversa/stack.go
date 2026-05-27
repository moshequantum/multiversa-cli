package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/detect"
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
	"github.com/moshequantum/multiversa-cli/internal/lang"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// newStackCmd installs the OS-level developer toolchain — Go, Rust,
// Python, Node, pnpm, Docker. It is intentionally separate from
// `multiversa init`, which installs the curated agentic engines
// (Engram, Graphify, Gentle, …) on top of an existing toolchain.
//
// Default mode is dry-run-by-print: every Plan is shown to the user
// before anything executes. --yes runs all missing tools without
// prompting; --only filters to a subset.
func newStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Install the OS-level developer toolchain (Go, Rust, Python, Node, pnpm, Docker).",
		Long: "Install or update the OS-level developer toolchain that the\n" +
			"Multiversa lab depends on. Distinct from `multiversa init`, which\n" +
			"installs the agentic engines (Engram, Graphify, Gentle, …) on\n" +
			"top of this foundation.\n\n" +
			"Default behavior shows the install plan and asks for confirmation\n" +
			"per tool. Use --yes to skip confirmation, --dry-run to print\n" +
			"commands without executing, --only=a,b to install a subset.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			yes, _ := cmd.Flags().GetBool("yes")
			only, _ := cmd.Flags().GetStringSlice("only")
			return runStack(stackOpts{dryRun: dryRun, yes: yes, only: only})
		},
	}
	cmd.Flags().Bool("dry-run", false, "Print install plans without executing them.")
	cmd.Flags().Bool("yes", false, "Install every missing tool without per-step confirmation.")
	cmd.Flags().StringSlice("only", nil, "Comma-separated tool IDs to operate on (e.g. --only=rust,pnpm).")
	return cmd
}

type stackOpts struct {
	dryRun bool
	yes    bool
	only   []string
}

func runStack(opts stackOpts) error {
	// 1. Detect once. We need OS + pkgMgr for every Plan.
	report := detect.Run()
	fmt.Println(theme.Accent.Render("multiversa stack"))
	fmt.Println(theme.Dim.Render(fmt.Sprintf("host: %s/%s · %s · pkg-mgr: %s",
		report.OS.Kind, report.OS.Arch, report.OS.Distro, displayPkgMgr(report.OS.PkgMgr))))
	fmt.Println()

	// 2. Build the working set.
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

	if len(planned) == 0 {
		fmt.Println(theme.Warn.Render("No matching tools for --only=" + strings.Join(opts.only, ",")))
		return nil
	}

	// 3. Print summary table.
	for _, tp := range planned {
		printToolRow(tp)
	}
	fmt.Println()

	// 4. Execute.
	if opts.dryRun {
		fmt.Println(theme.Dim.Render("Dry run — nothing installed. Re-run without --dry-run to apply."))
		return nil
	}

	var installed, skipped, failed int
	for _, tp := range planned {
		if tp.installed {
			skipped++
			continue
		}
		if tp.err != nil {
			fmt.Printf("%s %s: %v\n", theme.Warn.Render("⚠"), tp.tool.DisplayName(), tp.err)
			failed++
			continue
		}
		if !opts.yes && !confirmInstall(tp) {
			fmt.Printf("%s %s skipped\n", theme.Dim.Render("·"), tp.tool.DisplayName())
			skipped++
			continue
		}
		if err := executePlan(tp.plan); err != nil {
			fmt.Printf("%s %s failed: %v\n", theme.Warn.Render("✗"), tp.tool.DisplayName(), err)
			failed++
			continue
		}
		fmt.Printf("%s %s installed\n", theme.Accent.Render("✓"), tp.tool.DisplayName())
		installed++
	}

	// 5. Summary.
	fmt.Println()
	fmt.Println(theme.Dim.Render(fmt.Sprintf(
		"Done: %d installed · %d skipped · %d failed", installed, skipped, failed)))
	if failed > 0 {
		return fmt.Errorf("%d tool(s) failed", failed)
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

func printToolRow(tp toolPlan) {
	id := lipglossPad(tp.tool.ID(), 10)
	switch {
	case tp.installed:
		fmt.Printf("  %s %s %s\n", theme.Accent.Render("✓"), id, theme.Dim.Render("already installed"))
	case tp.err != nil:
		fmt.Printf("  %s %s %s\n", theme.Warn.Render("⚠"), id, theme.Warn.Render(tp.err.Error()))
	default:
		fmt.Printf("  %s %s %s\n", theme.Dim.Render("·"), id, theme.Body.Render(planSummary(tp.plan)))
	}
}

func planSummary(p lang.Plan) string {
	switch {
	case p.Shell != "":
		return truncate(p.Shell, 70)
	case p.Program != "":
		return truncate(p.Program+" "+strings.Join(p.Args, " "), 70)
	}
	return "(no plan)"
}

func confirmInstall(tp toolPlan) bool {
	fmt.Printf("\n%s %s\n", theme.Accent.Render("install"), tp.tool.DisplayName())
	if tp.plan.Shell != "" {
		fmt.Println("  " + theme.Body.Render("$ "+tp.plan.Shell))
	} else {
		fmt.Println("  " + theme.Body.Render("$ "+tp.plan.Program+" "+strings.Join(tp.plan.Args, " ")))
	}
	if tp.plan.Notes != "" {
		fmt.Println("  " + theme.Dim.Render(tp.plan.Notes))
	}
	fmt.Print(theme.Label.Render("proceed? [y/N] "))
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil {
		return false
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes"
}

func executePlan(p lang.Plan) error {
	if p.Shell != "" {
		// We deliberately use `sh -c` rather than parsing the shell line
		// ourselves: the curl-pipe-shell pattern is the canonical
		// install path for rustup, pnpm, nvm, and pyenv.
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
		return "(none detected)"
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

// lipglossPad pads a string with spaces to a fixed width without
// pulling lipgloss into this file. The output is plain ASCII suitable
// for left-aligned columns; rendering is applied separately.
func lipglossPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
