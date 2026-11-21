package legalparty

import canonical "github.com/larsartmann/go-legalparty"

type (
	LEI            = canonical.LEI
	RegisterRef    = canonical.RegisterRef
	VATID          = canonical.VATID
	TaxNumber      = canonical.TaxNumber
	CountryCode    = canonical.CountryCode
	LegalFormClass = canonical.LegalFormClass
)

var (
	ParseLEI            = canonical.ParseLEI
	NewRegisterRef      = canonical.NewRegisterRef
	ParseVATID          = canonical.ParseVATID
	ParseTaxNumber      = canonical.ParseTaxNumber
	ParseCountryCode    = canonical.ParseCountryCode
	ParseLegalFormClass = canonical.ParseLegalFormClass
)

const (
	LegalFormNaturalPerson = canonical.LegalFormNaturalPerson
	LegalFormLegalEntity   = canonical.LegalFormLegalEntity
	LegalFormPartnership   = canonical.LegalFormPartnership
	LegalFormPublicBody    = canonical.LegalFormPublicBody
	LegalFormNonProfit     = canonical.LegalFormNonProfit
)
