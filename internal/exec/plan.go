package exec

import "github.com/moshequantum/multiversa-cli/internal/lang"

// RunPlan executes an install plan. Shell plans (curl-pipe style) use
// sh -c; program plans invoke the binary directly.
func RunPlan(p lang.Plan) error {
	if p.Shell != "" {
		return Run("sh", "-c", p.Shell).Err
	}
	return Run(p.Program, p.Args...).Err
}
