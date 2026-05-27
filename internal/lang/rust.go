package lang

// Rust installs the official rustup toolchain. rustup is the canonical
// way to manage Rust across every OS, including Windows (with the
// rustup-init.exe shim). User-mode install into ~/.cargo and ~/.rustup
// keeps it portable and avoids sudo.
type Rust struct{}

func (Rust) ID() string          { return "rust" }
func (Rust) DisplayName() string { return "Rust" }
func (Rust) Description() string { return "Systems language. rustup-managed, user-mode install." }
func (Rust) Probe() string       { return "rustc" }
func (r Rust) Installed() bool   { return installed(r.Probe()) }

func (Rust) PlanFor(osKind, _ string) (Plan, error) {
	switch osKind {
	case "darwin", "linux":
		return Plan{
			Shell: "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --no-modify-path",
			Notes: "rustup → ~/.cargo, ~/.rustup. Adds nothing to PATH; we'll print the line to source.",
		}, nil
	case "windows":
		return Plan{
			Program: "winget",
			Args:    []string{"install", "--id", "Rustlang.Rustup", "-e", "--source", "winget"},
			Notes:   "winget installs rustup; the user must complete `rustup default stable` in a new shell.",
		}, nil
	}
	return Plan{}, ErrUnsupportedOS
}
