package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/credits"
	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/version"
	"github.com/moshequantum/multiversa-cli/internal/wizard"
)

func main() {
	root := &cobra.Command{
		Use:   "multiversa",
		Short: "Agnostic pre-configurator for the curated agentic stack.",
		Long: theme.Accent.Render("Multiversa") + " " + theme.Dim.Render("orchestrates Engram, Graphify, gentle-ai, gentle-pi, codegraph, MiroFish.") + "\n" +
			theme.Dim.Render("Multiversa does not author these engines — see `multiversa credits`.") + "\n\n" +
			theme.Body.Render("\"La IA propone, tú decides.\""),
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	root.AddCommand(
		newInitCmd(),
		newCreditsCmd(),
		newVersionCmd(),
		newDetectCmd(),
		newDoctorCmd(),
		newStackCmd(),
		newWorkspaceCmd(),
		newUSBCmd(),
		newManifestCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, theme.Warn.Render("error: ")+err.Error())
		os.Exit(1)
	}
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run the interactive setup wizard.",
		Long: "Run the interactive setup wizard. By default (v0.2), the wizard\n" +
			"executes real install commands for each selected engine. Use --dry-run\n" +
			"to preview the commands without running them.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return wizard.RunWith(wizard.Options{DryRun: dryRun})
		},
	}
	cmd.Flags().Bool("dry-run", false, "Preview install commands without executing them.")
	return cmd
}

func newCreditsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credits",
		Short: "Print attribution for the orchestrated stack.",
		Run: func(cmd *cobra.Command, args []string) {
			credits.Print(os.Stdout)
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("multiversa", version.Full())
		},
	}
}

// newDetectCmd is the canonical environment scanner. It is read-only:
// no installs, no network, no mutation. The output also drives the
// `/lab-setup` Claude Code skill, so the report shape is intentionally
// stable.
func newDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "detect",
		Aliases: []string{"scan"},
		Short:   "Scan the host: OS, package manager, dev stack, Multiversa state.",
		Long: "Run a read-only scan of the local environment. Reports OS and\n" +
			"package manager, the developer toolchain (Go, Rust, Python, Node,\n" +
			"pnpm, Docker, …), the Multiversa CLI state, and the curated engines.\n\n" +
			"This command is safe to run anywhere: it never installs, fetches,\n" +
			"or modifies anything. Use `multiversa init` afterwards to act on\n" +
			"the findings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			report := detect.Run()
			report.Render(os.Stdout)
			return nil
		},
	}
}

// newDoctorCmd is kept as an ergonomic alias for users who reach for
// `doctor` out of habit (npm, brew, dotnet, etc. all have one). It
// delegates to the same detect report so there is one source of truth.
func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "doctor",
		Short:  "Alias of `multiversa detect`.",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			report := detect.Run()
			report.Render(os.Stdout)
			return nil
		},
	}
}

func newManifestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "manifest",
		Short: "Print or edit the multiversa.toml manifest.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(theme.Accent.Render("multiversa manifest"))
			fmt.Println(theme.Dim.Render("Schema: multiversa.toml.example  ·  Editing UI: v0.2."))
			return nil
		},
	}
}
