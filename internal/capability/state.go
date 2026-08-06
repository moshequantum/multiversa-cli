// Package capability defines the shared lifecycle used by every runtime,
// engine, agent, connection and tenant resource that Multiversa audits.
package capability

// State is evidence-backed capability state. Values are intentionally stable:
// they are part of the JSON contract consumed by agents and Studio.
type State string

const (
	Absent          State = "absent"
	Detected        State = "detected"
	Installed       State = "installed"
	Configured      State = "configured"
	Connected       State = "connected"
	Indexed         State = "indexed"
	Healthy         State = "healthy"
	Drifted         State = "drifted"
	UpdateAvailable State = "update_available"
	Blocked         State = "blocked"
)

// Ready reports whether the capability can currently be used. A configured
// component is not assumed connected; callers that require connectivity must
// compare the exact state instead.
func (s State) Ready() bool {
	switch s {
	case Installed, Configured, Connected, Indexed, Healthy:
		return true
	default:
		return false
	}
}
