// Package detect scans the local environment for OS, package manager,
// developer toolchain, and Multiversa state. Used by `multiversa detect`
// and `multiversa doctor` to render a single consultive report.
//
// Detection is read-only: no installs, no network, no mutation.
package detect

import "github.com/moshequantum/multiversa-cli/internal/stack"

// Report is the canonical detection result. Render() turns it into a
// Charm-styled string for the terminal; the same struct can be JSON-encoded
// for the skill layer in Claude Code.
type Report struct {
	OS         OSInfo          `json:"os"`
	Tools      []Tool          `json:"tools"`
	Multiversa MultiversaState `json:"multiversa"`
	Agents     []AgentState    `json:"agents"`
}

// Summary is the aggregate readiness view included in JSON output so
// agents can branch on totals without re-deriving them.
type Summary struct {
	ToolsReady   int `json:"tools_ready"`
	ToolsTotal   int `json:"tools_total"`
	EnginesReady int `json:"engines_ready"`
	EnginesTotal int `json:"engines_total"`
	AgentsReady  int `json:"agents_ready"`
	AgentsTotal  int `json:"agents_total"`
}

// Summarize computes the Summary for this report.
func (r Report) Summarize() Summary {
	ti, tt := r.ReadyTools()
	ei, et := r.ReadyEngines()
	ai, at := r.ReadyAgents()
	return Summary{ToolsReady: ti, ToolsTotal: tt, EnginesReady: ei, EnginesTotal: et, AgentsReady: ai, AgentsTotal: at}
}

// Run executes a full local scan and returns the populated Report.
// It does not return an error — partial data is preferable to none.
func Run() Report {
	return Report{
		OS:         detectOS(),
		Tools:      detectTools(),
		Multiversa: detectMultiversa(stack.List()),
		Agents:     detectAgents(),
	}
}

// ReadyTools returns how many of the inspected tools are installed.
func (r Report) ReadyTools() (installed, total int) {
	for _, t := range r.Tools {
		if !t.Advisory {
			total++
			if (t.State == "" && t.Installed) || t.State.Ready() {
				installed++
			}
		}
	}
	return
}

// ReadyEngines returns how many Multiversa engines are installed.
func (r Report) ReadyEngines() (installed, total int) {
	for _, e := range r.Multiversa.Engines {
		total++
		if (e.State == "" && e.Installed) || e.State.Ready() {
			installed++
		}
	}
	return
}

// ReadyAgents returns how many optional agent runtimes are usable locally.
func (r Report) ReadyAgents() (ready, total int) {
	for _, a := range r.Agents {
		total++
		if a.State.Ready() {
			ready++
		}
	}
	return
}
