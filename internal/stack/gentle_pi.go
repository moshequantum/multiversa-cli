package stack

type GentlePi struct{}

func (GentlePi) ID() string          { return "gentle-pi" }
func (GentlePi) DisplayName() string { return "gentle-pi" }
func (GentlePi) Author() string      { return "Gentleman-Programming" }
func (GentlePi) Repo() string        { return "https://github.com/Gentleman-Programming/gentle-pi" }
func (GentlePi) License() string     { return "MIT" }
func (GentlePi) OptIn() bool         { return false }

func (g GentlePi) Install(version string) error {
	// TODO: `npm i -g gentle-pi` (TypeScript package) — confirm distribution channel from upstream.
	return ErrNotImplemented
}

func (g GentlePi) Update() error           { return ErrNotImplemented }
func (g GentlePi) Status() (Status, error) { return Status{}, ErrNotImplemented }
func (g GentlePi) Uninstall() error        { return ErrNotImplemented }
