package lang

// Python uses pyenv on Unix for per-project version management. On
// Windows we install the latest official Python via winget — pyenv-win
// exists but adds an extra dependency we don't currently audit.
type Python struct{}

func (Python) ID() string          { return "python" }
func (Python) DisplayName() string { return "Python" }
func (Python) Description() string { return "Python runtime. pyenv-managed on Unix, official MSI on Windows." }
func (Python) Probe() string       { return "python3" }
func (p Python) Installed() bool   { return installed(p.Probe()) }

func (Python) PlanFor(osKind, _ string) (Plan, error) {
	switch osKind {
	case "darwin", "linux":
		return Plan{
			Shell: "curl -fsSL https://pyenv.run | bash",
			Notes: "pyenv → ~/.pyenv. After install, run `pyenv install 3.12` and `pyenv global 3.12`.",
		}, nil
	case "windows":
		return Plan{
			Program: "winget",
			Args:    []string{"install", "--id", "Python.Python.3.12", "-e", "--source", "winget"},
			Notes:   "Latest Python 3.12 from python.org via winget. pyenv-win is an option for version switching.",
		}, nil
	}
	return Plan{}, ErrUnsupportedOS
}
