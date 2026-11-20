package legalparty

import (
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

func TestProfileValidate_Entities(t *testing.T) {
	t.Run("anchored GmbH with LEI and register is valid", func(t *testing.T) {
		requireNoError(t, gmbhProfile().Validate())
	})

	t.Run("entity with LEI only is valid", func(t *testing.T) {
		p := gmbhProfile()
		p.Register = nil
		requireNoError(t, p.Validate())
	})

	t.Run("entity with register only is valid", func(t *testing.T) {
		p := gmbhProfile()
		p.LEI = ""
		requireNoError(t, p.Validate())
	})

	t.Run("entity without anchor is rejected", func(t *testing.T) {
		p := gmbhProfile()
		p.LEI = ""
		p.Register = nil
		requireErrorCode(t, p.Validate(), "legalparty.profile.registry_anchor_required")
	})

	t.Run("partnership with register is valid", func(t *testing.T) {
		ref, _ := NewRegisterRef("de", "HRA 4098")
		p := LegalPartyProfile{
			LegalName:      "Muster Bau GbR",
			LegalForm:      LegalFormPartnership,
			OtherLegalForm: "Gesellschaft bürgerlichen Rechts",
			Country:        "DE",
			RegisteredAddr: testAddress,
			Register:       &ref,
		}
		requireNoError(t, p.Validate())
	})

	t.Run("public body needs no anchor", func(t *testing.T) {
		p := LegalPartyProfile{
			LegalName:      "Stadtwerke Musterstadt",
			LegalForm:      LegalFormPublicBody,
			Country:        "DE",
			RegisteredAddr: testAddress,
		}
		requireNoError(t, p.Validate())
	})
}

func TestProfileValidate_NaturalPersons(t *testing.T) {
	t.Run("freelancer without anchor is valid", func(t *testing.T) {
		requireNoError(t, freelancerProfile().Validate())
	})

	t.Run("registered merchant with register is valid", func(t *testing.T) {
		ref, _ := NewRegisterRef("de", "HRA 51234")
		p := freelancerProfile()
		p.Register = &ref
		requireNoError(t, p.Validate())
	})

	t.Run("natural person cannot hold a LEI", func(t *testing.T) {
		p := freelancerProfile()
		p.LEI = LEI(leiLiveNL)
		requireErrorCode(t, p.Validate(), "legalparty.profile.natural_person_lei")
	})

	t.Run("natural person cannot carry an ELF code", func(t *testing.T) {
		p := freelancerProfile()
		p.LegalFormCode = "2HBR"
		requireErrorCode(t, p.Validate(), "legalparty.profile.natural_person_legal_form_code")
	})
}

// TestProfileValidate_SameFactsDifferentOutcomes is issue #45's third
// verification bullet: identical buyer-role data validates differently by
// class — the GmbH needs its registry anchor, the sole trader does not.
func TestProfileValidate_SameFactsDifferentOutcomes(t *testing.T) {
	sameFacts := func(class LegalFormClass) LegalPartyProfile {
		return LegalPartyProfile{
			LegalName:      "Muster GmbH",
			LegalForm:      class,
			Country:        "DE",
			RegisteredAddr: testAddress,
		}
	}

	requireErrorCode(t, sameFacts(LegalFormLegalEntity).Validate(),
		"legalparty.profile.registry_anchor_required")
	requireNoError(t, sameFacts(LegalFormNaturalPerson).Validate())
}

func TestProfileValidate_IdentityFacts(t *testing.T) {
	t.Run("empty name rejected", func(t *testing.T) {
		p := gmbhProfile()
		p.LegalName = "   "
		requireErrorCode(t, p.Validate(), "legalparty.profile.name_required")
	})

	t.Run("unknown class rejected", func(t *testing.T) {
		p := gmbhProfile()
		p.LegalForm = "gmbh"
		requireErrorCode(t, p.Validate(), "legalparty.profile.legal_form_invalid")
	})

	t.Run("bad country rejected", func(t *testing.T) {
		p := gmbhProfile()
		p.Country = "ZZ"
		requireErrorCode(t, p.Validate(), "legalparty.profile.country_invalid")
	})

	t.Run("bad billing email rejected", func(t *testing.T) {
		p := gmbhProfile()
		p.BillingEmail = identitymodel.Email("not-an-email")
		requireErrorCode(t, p.Validate(), "legalparty.profile.billing_email_invalid")
	})

	t.Run("bad LEI format propagates", func(t *testing.T) {
		p := gmbhProfile()
		p.LEI = LEI("984500663D3E060B3G68")
		requireErrorCode(t, p.Validate(), "legalparty.lei.checksum_failed")
	})

	t.Run("bad ELF code rejected", func(t *testing.T) {
		p := gmbhProfile()
		p.LegalFormCode = "GmbH"
		requireErrorCode(t, p.Validate(), "legalparty.profile.legal_form_code_invalid")
	})

	t.Run("nil register skips validation", func(t *testing.T) {
		p := gmbhProfile()
		p.Register = nil
		requireNoError(t, p.Validate())
	})

	t.Run("bad register propagates", func(t *testing.T) {
		p := gmbhProfile()
		p.LEI = ""
		p.Register = &RegisterRef{Registry: "", Number: "HRB 1"}
		requireErrorCode(t, p.Validate(), "legalparty.register.registry_required")
	})
}
