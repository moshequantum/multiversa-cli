package lang

// Docker is system-level on Linux (apt/dnf/pacman repo) and a desktop
// app on macOS/Windows. We don't auto-launch the GUI installer; we
// surface the right command and let the user click through. This keeps
// the consultive contract honest: the wizard never silently swallows a
// GUI prompt.
type Docker struct{}

func (Docker) ID() string          { return "docker" }
func (Docker) DisplayName() string { return "Docker" }
func (Docker) Description() string { return "Container runtime. Engine on Linux, Desktop on macOS/Windows." }
func (Docker) Probe() string       { return "docker" }
func (d Docker) Installed() bool   { return installed(d.Probe()) }

func (Docker) PlanFor(osKind, pkgMgr string) (Plan, error) {
	switch osKind {
	case "darwin":
		if pkgMgr == "brew" {
			return Plan{
				Program: "brew",
				Args:    []string{"install", "--cask", "docker"},
				Notes:   "Docker Desktop for Mac (cask). Launch it once from /Applications after install.",
			}, nil
		}
		return Plan{}, ErrUnsupportedOS
	case "linux":
		switch pkgMgr {
		case "apt":
			return Plan{
				Shell: "curl -fsSL https://get.docker.com | sh",
				Notes: "Official Docker convenience script. Run `sudo usermod -aG docker $USER` and re-login after.",
			}, nil
		case "dnf":
			return Plan{
				Shell: "curl -fsSL https://get.docker.com | sh",
				Notes: "Official Docker convenience script. Run `sudo usermod -aG docker $USER` and re-login after.",
			}, nil
		case "pacman":
			return Plan{
				Program: "sudo",
				Args:    []string{"pacman", "-S", "--noconfirm", "docker"},
				Notes:   "Enable and start: `sudo systemctl enable --now docker`. Add user: `sudo usermod -aG docker $USER`.",
			}, nil
		}
		return Plan{}, ErrUnsupportedOS
	case "windows":
		return Plan{
			Program: "winget",
			Args:    []string{"install", "--id", "Docker.DockerDesktop", "-e", "--source", "winget"},
			Notes:   "Docker Desktop for Windows. Requires WSL2 backend (winget will prompt).",
		}, nil
	}
	return Plan{}, ErrUnsupportedOS
}
