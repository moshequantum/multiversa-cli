package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	alertledger "github.com/moshequantum/multiversa-cli/internal/alerts"
	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/doctor"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type alertsJSON struct {
	Path    string              `json:"path"`
	Summary alertledger.Summary `json:"summary"`
	Alerts  []alertledger.Alert `json:"alerts"`
}

func newAlertsCmd() *cobra.Command {
	var jsonOut, all, noRefresh bool
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "Actualiza y muestra el ledger local de hallazgos.",
		Long: "Mantiene ~/.multiversa/alerts.json con primera/última aparición y resolución.\n" +
			"No envía datos ni aplica correcciones.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := alertledger.DefaultPath()
			if err != nil {
				return err
			}
			var ledger alertledger.Ledger
			if noRefresh {
				ledger, err = alertledger.Load(path)
			} else {
				diagnosis := doctor.Run(detect.Run())
				ledger, err = alertledger.Reconcile(path, diagnosis.Findings, time.Now())
			}
			if err != nil {
				return err
			}
			shown := ledger.Alerts
			if !all {
				shown = ledger.OpenAlerts()
			}
			payload := alertsJSON{Path: path, Summary: ledger.Summary(), Alerts: shown}
			if jsonOut {
				return agentout.Emit(os.Stdout, "alerts", payload)
			}
			fmt.Println(theme.Accent.Render("multiversa alerts") + theme.Dim.Render(" · "+path))
			if len(shown) == 0 {
				fmt.Println(theme.Accent.Render("✓ sin alertas abiertas"))
				return nil
			}
			for _, a := range shown {
				fmt.Printf("  [%s] %-8s %s\n", a.Finding.Severity, a.State, a.Finding.Title)
			}
			fmt.Println(theme.Dim.Render("\nLa IA propone; usa doctor para ver evidencia y acciones."))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.alerts/v1).")
	cmd.Flags().BoolVar(&all, "all", false, "Incluye alertas resueltas.")
	cmd.Flags().BoolVar(&noRefresh, "no-refresh", false, "Lee el ledger sin ejecutar un diagnóstico nuevo.")
	return cmd
}
