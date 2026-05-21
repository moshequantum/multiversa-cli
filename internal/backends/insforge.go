package backends

type InsForge struct{}

func (InsForge) ID() string          { return "insforge" }
func (InsForge) DisplayName() string { return "InsForge (Postgres + Auth + Storage + Functions)" }
func (InsForge) Init() error         { return ErrNotImplemented }
func (InsForge) Auth() error         { return ErrNotImplemented }
func (InsForge) Sync() error         { return ErrNotImplemented }
func (InsForge) Pull() error         { return ErrNotImplemented }
func (InsForge) Push() error         { return ErrNotImplemented }
