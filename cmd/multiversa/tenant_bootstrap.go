package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/graphify"
	"github.com/moshequantum/multiversa-cli/internal/tenant"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// Variables keep the Cobra surface testable without launching Graphify. The
// production values still call the tenant package directly.
var (
	runTenantBootstrap = tenant.Bootstrap
	newBootstrapClient = func() *graphify.Client {
		return graphify.New(graphify.OSRunner{})
	}
)

func newTenantBootstrapCmd() *cobra.Command {
	var (
		jsonOut                   bool
		dryRun, activate          bool
		name, kind, osName, owner string
		brand, voice, language    string
		taboos                    []string
		route, engagement         string
		backend, model, mode      string
		sourceURLs, pillarSpecs   []string
		author, contributor       string
		skipExtract, noCluster    bool
		forceExtract              bool
		timeout                   time.Duration
	)

	cmd := &cobra.Command{
		Use:          "bootstrap <slug>",
		Short:        "Crea o reanuda un OS e ingiere sus fuentes con Graphify.",
		SilenceUsage: true,
		Long: "Crea el tenant aislado, materializa su corpus con procedencia, ingiere\n" +
			"cada --source y extrae un grafo validado. Es reanudable: las fuentes ya\n" +
			"registradas se omiten. --dry-run muestra el plan sin escribir ni ejecutar.",
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if route != "lab" && route != "group" {
				return fmt.Errorf("route inválido %q: usa lab o group", route)
			}
			switch engagement {
			case "self-service", "consulting", "implementation", "managed":
				return nil
			default:
				return fmt.Errorf("engagement inválido %q", engagement)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pillars, err := parsePillarSpecs(pillarSpecs)
			if err != nil {
				return emitBootstrapError(cmd, jsonOut, "invalid_pillar", err)
			}
			sources := make([]tenant.BootstrapSource, 0, len(sourceURLs))
			for _, sourceURL := range sourceURLs {
				sources = append(sources, tenant.BootstrapSource{
					URL: sourceURL, Author: author, Contributor: contributor,
				})
			}
			opts := tenant.BootstrapOptions{
				Name: name, Kind: kind, OSName: osName, Owner: owner,
				Brand: brand, Voice: voice, Language: language, Taboos: taboos,
				Route: route, Engagement: engagement, Pillars: pillars,
				Backend: backend, Model: model, Mode: mode,
				SkipExtract: skipExtract, NoCluster: noCluster, Force: forceExtract,
				Sources: sources, DryRun: dryRun, Activate: activate,
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			result, err := runTenantBootstrap(ctx, args[0], opts, newBootstrapClient())
			if err != nil {
				return emitBootstrapError(cmd, jsonOut, "bootstrap_failed", err)
			}
			if jsonOut {
				return agentout.Emit(cmd.OutOrStdout(), "tenant-bootstrap", struct {
					Action string                  `json:"action"`
					Result *tenant.BootstrapResult `json:"result"`
				}{Action: "bootstrap", Result: result})
			}

			if result.DryRun {
				fmt.Fprintln(cmd.OutOrStdout(), theme.Accent.Render("tenant bootstrap · dry-run"))
				for _, step := range result.Plan {
					fmt.Fprintln(cmd.OutOrStdout(), theme.Dim.Render("  · "+step))
				}
				return nil
			}
			verb := "creado"
			if result.Resumed {
				verb = "reanudado"
			}
			fmt.Fprintln(cmd.OutOrStdout(), theme.Accent.Render("✓ tenant "+verb+" · "+result.Slug))
			fmt.Fprintf(cmd.OutOrStdout(), "  fuentes   %d añadidas · %d omitidas\n", result.Added, result.Skipped)
			fmt.Fprintln(cmd.OutOrStdout(), theme.Dim.Render("  grafo     "+result.GraphPath))
			if result.Activated {
				fmt.Fprintln(cmd.OutOrStdout(), theme.Accent.Render("  activo     sí"))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Nombre visible del OS del proyecto.")
	cmd.Flags().StringVar(&kind, "kind", "project-os", "Compatibilidad legacy; siempre crea project-os.")
	cmd.Flags().StringVar(&osName, "os-name", "", "Nombre propio del OS; por defecto usa --name.")
	cmd.Flags().StringVar(&owner, "owner", "", "Persona u organización con autoridad sobre el OS.")
	cmd.Flags().StringVar(&brand, "brand", "", "Marca principal que ancla el grafo.")
	cmd.Flags().StringVar(&voice, "voice", "", "Descripción breve de la voz de marca.")
	cmd.Flags().StringVar(&language, "language", "es-LA", "Idioma principal del OS.")
	cmd.Flags().StringArrayVar(&taboos, "taboo", nil, "Límite duro de identidad; repetible.")
	cmd.Flags().StringVar(&route, "route", "lab", "Frontera primaria: lab o group.")
	cmd.Flags().StringVar(&engagement, "engagement", "self-service", "Modalidad del trabajo.")
	cmd.Flags().StringArrayVar(&pillarSpecs, "pillar", nil, "Pilar \"Nombre[=métrica[=peso]]\"; repetible.")
	cmd.Flags().StringArrayVar(&sourceURLs, "source", nil, "URL pública aprobada para ingerir; repetible.")
	cmd.Flags().StringVar(&author, "author", "", "Autor atribuido a las fuentes de esta ejecución.")
	cmd.Flags().StringVar(&contributor, "contributor", "", "Persona que incorporó las fuentes al corpus.")
	cmd.Flags().StringVar(&backend, "backend", "", "Backend Graphify: gemini, claude, openai, ollama, etc.")
	cmd.Flags().StringVar(&model, "model", "", "Modelo específico para la extracción Graphify.")
	cmd.Flags().StringVar(&mode, "mode", "", "Modo Graphify; usa deep para relaciones semánticas más ricas.")
	cmd.Flags().BoolVar(&skipExtract, "skip-extract", false, "Descarga y registra el corpus sin ejecutar el modelo todavía.")
	cmd.Flags().BoolVar(&noCluster, "no-cluster", false, "Omite clustering al extraer el grafo.")
	cmd.Flags().BoolVar(&forceExtract, "force-extract", false, "Fuerza una reextracción completa sin sobreescribir el tenant.")
	cmd.Flags().BoolVar(&activate, "activate", false, "Activa el tenant sólo después de validar el grafo.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el plan sin escribir ni ejecutar Graphify.")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "Tiempo máximo total para Graphify.")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.tenant-bootstrap/v1).")
	return cmd
}

func emitBootstrapError(cmd *cobra.Command, jsonOut bool, code string, err error) error {
	if jsonOut {
		if emitErr := agentout.EmitError(cmd.OutOrStdout(), "tenant-bootstrap", code, err.Error(),
			"Corrige los datos o usa --dry-run para revisar el plan."); emitErr != nil {
			return emitErr
		}
	}
	return err
}
