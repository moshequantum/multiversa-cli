package lang

// Pnpm installs pnpm via the official standalone script. This is the
// only Node package manager Multiversa supports — `npm` is banned by
// policy (see CONSTITUTION.md). The standalone installer does not
// require Node to be present yet; it brings its own bundled runtime.
type Pnpm struct{}

func (Pnpm) ID() string          { return "pnpm" }
func (Pnpm) DisplayName() string { return "pnpm" }
func (Pnpm) Description() string { return "JS/TS package manager. Multiversa policy: pnpm only, never npm." }
func (Pnpm) Probe() string       { return "pnpm" }
func (p Pnpm) Installed() bool   { return installed(p.Probe()) }

func (Pnpm) PlanFor(osKind, _ string) (Plan, error) {
	switch osKind {
	case "darwin", "linux":
		return Plan{
			Shell: "curl -fsSL https://get.pnpm.io/install.sh | sh -",
			Notes: "Standalone installer; ships its own Node bundle. Drops binary in ~/.local/share/pnpm.",
		}, nil
	case "windows":
		return Plan{
			Program: "powershell",
			Args:    []string{"-Command", "iwr https://get.pnpm.io/install.ps1 -useb | iex"},
			Notes:   "PowerShell variant of the same standalone installer.",
		}, nil
	}
	return Plan{}, ErrUnsupportedOS
}
