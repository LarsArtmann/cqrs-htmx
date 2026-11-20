package legalparty

import errorfamily "github.com/larsartmann/go-error-family"

// LegalFormClass classifies the legal personality of a party. The five
// values are peers, not a hierarchy: a GmbH (legal_entity) and a GbR
// (partnership) and a freelance natural person (natural_person) all bill
// differently, and the class — not a boolean — decides which document
// requirements apply (register number, legal-form suffix, VAT ID vs
// Steuernummer).
type LegalFormClass string

const (
	// LegalFormNaturalPerson is a human being billed personally (freelancer,
	// private individual, sole trader under a business name). Sole traders
	// are natural persons, not a fourth class.
	LegalFormNaturalPerson LegalFormClass = "natural_person"
	// LegalFormLegalEntity has a legal personality separate from its owners
	// (GmbH, UG, AG, Ltd, SARL, LLC).
	LegalFormLegalEntity LegalFormClass = "legal_entity"
	// LegalFormPartnership is legally capable without being a juristic
	// person; partners are liable (GbR, OHG, KG, PartG).
	LegalFormPartnership LegalFormClass = "partnership"
	// LegalFormPublicBody is a public-law entity (Körperschaft des
	// öffentlichen Rechts). Public-law bodies may sit outside the commercial
	// registers, so a registry anchor is optional for them.
	LegalFormPublicBody LegalFormClass = "public_body"
	// LegalFormNonProfit is an association or non-profit (e.V., gGmbH).
	LegalFormNonProfit LegalFormClass = "non_profit"
)

// ParseLegalFormClass converts a raw string into a LegalFormClass.
func ParseLegalFormClass(s string) (LegalFormClass, error) {
	c := LegalFormClass(s)
	if !c.IsValid() {
		return "", errorfamily.NewRejection("legalparty.legal_form.invalid",
			"unknown legal form class "+s+" (want natural_person, legal_entity, partnership, public_body, non_profit)")
	}

	return c, nil
}

// IsValid reports whether the class is one of the five defined values.
func (c LegalFormClass) IsValid() bool {
	switch c {
	case LegalFormNaturalPerson, LegalFormLegalEntity, LegalFormPartnership,
		LegalFormPublicBody, LegalFormNonProfit:
		return true
	default:
		return false
	}
}

// RequiresRegistryAnchor reports whether an opened profile of this class
// must anchor to an external registry (a LEI or a commercial register
// reference). Legal entities and partnerships owe the register their
// existence; the anchor is the evidence.
func (c LegalFormClass) RequiresRegistryAnchor() bool {
	switch c {
	case LegalFormLegalEntity, LegalFormPartnership:
		return true
	case LegalFormNaturalPerson, LegalFormPublicBody, LegalFormNonProfit:
		return false
	default:
		return false
	}
}

// AllowsLEI reports whether a party of this class can hold a Legal Entity
// Identifier. GLEIF's scope covers legal entities only: natural persons
// (including sole traders) cannot hold a LEI.
func (c LegalFormClass) AllowsLEI() bool {
	return c != LegalFormNaturalPerson
}
