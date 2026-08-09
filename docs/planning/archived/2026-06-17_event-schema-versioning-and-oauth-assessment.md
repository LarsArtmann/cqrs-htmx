# Assessment: Event Schema Versioning & OAuth2/OIDC

**Date:** 2026-06-17 | **Type:** Feasibility / scoping | **Status:** Assessed, not started

Two high-impact features from the backlog. Both are real features that warrant
ADR-level design before implementation. This document scopes each honestly so
they can be planned into dedicated sessions rather than half-built.

---

## 1. Event Schema Versioning (upcasters / migrations)

### Current state

The **foundation already exists**: every event payload struct carries a
`SchemaVersion int` field (`usermgmt/es_events.go`), and there is a single
`currentSchemaVersion = 1` constant (`es_constants.go:49`). All events are
written at version 1.

**What's missing:** the decode path does not act on `schema_version`.
`unmarshalPayload[T]` (`es_state.go:28`) calls `event.DecodePayload[T]` directly
with a JSON codec. If a payload struct gains a required field, old events in the
store will fail to decode (or silently zero-fill), corrupting the `foldUser`
replay. There is no registry of version-to-version transformers ("upcasters").

### Recommended design

An upcaster is a pure function `rawV(n) -> rawV(n+1)` applied to the **raw JSON
bytes** before typed decoding, chained by the `schema_version` the bytes carry:

```
load event → read schema_version → apply upcasters[v→current] → decode into T
```

**Concrete API proposal** (in `usermgmt`):

```go
// Upcaster transforms a raw payload from oldVersion to the next version.
type Upcaster func(raw []byte) ([]byte, error)

// UpcasterRegistry holds per-event-type, per-version transformers.
type UpcasterRegistry struct{ /* ... */ }

// Register chains an upcaster for (eventType, fromVersion).
func (r *UpcasterRegistry) Register(eventType event.Type, fromVersion int, u Upcaster)

// Upcast applies the full chain from the event's schema_version to currentSchemaVersion.
func (r *UpcasterRegistry) Upcast(evt event.Event) ([]byte, error)
```

`unmarshalPayload` would then optionally consult a registry before decoding.

### Effort & risk

| Aspect        | Detail                                                                                                                                             |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Effort        | Medium (1 focused session). API is small; the work is wiring it into `foldUser` + projection decode paths and testing.                             |
| Risk          | Low if added **before** the first breaking change. Adding it reactively after a schema change is harder (events already written at the new shape). |
| Urgency       | **Do it before usermgmt payload schemas evolve.** Today all events are v1, so this is the cheapest it will ever be.                                |
| Compatibility | The `schema_version` field is already persisted on every event — no data migration needed to introduce upcasters.                                  |

### Recommendation

**Worth a dedicated ADR + implementation before the first real schema change.**
Not speculative: it is defensive infrastructure that becomes painful to
retrofit. The existing `SchemaVersion` field means the on-disk format is already
forward-compatible.

---

## 2. OAuth2 / OIDC Integration (alternative to WebAuthn)

### Current state

`usermgmt` is **passwordless via WebAuthn only**. There is no password login
and no external identity provider (IdP) integration. Sessions are created
exclusively by `FinishLogin` after a WebAuthn ceremony.

### What it would add

An OAuth2/OIDC flow lets users sign in via an external IdP (Google, GitHub,
Auth0, etc.) instead of (or alongside) a passkey. The flow:

1. Redirect to IdP authorization endpoint.
2. IdP redirects back with an authorization code.
3. Server exchanges code for tokens, fetches userinfo.
4. **Link or provision a user**: if the IdP subject matches an existing user,
   create a session; otherwise register a new user (or link to current user).

### Design questions that need answers (not yet decided)

- **New event type?** `ExternalAccountLinked` (IdP subject + provider) as a new
  event on the User aggregate — this is the clean, event-sourced approach.
- **One user, many IdPs?** A user should be able to link multiple external
  accounts (Google + GitHub). This implies a `Credentials`-like list for
  external accounts, mirroring the WebAuthn credential model.
- **Provisioning policy:** auto-create on first login, or require pre-registration?
- **Library choice:** `golang.org/x/oauth2` (low-level, no new heavy dep) vs a
  full OIDC library (`coreos/go-oidc`). The library principle favors
  `golang.org/x/oauth2` (already a transitive dep via other modules).
- **Session creation:** reuse the existing `SessionStore` — OAuth login should
  produce the same `Session` type as WebAuthn login for uniform downstream behavior.

### Effort & risk

| Aspect    | Detail                                                                                                                                               |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Effort    | High (2+ sessions). New event/command, HTTP callback handler, token exchange, userinfo fetch, user-linking logic, tests with a fake IdP.             |
| Risk      | Medium. Security-sensitive (token handling, state/CSRF on the callback, account-takeover via IdP email reuse). Needs careful threat modeling.        |
| Dep noise | `golang.org/x/oauth2` adds a stdlib-adjacent dep; acceptable and widely trusted.                                                                     |
| Scope fit | Aligns with "alternative to WebAuthn" but **dilutes the passwordless story**. Should be opt-in (separate config + handlers), not replacing WebAuthn. |

### Recommendation

**Genuinely valuable but defer until there is concrete consumer demand.** Unlike
schema versioning, this is additive (not defensive) and carries real
security-design weight. When taken on:

1. Write an ADR first (event model, provisioning policy, library choice).
2. Mirror the WebAuthn pattern: new `ExternalAccount` value type + `LinkExternalAccount`/`UnlinkExternalAccount` commands + events.
3. Reuse the existing `SessionStore` and HTTP `AuthHandler` registration pattern.
4. Add a fake-IdP integration test (like the virtual-authenticator WebAuthn tests).

---

## Priority

| Feature                 | Verdict                                        | Why                                                                 |
| ----------------------- | ---------------------------------------------- | ------------------------------------------------------------------- |
| Event schema versioning | **Do soon** (defensive, cheap now)             | Foundation exists; retrofitting after a breaking change is painful. |
| OAuth2/OIDC             | **Defer to demand** (additive, security-heavy) | Real value but large surface; should not be rushed.                 |
