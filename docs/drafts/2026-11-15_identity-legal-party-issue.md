## Why

`identity-model` models **who can log in** (User), **machine actors** (Bot/System/Service, `identity-model/id.go:161`), and a **tenancy boundary** (Tenant). None of these is the thing an invoice is addressed to.

Billing needs a fourth concept: a **legal party** — an entity with a registered name, a registered address, a legal form, and tax identifiers, which can enter a contract and receive a tax document. The closest thing today is `Tenant`, and it carries two strings and a lifecycle:

```go
// identity-model/fold.go:61
type TenantState struct {
	Name          string
	DisplayName   string
	Suspended     bool
	SuspendReason string
	Deleted       bool
	DeleteReason  string
}
```

The glossary is explicit that a Tenant is an "organizational boundary for multi-tenancy" (`docs/DOMAIN_LANGUAGE.md:58`) — authorization scoping, not legal personality.

## What breaks

Two failures, both visible on the invoices side:

1. **Role and legal form are conflated into one axis.** The invoices project models the party _role_ (`internal/models/party_type.go:8`, `PartyType` = seller/buyer/end_customer/bank) but never the _legal form_ (`internal/models/enhanced_invoice_entities.go:35` carries address and `VATNumber`, no form, no register number). A GmbH and a freelance natural person are both `buyer`, yet the document requirements differ: legal-form suffix in the name, register number, VAT ID vs Steuernummer. The glossary already records that a Steuernummer must not be validated as `models.TaxID` because the two identifiers are not interchangeable — that distinction has no home in identity-model.
2. **A tenant can be deleted; an invoiced party cannot.** `TenantDeletedPayload` is a tombstone (`identity-model/events.go:129`). Tax retention (GoBD, 10 years in DE) requires the party behind an issued invoice to survive. If the billing party is the tenant, retention and erasure pull in opposite directions with no type to express the resolution.

## Legal form is not a boolean

`is_company` cannot express the cases this project already bills against. Three values are the minimum, and they are not a hierarchy:

| Value                              | Example                        | Legal personality                      | Billing consequence                                                |
| ---------------------------------- | ------------------------------ | -------------------------------------- | ------------------------------------------------------------------ |
| Natural person                     | freelancer, private individual | the human                              | name + address; USt-IdNr. optional; Steuernummer if DE             |
| Legal entity (juristische Person)  | GmbH, UG, AG, Ltd              | separate from its owners               | legal form + register number + VAT ID + registered address         |
| Partnership (Personengesellschaft) | GbR, OHG, KG, PartG            | legally capable, not a juristic person | partners are liable; form suffix in the name; VAT ID on the entity |

Depending on jurisdiction also: public-law body (Körperschaft des öffentlichen Rechts), association / non-profit (e.V., gGmbH). What must stay out: Bot/System/Service actors are never billable, and a _sole trader_ is a natural person billed under a business name — not a fourth value.

## Proposal

- [ ] Add `LegalForm` as a typed enum plus a jurisdiction field, not a bool: `natural_person`, `legal_entity`, `partnership`, `public_body`, `non_profit`.
- [ ] Add `LegalParty` as its own aggregate/stream (name, registered address, legal form, jurisdiction, register number, billing contact email): `legal_party.opened` / `.profile_updated` / `.closed` — never deleted.
- [ ] Keep tax identifiers as separate typed value objects: VAT ID (`DE` + 9 digits / PL NIP) and Steuernummer are different types; a `TaxID` that accepts both will reject valid Steuernummern.
- [ ] Model the link explicitly: `Tenant --bill_to--> LegalParty` (0..1 default). A tenant is a login/authorization scope, not automatically the payer, and one payer may cover several tenants.
- [ ] Add the `ActorIDAsUserID`-style guard for the new axis: a `LegalParty` must not be constructible from a bot actor, and an invoice must reference a `LegalParty`, not a `User`.

One decision is needed before the first schema: does `LegalParty` live in `identity-model` (types-only, zero infrastructure, so it fits mechanically) or in a sibling `billing-model`? My preference is the sibling — address, VAT and retention are billing concerns, and the identity-model README pins its scope to identity and access.

## To verify

- [ ] `LegalForm` round-trips through the event codec and rejects an unknown value.
- [ ] A `LegalParty` survives tenant deletion and user deletion (retention test).
- [ ] A natural person and a GmbH with identical buyer-role data produce different invoice validation outcomes (name suffix, register number required for the entity).

💘 Generated with Crush (DeepSeek-V4.1-Flash)
