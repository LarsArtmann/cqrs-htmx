package legalparty

import "testing"

// The canonical identifier vectors are owned and exhaustively pinned
// upstream (github.com/larsartmann/go-legalparty). This file only pins the
// aliases.go bridge: every aliased parser must resolve to the real
// canonical validator — one accept per alias, plus one reject proving the
// wiring is not a stub that accepts everything.
func TestAliasWiring(t *testing.T) {
	t.Parallel()

	for name, parse := range map[string]func(string) error{
		"ParseLEI":            wrap2(ParseLEI),
		"ParseVATID":          wrap2(ParseVATID),
		"ParseCountryCode":    wrap2(ParseCountryCode),
		"ParseLegalFormClass": wrap2(ParseLegalFormClass),
	} {
		t.Run(name+" accepts a canonical value", func(t *testing.T) {
			t.Parallel()

			if err := parse(canonicalAliasVector[name]); err != nil {
				t.Fatalf("%s alias must accept %q: %v", name, canonicalAliasVector[name], err)
			}
		})
	}

	if _, err := ParseLEI("short"); err == nil {
		t.Fatal("ParseLEI alias must reject invalid input (wiring must not be a stub)")
	}

	if _, err := ParseTaxNumber("DE", "217/8150/8173"); err != nil {
		t.Fatalf("ParseTaxNumber alias must accept the DE Steuernummer: %v", err)
	}

	if _, err := ParseTaxNumber("DE", "12345"); err == nil {
		t.Fatal("ParseTaxNumber alias must reject an invalid DE body")
	}

	if _, err := NewRegisterRef("HRB", "12345"); err != nil {
		t.Fatalf("NewRegisterRef alias must construct: %v", err)
	}
}

func wrap2[V any](f func(string) (V, error)) func(string) error {
	return func(s string) error {
		_, err := f(s)

		return err
	}
}

var canonicalAliasVector = map[string]string{
	"ParseLEI":            "984500663D3E060B3G67",
	"ParseVATID":          "DE136695976",
	"ParseCountryCode":    "DE",
	"ParseLegalFormClass": "legal_entity",
}

// leiLiveDE and leiLiveNL are live GLEIF records (fetched 2026-11-20 from
// api.gleif.org; checksum-invalid variants 404 on the registry) — the
// fixture LEIs for the aggregate tests in this module.
const (
	leiLiveDE = "984500663D3E060B3G67"
	leiLiveNL = "7245007CS2LXYD2JCT52"
)
