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
	OS         OSInfo
	Tools      []Tool
	Multiversa MultiversaState
}

// Run executes a full local scan and returns the populated Report.
// It does not return an error — partial data is preferable to none.
func Run() Report {
	return Report{
		OS:         detectOS(),
		Tools:      detectTools(),
		Multiversa: detectMultiversa(stack.Registry()),
	}
}

// ReadyTools returns how many of the inspected tools are installed.
func (r Report) ReadyTools() (installed, total int) {
	for _, t := range r.Tools {
		if !t.Advisory {
			total++
			if t.Installed {
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
		if e.Installed {
			installed++
		}
	}
	return
}
