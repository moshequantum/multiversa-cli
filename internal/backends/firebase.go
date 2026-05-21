package backends

type Firebase struct{}

func (Firebase) ID() string          { return "firebase" }
func (Firebase) DisplayName() string { return "Firebase (Firestore + Storage)" }
func (Firebase) Init() error         { return ErrNotImplemented }
func (Firebase) Auth() error         { return ErrNotImplemented }
func (Firebase) Sync() error         { return ErrNotImplemented }
func (Firebase) Pull() error         { return ErrNotImplemented }
func (Firebase) Push() error         { return ErrNotImplemented }
