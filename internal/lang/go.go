package lang

// Go (the language) uses the package manager when available. We don't
// install from the official tarball by default because that requires
// writing to /usr/local without sudo, which breaks the no-sudo
// portability promise.
type Go struct{}

func (Go) ID() string          { return "go" }
func (Go) DisplayName() string { return "Go" }
func (Go) Description() string { return "Go toolchain. Installed via the host package manager." }
func (Go) Probe() string       { return "go" }
func (g Go) Installed() bool   { return installed(g.Probe()) }

func (Go) PlanFor(osKind, pkgMgr string) (Plan, error) {
	switch osKind {
	case "darwin":
		if pkgMgr == "brew" {
			return Plan{Program: "brew", Args: []string{"install", "go"}}, nil
		}
		return Plan{}, ErrUnsupportedOS
	case "linux":
		switch pkgMgr {
		case "apt":
			return Plan{
				Program: "sudo",
				Args:    []string{"apt", "install", "-y", "golang-go"},
				Notes:   "Ubuntu/Debian package may lag behind upstream — fine for Multiversa CLI work.",
			}, nil
		case "dnf":
			return Plan{Program: "sudo", Args: []string{"dnf", "install", "-y", "golang"}}, nil
		case "pacman":
			return Plan{Program: "sudo", Args: []string{"pacman", "-S", "--noconfirm", "go"}}, nil
		}
		return Plan{}, ErrUnsupportedOS
	case "windows":
		return Plan{
			Program: "winget",
			Args:    []string{"install", "--id", "GoLang.Go", "-e", "--source", "winget"},
		}, nil
	}
	return Plan{}, ErrUnsupportedOS
}
