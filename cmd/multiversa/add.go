package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/adapters"
	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/manifest"
	"github.com/moshequantum/multiversa-cli/internal/stack"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// plannedJSON is the payload emitted by add/connect/backend under --json
// while their real implementations land in a future release. Agents should branch
// on status: "planned" means the command validated but executed nothing.
type plannedJSON struct {
	Kind           string   `json:"kind"` // "engine" | "agent" | "backend"
	Target         string   `json:"target"`
	Status         string   `json:"status"` // "planned" | "active"
	PlannedVersion string   `json:"planned_version,omitempty"`
	Available      []string `json:"available"`
	Workaround     string   `json:"workaround,omitempty"`
}

// newAddCmd registers `multiversa add <engine>`.
// v0.4.x: validates the engine name against the registry and prints
// a planned-feature notice. Full single-engine install ships in a future release.
func newAddCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "add <engine>",
		Short: "Add a single engine to the stack (próximamente).",
		Long: "Add a single engine to your Multiversa stack without re-running the\n" +
			"full init wizard. Planned for a future release.\n\n" +
			"Available engines: " + engineList(),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine := strings.ToLower(args[0])
			if !engineExists(engine) {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "add", "unknown_engine",
						"motor desconocido: "+engine,
						"Motores disponibles: "+engineList())
				} else {
					fmt.Fprintf(os.Stderr, "%s motor desconocido: %q\n%s\n",
						theme.Warn.Render("error:"),
						engine,
						theme.Dim.Render("Motores disponibles: "+engineList()),
					)
				}
				os.Exit(1)
			}
			if jsonOut {
				return agentout.Emit(os.Stdout, "add", plannedJSON{
					Kind: "engine", Target: engine, Status: "planned", PlannedVersion: "",
					Available:  engineIDs(),
					Workaround: "multiversa lab -> Capa Técnica -> Stack base",
				})
			}
			fmt.Println(theme.Accent.Render("multiversa add · " + engine))
			fmt.Println(theme.Dim.Render("Instalación individual planificada — llegará en una próxima versión."))
			fmt.Println(theme.Body.Render("Por ahora, usa `multiversa lab` → Capa Técnica → Stack base."))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.add/v1).")
	return cmd
}

// newConnectCmd registers `multiversa connect <agent>`.
func newConnectCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "connect <agent>",
		Short: "Conecta un agente de IA al servidor MCP de Multiversa.",
		Long: "Configura el agente seleccionado para conectarse al servidor MCP de\n" +
			"Multiversa, modificando ~/.claude.json, ~/.cursor/mcp.json u otras\n" +
			"configuraciones del agente según sea necesario.\n\n" +
			"Agentes soportados: " + agentList(),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := strings.ToLower(args[0])
			adapter, err := adapters.Resolve(agent)
			if err != nil {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "connect", "unknown_agent",
						"agente desconocido: "+agent,
						"Agentes soportados: "+agentList())
				} else {
					fmt.Fprintf(os.Stderr, "%s agente desconocido: %q\n%s\n",
						theme.Warn.Render("error:"),
						agent,
						theme.Dim.Render("Agentes soportados: "+agentList()),
					)
				}
				os.Exit(1)
			}

			report := detect.Run()
			var enabledEngines []string
			for _, eng := range report.Multiversa.Engines {
				if eng.Installed {
					enabledEngines = append(enabledEngines, eng.ID)
				}
			}

			opts := adapters.ConnectOptions{
				EnabledEngines: enabledEngines,
				Manifest:       manifest.DefaultPath,
			}

			err = adapter.Connect(opts)
			if err != nil {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "connect", "connect_failed", err.Error(), "")
					os.Exit(1)
				}
				return err
			}

			if jsonOut {
				return agentout.Emit(os.Stdout, "connect", struct {
					Agent  string   `json:"agent"`
					Status string   `json:"status"`
					Active []string `json:"active_engines"`
				}{agent, "connected", enabledEngines})
			}

			fmt.Println(theme.Accent.Render("✓ " + adapter.DisplayName() + " conectado exitosamente"))
			fmt.Println(theme.Dim.Render("  Servidor MCP 'multiversa' registrado en la configuración del agente."))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.connect/v1).")
	return cmd
}

