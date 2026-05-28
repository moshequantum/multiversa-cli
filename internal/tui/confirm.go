package tui

import "strings"

// ConfirmDecision interprets a user response to a yes/no prompt.
// The contract is intentionally strict for destructive operations:
// only an explicit "y", "Y", "yes", or "YES" returns true. Everything
// else — including blank, spaces, "no", ambiguous strings — returns
// false. This is the no-default-yes rule baked into v0.4.0 specs.
func ConfirmDecision(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true
	}
	return false
}
