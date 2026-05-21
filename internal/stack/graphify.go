package stack

type Graphify struct{}

func (Graphify) ID() string          { return "graphify" }
func (Graphify) DisplayName() string { return "Graphify" }
func (Graphify) Author() string      { return "Safi Shamsi" }
func (Graphify) Repo() string        { return "https://github.com/safishamsi/graphify" }
func (Graphify) License() string     { return "MIT" }
func (Graphify) OptIn() bool         { return false }

func (g Graphify) Install(version string) error {
	// TODO: `pipx install graphify` (preferred) or `uv tool install graphify`.
	return ErrNotImplemented
}

func (g Graphify) Update() error           { return ErrNotImplemented }
func (g Graphify) Status() (Status, error) { return Status{}, ErrNotImplemented }
func (g Graphify) Uninstall() error        { return ErrNotImplemented }
