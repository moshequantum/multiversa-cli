package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/credits"
	"github.com/moshequantum/multiversa-cli/internal/manifest"
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
		newLabCmd(),
		newInitCmd(),
		newDetectCmd(),
		newStackCmd(),
		newWorkspaceCmd(),
		newUSBCmd(),
		newAddCmd(),
		newConnectCmd(),
		newBackendCmd(),
		newDoctorCmd(),
		newUpdatesCmd(),
		newCreditsCmd(),
		newVersionCmd(),
		newManifestCmd(),
		newMCPCmd(),
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
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "credits",
		Short: "Print attribution for the orchestrated stack.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOut {
				return agentout.Emit(os.Stdout, "credits", struct {
					Sources []credits.Source `json:"sources"`
				}{credits.Sources})
			}
			credits.Print(os.Stdout)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit upstream attribution as a JSON envelope (schema multiversa.credits/v1).")
	return cmd
}

func newVersionCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOut {
				return agentout.Emit(os.Stdout, "version", struct {
					Version string `json:"version"`
					Commit  string `json:"commit"`
					Date    string `json:"date"`
				}{version.Version, version.Commit, version.Date})
			}
			fmt.Println("multiversa", version.Full())
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit version info as a JSON envelope (schema multiversa.version/v1).")
	return cmd
}

// manifestJSON is the payload of `multiversa manifest --json`
// (schema multiversa.manifest/v1). Secrets (anon_key) never serialize.
type manifestJSON struct {
	Path     string             `json:"path"`
	Exists   bool               `json:"exists"`
	Manifest *manifest.Manifest `json:"manifest"`
}

func newManifestCmd() *cobra.Command {
	var jsonOut bool
	var path string
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Print the parsed multiversa.toml manifest.",
		Long: "Print the multiversa.toml manifest, parsed and validated. If the file\n" +
			"does not exist, the default manifest schema is shown instead so agents\n" +
			"and humans can see what `multiversa init` would generate.",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := manifest.Load(path)
			exists := err == nil
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					// Malformed TOML or unreadable file — a real error either way.
					if jsonOut {
						_ = agentout.EmitError(os.Stdout, "manifest", "manifest_invalid", err.Error(),
							"Fix the TOML syntax in "+path+" or regenerate it with `multiversa init`.")
						os.Exit(1)
					}
					return err
				}
				m = manifest.Default()
			}
			if jsonOut {
				return agentout.Emit(os.Stdout, "manifest", manifestJSON{Path: path, Exists: exists, Manifest: m})
			}
			fmt.Println(theme.Accent.Render("multiversa manifest") + theme.Dim.Render(" · "+path))
			if !exists {
				fmt.Println(theme.Warn.Render("⚠ " + path + " no existe — mostrando el manifiesto por defecto."))
			}
			fmt.Printf("  profile   %s\n  ethic     %s\n  version   %s\n  backend   %s\n",
				m.Multiversa.Profile, m.Multiversa.Ethic, m.Multiversa.Version, m.Backend.Provider)
			if len(m.Agents.Enabled) > 0 {
				fmt.Printf("  agents    %s\n", strings.Join(m.Agents.Enabled, " · "))
			}
			for id, src := range m.Stack {
				fmt.Printf("  engine    %-10s %s\n", id, src)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit the manifest as a JSON envelope (schema multiversa.manifest/v1).")
	cmd.Flags().StringVar(&path, "path", manifest.DefaultPath, "Path to the multiversa.toml manifest.")
	return cmd
}
