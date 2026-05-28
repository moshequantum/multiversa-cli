package profile

// Layer is one of the consultive configuration layers in the
// `multiversa lab` meta-wizard. Each layer groups a related set of
// Steps (e.g. detect/stack/init in Técnica) and is shown as a
// section in the sidebar.
//
// Negocio (clients/billing/contracts) is intentionally absent here.
// It belongs to MultiversaGroup, the private commercial entity,
// not to the MIT Lab.
type Layer string

const (
	// Tecnica covers the OS-level toolchain and curated engines.
	Tecnica Layer = "tecnica"
	// Identitaria covers the user's identity, brain, and private
	// workspace (SSH/GPG/repos/vault).
	Identitaria Layer = "identitaria"
	// Operacional covers operational artifacts that aren't strictly
	// dev-tool nor identity: encrypted USB, credits attribution,
	// housekeeping.
	Operacional Layer = "operacional"
)

// AllLayers returns the layers shown in `multiversa lab`, in the
// order they appear in the sidebar.
func AllLayers() []Layer {
	return []Layer{Tecnica, Identitaria, Operacional}
}

// DisplayName returns the human label rendered in the sidebar.
// Display names are always Spanish — the Lab tone is bilingual but
// the layer names themselves are part of the brand vocabulary.
func (l Layer) DisplayName() string {
	switch l {
	case Tecnica:
		return "Técnica"
	case Identitaria:
		return "Identitaria"
	case Operacional:
		return "Operacional"
	}
	return string(l)
}

// Tagline returns a one-line description shown under each layer
// header in the sidebar.
func (l Layer) Tagline() string {
	switch l {
	case Tecnica:
		return "toolchain + engines"
	case Identitaria:
		return "tu identidad y Brain"
	case Operacional:
		return "USB encriptado · atribución"
	}
	return ""
}
