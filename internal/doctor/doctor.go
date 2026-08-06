// Package doctor turns factual detection into evidence-backed diagnostics.
// It is read-only: findings propose actions but never apply them.
package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/moshequantum/multiversa-cli/internal/capability"
	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/tenant"
)

type Severity string

const (
	P0 Severity = "P0"
	P1 Severity = "P1"
	P2 Severity = "P2"
	P3 Severity = "P3"
)

type Finding struct {
	ID               string   `json:"id"`
	Severity         Severity `json:"severity"`
	Component        string   `json:"component"`
	Title            string   `json:"title"`
	Detail           string   `json:"detail"`
	Evidence         []string `json:"evidence,omitempty"`
	Missing          []string `json:"missing,omitempty"`
	NextActions      []string `json:"next_actions,omitempty"`
	RequiresApproval bool     `json:"requires_approval"`
}

type Summary struct {
	P0   int `json:"p0"`
	P1   int `json:"p1"`
	P2   int `json:"p2"`
	P3   int `json:"p3"`
	Open int `json:"open"`
}

type Report struct {
	State    capability.State `json:"state"`
	Summary  Summary          `json:"summary"`
	Findings []Finding        `json:"findings"`
}

type TenantState struct {
	Slug          string
	Active        bool
	VaultOK       bool
	GraphEngine   string
	GraphIndexed  bool
	GraphEvidence []string
}

type CronState struct {
	Supported bool
	Readable  bool
	Installed bool
	Entry     string
	Binary    string
	HasPATH   bool
	Error     string
}

// Run evaluates the current local machine without modifying it.
func Run(report detect.Report) Report {
	return Analyze(report, inspectTenants(), inspectCron())
}

// Analyze is the deterministic core used by tests and alternative frontends.
func Analyze(report detect.Report, tenants []TenantState, cron CronState) Report {
	var findings []Finding

	if len(report.Multiversa.CLIBinaries) > 1 {
		evidence := make([]string, 0, len(report.Multiversa.CLIBinaries))
		versions := map[string]bool{}
		for _, b := range report.Multiversa.CLIBinaries {
			marker := ""
			if b.Active {
				marker = " (PATH activo)"
			}
			evidence = append(evidence, b.Path+" · "+orUnknown(b.Version)+marker)
			versions[b.Version] = true
		}
		detail := "Hay más de un binario de Multiversa; cualquier automatización puede ejecutar una copia distinta."
		if len(versions) > 1 {
			detail = "Los binarios de Multiversa reportan builds diferentes; PATH y automatizaciones pueden observar comportamientos distintos."
		}
		findings = append(findings, Finding{
			ID: "cli.duplicate-binaries", Severity: P2, Component: "multiversa-cli",
			Title: "Binarios duplicados de Multiversa", Detail: detail, Evidence: evidence,
			NextActions: []string{"select-canonical-user-binary", "repoint-automation-to-canonical-binary"}, RequiresApproval: true,
		})
	}

	for _, t := range report.Tools {
		if t.State != capability.Blocked {
			continue
		}
		findings = append(findings, Finding{
			ID: "tool." + t.Name + ".probe-blocked", Severity: P2, Component: t.Name,
			Title: "Herramienta detectada pero no verificable", Detail: t.ProbeError,
			Evidence: []string{"binary:" + t.Path}, Missing: []string{"working-version-probe"},
			NextActions: []string{"repair-runtime:" + t.Name}, RequiresApproval: true,
		})
	}

	for _, e := range report.Multiversa.Engines {
		if e.State != capability.Blocked {
			continue
		}
		severity := P2
		if !e.OptIn {
			severity = P1
		}
		findings = append(findings, Finding{
			ID: "engine." + e.ID + ".blocked", Severity: severity, Component: e.ID,
			Title:    "Integración instalada pero bloqueada",
			Detail:   "Existe evidencia del componente, pero faltan requisitos para ejecutarlo de forma confiable.",
			Evidence: e.Evidence, Missing: e.Missing, NextActions: e.NextActions, RequiresApproval: true,
		})
	}

	for _, a := range report.Agents {
		if a.State == capability.Blocked {
			findings = append(findings, Finding{
				ID: "agent." + a.ID + ".blocked", Severity: P2, Component: a.ID,
				Title: "Runtime de agente bloqueado", Detail: "Se encontraron artefactos, pero el runtime no está utilizable.",
				Evidence: a.Evidence, Missing: a.Missing, NextActions: a.NextActions, RequiresApproval: true,
			})
		}
		if a.ID == "hermes" && a.Installed && a.Configured && !a.Connected {
			findings = append(findings, Finding{
				ID: "agent.hermes.mcp-not-connected", Severity: P3, Component: "hermes",
				Title:    "Hermes aún no consume el MCP de Multiversa",
				Detail:   "Hermes está instalado y configurado, pero no existe una entrada mcp_servers.multiversa.",
				Evidence: a.Evidence, Missing: []string{"mcp:multiversa"},
				NextActions: []string{"multiversa connect hermes"}, RequiresApproval: true,
			})
		}
	}

	for _, t := range tenants {
		if !t.VaultOK {
			findings = append(findings, Finding{
				ID: "tenant." + t.Slug + ".vault-permissions", Severity: P0, Component: "tenant:" + t.Slug,
				Title: "Vault ausente o con permisos inseguros", Detail: "El vault del tenant debe existir con permisos 0700.",
				Missing: []string{"vault:0700"}, NextActions: []string{"repair-tenant-vault-permissions"}, RequiresApproval: true,
			})
		}
		if t.GraphEngine != "" && !t.GraphIndexed {
			findings = append(findings, Finding{
				ID: "tenant." + t.Slug + ".graph-not-indexed", Severity: P2, Component: "tenant:" + t.Slug,
				Title:    "Grafo declarado pero no indexado",
				Detail:   fmt.Sprintf("El manifiesto declara %s, pero no hay artefactos válidos de índice.", t.GraphEngine),
				Evidence: t.GraphEvidence, Missing: []string{"graph-index"},
				NextActions: []string{"bootstrap-or-index-tenant:" + t.Slug}, RequiresApproval: true,
			})
		}
	}

	if cron.Supported && !cron.Readable {
		findings = append(findings, Finding{
			ID: "updates.cron-unreadable", Severity: P2, Component: "updates-cron",
			Title: "No se pudo verificar el cron de actualizaciones", Detail: cron.Error,
			Missing: []string{"readable-crontab"}, NextActions: []string{"inspect-crontab-permissions"}, RequiresApproval: true,
		})
	} else if cron.Installed {
		if cron.Binary != "" && report.Multiversa.CLIPath != "" && !samePath(cron.Binary, report.Multiversa.CLIPath) {
			findings = append(findings, Finding{
				ID: "updates.cron-binary-drift", Severity: P2, Component: "updates-cron",
				Title:       "El cron ejecuta otro binario de Multiversa",
				Detail:      "El watcher diario no usa el mismo binario que selecciona el PATH interactivo.",
				Evidence:    []string{"cron:" + cron.Binary, "path:" + report.Multiversa.CLIPath},
				NextActions: []string{"repoint-cron-to-canonical-binary"}, RequiresApproval: true,
			})
		}
		if !cron.HasPATH {
			findings = append(findings, Finding{
				ID: "updates.cron-path-incomplete", Severity: P2, Component: "updates-cron",
				Title:       "El cron no declara un PATH reproducible",
				Detail:      "Los motores instalados en rutas de usuario pueden aparecer como ausentes durante el chequeo diario.",
				Evidence:    []string{"cron-entry-without-explicit-path"},
				NextActions: []string{"add-explicit-user-path-to-cron"}, RequiresApproval: true,
			})
		}
	}

	sortFindings(findings)
	return Report{State: overallState(findings), Summary: summarize(findings), Findings: findings}
}

