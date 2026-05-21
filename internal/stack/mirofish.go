package stack

import (
	xexec "github.com/moshequantum/multiversa-cli/internal/exec"
)

// MiroFish is invoked external-only because of AGPL-3.0. Multiversa never
// embeds, vendors, forks, or `go install`s MiroFish source. We pull the
// upstream container image and treat it as a separate service.
type MiroFish struct {
	AgplAcknowledged bool
}

func (MiroFish) ID() string          { return "mirofish" }
func (MiroFish) DisplayName() string { return "MiroFish" }
func (MiroFish) Author() string      { return "666ghj" }
func (MiroFish) Repo() string        { return "https://github.com/666ghj/MiroFish" }
func (MiroFish) License() string     { return "AGPL-3.0" }
func (MiroFish) OptIn() bool         { return true }
func (MiroFish) Prereq() string      { return "docker" }

func (m MiroFish) Command(version string) []string {
	tag := version
	if tag == "" {
		tag = "latest"
	}
	// Pulled as an external service. Confirm canonical image registry with
	// upstream (see .outreach/baifu.md) — fallback to GHCR placeholder.
	return []string{"docker", "pull", "ghcr.io/666ghj/mirofish:" + tag}
}

func (m MiroFish) Install(version string) error {
	if !m.AgplAcknowledged {
		return ErrAgplConsentRequired
	}
	cmd := m.Command(version)
	return xexec.Run(cmd[0], cmd[1:]...).Err
}

func (m MiroFish) Status() (Status, error) {
	if !xexec.Check("docker") {
		return Status{}, nil
	}
	r := xexec.Run("docker", "image", "inspect", "ghcr.io/666ghj/mirofish:latest", "--format", "{{.Created}}")
	if r.Err != nil {
		return Status{Installed: false}, nil
	}
	return Status{Installed: true, Version: r.LastLine()}, nil
}

func (m MiroFish) Uninstall() error { return ErrNotImplemented }