// newDisconnectCmd registers `multiversa disconnect <agent>`.
func newDisconnectCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "disconnect <agent>",
		Short: "Desconecta un agente de IA del servidor MCP de Multiversa.",
		Long:  "Remueve la definición del servidor MCP de Multiversa de las configuraciones del agente.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := strings.ToLower(args[0])
			adapter, err := adapters.Resolve(agent)
			if err != nil {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "disconnect", "unknown_agent",
						"agente desconocido: "+agent,
						"Agentes soportados: "+agentList())
				} else {
					fmt.Fprintf(os.Stderr, "%s agente desconocido: %q\n%s\n",
						theme.Warn.Render("error:"),
						agent,
						theme.Dim.Render("Agentes soportados: "+agentList()),
					)
				}
				os.Exit(1)
			}

			err = adapter.Disconnect()
			if err != nil {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "disconnect", "disconnect_failed", err.Error(), "")
					os.Exit(1)
				}
				return err
			}

			if jsonOut {
				return agentout.Emit(os.Stdout, "disconnect", struct {
					Agent  string `json:"agent"`
					Status string `json:"status"`
				}{agent, "disconnected"})
			}

			fmt.Println(theme.Accent.Render("✓ " + adapter.DisplayName() + " desconectado exitosamente"))
			fmt.Println(theme.Dim.Render("  Servidor MCP 'multiversa' removido de la configuración del agente."))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.disconnect/v1).")
	return cmd
}

// newBackendCmd registers `multiversa backend <name>`.
// v0.4.x: validates the backend name and prints a planned-feature notice.
// Standalone backend configuration ships in a future release.
func newBackendCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "backend <name>",
		Short: "Configure an optional remote backend (próximamente).",
		Long: "Configure the remote backend for Multiversa's persistent memory and\n" +
			"knowledge graph (Engram + Graphify). Local SQLite is the default and\n" +
			"requires no configuration. Planned for a future release.\n\n" +
			"Available backends: local (default) · supabase · firebase · insforge",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.ToLower(args[0])
			valid := map[string]bool{"local": true, "supabase": true, "firebase": true, "insforge": true}
			if !valid[name] {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "backend", "unknown_backend",
						"backend desconocido: "+name,
						"Backends disponibles: local · supabase · firebase · insforge")
				} else {
					fmt.Fprintf(os.Stderr, "%s backend desconocido: %q\n%s\n",
						theme.Warn.Render("error:"),
						name,
						theme.Dim.Render("Backends disponibles: local · supabase · firebase · insforge"),
					)
				}
				os.Exit(1)
			}
			if name == "local" {
				if jsonOut {
					return agentout.Emit(os.Stdout, "backend", plannedJSON{
						Kind: "backend", Target: name, Status: "active",
						Available: backendIDs(),
					})
				}
				fmt.Println(theme.Accent.Render("multiversa backend · local"))
				fmt.Println(theme.Dim.Render("Local SQLite está activo por defecto — no requiere configuración."))
				return nil
			}
			if jsonOut {
				return agentout.Emit(os.Stdout, "backend", plannedJSON{
					Kind: "backend", Target: name, Status: "planned", PlannedVersion: "",
					Available:  backendIDs(),
					Workaround: "configura el backend durante multiversa init",
				})
			}
			fmt.Println(theme.Accent.Render("multiversa backend · " + name))
			fmt.Println(theme.Dim.Render("Configuración de backend remoto planificada — llegará en una próxima versión."))
			fmt.Println(theme.Body.Render("Por ahora, configura el backend durante `multiversa init`."))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.backend/v1).")
	return cmd
}

// engineIDs returns the registered engine IDs in registry order.
func engineIDs() []string {
	ids := make([]string, 0, len(stack.Registry()))
	for _, e := range stack.Registry() {
		ids = append(ids, e.ID())
	}
	return ids
}

// engineList returns a comma-joined list of known engine IDs for help text.
func engineList() string {
	return strings.Join(engineIDs(), " · ")
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

// agentIDs returns the known agent adapter IDs.
func agentIDs() []string {
	return []string{"claude-code", "cursor", "codex", "gemini-cli", "opencode", "aider", "cline", "continue", "roo-code", "generic-mcp"}
}

// agentList returns a comma-joined list of known agent IDs for help text.
func agentList() string {
	return strings.Join(agentIDs(), " · ")
}

// agentExists checks whether the given id matches a known agent adapter.
func agentExists(id string) bool {
	for _, a := range agentIDs() {
		if a == id {
			return true
		}
	}
	return false
}

// backendIDs returns the supported backend provider IDs.
func backendIDs() []string {
	return []string{"local", "supabase", "firebase", "insforge"}
}
