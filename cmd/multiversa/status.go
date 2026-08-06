package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	alertledger "github.com/moshequantum/multiversa-cli/internal/alerts"
	"github.com/moshequantum/multiversa-cli/internal/capability"
	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/doctor"
	"github.com/moshequantum/multiversa-cli/internal/tenant"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type statusJSON struct {
	State        capability.State       `json:"state"`
	ActiveTenant *tenant.Info           `json:"active_tenant,omitempty"`
	Detection    detect.Summary         `json:"detection"`
	CLI          detect.MultiversaState `json:"cli"`
	Agents       []detect.AgentState    `json:"agents"`
	Doctor       doctor.Summary         `json:"doctor"`
	Alerts       alertledger.Summary    `json:"alerts"`
	LedgerPath   string                 `json:"ledger_path,omitempty"`
	LedgerError  string                 `json:"ledger_error,omitempty"`
	NextAction   *doctor.Finding        `json:"next_action,omitempty"`
}

func buildStatus(refreshLedger bool, now time.Time) statusJSON {
	detection := detect.Run()
	diagnosis := doctor.Run(detection)
	data := statusJSON{
		State: detectionState(diagnosis), Detection: detection.Summarize(),
		CLI: detection.Multiversa, Agents: detection.Agents, Doctor: diagnosis.Summary,
	}
	if infos, err := tenant.List(); err == nil {
		active := tenant.Active()
		for i := range infos {
			if infos[i].Slug == active {
				copy := infos[i]
				data.ActiveTenant = &copy
				break
			}
		}
	}
	if path, err := alertledger.DefaultPath(); err == nil {
		data.LedgerPath = path
		var ledger alertledger.Ledger
		if refreshLedger {
			ledger, err = alertledger.Reconcile(path, diagnosis.Findings, now)
		} else {
			ledger, err = alertledger.Load(path)
		}
		if err != nil {
			data.LedgerError = err.Error()
		} else {
			data.Alerts = ledger.Summary()
		}
	} else {
		data.LedgerError = err.Error()
	}
	if len(diagnosis.Findings) > 0 {
		next := diagnosis.Findings[0]
		data.NextAction = &next
	}
	return data
}

func detectionState(diagnosis doctor.Report) capability.State {
	return diagnosis.State
}

func renderStatus(w io.Writer, data statusJSON) {
	fmt.Fprintln(w, theme.Accent.Render("multiversa status")+theme.Dim.Render(" · vista diaria del operador"))
	fmt.Fprintf(w, "  estado     %s\n", data.State)
	if data.ActiveTenant == nil {
		fmt.Fprintln(w, "  tenant     ninguno activo")
	} else {
		name := data.ActiveTenant.Name
		if name == "" {
			name = data.ActiveTenant.Slug
		}
		fmt.Fprintf(w, "  tenant     %s (%s)\n", name, data.ActiveTenant.Slug)
	}
	fmt.Fprintf(w, "  cli        %s · %s\n", data.CLI.CLIPath, data.CLI.CLIVersion)
	fmt.Fprintf(w, "  stack      %d/%d motores · %d/%d agentes\n", data.Detection.EnginesReady, data.Detection.EnginesTotal, data.Detection.AgentsReady, data.Detection.AgentsTotal)
	fmt.Fprintf(w, "  alertas    %d abiertas · P0:%d · P1:%d · P2:%d · P3:%d\n", data.Alerts.Open, data.Alerts.P0, data.Alerts.P1, data.Alerts.P2, data.Alerts.P3)
	if data.NextAction != nil {
		fmt.Fprintln(w, "\n"+theme.Label.Render("Próxima acción aprobable"))
		fmt.Fprintf(w, "  [%s] %s\n", data.NextAction.Severity, data.NextAction.Title)
		if len(data.NextAction.NextActions) > 0 {
			fmt.Fprintf(w, "  propone    %s\n", data.NextAction.NextActions[0])
		}
	}
	if data.LedgerError != "" {
		fmt.Fprintln(w, "\n"+theme.Warn.Render("⚠ ledger: "+data.LedgerError))
	}
	fmt.Fprintln(w, theme.Dim.Render("\nVer evidencia: multiversa doctor · Historial: multiversa alerts"))
}

func newStatusCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Resume tenant activo, stack, salud, alertas y próxima acción.",
		RunE: func(cmd *cobra.Command, args []string) error {
			data := buildStatus(true, time.Now())
			if jsonOut {
				return agentout.Emit(os.Stdout, "status", data)
			}
			renderStatus(os.Stdout, data)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.status/v1).")
	return cmd
}
