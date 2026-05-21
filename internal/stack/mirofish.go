package stack

import "errors"

// ErrAgplConsentRequired must be returned if a user attempts MiroFish install
// without acknowledging the AGPL-3.0 disclaimer. Multiversa never embeds
// MiroFish source — it is invoked as an external service only.
var ErrAgplConsentRequired = errors.New("AGPL-3.0 disclaimer not acknowledged — MiroFish must be invoked external-only")

type MiroFish struct {
	AgplAcknowledged bool
}

func (MiroFish) ID() string          { return "mirofish" }
func (MiroFish) DisplayName() string { return "MiroFish" }
func (MiroFish) Author() string      { return "666ghj" }
func (MiroFish) Repo() string        { return "https://github.com/666ghj/MiroFish" }
func (MiroFish) License() string     { return "AGPL-3.0" }
func (MiroFish) OptIn() bool         { return true }

func (m MiroFish) Install(version string) error {
	if !m.AgplAcknowledged {
		return ErrAgplConsentRequired
	}
	// TODO: pull official Docker image or run the upstream installer in an
	// isolated environment. NEVER `go install` or vendor source — that would
	// pull AGPL into Multiversa.
	return ErrNotImplemented
}

func (m MiroFish) Update() error           { return ErrNotImplemented }
func (m MiroFish) Status() (Status, error) { return Status{}, ErrNotImplemented }
func (m MiroFish) Uninstall() error        { return ErrNotImplemented }
