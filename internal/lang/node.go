package lang

// Node uses nvm on Unix and the official MSI on Windows. We prefer nvm
// because it lets Moshe pin per-project Node versions without root.
// The install drops nvm into ~/.nvm; activating it requires sourcing
// nvm.sh, which the post-install message reminds the user to do.
type Node struct{}

func (Node) ID() string          { return "node" }
func (Node) DisplayName() string { return "Node.js" }
func (Node) Description() string { return "JS runtime. nvm-managed on Unix, official MSI on Windows." }
func (Node) Probe() string       { return "node" }
func (n Node) Installed() bool   { return installed(n.Probe()) }

func (Node) PlanFor(osKind, _ string) (Plan, error) {
	switch osKind {
	case "darwin", "linux":
		return Plan{
			Shell: `bash -c "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.4/install.sh | bash && export NVM_DIR=\"$HOME/.nvm\" && . \"$NVM_DIR/nvm.sh\" && nvm install --lts"`,
			Notes: "nvm → ~/.nvm. Installs latest LTS. Source ~/.nvm/nvm.sh in your shell init to use it.",
		}, nil
	case "windows":
		return Plan{
			Program: "winget",
			Args:    []string{"install", "--id", "OpenJS.NodeJS.LTS", "-e", "--source", "winget"},
			Notes:   "Installs Node.js LTS. For version switching on Windows, install nvm-windows separately.",
		}, nil
	}
	return Plan{}, ErrUnsupportedOS
}
