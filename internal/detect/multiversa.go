package detect

import (
	"os"
	"path/filepath"

	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
	"github.com/moshequantum/multiversa-cli/internal/stack"
)

// MultiversaState describes how much of the Multiversa ecosystem is wired
// up on this host: the CLI itself, each curated engine, and any locally
// detected repos.
type MultiversaState struct {
	CLIInstalled bool
	CLIVersion   string
	HomeDir      string   // ~/.multiversa, if it exists
	Engines      []EngineState
	Repos        []string // detected repo paths under common locations
}

// EngineState is the per-engine slice of the Multiversa report.
type EngineState struct {
	ID        string
	Name      string
	Author    string
	Installed bool
	Version   string
	OptIn     bool
}

func detectMultiversa(registry map[string]stack.Engine) MultiversaState {
	st := MultiversaState{}

	// 1. CLI itself.
	if xexec.Check("multiversa") {
		st.CLIInstalled = true
		if r := xexec.Run("multiversa", "version"); r.Err == nil {
			st.CLIVersion = r.LastLine()
		}
	}

	// 2. ~/.multiversa state directory.
	if home, err := os.UserHomeDir(); err == nil {
		mvHome := filepath.Join(home, ".multiversa")
		if fi, err := os.Stat(mvHome); err == nil && fi.IsDir() {
			st.HomeDir = mvHome
		}
	}

	// 3. Engines (delegated to the stack registry — single source of truth).
	for id, eng := range registry {
		es := EngineState{
			ID:     id,
			Name:   eng.DisplayName(),
			Author: eng.Author(),
			OptIn:  eng.OptIn(),
		}
		if status, err := eng.Status(); err == nil {
			es.Installed = status.Installed
			es.Version = status.Version
		}
		st.Engines = append(st.Engines, es)
	}

	// 4. Local repos (best-effort, only well-known paths).
	st.Repos = detectKnownRepos()
	return st
}

// detectKnownRepos checks the canonical Multiversa workspace path and
// returns any sub-paths that look like git repos.
//
// We deliberately do NOT walk the filesystem — that would be slow and
// leak private paths into reports. We only check expected locations.
func detectKnownRepos() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []string{
		filepath.Join(home, "Documents", "01_Multiversa", "Lab", "repo"),
		filepath.Join(home, "Documents", "01_Multiversa", "Group", "repo"),
		filepath.Join(home, "Documents", "01_Multiversa", "Shared", "multiversa-cli"),
	}
	var found []string
	for _, p := range candidates {
		if _, err := os.Stat(filepath.Join(p, ".git")); err == nil {
			found = append(found, p)
		}
	}
	return found
}