func inspectTenants() []TenantState {
	infos, err := tenant.List()
	if err != nil {
		return nil
	}
	out := make([]TenantState, 0, len(infos))
	for _, info := range infos {
		st := TenantState{Slug: info.Slug, Active: info.Active, VaultOK: info.VaultOK}
		m, _, err := tenant.Load(info.Slug)
		if err == nil {
			st.GraphEngine = m.Graph.Engine
		}
		if info.Dir != "" {
			for _, rel := range []string{
				filepath.Join("graph", "graphify-out", "graph.json"),
				filepath.Join("graph", "graphify-out", "manifest.json"),
			} {
				p := filepath.Join(info.Dir, rel)
				if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Size() > 0 {
					st.GraphEvidence = append(st.GraphEvidence, "artifact:"+p)
				}
			}
			st.GraphIndexed = len(st.GraphEvidence) == 2
		}
		out = append(out, st)
	}
	return out
}

func inspectCron() CronState {
	st := CronState{Supported: true}
	if _, err := exec.LookPath("crontab"); err != nil {
		st.Supported = false
		return st
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "crontab", "-l").CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(strings.ToLower(msg), "no crontab") {
			st.Readable = true
			return st
		}
		st.Error = strings.TrimSpace(fmt.Sprintf("%v: %s", err, msg))
		return st
	}
	st.Readable = true
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "# multiversa-updates") {
			continue
		}
		st.Installed = true
		st.Entry = strings.TrimSpace(line)
		st.Binary, st.HasPATH = parseCronEntry(line)
		break
	}
	return st
}

func parseCronEntry(line string) (binary string, hasPATH bool) {
	fields := strings.Fields(line)
	if len(fields) <= 5 {
		return "", false
	}
	for _, field := range fields[5:] {
		if strings.HasPrefix(field, "PATH=") || strings.HasPrefix(field, "env=PATH=") {
			hasPATH = true
			continue
		}
		if strings.Contains(field, "=") && !strings.Contains(field, "/") {
			continue
		}
		return field, hasPATH
	}
	return "", hasPATH
}

func samePath(a, b string) bool {
	aa, errA := filepath.EvalSymlinks(a)
	bb, errB := filepath.EvalSymlinks(b)
	if errA == nil {
		a = aa
	}
	if errB == nil {
		b = bb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func summarize(findings []Finding) Summary {
	s := Summary{Open: len(findings)}
	for _, f := range findings {
		switch f.Severity {
		case P0:
			s.P0++
		case P1:
			s.P1++
		case P2:
			s.P2++
		case P3:
			s.P3++
		}
	}
	return s
}

func overallState(findings []Finding) capability.State {
	for _, f := range findings {
		if f.Severity == P0 || f.Severity == P1 {
			return capability.Blocked
		}
	}
	for _, f := range findings {
		if f.Severity == P2 {
			return capability.Drifted
		}
	}
	return capability.Healthy
}

func sortFindings(findings []Finding) {
	rank := map[Severity]int{P0: 0, P1: 1, P2: 2, P3: 3}
	sort.SliceStable(findings, func(i, j int) bool {
		if rank[findings[i].Severity] != rank[findings[j].Severity] {
			return rank[findings[i].Severity] < rank[findings[j].Severity]
		}
		return findings[i].ID < findings[j].ID
	})
}

func orUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "versión desconocida"
	}
	return s
}
