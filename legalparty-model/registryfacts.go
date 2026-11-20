package legalparty

import (
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// RegistrySource names the external registry confirmed facts came from.
type RegistrySource string

const (
	// RegistrySourceGLEIF is the Global LEI Foundation's LEI registry
	// (api.gleif.org). Data is published under CC BY 4.0; consumers must
	// attribute the source when displaying it.
	RegistrySourceGLEIF RegistrySource = "gleif"
	// RegistrySourceOpenCorporates is the OpenCorporates company registry
	// aggregator (api.opencorporates.com). API terms require attribution;
	// see the opencorporates package docs.
	RegistrySourceOpenCorporates RegistrySource = "opencorporates"
)

// IsValid reports whether the source is one of the known registries.
func (s RegistrySource) IsValid() bool {
	switch s {
	case RegistrySourceGLEIF, RegistrySourceOpenCorporates:
		return true
	default:
		return false
	}
}

// RegistryFacts is a provenance-stamped snapshot of what an external
// registry says about a party. Facts never mutate the aggregate on their
// own: an adapter fetches them, a human (or policy) decides, and they are
// applied through UpdateLegalPartyProfile — the event then carries the
// facts next to the resulting profile, so the journal answers "what did we
// know, from where, when".
type RegistryFacts struct {
	// Source names the registry the facts came from.
	Source RegistrySource `json:"source"`
	// Register echoes the register reference that was resolved, when the
	// lookup was register-keyed.
	Register *RegisterRef `json:"register,omitempty"`
	// LEI echoes the resolved LEI, when the registry mints or confirms one.
	LEI LEI `json:"lei,omitempty"`
	// LegalName is the registry's current name for the entity.
	LegalName string `json:"legal_name"`
	// LegalFormCode is the ISO 20275 ELF code when the registry speaks one
	// (GLEIF).
	LegalFormCode string `json:"legal_form_code,omitempty"`
	// LegalFormLabel is the registry-native legal-form label when the
	// registry has no ELF code (OpenCorporates company_type).
	LegalFormLabel string `json:"legal_form_label,omitempty"`
	// Status is the registry's entity status (ACTIVE/INACTIVE/...).
	Status string `json:"status,omitempty"`
	// Country is the registry's jurisdiction, when resolvable.
	Country CountryCode `json:"country,omitempty"`
	// RegisteredAddr is the registered address on file, when the registry
	// exposes one.
	RegisteredAddr *RegisteredAddress `json:"registered_address,omitempty"`
	// FetchedAt is when the adapter fetched the facts (UTC).
	FetchedAt time.Time `json:"fetched_at"`
}

// Validate enforces that the facts are attributable: a known source, a
// named entity, and a fetch timestamp.
func (f RegistryFacts) Validate() error {
	if !f.Source.IsValid() {
		return errorfamily.NewRejection("legalparty.registry_facts.source_invalid",
			"unknown registry source "+string(f.Source))
	}
	if f.LegalName == "" {
		return errorfamily.NewRejection("legalparty.registry_facts.name_required",
			"registry facts carry the registry's legal name")
	}
	if f.FetchedAt.IsZero() {
		return errorfamily.NewRejection("legalparty.registry_facts.fetched_at_required",
			"registry facts carry the fetch timestamp")
	}

	return nil
}

// LEIResolver resolves registry facts for a LEI holder. GLEIF is the
// canonical implementation.
type LEIResolver interface {
	// ResolveByLEI fetches the registry's current record for the entity
	// identified by the LEI. Returns [ErrRegistryNotFound] when no entity
	// matches and [ErrRegistryUnavailable] when the registry cannot be
	// reached.
	ResolveByLEI(ctx context.Context, lei LEI) (RegistryFacts, error)
}

// RegisterResolver resolves registry facts by commercial register
// reference. OpenCorporates is the canonical implementation.
type RegisterResolver interface {
	// ResolveByRegister fetches the registry's current record for the
	// register entry. Returns [ErrRegistryNotFound] when no entry matches
	// and [ErrRegistryUnavailable] when the registry cannot be reached.
	ResolveByRegister(ctx context.Context, ref RegisterRef) (RegistryFacts, error)
}
