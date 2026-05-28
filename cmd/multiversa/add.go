package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/stack"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// newAddCmd registers `multiversa add <engine>`.
// v0.4.x: validates the engine name against the registry and prints
// a planned-feature notice. Full single-engine install ships in v0.5.0.
func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <engine>",
		Short: "Add a single engine to the stack (v0.5.0).",
		Long: "Add a single engine to your Multiversa stack without re-running the\n" +
			"full init wizard. Planned for v0.5.0.\n\n" +
			"Available engines: " + engineList(),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine := strings.ToLower(args[0])
			if !engineExists(engine) {
				fmt.Fprintf(os.Stderr, "%s engine desconocido: %q\n%s\n",
					theme.Warn.Render("error:"),
					engine,
					theme.Dim.Render("Motores disponibles: "+engineList()),
				)
				os.Exit(1)
			}
			fmt.Println(theme.Accent.Render("multiversa add · "+engine))
			fmt.Println(theme.Dim.Render("Instalación individual planificada para v0.5.0."))
			fmt.Println(theme.Body.Render("Por ahora, usa `multiversa lab` → Capa Técnica → Stack base."))
			return nil
		},
	}
}

// newConnectCmd registers `multiversa connect <agent>`.
// v0.4.x: validates the agent name and prints a planned-feature notice.
// Full agent wiring (CLAUDE.md / .cursorrules / mcp config) ships in v0.5.0.
func newConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect <agent>",
		Short: "Wire an AI agent connector (v0.5.0).",
		Long: "Configure the selected AI agent to work with the installed Multiversa\n" +
			"stack — writing CLAUDE.md, .cursorrules, MCP config, or agent-specific\n" +
			"settings as needed. Planned for v0.5.0.\n\n" +
			"Supported agents: " + agentList(),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := strings.ToLower(args[0])
			if !agentExists(agent) {
				fmt.Fprintf(os.Stderr, "%s agente desconocido: %q\n%s\n",
					theme.Warn.Render("error:"),
					agent,
					theme.Dim.Render("Agentes soportados: "+agentList()),
				)
				os.Exit(1)
			}
			fmt.Println(theme.Accent.Render("multiversa connect · " + agent))
			fmt.Println(theme.Dim.Render("Cableado de agente planificado para v0.5.0."))
			fmt.Println(theme.Body.Render("Por ahora, el wizard `multiversa init` configura el agente de tu elección."))
			return nil
		},
	}
}

// newBackendCmd registers `multiversa backend <name>`.
// v0.4.x: validates the backend name and prints a planned-feature notice.
// Standalone backend configuration ships in v0.5.0.
func newBackendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backend <name>",
		Short: "Configure an optional remote backend (v0.5.0).",
		Long: "Configure the remote backend for Multiversa's persistent memory and\n" +
			"knowledge graph (Engram + Graphify). Local SQLite is the default and\n" +
			"requires no configuration. Planned for v0.5.0.\n\n" +
			"Available backends: local (default) · supabase · firebase · insforge",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.ToLower(args[0])
			valid := map[string]bool{"local": true, "supabase": true, "firebase": true, "insforge": true}
			if !valid[name] {
				fmt.Fprintf(os.Stderr, "%s backend desconocido: %q\n%s\n",
					theme.Warn.Render("error:"),
					name,
					theme.Dim.Render("Backends disponibles: local · supabase · firebase · insforge"),
				)
				os.Exit(1)
			}
			if name == "local" {
				fmt.Println(theme.Accent.Render("multiversa backend · local"))
				fmt.Println(theme.Dim.Render("Local SQLite está activo por defecto — no requiere configuración."))
				return nil
			}
			fmt.Println(theme.Accent.Render("multiversa backend · " + name))
			fmt.Println(theme.Dim.Render("Configuración de backend remoto planificada para v0.5.0."))
			fmt.Println(theme.Body.Render("Por ahora, configura el backend durante `multiversa init`."))
			return nil
		},
	}
}

// engineList returns a comma-joined list of known engine IDs for help text.
func engineList() string {
	ids := make([]string, 0, len(stack.Registry()))
	for _, e := range stack.Registry() {
		ids = append(ids, e.ID())
	}
	return strings.Join(ids, " · ")
}

// engineExists checks whether the given id matches a registered engine.
func engineExists(id string) bool {
	for _, e := range stack.Registry() {
		if e.ID() == id {
			return true
		}
	}
	return false
}

// agentList returns a comma-joined list of known agent IDs for help text.
func agentList() string {
	return "claude-code · cursor · codex · gemini-cli · opencode · aider · cline · continue · roo-code · generic-mcp"
}

// agentExists checks whether the given id matches a known agent adapter.
func agentExists(id string) bool {
	known := map[string]bool{
		"claude-code": true, "cursor": true, "codex": true,
		"gemini-cli": true, "opencode": true, "aider": true,
		"cline": true, "continue": true, "roo-code": true, "generic-mcp": true,
	}
	return known[id]
}
