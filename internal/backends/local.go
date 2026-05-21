package backends

type Local struct{}

func (Local) ID() string          { return "local" }
func (Local) DisplayName() string { return "Local SQLite (no remote sync)" }
func (Local) Init() error         { return nil } // nothing to init — engines bring their own SQLite
func (Local) Auth() error         { return nil }
func (Local) Sync() error         { return nil }
func (Local) Pull() error         { return nil }
func (Local) Push() error         { return nil }
