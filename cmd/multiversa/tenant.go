// multiversa tenant — isolated, replicable tenant profiles.
//
// Each tenant is one directory under ~/.multiversa/tenants/<slug>/
// holding its DNA (multiversa.toml), its vault (0700, opaque to every
// Multiversa surface), its graph, and its memory. `tenant new` scaffolds
// from a template (agency = ElevatOS shape, personal-brand = PulseOS
// shape), `tenant use` switches context atomically, and everything is
// readable by agents via --json (schema multiversa.tenant/v1).
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/manifest"
	"github.com/moshequantum/multiversa-cli/internal/tenant"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

func newTenantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Perfiles de tenant aislados: ADN, vault, grafo y memoria por cliente.",
		Long: "Gestiona perfiles de tenant bajo ~/.multiversa/tenants/<slug>/. Cada perfil\n" +
			"es una instalación aislada y replicable: su manifiesto (ADN de la marca),\n" +
			"su vault (0700 — Multiversa nunca lee su contenido), su grafo de\n" +
			"conocimiento anclado a la identidad, y su memoria Engram.",
	}
	cmd.AddCommand(newTenantNewCmd(), newTenantListCmd(), newTenantShowCmd(), newTenantUseCmd())
	return cmd
}

func newTenantNewCmd() *cobra.Command {
	var jsonOut bool
	var name, kind string
	var pillarSpecs []string
	cmd := &cobra.Command{
		Use:   "new <slug>",
		Short: "Crea un perfil de tenant desde una plantilla (nunca sobreescribe).",
		Long: "Crea ~/.multiversa/tenants/<slug>/ con manifiesto pre-llenado, vault 0700,\n" +
			"y directorios de grafo y memoria.\n\n" +
			"Plantillas (--kind): personal-os (default) · personal-brand (perfil PulseOS)\n" +
			"· agency (perfil ElevatOS) · team\n\n" +
			"Con --pillar se sustituyen los pilares de la plantilla. Formato:\n" +
			"  --pillar \"Nombre\"                  (métrica vacía, peso 1.0)\n" +
			"  --pillar \"Nombre=métrica\"          (peso 1.0)\n" +
			"  --pillar \"Nombre=métrica=0.7\"      (peso explícito)\n" +
			"Repetible. El id se deriva del nombre.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pillars, err := parsePillarSpecs(pillarSpecs)
			if err != nil {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "tenant", "invalid_pillar", err.Error(), "")
					os.Exit(1)
				}
				return err
			}
			m, dir, err := tenant.New(args[0], name, kind, pillars...)
			if err != nil {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "tenant", "tenant_create_failed", err.Error(), "")
					os.Exit(1)
				}
				return err
			}
			if jsonOut {
				return agentout.Emit(os.Stdout, "tenant", struct {
					Action   string             `json:"action"`
					Dir      string             `json:"dir"`
					Manifest *manifest.Manifest `json:"manifest"`
				}{"created", dir, m})
			}
			fmt.Println(theme.Accent.Render("✓ tenant creado · " + m.Tenant.Name))
			fmt.Println(theme.Dim.Render("  " + dir))
			fmt.Println(theme.Body.Render("  Edita el ADN en " + dir + "/multiversa.toml y actívalo con:"))
			fmt.Println(theme.Body.Render("  multiversa tenant use " + args[0]))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nombre visible del tenant (ej. \"PulseOS — Cintia Larizatti\").")
	cmd.Flags().StringVar(&kind, "kind", "personal-os", "Plantilla: personal-os · personal-brand · agency · team.")
	cmd.Flags().StringArrayVar(&pillarSpecs, "pillar", nil, "Pilar \"Nombre[=métrica[=peso]]\". Repetible; sustituye los de la plantilla.")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.tenant/v1).")
	return cmd
}

