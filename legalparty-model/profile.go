package legalparty

import (
	"regexp"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// legalFormCodePattern is the ISO 20275 ELF code shape: exactly four
// alphanumeric characters. The reserved codes 8888 (form exists but is not
// yet in the list) and 9999 (no separate legal form) travel in
// [LegalPartyProfile.OtherLegalForm] instead.
var legalFormCodePattern = regexp.MustCompile(`^[A-Z0-9]{4}$`)

// LegalPartyProfile is the descriptive surface of a legal party: who it is
// in the eyes of a registry and a tax authority. It is a value object —
// the profile_updated event replaces it wholesale, and every field is
// validated together by [LegalPartyProfile.Validate].
//
// Actor attribution deliberately lives on the commands (and from there in
// event metadata via the fleet's actor-enrichment chain), not in the
// profile: a party's identity facts and the actors maintaining them are
// different axes.
type LegalPartyProfile struct {
	// LegalName is the registered name, including the legal-form suffix a
	// jurisdiction requires ("Muster GmbH", "ACME e.K.").
	LegalName string `json:"legal_name"`
	// LegalForm is the personality class (see [LegalFormClass]).
	LegalForm LegalFormClass `json:"legal_form"`
	// LegalFormCode is the ISO 20275 ELF code when the registry speaks one
	// (GLEIF carries it); empty for natural persons.
	LegalFormCode string `json:"legal_form_code,omitempty"`
	// OtherLegalForm carries the registry-native legal-form label when no
	// ELF code applies ("Gesellschaft bürgerlichen Rechts", "Private
	// company limited by shares").
	OtherLegalForm string `json:"other_legal_form,omitempty"`
	// Country is the jurisdiction of registration.
	Country CountryCode `json:"country"`
	// RegisteredAddr is the official registered address.
	RegisteredAddr RegisteredAddress `json:"registered_address"`
	// BillingEmail is the optional billing contact address.
	BillingEmail identitymodel.Email `json:"billing_email,omitempty"`
	// LEI is the Legal Entity Identifier when one exists. Natural persons
	// cannot hold one.
	LEI LEI `json:"lei,omitempty"`
	// Register anchors the party to its commercial/public register entry.
	// Required for legal entities and partnerships.
	Register *RegisterRef `json:"register,omitempty"`
}

// Validate enforces the profile invariants: identity facts present, class
// consistent with the identifiers carried, and registry anchors required
// where the class owes the register its existence.
func (p LegalPartyProfile) Validate() error {
	if err := p.validateIdentity(); err != nil {
		return err
	}
	if err := p.validateIdentifiers(); err != nil {
		return err
	}

	return p.validateClassConsistency()
}

// validateIdentity checks the required identity facts.
func (p LegalPartyProfile) validateIdentity() error {
	if strings.TrimSpace(p.LegalName) == "" {
		return errorfamily.NewRejection("legalparty.profile.name_required",
			"a legal party carries its registered name")
	}
	if !p.LegalForm.IsValid() {
		return errorfamily.NewRejection("legalparty.profile.legal_form_invalid",
			"unknown legal form class "+string(p.LegalForm))
	}
	if !p.Country.IsValid() {
		return errorfamily.NewRejection("legalparty.profile.country_invalid",
			"a legal party names its jurisdiction (ISO 3166-1 alpha-2)")
	}

	return p.RegisteredAddr.Validate()
}

// validateIdentifiers format-checks every identifier the profile carries.
func (p LegalPartyProfile) validateIdentifiers() error {
	if p.BillingEmail != "" {
		if _, err := identitymodel.ParseEmail(string(p.BillingEmail)); err != nil {
			return errorfamily.NewRejection("legalparty.profile.billing_email_invalid",
				"the billing contact email does not validate")
		}
	}
	if p.LEI != "" {
		if _, err := ParseLEI(string(p.LEI)); err != nil {
			return err
		}
	}
	if p.LegalFormCode != "" {
		if !legalFormCodePattern.MatchString(p.LegalFormCode) {
			return errorfamily.NewRejection("legalparty.profile.legal_form_code_invalid",
				"an ISO 20275 ELF code is four alphanumeric characters, got "+p.LegalFormCode)
		}
	}
	if p.Register != nil {
		if err := p.Register.Validate(); err != nil {
			return errorfamily.WrapRejection(err, "legalparty.profile.register_invalid",
				"the register reference on a legal party profile does not validate")
		}
	}

	return nil
}

// validateClassConsistency enforces the class-dependent identifier rules:
// natural persons hold neither LEI nor ELF code, and legal entities and
// partnerships carry a registry anchor.
func (p LegalPartyProfile) validateClassConsistency() error {
	if p.LegalForm == LegalFormNaturalPerson && p.LegalFormCode != "" {
		return errorfamily.NewRejection("legalparty.profile.natural_person_legal_form_code",
			"a natural person carries no ISO 20275 ELF code; use OtherLegalForm for the trade name")
	}
	if p.LEI != "" && !p.LegalForm.AllowsLEI() {
		return errorfamily.NewRejection("legalparty.profile.natural_person_lei",
			"a natural person cannot hold a LEI (GLEIF's scope covers legal entities only)")
	}
	if p.LegalForm.RequiresRegistryAnchor() && p.LEI == "" && p.Register == nil {
		return errorfamily.NewRejection("legalparty.profile.registry_anchor_required",
			"a "+string(p.LegalForm)+" carries a LEI or a register reference as evidence of its registration")
	}

	return nil
}
