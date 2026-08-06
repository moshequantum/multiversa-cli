package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moshequantum/multiversa-cli/internal/agentout"
	"github.com/moshequantum/multiversa-cli/internal/detect"
	"github.com/moshequantum/multiversa-cli/internal/doctor"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

type doctorJSON struct {
	Report doctor.Report `json:"report"`
}

func runDoctor(stdout io.Writer, jsonOut bool) error {
	report := doctor.Run(detect.Run())
	if jsonOut {
		return agentout.Emit(stdout, "doctor", doctorJSON{Report: report})
	}
	renderDoctor(stdout, report)
	return nil
}

func renderDoctor(w io.Writer, report doctor.Report) {
	fmt.Fprintln(w, theme.Accent.Render("multiversa doctor")+theme.Dim.Render(" · diagnóstico local · solo lectura"))
	fmt.Fprintf(w, "  estado     %s\n", report.State)
	fmt.Fprintf(w, "  alertas    P0:%d · P1:%d · P2:%d · P3:%d\n\n", report.Summary.P0, report.Summary.P1, report.Summary.P2, report.Summary.P3)
	if len(report.Findings) == 0 {
		fmt.Fprintln(w, theme.Accent.Render("✓ sin hallazgos abiertos"))
		return
	}
	for _, f := range report.Findings {
		line := fmt.Sprintf("[%s] %s — %s", f.Severity, f.Title, f.Detail)
		if f.Severity == doctor.P0 || f.Severity == doctor.P1 {
			fmt.Fprintln(w, theme.Warn.Render(line))
		} else {
			fmt.Fprintln(w, theme.Body.Render(line))
		}
		if len(f.Missing) > 0 {
			fmt.Fprintln(w, theme.Dim.Render("     falta: "+strings.Join(f.Missing, ", ")))
		}
		if len(f.NextActions) > 0 {
			fmt.Fprintln(w, theme.Dim.Render("     propone: "+strings.Join(f.NextActions, " · ")))
		}
	}
	fmt.Fprintln(w, theme.Dim.Render("\nLa IA propone; ninguna corrección fue aplicada."))
}

func newDoctorCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica incoherencias del host con evidencia, sin modificar nada.",
		Long: "Evalúa el inventario local y reporta binarios duplicados, runtimes bloqueados,\n" +
			"índices ausentes, permisos inseguros y deriva del cron. Nunca aplica correcciones.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(os.Stdout, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit a JSON envelope (schema multiversa.doctor/v1).")
	return cmd
}
