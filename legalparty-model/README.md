# legalparty-model

Event-sourced domain model for **legal parties** — the entities an invoice is addressed to. Companion module to `identity-model` for billing, built per [cqrs-htmx#45](https://github.com/LarsArtmann/cqrs-htmx/issues/45).

> **Why not `billing-model`?** That name is reserved by the Stripe billing plan of record (see `docs/planning/2026-11-20_20-25_SUPERB-stripe-billing-v3-execution-plan.md`, milestones M03/M23 with their vocabulary-purity gate). Legal parties are a different bounded context from subscriptions and processor references, so this module ships under its own name.

## The problem it solves

`identity-model` models **who can log in** (`User`), **machine actors** (`Bot`/`System`/`Service`), and a **tenancy boundary** (`Tenant`). None of these is the thing an invoice is addressed to: a party with a registered name, a registered address, a legal form, a jurisdiction, and tax identifiers that can enter a contract and receive a tax document.

| #45 failure mode | What this module does |
| --- | --- |
| Role and legal form conflated into one axis | `LegalFormClass` is a five-value enum (`natural_person`, `legal_entity`, `partnership`, `public_body`, `non_profit`), never a bool; per-class rules are explicit |
| `TenantDeleted` is a tombstone; retention and erasure pull in opposite directions | `LegalPartyClosed` is terminal and **never destructive** — no delete event exists, the profile survives for tax retention (GoBD, 10 years in DE) |
| One `TaxID` rejects valid Steuernummern | `VATID` (prefixed EU form, full ISO 7064 Mod 11,10 checksum for DE) and `TaxNumber` (prefix-less national forms) are separate types |
| Bots must not be billable parties | The decider rejects `ActorBot` and unattributed actors on every command; invoice-side types take `LegalPartyID`, never `UserID` |

## Install

```
go get github.com/larsartmann/cqrs-htmx/legalparty-model/v4
```

## Quick example

```go
package main

import (
	"context"
	"time"

	"github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/legalparty-model/v4"
	"github.com/larsartmann/cqrs-htmx/legalparty-model/v4/gleif"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func main() {
	addr, _ := legalparty.NewRegisteredAddress("Kurfürstendamm 11", "Berlin", "10719", "", "DE")
	profile := legalparty.LegalPartyProfile{
		LegalName:      "ANDHOFF Wohnbau GmbH",
		LegalForm:      legalparty.LegalFormLegalEntity,
		LegalFormCode:  "2HBR", // ISO 20275 ELF code (Gesellschaft mit beschränkter Haftung)
		Country:        "DE",
		RegisteredAddr: addr,
		LEI:            legalparty.LEI("984500663D3E060B3G67"), // mod-97 checksum validated
	}

	// 1. Enrich from the registry before humans confirm anything.
	client := gleif.NewClient(nil) // api.gleif.org, no key needed, 60 req/min
	facts, err := client.ResolveByLEI(context.Background(), profile.LEI)
	_ = err // map to your error handling; ErrRegistryNotFound / ErrRegistryUnavailable are the sentinels

	// 2. Dispatch: decider guards (no bots, anchors required, closed is terminal).
	decide := legalparty.DecideOpenLegalParty(
		id.NewStreamID(), profile,
		identitymodel.NewActorID(identitymodel.ActorUser, "01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		time.Now(),
	)
	events, err := decide(legalparty.LegalPartyState{}, event.Version(0))
	_ = events
	_ = err

	// 3. Wire the aggregate like any go-cqrs-lite decider (system.DomainConfig
	//    or decider.Repository — see systemadapter for the consumer pattern).
	var _ decider.Decider[legalparty.LegalPartyState] = legalparty.LegalPartyDecider()
}
```

Consumers confirm fetched facts through `DecideUpdateLegalPartyProfile` with the `*RegistryFacts` attached — the event then carries the profile **and** its provenance (source, fetch time, attribution URL), so the journal answers "what did we know, from where, when". Facts never mutate an aggregate on their own.

## Domain surface

| Concern | Symbols |
| --- | --- |
| Aggregate | `LegalPartyState`, `FoldLegalParty`, `LegalPartyDecider()`, `AggregateTypeLegalParty` |
| Events | `LegalPartyOpened` / `LegalPartyProfileUpdated` / `LegalPartyClosed` (no delete exists) |
| Commands | `OpenLegalParty` / `UpdateLegalPartyProfile` / `CloseLegalParty` (actor guard: user/system/service only) |
| Identity | `LegalPartyID` (ULID-backed brand), `LegalFormClass`, `CountryCode` (ISO 3166-1 assigned set) |
| Identifiers | `LEI` (ISO 17442 mod-97), `RegisterRef`, `VATID` (DE checksum per ISO 7064 Mod 11,10; AT/PL/NL + generic formats), `TaxNumber` (DE 10-13 digits; rejects prefixed VAT shapes) |
| Registry | `RegistryFacts` + `LEIResolver` / `RegisterResolver` ports, `gleif.Client`, `opencorporates.Client` |

### Registry integrations

- **GLEIF** (`gleif`): resolves `api.gleif.org/api/v1/lei-records/{lei}` (unauthenticated, 60 req/min; 404/429 are HTML and mapped by status code). `ResolveByRegister` uses GLEIF's exact-match `filter[entity.registeredAs]` — the number must be spelled as the national register writes it (`HRB 127525`, not `HRB127525`). Data is CC BY 4.0: attribute "Source: GLEIF" (`RegistryFacts.SourceURL` carries the record link).
- **OpenCorporates** (`opencorporates`): resolves `/v0.4/companies/{jurisdiction_code}/{company_number}` with the `X-API-TOKEN` header (token required; open plans 50 req/day, hard cap 5 req/s; rate limit answers 403). German records keep `company_type` null and the full address inside `street_address`, so the typed address projection only fills when `locality` exists — partial structure is never pretended. **ODbL attribution is mandatory**: display a hyperlink reading "from OpenCorporates" (the `SourceURL` field) and "Powered by OpenCorporates" for API-derived distributions; share-alike applies to open-data accounts.

Status and normalization are honest rather than clever: GLEIF's literal `"NULL"` entity status maps to empty, `US-IA`/`us_mi` style jurisdictions map to their country, and reserved ELF codes `8888`/`9999` leave `LegalFormCode` unset.

## Actor guard

Legal parties are maintained by human or trusted-infrastructure actors:

```go
identitymodel.ActorUser / identitymodel.ActorSystem / identitymodel.ActorService  // allowed
identitymodel.ActorBot / unattributed (zero) ActorID                             // ErrBotActorForbidden
```

Actor attribution rides event metadata through the fleet's standard actor-enrichment chain — payloads carry identity facts, never maintainers.

## Scope notes

- **Payloads are nested** (`Profile LegalPartyProfile`) rather than identity-model's flat style: `profile_updated` replaces the profile wholesale, so one value object is the single validation and evolution surface. `SchemaVersion: 1` pins the shape.
- **No invoice aggregate here.** Invoices are the Stripe-billing lane's concern; when they land, their party reference type is `LegalPartyID` — the "invoice references a `LegalParty`, not a `User`" rule from #45 is then a compile-time fact.
- **No projection/read model, no HTTP surface.** Like `identity-model`, this module declares the vocabulary and the decision logic; consumers wire projections and routes (see `systemadapter` for the `system.DomainConfig` pattern and a `metaengine.projectionadapter.TypeDecoder` example).
- **Validation depth is documented, not universal**: VAT checksums run for DE (verified against the BMF/BZSt example `DE136695976` and the python-stdnum reference vectors); other jurisdictions are format-checked. Checksum-valid ≠ registered — verify registration via VIES separately.

## The bill-to aggregate (issue #45)

Beside the party aggregate, the module ships a second, smaller aggregate:
**BillTo** — the link that answers "which legal party receives this
tenant's invoices". One stream per tenant, keyed by the tenant ID itself
(`DeriveBillToStreamID`), so there is exactly one link per tenant by
construction and no join table.

```go
stream := legalparty.DeriveBillToStreamID("tenant-01M…")

// open (the tenant's first billing link)
DecideOpenBillTo(stream, partyID, actor, time.Now())

// re-point bills at another party (the previous party stays journaled)
DecideRedirectBillTo(stream, otherPartyID, "holding restructure", actor, time.Now())

// terminal, retained (tax retention: closure never deletes anything)
DecideCloseBillTo(stream, "tenant churned", actor, time.Now())
```

Lifecycle mirrors the party aggregate: the same human/system/service actor
guard (`ErrBotActorForbidden` for bots and unattributed commands), typed
rejections for malformed input (`party_required`, `redirect_reason_required`,
`close_reason_required`), conflicts for state-machine violations
(`already_exists`, `closed`), and `ErrBillToNotFound` / `ErrBillToClosed`
sentinels. Redirecting to the party already named is an idempotent no-op.
Events: `BillToOpened`, `BillToRedirected`, `BillToClosed` — commands:
`OpenBillTo`, `RedirectBillTo`, `CloseBillTo`.
