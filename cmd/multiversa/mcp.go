// multiversa mcp serve — the CLI as a native MCP server.
//
// Phase 2 of the agent bridge: the same read-only surfaces exposed via
// --json (schemas multiversa.*/v1) become typed MCP tools over stdio,
// so Claude Code, Claude Desktop, Codex, or any MCP client can drive
// the orchestrator natively. Mutating operations are deliberately NOT
// exposed: the MCP surface proposes, the human decides at the TUI.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	alertledger "github.com/moshequantum/multiversa-cli/internal/alerts"
	"github.com/moshequantum/multiversa-cli/internal/credits"
	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/doctor"
	"github.com/moshequantum/multiversa-cli/internal/manifest"
	"github.com/moshequantum/multiversa-cli/internal/tenant"
	"github.com/moshequantum/multiversa-cli/internal/theme"
	"github.com/moshequantum/multiversa-cli/internal/upstream"
	"github.com/moshequantum/multiversa-cli/internal/version"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Superficie MCP del orquestador.",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Sirve las superficies de solo lectura del CLI como tools MCP por stdio.",
		Long: "Levanta un servidor Model Context Protocol sobre stdin/stdout que expone\n" +
			"detect, doctor, status, alerts, version, manifest, credits y updates como tools tipados.\n" +
			"Solo lectura por diseño: ninguna operación mutante se expone vía MCP.\n\n" +
			"Registro en Claude Code:\n" +
			"  claude mcp add multiversa -- multiversa mcp serve",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, theme.Dim.Render("multiversa mcp serve · stdio · solo lectura · \"la IA propone, tú decides\""))
			return runMCPServer(cmd.Context())
		},
	})
	return cmd
}

// versionInfo mirrors the multiversa.version/v1 payload.
type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// manifestArgs is the input schema for the manifest tool.
type manifestArgs struct {
	Path string `json:"path,omitempty" jsonschema:"ruta al multiversa.toml; por defecto ./multiversa.toml"`
}

// creditsOut wraps the attribution list, mirroring multiversa.credits/v1.
type creditsOut struct {
	Sources []credits.Source `json:"sources"`
}

// runMCPServer wires every read-only surface as a typed MCP tool and
// blocks serving a single stdio session.
func runMCPServer(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "multiversa",
		Title:   "Multiversa CLI — orquestador del stack agentic curado",
		Version: version.Version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name: "detect",
		Description: "Escaneo de solo lectura del host: SO, gestor de paquetes, toolchain de " +
			"desarrollo, estado del CLI Multiversa y motores curados instalados. Nunca modifica nada.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, detectJSON, error) {
		report := detect.Run()
		return nil, detectJSON{Report: report, Summary: report.Summarize()}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "doctor",
		Description: "Diagnóstico local de solo lectura con evidencia: deriva de binarios/cron, " +
			"runtimes bloqueados, permisos e índices ausentes. Nunca aplica correcciones.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, doctorJSON, error) {
		report := doctor.Run(detect.Run())
		return nil, doctorJSON{Report: report}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "status",
		Description: "Vista diaria de solo lectura: tenant activo, salud, stack, alertas persistidas y " +
			"próxima acción aprobable. No refresca ni modifica el ledger desde MCP.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, statusJSON, error) {
		return nil, buildStatus(false, time.Now()), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "alerts",
		Description: "Lee el ledger local de alertas de Multiversa sin refrescarlo ni modificarlo.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, alertsJSON, error) {
		path, err := alertledger.DefaultPath()
		if err != nil {
			return nil, alertsJSON{}, err
		}
		ledger, err := alertledger.Load(path)
		if err != nil {
			return nil, alertsJSON{}, err
		}
		return nil, alertsJSON{Path: path, Summary: ledger.Summary(), Alerts: ledger.OpenAlerts()}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "version",
		Description: "Versión, commit y fecha de build del CLI Multiversa.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, versionInfo, error) {
		return nil, versionInfo{Version: version.Version, Commit: version.Commit, Date: version.Date}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "manifest",
		Description: "Lee y parsea el manifiesto declarativo multiversa.toml (perfil, ética, stack, " +
			"agentes, backend). Si no existe, devuelve el manifiesto por defecto con exists=false. " +
			"Los secretos (anon_key) nunca se serializan.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in manifestArgs) (*mcp.CallToolResult, manifestJSON, error) {
		path := in.Path
		if path == "" {
			path = manifest.DefaultPath
		}
		m, err := manifest.Load(path)
		exists := err == nil
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, manifestJSON{}, fmt.Errorf("manifest inválido en %s: %w", path, err)
			}
			m = manifest.Default()
		}
		return nil, manifestJSON{Path: path, Exists: exists, Manifest: m}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "credits",
		Description: "Atribución upstream completa del stack curado: autor, repo, licencia y rol de " +
			"cada motor. Multiversa orquesta estos motores, no los autoría.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, creditsOut, error) {
		return nil, creditsOut{Sources: credits.Sources}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "updates",
		Description: "Vigilante de releases: consulta el último release publicado de cada motor curado " +
			"y del propio CLI contra lo instalado localmente. Reporta qué actualizar; NUNCA actualiza — " +
			"aplicar la actualización es decisión humana.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, updatesJSON, error) {
		results := upstream.Check(watchedComponents(), 15*time.Second)
		available := 0
		for _, r := range results {
			if r.UpdateAvailable {
				available++
			}
		}
		return nil, updatesJSON{
			CheckedAt:        time.Now().UTC().Format(time.RFC3339),
			UpdatesAvailable: available,
			Components:       results,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "tenant_list",
		Description: "Lista los perfiles de tenant (~/.multiversa/tenants): slug, tipo, dueño, cuál está " +
			"activo y si su vault mantiene permisos 0700. El contenido del vault nunca se lee ni expone.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, tenantListOut, error) {
		infos, err := tenant.List()
		if err != nil {
			return nil, tenantListOut{}, err
		}
		return nil, tenantListOut{Active: tenant.Active(), Tenants: infos}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "tenant_show",
		Description: "Muestra el ADN completo de un tenant por slug: identidad, pilares, scoring, grafo, " +
			"sync y deploy. Los secretos del vault jamás se serializan.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in tenantShowArgs) (*mcp.CallToolResult, tenantShowOut, error) {
		m, path, err := tenant.Load(in.Slug)
		if err != nil {
			return nil, tenantShowOut{}, err
		}
		return nil, tenantShowOut{Path: path, Manifest: m}, nil
	})

	return server.Run(ctx, &mcp.StdioTransport{})
}

// tenantListOut mirrors the multiversa.tenant/v1 list payload.
type tenantListOut struct {
	Active  string        `json:"active,omitempty"`
	Tenants []tenant.Info `json:"tenants"`
}

// tenantShowArgs selects the tenant to inspect.
type tenantShowArgs struct {
	Slug string `json:"slug" jsonschema:"slug kebab-case del tenant, ver tenant_list"`
}

// tenantShowOut mirrors the multiversa.tenant/v1 show payload.
type tenantShowOut struct {
	Path     string             `json:"path"`
	Manifest *manifest.Manifest `json:"manifest"`
}
