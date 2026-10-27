# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Strict branded-ID reconstruction (S38 a/b):** `ParseTenantID`/`ParseBotID` — the strict counterparts to `NewTenantID`/`NewBotID`. They strip exactly one leading display prefix (`Tenant:`/`Bot:`) and reject empty or whitespace-padded values (Rejection family) — the ambiguous forms that would otherwise silently become distinct identities. `FoldMembership` now reconstructs `TenantID` through `ParseTenantID`, so journaled events with ambiguous tenant references are rejected at replay time. `GenerateUserID` preserves ULID time-ordering (minted later ⇒ sorts later across millisecond boundaries; pinned by test).

- **`UserState.IsActive`/`BotState.IsActive` (S38c, additive):** the uniform write-path liveness gate — `Exists() && !Deleted` for users (a deleted user still HAS history, so `Exists` keeps its "has events" meaning), the existing-tombstone mirror for bots. `MembershipState.Exists` already folds `Removed`, so no split was needed there; documented on the type.

### Changed

- **`NotConfigured` errors are now Rejection, not Transient (S38e):** `ErrEmailVerificationNotConfigured`, `ErrTOTPNotConfigured`, and `ErrOAuthNotConfigured` answer 400-class instead of 503 — a missing deployment configuration is permanent for this deployment, and a retryable signal would tell clients a retry can help when it cannot. Consumers mapping error families to HTTP statuses see the correction; none of the in-tree suites pinned the old 503s.

- **Command slice accessors return copies (S38f):** `RegisterUserCmd.Roles`, `RemoveCredentialCmd.CredentialID`, `EnableTOTPCmd.Secret`, `AddMemberCmd.Roles`, `UpdateMemberRolesCmd.Roles`, `RegisterBotCmd.TokenHash`, `RegisterBotCmd.Scopes` now return `slices.Clone` — mutating a returned slice can no longer write through to the command (commands are immutable once constructed; the TOTP secret and bot token hash are credential material).

### Fixed

- **`NewAuthz` no longer seeds `g2` into custom models without a `g2` section (S38d):** `AddNamedGroupingPolicy("g2", ...)` errors against an undeclared section, so a consumer model without `g2` failed construction. Seeding is now guarded (and extracted into `seedDefaultRoleHierarchy`); default-model behavior is unchanged.

- **Role resolution for actors is strict (S38g):** `RolesForActor`/`ImplicitRolesForActor` and `ActorIDAsUserID` parse actor identifiers through `ParseUserID` instead of the deprecated hash-masking `NewUserID` — a non-ULID subject now surfaces as a parse failure instead of silently colliding with an unrelated synthetic user.

## [v4.14.0] - 2026-10-25

### Added

- **WebAuthn credential-use recording — the clone-detection data model.** `CredentialUse` (credential ID, signature counter, UP/UV/backup flags observed during one successful assertion), the `RecordCredentialUse` command + `CredentialUsed` event (payload carries counter + flags), `WebAuthnCredential.CounterRegressed(next)` implementing the WebAuthn L3 §7.2 step-17 clone signal (0→0 is not a regression; any non-zero count that fails to advance is), and fold/read-model support so the LAST assertion's state is what the next one compares against. Without it, clone detection compared every assertion to the registration-time counter and never fired.

### Changed

- **`WebAuthnProvider.FinishLogin` now returns `(credentialJSON []byte, err error)`.** The provider returns the credential state the assertion produced so the service can persist it. Technically breaking for custom implementors of the interface (the in-tree `usermgmt/webauthn` provider is updated in the same train); shipped as a minor following the v4.7.0 precedent — the interface has one official implementation and keeping clone detection dead was the greater harm.

### Fixed

- **Role hierarchy now fires, and `Authz.Apply` is atomic as documented.** Casbin keeps one role manager per role definition, so the `g2` edges `NewAuthz` seeded never composed with the domain-scoped `g` grants that assign roles: a user granted `user` was denied what `viewer` allows (admin only looked correct because the seed policy gives it `*/*/*`). The hierarchy is mirrored into `g` per domain by `EnsureRoleHierarchy` (called from `AddGroupPolicy`, `Apply`, and `NewAuthz`; idempotent, replay-safe, tenant-scoped), and `UsersForRole` no longer reports role names as subjects. `Apply` now rolls the whole batch back when one operation fails instead of leaving earlier operations committed. Consumers listing `GroupPolicies()` should count grants by non-role subjects: the hierarchy edges are rows too.

## [v4.13.0] - 2026-10-19

### Added

