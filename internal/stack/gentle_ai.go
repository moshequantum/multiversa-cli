package stack

type GentleAI struct{}

func (GentleAI) ID() string          { return "gentle-ai" }
func (GentleAI) DisplayName() string { return "gentle-ai" }
func (GentleAI) Author() string      { return "Gentleman-Programming" }
func (GentleAI) Repo() string        { return "https://github.com/Gentleman-Programming/gentle-ai" }
func (GentleAI) License() string     { return "MIT" }
func (GentleAI) OptIn() bool         { return false }

func (g GentleAI) Install(version string) error {
	// TODO: `go install github.com/Gentleman-Programming/gentle-ai/cmd/gentle@VERSION`
	return ErrNotImplemented
}

func (g GentleAI) Update() error           { return ErrNotImplemented }
func (g GentleAI) Status() (Status, error) { return Status{}, ErrNotImplemented }
func (g GentleAI) Uninstall() error        { return ErrNotImplemented }
