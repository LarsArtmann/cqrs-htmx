package legalparty

import (
	"strconv"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// LEI is a Legal Entity Identifier (ISO 17442): a 20-character,
// globally unique identifier for legal entities that participate in
// financial transactions, issued under GLEIF governance. The final two
// characters are the mod-97 check digits.
//
// Natural persons cannot hold a LEI (GLEIF's scope covers legal entities
// only); [LegalFormClass.AllowsLEI] enforces that at the profile level.
type LEI string

// ParseLEI validates and constructs a LEI. Input is uppercased and
// trimmed, must be exactly 20 characters of [A-Z0-9], and must pass the
// ISO 17442 mod-97 checksum.
func ParseLEI(s string) (LEI, error) {
	v := strings.ToUpper(strings.TrimSpace(s))
	if len(v) != 20 {
		return "", errorfamily.NewRejection("legalparty.lei.invalid",
			"a LEI is exactly 20 characters, got "+strconv.Itoa(len(v))+" in "+s)
	}
	for _, r := range v {
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') {
			return "", errorfamily.NewRejection("legalparty.lei.invalid",
				"a LEI carries only digits and uppercase letters, got "+s)
		}
	}
	if !validLEIChecksum(v) {
		return "", errorfamily.NewRejection("legalparty.lei.checksum_failed",
			"LEI "+v+" fails the ISO 17442 mod-97 checksum")
	}

	return LEI(v), nil
}

// validLEIChecksum applies the ISO 17442 check: map every character to its
// numeric form (digits as-is, A=10 .. Z=35), interpret the whole 20-character
// string as one decimal number, and require number mod 97 == 1.
func validLEIChecksum(code string) bool {
	remainder := 0
	for _, r := range code {
		switch {
		case r >= '0' && r <= '9':
			remainder = (remainder*10 + int(r-'0')) % 97
		case r >= 'A' && r <= 'Z':
			v := int(r-'A') + 10
			remainder = (remainder*10 + v/10) % 97
			remainder = (remainder*10 + v%10) % 97
		default:
			return false
		}
	}

	return remainder == 1
}

// RegisterRef points at an entry in a commercial or public register
// (Handelsregister HRB, Companies House number, RCS number, ...). Registry
// is an opaque registry/jurisdiction code owned by whoever resolves the
// reference (an OpenCorporates jurisdiction code such as "de_munich", a
// national register key) — the domain never inspects its semantics.
type RegisterRef struct {
	Registry string `json:"registry"`
	Number   string `json:"number"`
}

// NewRegisterRef validates and constructs a RegisterRef.
func NewRegisterRef(registry, number string) (RegisterRef, error) {
	r := RegisterRef{Registry: strings.TrimSpace(registry), Number: strings.TrimSpace(number)}
	if err := r.Validate(); err != nil {
		return RegisterRef{}, err
	}

	return r, nil
}

// Validate enforces that both the registry code and the number are present.
func (r RegisterRef) Validate() error {
	if strings.TrimSpace(r.Registry) == "" {
		return errorfamily.NewRejection("legalparty.register.registry_required",
			"a register reference names its registry (e.g. the jurisdiction code)")
	}
	if strings.TrimSpace(r.Number) == "" {
		return errorfamily.NewRejection("legalparty.register.number_required",
			"a register reference carries the registry entry number")
	}

	return nil
}

// normalizeTaxIdentifier uppercases and strips the separator characters
// national tax forms carry (spaces, slashes, dashes, dots).
func normalizeTaxIdentifier(s string) string {
	v := strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if r == ' ' || r == '/' || r == '-' || r == '.' {
			continue
		}
		b.WriteRune(r)
	}

	return b.String()
}

// VATID is a value-added tax identifier in its canonical EU-prefixed form
// ("DE136695976", "PL1234567890"). It is deliberately a different type from
// [TaxNumber]: a VAT ID starts with a country prefix, a national tax number
// never does, and conflating the two rejects valid values on both sides.
//
// Germany validates the full ISO 7064 Mod 11,10 checksum; the other known
// forms are format-validated. Unprefixed domestic forms (e.g. a bare NIP)
// must be normalized to the prefixed form by the caller.
type VATID string

// ParseVATID validates and constructs a VATID in canonical prefixed form.
func ParseVATID(s string) (VATID, error) {
	v := normalizeTaxIdentifier(s)
	if len(v) < 3 {
		return "", errorfamily.NewRejection("legalparty.vat_id.required",
			"a VAT ID carries a country prefix and a national part, got "+s)
	}

	cc, err := ParseCountryCode(v[:2])
	if err != nil {
		return "", errorfamily.NewRejection("legalparty.vat_id.prefix_invalid",
			"a VAT ID starts with a country prefix: "+err.Error())
	}

	national := v[2:]
	if cc == "AT" && len(national) > 0 && national[0] == 'U' {
		national = national[1:]
	}

	if err := validateVATNationalPart(cc, national); err != nil {
		return "", err
	}

	return VATID(v), nil
}

func validateVATNationalPart(cc CountryCode, national string) error {
	if err := validateVATCharacters(cc, national); err != nil {
		return err
	}

	return validateVATLength(cc, national)
}

