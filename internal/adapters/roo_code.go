package adapters

type RooCode struct{}

func (RooCode) ID() string                          { return "roo-code" }
func (RooCode) DisplayName() string                 { return "Roo Code" }
func (RooCode) Detect() bool                        { return false } // VSCode extension — detect via marketplace lookup TBD
func (RooCode) Connect(opts ConnectOptions) error   { return ErrNotImplemented }
func (RooCode) Disconnect() error                   { return ErrNotImplemented }
