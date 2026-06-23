package profile

import (
	"encoding/json"

	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

// MirrorEngram persists a JSON snapshot of the profile into Engram
// under topic_key "multiversa/profile". If the `engram` binary is
// not on PATH, the call is a no-op — Engram integration is opt-in
// and must never block the local TOML write.
func MirrorEngram(p Profile) error {
	if !xexec.Check("engram") {
		return nil
	}
	payload, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	r := xexec.Run("engram", "save",
		"multiversa/profile",
		string(payload),
		"--type", "config",
		"--project", "multiversa")
	return r.Err
}