// validateVATCharacters checks the national part's alphabet: digits only,
// except the Dutch form which carries the literal B separator.
func validateVATCharacters(cc CountryCode, national string) error {
	for _, r := range national {
		switch {
		case r >= '0' && r <= '9':
			continue
		case r >= 'A' && r <= 'Z':
			if cc != "NL" {
				return errorfamily.NewRejection("legalparty.vat_id.national_invalid",
					"the national part of a "+string(cc)+" VAT ID carries only digits")
			}
		default:
			return errorfamily.NewRejection("legalparty.vat_id.national_invalid",
				"the national part of a VAT ID carries only uppercase letters and digits")
		}
	}

	return nil
}

// validateVATLength enforces the per-jurisdiction national-part shape.
func validateVATLength(cc CountryCode, national string) error {
	switch cc {
	case "DE":
		return validateDEVAT(national)
	case "NL":
		return validateNLVAT(national)
	case "AT", "PL":
		return validateFixedLengthVAT(cc, national)
	default:
		return validateGenericVAT(national)
	}
}

func validateDEVAT(national string) error {
	if len(national) != 9 {
		return errorfamily.NewRejection("legalparty.vat_id.national_invalid",
			"a German USt-IdNr. is DE followed by exactly 9 digits")
	}
	if !validDEVATChecksum(national) {
		return errorfamily.NewRejection("legalparty.vat_id.checksum_failed",
			"USt-IdNr. "+national+" fails the ISO 7064 Mod 11,10 checksum")
	}

	return nil
}

func validateNLVAT(national string) error {
	if len(national) != 12 || national[9] != 'B' {
		return errorfamily.NewRejection("legalparty.vat_id.national_invalid",
			"a Dutch BTW-nummer is NL followed by 9 digits, B, and 2 digits")
	}

	return nil
}

// validateFixedLengthVAT checks the single-length national forms
// (AT: 8 digits, PL: 10 digits).
func validateFixedLengthVAT(cc CountryCode, national string) error {
	lengths := map[CountryCode]int{"AT": 8, "PL": 10}
	if len(national) != lengths[cc] {
		return errorfamily.NewRejection("legalparty.vat_id.national_invalid",
			"the national part of a "+string(cc)+" VAT ID is "+strconv.Itoa(lengths[cc])+" digits")
	}

	return nil
}

// validateGenericVAT checks jurisdictions without a dedicated rule.
func validateGenericVAT(national string) error {
	if len(national) < 8 || len(national) > 12 {
		return errorfamily.NewRejection("legalparty.vat_id.national_invalid",
			"the national part of this VAT ID is 8 to 12 characters")
	}

	return nil
}

// validDEVATChecksum applies the ISO 7064 Mod 11,10 checksum to the full
// nine-digit German national part. Faithful port of the published reference
// implementation (python-stdnum, stdnum/iso7064/mod_11_10.py): the checksum
// runs over ALL digits with a seed of 5 and is valid iff it equals 1.
// Verified vectors: BMF/BZSt example DE136695976 validates, DE136695978
// fails, calc('79462') == '3', calc('00200667308') == '5'.
func validDEVATChecksum(national string) bool {
	check := 5
	for i := range len(national) {
		d := int(national[i] - '0')
		base := check
		if base == 0 {
			base = 10
		}
		check = ((base*2)%11 + d) % 10
	}

	return check == 1
}

// TaxNumber is a national direct-tax number (the German Steuernummer, UK
// UTR, ...). Deliberately a different type from [VATID]: tax numbers carry
// no country prefix, so a prefixed VAT-shaped value is rejected here rather
// than silently accepted. Germany validates the 10-13 digit national forms;
// other jurisdictions are format-checked.
type TaxNumber string

// ParseTaxNumber validates and constructs a TaxNumber for the given
// jurisdiction.
func ParseTaxNumber(country CountryCode, s string) (TaxNumber, error) {
	v := normalizeTaxIdentifier(s)
	if v == "" {
		return "", errorfamily.NewRejection("legalparty.tax_number.required",
			"a tax number is required")
	}

	if isPrefixedVATShape(v) {
		return "", errorfamily.NewRejection("legalparty.tax_number.vat_shape",
			"a tax number carries no country prefix; parse "+v+" as a VAT ID instead")
	}

	if err := validateTaxNumberBody(country, v); err != nil {
		return "", err
	}

	return TaxNumber(v), nil
}

// validateTaxNumberBody enforces the per-jurisdiction shape after separator
// removal: Germany takes the 10-13 digit national forms, everyone else is
// format-checked.
func validateTaxNumberBody(country CountryCode, v string) error {
	if country == "DE" {
		if len(v) < 10 || len(v) > 13 {
			return errorfamily.NewRejection("legalparty.tax_number.national_invalid",
				"a German Steuernummer has 10 to 13 digits (legacy or ELSTER form)")
		}
	} else if len(v) < 5 || len(v) > 16 {
		return errorfamily.NewRejection("legalparty.tax_number.national_invalid",
			"a "+string(country)+" tax number is 5 to 16 characters after separator removal")
	}
	for _, r := range v {
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') {
			return errorfamily.NewRejection("legalparty.tax_number.national_invalid",
				"a tax number carries only letters and digits after separator removal")
		}
	}

	return nil
}

// isPrefixedVATShape reports whether the value is a two-letter country
// prefix followed by digits only — the canonical VAT ID shape.
func isPrefixedVATShape(v string) bool {
	if len(v) < 3 || v[0] < 'A' || v[0] > 'Z' || v[1] < 'A' || v[1] > 'Z' {
		return false
	}
	for _, r := range v[2:] {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
