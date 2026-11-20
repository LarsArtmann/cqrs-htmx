package legalparty

import (
	"strings"

	brandid "github.com/larsartmann/go-branded-id"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/oklog/ulid/v2"
)

type legalPartyBrand struct{}

func (legalPartyBrand) Name() string { return "LegalParty" }

// LegalPartyID is a branded identifier for a legal party aggregate.
type LegalPartyID = brandid.ID[legalPartyBrand, string]

// NewLegalPartyID constructs a LegalPartyID from a string value. For fresh
// system-minted identifiers use [GenerateLegalPartyID].
func NewLegalPartyID(s string) LegalPartyID { return brandid.NewID[legalPartyBrand](s) }

// GenerateLegalPartyID mints a fresh ULID-backed LegalPartyID.
func GenerateLegalPartyID() LegalPartyID { return NewLegalPartyID(ulid.Make().String()) }

// ParseLegalPartyID converts a string to a LegalPartyID. The strict path for
// RECONSTRUCTION (external input, invoice references): it strips the display
// prefix ("LegalParty:", as emitted by String()), rejects empty or padded
// values, and requires the ULID form every minted identifier carries. For
// arbitrary trusted input use [NewLegalPartyID].
func ParseLegalPartyID(s string) (LegalPartyID, error) {
	v, err := parseBrandedComponent(s, "LegalParty")
	if err != nil {
		return LegalPartyID{}, errorfamily.Wrapf(err, event.Rejection,
			"legalparty.legal_party_id.invalid", "invalid legal party id %q", s)
	}

	u, err := ulid.ParseStrict(v)
	if err != nil {
		return LegalPartyID{}, errorfamily.Wrapf(err, event.Rejection,
			"legalparty.legal_party_id.invalid", "invalid legal party id %q", s)
	}

	return NewLegalPartyID(u.String()), nil
}

// parseBrandedComponent strips a single exact "<Brand>:" display prefix and
// rejects empty or whitespace-padded values — the ambiguous forms that would
// otherwise silently become distinct identities.
func parseBrandedComponent(s, brand string) (string, error) {
	v := strings.TrimPrefix(s, brand+":")

	if v == "" || v != strings.TrimSpace(v) {
		return "", errorfamily.NewRejection("legalparty.branded_id.invalid",
			"value is empty or whitespace-padded")
	}

	return v, nil
}
