package profile

import (
	"encoding/json"
	"os/exec"
)

// MirrorEngram persists a JSON snapshot of the profile into Engram
// under topic_key "multiversa/profile". If the `engram` binary is
// not on PATH, the call is a no-op — Engram integration is opt-in
// and must never block the local TOML write.
//
// Returns the engram CLI exit error if the call ran but failed,
// nil otherwise. Callers should ignore non-nil returns for routine
// flows; the function is logging-only.
func MirrorEngram(p Profile) error {
	if !engramAvailable() {
		return nil
	}
	payload, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	// Use --project=multiversa so the mirror lives alongside other
	// CLI-related memories rather than scattered globally.
	cmd := exec.Command("engram", "save",
		"multiversa/profile",
		string(payload),
		"--type", "config",
		"--project", "multiversa")
	return cmd.Run()
}

// engramAvailable returns true if the engram binary can be located
// on PATH. We do not run `engram version` to keep the check cheap.
func engramAvailable() bool {
	_, err := exec.LookPath("engram")
	return err == nil
}
