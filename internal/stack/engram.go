package stack

type Engram struct{}

func (Engram) ID() string          { return "engram" }
func (Engram) DisplayName() string { return "Engram" }
func (Engram) Author() string      { return "Gentleman-Programming" }
func (Engram) Repo() string        { return "https://github.com/Gentleman-Programming/engram" }
func (Engram) License() string     { return "MIT" }
func (Engram) OptIn() bool         { return false }

func (e Engram) Install(version string) error {
	// TODO: `go install github.com/Gentleman-Programming/engram/cmd/engram@VERSION`
	// or fetch a release binary from GitHub and place under ~/.multiversa/bin.
	return ErrNotImplemented
}

func (e Engram) Update() error            { return ErrNotImplemented }
func (e Engram) Status() (Status, error)  { return Status{}, ErrNotImplemented }
func (e Engram) Uninstall() error         { return ErrNotImplemented }
