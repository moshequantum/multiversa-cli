package backends

type Supabase struct{}

func (Supabase) ID() string          { return "supabase" }
func (Supabase) DisplayName() string { return "Supabase (Postgres + RLS + Storage)" }
func (Supabase) Init() error         { return ErrNotImplemented }
func (Supabase) Auth() error         { return ErrNotImplemented }
func (Supabase) Sync() error         { return ErrNotImplemented }
func (Supabase) Pull() error         { return ErrNotImplemented }
func (Supabase) Push() error         { return ErrNotImplemented }
