package legalparty

import "testing"

// LEI vectors: the first two are live GLEIF records fetched 2026-11-20
// (api.gleif.org/api/v1/lei-records), the checksum-invalid one was verified
// to 404 on the registry.
const (
	leiLiveDE = "984500663D3E060B3G67"
	leiLiveNL = "7245007CS2LXYD2JCT52"
)

func TestParseLEI(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		want     string
		wantCode string
	}{
		{"live German record", leiLiveDE, leiLiveDE, ""},
		{"live Dutch record", leiLiveNL, leiLiveNL, ""},
		{"lowercase normalizes", "7245007cs2lxyd2jct52", leiLiveNL, ""},
		{"checksum tampered", "7245007CS2LXYD2JCT53", "", "legalparty.lei.checksum_failed"},
		{"too short", "5493001KJTIIGC8Y1V1", "", "legalparty.lei.invalid"},
		{"bad character", "5493001KJTIIGC8Y1V1!", "", "legalparty.lei.invalid"},
		{"empty", "", "", "legalparty.lei.invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLEI(tt.in)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("ParseLEI(%q) must accept: %v", tt.in, err)
				}
				if string(got) != tt.want {
					t.Fatalf("ParseLEI(%q) = %q, want %q", tt.in, got, tt.want)
				}

				return
			}
			if err == nil {
				t.Fatalf("ParseLEI(%q) must reject, got %q", tt.in, got)
			}
			if got := errorCode(t, err); got != tt.wantCode {
				t.Fatalf("error code: got %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestParseVATID(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		want     string
		wantCode string
	}{
		{"BMF example passes checksum", "DE136695976", "DE136695976", ""},
		{"tampered checksum rejected", "DE136695978", "", "legalparty.vat_id.checksum_failed"},
		{"separators normalize", "de 136 695 976", "DE136695976", ""},
		{"austrian UID format", "ATU13585627", "ATU13585627", ""},
		{"polish NIP", "PL1234567890", "PL1234567890", ""},
		{"dutch BTW", "NL123456789B01", "NL123456789B01", ""},
		{"generic EU form", "FR12345678901", "FR12345678901", ""},
		{"no country prefix", "136695976", "", "legalparty.vat_id.prefix_invalid"},
		{"unassigned prefix", "ZZ123456789", "", "legalparty.vat_id.prefix_invalid"},
		{"german wrong length", "DE1366959", "", "legalparty.vat_id.national_invalid"},
		{"too short", "DE12", "", "legalparty.vat_id.national_invalid"},
		{"empty", "  ", "", "legalparty.vat_id.required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVATID(tt.in)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("ParseVATID(%q) must accept: %v", tt.in, err)
				}
				if string(got) != tt.want {
					t.Fatalf("ParseVATID(%q) = %q, want %q", tt.in, got, tt.want)
				}

				return
			}
			if err == nil {
				t.Fatalf("ParseVATID(%q) must reject, got %q", tt.in, got)
			}
			if got := errorCode(t, err); got != tt.wantCode {
				t.Fatalf("error code: got %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestParseTaxNumber(t *testing.T) {
	tests := []struct {
		name     string
		country  CountryCode
		in       string
		want     string
		wantCode string
	}{
		{"ELSTER 13-digit with separators", "DE", "28 93 0 815 0815 2", "2893081508152", ""},
		{"legacy 11-digit with slashes", "DE", "181/815/08155", "18181508155", ""},
		{"prefixed VAT shape rejected", "DE", "DE136695976", "", "legalparty.tax_number.vat_shape"},
		{"german too short", "DE", "12345", "", "legalparty.tax_number.national_invalid"},
		{"italian codice fiscale", "IT", "RSSMRA85M01H501Z", "RSSMRA85M01H501Z", ""},
		{"empty", "DE", " ", "", "legalparty.tax_number.required"},
		{"other country too short", "GB", "123", "", "legalparty.tax_number.national_invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaxNumber(tt.country, tt.in)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("ParseTaxNumber(%v, %q) must accept: %v", tt.country, tt.in, err)
				}
				if string(got) != tt.want {
					t.Fatalf("normalized %q, want %q", got, tt.want)
				}

				return
			}
			if err == nil {
				t.Fatalf("ParseTaxNumber(%v, %q) must reject, got %q", tt.country, tt.in, got)
			}
			if got := errorCode(t, err); got != tt.wantCode {
				t.Fatalf("error code: got %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestNewRegisterRef(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ref, err := NewRegisterRef("de", "B1204_HRB707415")
		if err != nil {
			t.Fatalf("must accept: %v", err)
		}
		if ref.Registry != "de" || ref.Number != "B1204_HRB707415" {
			t.Fatalf("unexpected ref: %+v", ref)
		}
	})

	t.Run("missing registry", func(t *testing.T) {
		if _, err := NewRegisterRef("", "HRB1"); err == nil {
			t.Fatal("must reject empty registry")
		}
	})

	t.Run("missing number", func(t *testing.T) {
		if _, err := NewRegisterRef("de", "  "); err == nil {
			t.Fatal("must reject empty number")
		}
	})
}

func TestParseCountryCode(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  CountryCode
		valid bool
	}{
		{"germany lowercase input", "de", "DE", true},
		{"netherlands", "NL", "NL", true},
		{"unassigned ZZ", "ZZ", "", false},
		{"user-assigned XK", "XK", "", false},
		{"three letters", "DEU", "", false},
		{"empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCountryCode(tt.in)
			if tt.valid {
				if err != nil {
					t.Fatalf("ParseCountryCode(%q) must accept: %v", tt.in, err)
				}
				if got != tt.want {
					t.Fatalf("got %q, want %q", got, tt.want)
				}

				return
			}
			if err == nil {
				t.Fatalf("ParseCountryCode(%q) must reject, got %q", tt.in, got)
			}
		})
	}
}

func TestLegalFormClass_Predicate(t *testing.T) {
	tests := []struct {
		class    LegalFormClass
		anchor   bool
		allowsLE bool
	}{
		{LegalFormNaturalPerson, false, false},
		{LegalFormLegalEntity, true, true},
		{LegalFormPartnership, true, true},
		{LegalFormPublicBody, false, true},
		{LegalFormNonProfit, false, true},
		// Values that never passed ParseLegalFormClass must allow nothing
		// (go-legalparty v0.1.1 aligned AllowsLEI with
		// RequiresRegistryAnchor on unknown input).
		{LegalFormClass(""), false, false},
		{LegalFormClass("corp"), false, false},
	}

	for _, tt := range tests {
		if tt.class.RequiresRegistryAnchor() != tt.anchor {
			t.Fatalf("%s RequiresRegistryAnchor = %v, want %v",
				tt.class, tt.class.RequiresRegistryAnchor(), tt.anchor)
		}
		if tt.class.AllowsLEI() != tt.allowsLE {
			t.Fatalf("%s AllowsLEI = %v, want %v", tt.class, tt.class.AllowsLEI(), tt.allowsLE)
		}
	}

	if _, err := ParseLegalFormClass("sole_proprietor"); err == nil {
		t.Fatal("unknown class must reject")
	}
	if _, err := ParseLegalFormClass("natural_person"); err != nil {
		t.Fatalf("known class must parse: %v", err)
	}
}
