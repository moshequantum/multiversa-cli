package stack

type CodeGraph struct{}

func (CodeGraph) ID() string          { return "codegraph" }
func (CodeGraph) DisplayName() string { return "codegraph" }
func (CodeGraph) Author() string      { return "Colby McHenry" }
func (CodeGraph) Repo() string        { return "https://github.com/colbymchenry/codegraph" }
func (CodeGraph) License() string     { return "MIT" }
func (CodeGraph) OptIn() bool         { return true }

func (c CodeGraph) Install(version string) error {
	// TODO: `npm i -g codegraph` — confirm package name with upstream.
	return ErrNotImplemented
}

func (c CodeGraph) Update() error           { return ErrNotImplemented }
func (c CodeGraph) Status() (Status, error) { return Status{}, ErrNotImplemented }
func (c CodeGraph) Uninstall() error        { return ErrNotImplemented }