// parsePillarSpecs turns repeated --pillar values into manifest pillars.
// The id and the default weight are filled in by internal/tenant, which is the
// single place that knows the manifest's rules.
func parsePillarSpecs(specs []string) ([]manifest.Pillar, error) {
	out := make([]manifest.Pillar, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, "=", 3)
		p := manifest.Pillar{Name: strings.TrimSpace(parts[0])}
		if p.Name == "" {
			return nil, fmt.Errorf("pilar sin nombre: %q", spec)
		}
		if len(parts) > 1 {
			p.Metric = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			raw := strings.TrimSpace(parts[2])
			w, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, fmt.Errorf("peso inválido %q en el pilar %q: debe ser un número", raw, p.Name)
			}
			if w <= 0 {
				return nil, fmt.Errorf("peso inválido %v en el pilar %q: debe ser mayor que cero", w, p.Name)
			}
			p.Weight = w
		}
		out = append(out, p)
	}
	return out, nil
}

func newTenantListCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista los perfiles de tenant y cuál está activo.",
		RunE: func(cmd *cobra.Command, args []string) error {
			infos, err := tenant.List()
			if err != nil {
				return err
			}
			if jsonOut {
				return agentout.Emit(os.Stdout, "tenant", struct {
					Action  string        `json:"action"`
					Active  string        `json:"active,omitempty"`
					Tenants []tenant.Info `json:"tenants"`
				}{"list", tenant.Active(), infos})
			}
			if len(infos) == 0 {
				fmt.Println(theme.Dim.Render("Sin tenants aún. Crea el primero: multiversa tenant new <slug>"))
				return nil
			}
			fmt.Println(theme.Accent.Render("multiversa tenant list"))
			for _, t := range infos {
				marker := "  "
				if t.Active {
					marker = theme.Accent.Render("▸ ")
				}
				vault := theme.Warn.Render("⚠ vault")
				if t.VaultOK {
					vault = theme.Dim.Render("vault 0700")
				}
				fmt.Printf("%s%-20s %-16s %s\n", marker, t.Slug, theme.Dim.Render(t.Kind), vault)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.tenant/v1).")
	return cmd
}

func newTenantShowCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "show <slug>",
		Short: "Muestra el ADN completo de un tenant (los secretos del vault jamás).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, path, err := tenant.Load(args[0])
			if err != nil {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "tenant", "tenant_not_found", err.Error(),
						"Lista los tenants con `multiversa tenant list`.")
					os.Exit(1)
				}
				return err
			}
			if jsonOut {
				return agentout.Emit(os.Stdout, "tenant", struct {
					Action   string             `json:"action"`
					Path     string             `json:"path"`
					Manifest *manifest.Manifest `json:"manifest"`
				}{"show", path, m})
			}
			fmt.Println(theme.Accent.Render("tenant · "+m.Tenant.Name) + theme.Dim.Render("  ("+m.Tenant.Kind+")"))
			fmt.Printf("  identidad  %s · voz: %s\n", m.Identity.Brand, m.Identity.Voice)
			for _, p := range m.Pillars {
				fmt.Printf("  pilar      %-14s %s (%s)\n", p.ID, p.Name, p.Metric)
			}
			fmt.Printf("  grafo      %s anclado a %s\n", m.Graph.Engine, m.Graph.Anchor)
			fmt.Printf("  sync       %v · %s · auto=%v\n", m.Sync.Providers, m.Sync.Mode, m.Sync.Auto)
			fmt.Printf("  deploy     %v\n", m.Deploy.Targets)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.tenant/v1).")
	return cmd
}

func newTenantUseCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "use <slug>",
		Short: "Activa un tenant — el contexto se intercambia completo y aislado.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := tenant.Use(args[0]); err != nil {
				if jsonOut {
					_ = agentout.EmitError(os.Stdout, "tenant", "tenant_not_found", err.Error(),
						"Lista los tenants con `multiversa tenant list`.")
					os.Exit(1)
				}
				return err
			}
			if jsonOut {
				return agentout.Emit(os.Stdout, "tenant", struct {
					Action string `json:"action"`
					Active string `json:"active"`
				}{"use", args[0]})
			}
			fmt.Println(theme.Accent.Render("✓ tenant activo · " + args[0]))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.tenant/v1).")
	return cmd
}
