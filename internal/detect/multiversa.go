package detect

import (
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"

	"github.com/moshequantum/multiversa-cli/internal/capability"
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
	"github.com/moshequantum/multiversa-cli/internal/stack"
)

// MultiversaState describes how much of the Multiversa ecosystem is wired
// up on this host: the CLI itself, each curated engine, and any locally
// detected repos.
type MultiversaState struct {
	CLIInstalled bool          `json:"cli_installed"`
	CLIVersion   string        `json:"cli_version,omitempty"`
	CLIPath      string        `json:"cli_path,omitempty"`
	CLIBinaries  []BinaryState `json:"cli_binaries,omitempty"`
	HomeDir      string        `json:"home_dir,omitempty"` // ~/.multiversa, if it exists
	Engines      []EngineState `json:"engines"`
	Repos        []string      `json:"repos,omitempty"` // detected repo paths under common locations
}

// EngineState is the per-engine slice of the Multiversa report.
type EngineState struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Author           string           `json:"author"`
	Installed        bool             `json:"installed"`
	Version          string           `json:"version,omitempty"`
	OptIn            bool             `json:"opt_in"`
	State            capability.State `json:"state"`
	Path             string           `json:"path,omitempty"`
	Evidence         []string         `json:"evidence,omitempty"`
	Missing          []string         `json:"missing,omitempty"`
	NextActions      []string         `json:"next_actions,omitempty"`
	RequiresApproval bool             `json:"requires_approval"`
}

// BinaryState records each known CLI installation without deciding which one
// should be deleted. Active is the binary currently selected by PATH.
type BinaryState struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Active  bool   `json:"active"`
}

func detectMultiversa(engines []stack.Engine) MultiversaState {
	st := MultiversaState{}

	// 1. CLI itself.
	if p, err := osexec.LookPath("multiversa"); err == nil {
		st.CLIInstalled = true
		st.CLIPath = p
		if r := xexec.Run("multiversa", "version"); r.Err == nil {
			st.CLIVersion = r.LastLine()
		}
	}
	st.CLIBinaries = detectCLIBinaries(st.CLIPath)

	// 2. ~/.multiversa state directory.
	if home, err := os.UserHomeDir(); err == nil {
		mvHome := filepath.Join(home, ".multiversa")
		if fi, err := os.Stat(mvHome); err == nil && fi.IsDir() {
			st.HomeDir = mvHome
		}
	}

	// 3. Engines (delegated to the stack registry — single source of truth).
	for _, eng := range engines {
		es := EngineState{
			ID:     eng.ID(),
			Name:   eng.DisplayName(),
			Author: eng.Author(),
			OptIn:  eng.OptIn(),
			State:  capability.Absent,
		}
		if status, err := eng.Status(); err == nil {
			es.Installed = status.Installed
			es.Version = status.Version
			es.Path = status.Path
			es.Evidence = append(es.Evidence, status.Evidence...)
			es.Missing = append(es.Missing, status.Missing...)
			switch {
			case status.Blocked:
				es.State = capability.Blocked
				es.RequiresApproval = true
				es.NextActions = []string{"review-and-repair:" + eng.ID()}
			case status.Installed:
				es.State = capability.Installed
			}
		}
		st.Engines = append(st.Engines, es)
	}

	// 4. Local repos (best-effort, only well-known paths).
	st.Repos = detectKnownRepos()
	return st
}

func detectCLIBinaries(active string) []BinaryState {
	home, _ := os.UserHomeDir()
	candidates := []string{active, filepath.Join(home, ".local", "bin", "multiversa"), filepath.Join(home, "go", "bin", "multiversa")}
	if filepath.Separator == '/' {
		candidates = append(candidates, "/usr/local/bin/multiversa")
	}
	seen := map[string]bool{}
	var out []BinaryState
	for _, p := range candidates {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() || fi.Mode()&0o111 == 0 {
			continue
		}
		b := BinaryState{Path: p, Active: p == active}
		if r := xexec.Run(p, "version"); r.Err == nil {
			b.Version = r.LastLine()
		}
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Active != out[j].Active {
			return out[i].Active
		}
		return out[i].Path < out[j].Path
	})
	return out
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
		filepath.Join(home, "Documentos", "Multiversa", "multiversa-cli"),
		filepath.Join(home, "Documentos", "Multiversa", "multiversagroup"),
		filepath.Join(home, "Documentos", "Multiversa", "multiversa-lab"),
	}
	var found []string
	for _, p := range candidates {
		if _, err := os.Stat(filepath.Join(p, ".git")); err == nil {
			found = append(found, p)
		}
	}
	return found
}
