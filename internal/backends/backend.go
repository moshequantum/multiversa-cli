// Package backends provides optional remote sync layers. Default is local
// SQLite only — backends are never required.
package backends

import (
	"errors"
	"fmt"
)

var ErrNotImplemented = errors.New("backend not implemented yet")

type Backend interface {
	ID() string
	DisplayName() string
	Init() error
	Auth() error
	Sync() error
	Pull() error
	Push() error
}

func Registry() map[string]Backend {
	return map[string]Backend{
		"local":    &Local{},
		"supabase": &Supabase{},
		"firebase": &Firebase{},
		"insforge": &InsForge{},
	}
}

func List() []Backend {
	order := []string{"local", "supabase", "firebase", "insforge"}
	reg := Registry()
	out := make([]Backend, 0, len(order))
	for _, id := range order {
		out = append(out, reg[id])
	}
	return out
}

func Resolve(id string) (Backend, error) {
	if b, ok := Registry()[id]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("unknown backend %q", id)
}
