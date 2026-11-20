package legalparty

import (
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// RegisteredAddress is the official address a legal party is registered
// under. Street, locality and country are the load-bearing fields; postal
// code and region stay optional because not every jurisdiction assigns
// them.
type RegisteredAddress struct {
	StreetAddress string      `json:"street_address"`
	Locality      string      `json:"locality"`
	PostalCode    string      `json:"postal_code,omitempty"`
	Region        string      `json:"region,omitempty"`
	Country       CountryCode `json:"country"`
}

// NewRegisteredAddress validates and constructs a RegisteredAddress.
func NewRegisteredAddress(
	street, locality, postalCode, region string, country CountryCode,
) (RegisteredAddress, error) {
	a := RegisteredAddress{
		StreetAddress: strings.TrimSpace(street),
		Locality:      strings.TrimSpace(locality),
		PostalCode:    strings.TrimSpace(postalCode),
		Region:        strings.TrimSpace(region),
		Country:       country,
	}
	if err := a.Validate(); err != nil {
		return RegisteredAddress{}, err
	}

	return a, nil
}

// Validate enforces the address invariants: street, locality and a valid
// country code are required, and a German postal code must be five digits
// when present.
func (a RegisteredAddress) Validate() error {
	if strings.TrimSpace(a.StreetAddress) == "" {
		return errorfamily.NewRejection("legalparty.address.street_required",
			"registered address street is required")
	}
	if strings.TrimSpace(a.Locality) == "" {
		return errorfamily.NewRejection("legalparty.address.locality_required",
			"registered address locality is required")
	}
	if !a.Country.IsValid() {
		return errorfamily.NewRejection("legalparty.address.country_invalid",
			"registered address country must be an assigned ISO 3166-1 alpha-2 code")
	}
	if a.Country == "DE" && a.PostalCode != "" && !isGermanPostalCode(a.PostalCode) {
		return errorfamily.NewRejection("legalparty.address.postal_code_invalid",
			"a German postal code (PLZ) must be exactly five digits")
	}

	return nil
}

func isGermanPostalCode(s string) bool {
	if len(s) != 5 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