- **Auth audit event vocabulary (cqrs-htmx#23):** `AggregateTypeAuth` stream type plus three event types — `LoginSucceeded`, `LoginFailed`, `RegistrationClosed` — with payload structs (`LoginSucceededPayload{Method,Provider}`, `LoginFailedPayload{Method,Reason,Provider}`, `RegistrationClosedPayload{SchemaVersion}`) and two bounded vocabularies: `AuthMethod` (webauthn | totp | oauth2) and `AuthFailureReason` (no_credentials, ceremony_failed, code_invalid, state_invalid, provider_mismatch, token_exchange, no_email). These are SECURITY AUDIT facts, not user state: emitters place each event on its OWN Auth-typed stream so login volume never grows the user aggregate, and reason classes are coarse by contract — never raw provider error text, never secrets. Additive; no existing event, fold, or payload changes.

### Fixed

- **Fold-aliasing invariant violations + guards (2026-10-18 full-code-review wave).** `FoldUser`'s `TOTPSecret` and `FoldBot`'s `TokenHash` retained payload-owned `[]byte` slices despite the package's own "every slice taken from a payload is defensively copied at the fold site" invariant (`doc.go`) — both now `slices.Clone`. `TenantCreated` clears stale `SuspendReason`/`DeleteReason` (a recreated tenant inherited its previous incarnation's reasons and failed `IsValid`). `MustParseUserID` routes through the brand-prefix-tolerant `ParseUserID` (previously panicked where its non-Must sibling succeeded). `User.MarshalJSON` marshals a nil receiver as `null` (previously nil-deref'd on `len(u.Credentials)`).

## [v4.12.3] - 2026-10-11

### Fixed

- **Canonical nested struct literals for embedded cores (exhaustruct, 2026-10-11).** `ExternalAccount` construction sites now use the explicit `ExternalAccountCore: ExternalAccountCore{...}` form instead of promoted-key literals — identical runtime behavior, unambiguous for readers and linters alike.
- **Go toolchain floor raised 1.27.1 -> 1.27.2 (security, 2026-10-11).** `go.mod` declares `go 1.27.2` (GO-2026-6603/6611/6613/6617 stdlib fixes); consumers on 1.27.2+ see no change.

# [v4.12.2] - 2026-10-08

### Changed

- **Fold defensive copies modernized to `slices.Clone` (no behavior change):** `FoldMembership` (roles ×2) and `FoldBot` (scopes) replace the `make`+`copy` pairs with `slices.Clone`; the payload-slices-alias-event-bytes invariant (every slice taken from a payload is defensively copied at the fold site) is now documented in `doc.go` so future folds keep it.
- **Dependency alignment:** `go-cqrs-lite id/v4 v4.7.0→v4.7.2` (race-red patch wave), indirects `snapshot/v4 v4.6.2`, `storage/memory/v4 v4.6.2`, `fxamacker/cbor/v2 v2.9.6`.

### Added

- **`TestUserID_ParseBrandPrefix_Policy` — the `ParseUserID` prefix-strip policy is pinned (6 subtests):** known brand prefixes strip and parse, unknown brands strip the same way (generic policy, not a whitelist), multi-colon and non-ULID tails reject, brand-only strings reject, and the leading-colon quirk is pinned as documented behavior. Mechanizes the D13 decision input (`docs/drafts/2026-10-07-d13-parseuserid-prefix-policy-memo.md`).

## [v4.12.1] - 2026-10-05

### Fixed

- `ParseUserID` (and the `NewUserID` fallback path) now tolerate a leading branded-id prefix (`StreamMarker:<ulid>`, as emitted by branded-id `String()`): the prefix is stripped before the strict ULID parse. Under go-cqrs-lite `id` v4.7+ display-form branding, prefixed strings previously failed the strict parse — or were SHA-256-hashed into a wrong-but-valid user by the deprecated `NewUserID` fallback. ULIDs never contain colons, so a colon is always a brand separator, never part of the identity. Mirrors go-cqrs-lite's `ParseStreamID` round-trip law; narrative: root CHANGELOG § Fixed (2026-10-05) and AGENTS gotcha 25.

## [v4.12.0] - 2026-09-22

_(family v4.12.x train release — the train receipt lives in the root CHANGELOG; no identity-model-specific entry was recorded at cut time.)_

## [v4.11.0] - 2026-09-19

### Changed

- **Go 1.27.1 floor + dependency carrier:** `go.mod` requires Go 1.27.1 (matching the 2026-09-19 fleet cut; `go-etag v0.4.0` and go-cqrs-lite's re-pin force the floor). No domain-type or behavior changes since v4.9.0 (verified by tree diff): this release is the coordinated-train cut carrying the dependency alignment — go-branded-id v0.5.1→v0.6.0, go-codec v0.2.0→v0.3.0, go-cqrs-lite command v4.8.1→v4.10.0, event v4.9.0→v4.11.0, id v4.5.0→v4.6.0, go-error-family v0.10.0→v0.10.1.

## [v4.3.0] – [v4.9.0] — 2026-08-14 → 2026-09-09

- Coordinated lockstep bumps riding the family release trains (identity-model shipped v4.8.0 and v4.9.0 in the 2026-08-14/2026-09-01 trains). No identity-model-specific feature entries were recorded in this file for those trains; the historical detail lives in the root `CHANGELOG.md` sections for those versions.

## [v4.2.0] - 2026-08-07

## [v4.1.1] - 2026-07-27

### Changed

- **Simplified `HasRole`/`HasAnyRole`** (`fold.go`, `membership.go`): replaced manual loops with `slices.Contains`/`slices.ContainsFunc` for clarity and correctness.
- **go-cqrs-lite v4.1.0**: all go-cqrs-lite module references updated to v4.1.0 (codec at v4.1.1).

### Added

- **Module metadata files**: `.editorconfig`, `.gitattributes`, `.gitignore`, `CONTRIBUTING.md`, `CHANGELOG.md` — standard project hygiene for an independently versioned Go module.

## [v4.1.0] - 2026-07-24

### Added

- **Pure domain types for event-sourced identity management** (ADR-0043): IDs (UserID, TenantID, BotID, ActorID), event payloads (22 structs), commands (19 structs with accessor methods), state structs + fold functions (FoldUser/FoldMembership/FoldTenant/FoldBot), Authz engine (Casbin-backed — ADR-0044), RBAC model + default policies + role hierarchy, domain errors (errorfamily-only, no HTTP dependency), crypto helpers, upcaster registry, exported constants (41 event/command/aggregate-type constants).
- **Casbin as a first-class dependency** (ADR-0044): the Authz engine wraps a Casbin enforcer, enabling declarative role/tenant authorization policies.
