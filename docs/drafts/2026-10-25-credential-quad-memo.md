# Credential 4× duplication memo (S53, 2026-10-25)

Read-only analysis, no code (feeds ROADMAP OQ28 / the owner's keep-vs-consolidate call).

## The four shapes

| Struct | Module | Visibility | Fields | Delta vs `CredentialCore` |
| --- | --- | --- | ---: | --- |
| `CredentialCore` (embedded in `WebAuthnCredential`) | identity-model | **exported, published** | 11 | — the canonical set (`ID`, `PublicKey`, `AttestationType`, `Transports`, `AAGUID`, `SignCount`, `UserPresent`, `UserVerified`, `BackupEligible`, `BackupState`, `Name`); the wrapper adds `CreatedAt` |
| `CredentialView` | systemadapter | exported, published | 11 | **field-for-field identical** (same names, same JSON tags) |
| `credentialData` | usermgmt/webauthn | private | 10 | minus `Name` |
| `webAuthnUserCred` | usermgmt | private | 10 | minus `Name` (twin of `credentialData`) |

A fifth partial shape exists by design: `CredentialUsedPayload` (the S32 event) carries
`CredentialID` + `SignCount` + `UP/UV/BackupState` — an assertion fact, not a credential
mirror; out of scope for consolidation.

## Boundary roles

1. **`CredentialCore` — domain core AND event payload.** Embedded in both
   `WebAuthnCredential` and `CredentialAddedPayload` (its doc comment states this
   contract). Adding a field here is a wire change to the published language.
2. **`CredentialView` — declarative projection view.** systemadapter already imports
   identity-model, so this mirror is NOT forced by a dependency boundary; it duplicates
   11 fields + tags that could be had by embedding.
3. **`credentialData` ↔ `webAuthnUserCred` — one wire contract, two sides.** The
   `WebAuthnProvider` seam passes the user as JSON across a duck-typed interface
   (usermgmt cannot import the webauthn module and vice versa; the interface is the
   only legal coupling). Each side needs its own struct to marshal/unmarshal; the
   duplication is the price of the seam, and both are private.

## Usage map

- `CredentialCore`: credential.go (constructors/clone), events.go (payload embed),
  fold.go (`applyCredentialUsed`/`applyCredentialRemoved`), usermgmt re-exports.
- `CredentialView`: systemadapter views.go (UserView), declarations.go (the two
  declarative fold sites + `updateCredentialUsed`).
- `credentialData`: provider.go ×6 (to/fromWebAuthnCredential, marshal/unmarshal user).
- `webAuthnUserCred`: webauthn_service.go ×5 (marshalWebAuthnUser, unmarshal).

## Recommendation (for the owner call)

- **Keep the webauthn pair; pin it.** The seam requires two structs. The risk is silent
  drift (a field added to one side only corrupts the wire). Mitigation without new
  coupling: a cross-side wire golden — marshal a `WebAuthnCredential` through
  `fromWebAuthnCredential` → JSON → `webAuthnUserCred` (usermgmt test) → JSON back →
  assert byte equality against the provider side's golden. ~20 lines, one test per
  side, zero imports across the seam. (`TestCredentialConversion_RoundTrip` covers the
  provider half already; the usermgmt half is the gap.)
- **Consolidate `CredentialView` by embedding** `identitymodel.CredentialCore`
  (anonymous embedding serializes to the same flat JSON — the declarative UserView
  bytes would not change; a byte-equality golden proves it). This removes the only
  duplication that has no architectural excuse. Caveat: it couples the view schema to
  the domain type — if systemadapter's view is meant to drift independently of
  identity-model (e.g. rename fields for a stable external schema), keep the flat
  mirror and instead add the same wire golden. That preference is the actual OQ28
  question.
- **Do not** chase a shared "credential wire" module for the webauthn pair — a new
  published module to hold 10 fields is over-engineering; the golden test achieves the
  safety at ~1% of the cost.

**Default if the owner does not rule:** implement the two wire goldens (both sides),
leave all four structs as they are, and revisit only when a 12th field arrives.
