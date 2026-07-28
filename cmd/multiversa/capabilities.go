package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type capabilitiesJSON struct {
	Protocols       []string `json:"protocols"`
	ProfileSchemas  []string `json:"profile_schemas"`
	EnvelopeSchemas []string `json:"envelope_schemas"`
	Commands        []string `json:"commands"`
	Features        []string `json:"features"`
}

func currentCapabilities() capabilitiesJSON {
	return capabilitiesJSON{
		Protocols:       []string{"multiversa.cli/v1", "mcp/2025-03-26"},
		ProfileSchemas:  []string{"0.2-read", "0.3-read-write"},
		EnvelopeSchemas: []string{"multiversa.cli-envelope/v1"},
		Commands: []string{
			"capabilities", "detect", "manifest", "tenant.list", "tenant.show",
			"tenant.new", "tenant.bootstrap", "tenant.connect", "tenant.use", "tenant.set-secret", "mcp.serve",
		},
		Features: []string{
			"project-os", "tenant-bootstrap", "tenant-llm-fallback", "routing.lab-group", "engagement-reference",
			"vault.stdin-secrets", "human-in-the-loop",
		},
	}
}

func newCapabilitiesCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "Declara los protocolos, schemas y capacidades que soporta este CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			caps := currentCapabilities()
			if jsonOut {
				return agentout.Emit(os.Stdout, "capabilities", caps)
			}
			fmt.Println(theme.Accent.Render("multiversa capabilities"))
			fmt.Println(theme.Body.Render("  protocol  " + caps.Protocols[0]))
			fmt.Println(theme.Body.Render("  profile   " + caps.ProfileSchemas[len(caps.ProfileSchemas)-1]))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.capabilities/v1).")
	return cmd
}
