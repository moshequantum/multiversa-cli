package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/credits"
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
		newDoctorCmd(),
		newManifestCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, theme.Warn.Render("error: ")+err.Error())
		os.Exit(1)
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Run the interactive setup wizard.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return wizard.Run()
		},
	}
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

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Inspect the local stack and agent wiring.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(theme.Accent.Render("multiversa doctor"))
			fmt.Println(theme.Dim.Render("Diagnostics implementation: v0.2."))
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
