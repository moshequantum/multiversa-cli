package stack

import (
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/lang"
)

// ToolPlan bundles a lang.Tool with its current installation state
// and the resolved Plan for display and execution.
type ToolPlan struct {
	Tool      lang.Tool
	Installed bool
	Plan      lang.Plan
	Err       error
}

// PlanSummary returns a short human-readable description of a Plan.
func PlanSummary(p lang.Plan) string {
	switch {
	case p.Shell != "":
		return planTruncate(p.Shell, 70)
	case p.Program != "":
		return planTruncate(p.Program+" "+strings.Join(p.Args, " "), 70)
	}
	return "(sin plan)"
}

func planTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
