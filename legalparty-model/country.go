package legalparty

import (
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// assignedCountryCodes is the ISO 3166-1 alpha-2 set of officially assigned
// country codes. User-assigned, transitional-reservation and
// exceptionally-reserved codes (XK for Kosovo, EU, ZZ, ...) are deliberately
// absent: a billing jurisdiction must resolve to an assigned country.
var assignedCountryCodes = []string{
	"AD", "AE", "AF", "AG", "AI", "AL", "AM", "AO", "AQ", "AR", "AS", "AT", "AU", "AW", "AX", "AZ",
	"BA", "BB", "BD", "BE", "BF", "BG", "BH", "BI", "BJ", "BL", "BM", "BN", "BO", "BQ", "BR", "BS", "BT", "BV", "BW", "BY", "BZ",
	"CA", "CC", "CD", "CF", "CG", "CH", "CI", "CK", "CL", "CM", "CN", "CO", "CR", "CU", "CV", "CW", "CX", "CY", "CZ",
	"DE", "DJ", "DK", "DM", "DO", "DZ",
	"EC", "EE", "EG", "EH", "ER", "ES", "ET",
	"FI", "FJ", "FK", "FM", "FO", "FR",
	"GA", "GB", "GD", "GE", "GF", "GG", "GH", "GI", "GL", "GM", "GN", "GP", "GQ", "GR", "GS", "GT", "GU", "GW", "GY",
	"HK", "HM", "HN", "HR", "HT", "HU",
	"ID", "IE", "IL", "IM", "IN", "IO", "IQ", "IR", "IS", "IT",
	"JE", "JM", "JO", "JP",
	"KE", "KG", "KH", "KI", "KM", "KN", "KP", "KR", "KW", "KY", "KZ",
	"LA", "LB", "LC", "LI", "LK", "LR", "LS", "LT", "LU", "LV", "LY",
	"MA", "MC", "MD", "ME", "MF", "MG", "MH", "MK", "ML", "MM", "MN", "MO", "MP", "MQ", "MR", "MS", "MT", "MU", "MV", "MW", "MX", "MY", "MZ",
	"NA", "NC", "NE", "NF", "NG", "NI", "NL", "NO", "NP", "NR", "NU", "NZ",
	"OM",
	"PA", "PE", "PF", "PG", "PH", "PK", "PL", "PM", "PN", "PR", "PS", "PT", "PW", "PY",
	"QA",
	"RE", "RO", "RS", "RU", "RW",
	"SA", "SB", "SC", "SD", "SE", "SG", "SH", "SI", "SJ", "SK", "SL", "SM", "SN", "SO", "SR", "SS", "ST", "SV", "SX", "SY", "SZ",
	"TC", "TD", "TF", "TG", "TH", "TJ", "TK", "TL", "TM", "TN", "TO", "TR", "TT", "TV", "TW", "TZ",
	"UA", "UG", "UM", "US", "UY", "UZ",
	"VA", "VC", "VE", "VG", "VI", "VN", "VU",
	"WF", "WS",
	"YE", "YT",
	"ZA", "ZM", "ZW",
}

var countryCodeSet = buildCountryCodeSet()

func buildCountryCodeSet() map[string]struct{} {
	set := make(map[string]struct{}, len(assignedCountryCodes))
	for _, c := range assignedCountryCodes {
		set[c] = struct{}{}
	}

	return set
}

// CountryCode is an ISO 3166-1 alpha-2 country code (the jurisdiction whose
// law governs the party's registration and tax identifiers).
type CountryCode string

// ParseCountryCode converts a raw string into a CountryCode. Input is
// uppercased, must be exactly two ASCII letters, and must be officially
// assigned by ISO 3166-1.
func ParseCountryCode(s string) (CountryCode, error) {
	v := strings.ToUpper(strings.TrimSpace(s))
	if len(v) != 2 || v[0] < 'A' || v[0] > 'Z' || v[1] < 'A' || v[1] > 'Z' {
		return "", errorfamily.NewRejection("legalparty.country.invalid",
			"invalid country code "+s+" (want an ISO 3166-1 alpha-2 code)")
	}

	if _, ok := countryCodeSet[v]; !ok {
		return "", errorfamily.NewRejection("legalparty.country.unassigned",
			"country code "+v+" is not an officially assigned ISO 3166-1 alpha-2 code")
	}

	return CountryCode(v), nil
}

// IsValid reports whether the code is a non-zero, officially assigned
// ISO 3166-1 alpha-2 code.
func (c CountryCode) IsValid() bool {
	if c == "" {
		return false
	}

	_, err := ParseCountryCode(string(c))

	return err == nil
}

// String returns the canonical uppercase form.
func (c CountryCode) String() string { return string(c) }
