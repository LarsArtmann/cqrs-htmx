# ADR-0043: Extract identity-model as Domain Source of Truth

## Status

ACCEPTED — 2026-07-23

## Context

usermgmt was a monolithic module containing ALL domain logic: IDs, events,
commands, state structs, fold functions, Authz engine, errors, crypto helpers,
session/user/membership types, and the upcaster registry. Every consumer
(adminui, loginpage, integration_test, examples) depended on usermgmt for
domain types, creating tight coupling between the "what" (domain model) and the
"how" (CQRS infrastructure, SQL stores, HTTP handlers).

This made it impossible to:

- Share domain types without pulling in the full usermgmt infrastructure
- Test domain logic in isolation
- Evolve the domain model independently of the persistence/transport layer
- Reason about the authorization model without a SQL event store

## Decision

**Extract a new `identity-model` module as the single source of truth for all
identity domain types, logic, and the Casbin-backed Authz engine.**

Concrete changes:

- **identity-model/v4** — New module containing: IDs (UserID, TenantID, BotID,
  ActorID), events (22 payload structs), commands (19 structs with accessor
  methods), fold functions (FoldUser, FoldMembership, FoldTenant, FoldBot),
  state structs, Authz engine (Casbin-backed), Session, User, Membership,
  ExternalAccount, WebAuthnCredential, crypto helpers, domain errors
  (errorfamily-only, no HTTP dependency), upcaster registry, and exported
  constants (event types, command types, aggregate types, schema version).

- **usermgmt** — Rewired to use type aliases (`type UserID =
identitymodel.UserID`) and var aliases (`var foldUser =
identitymodel.FoldUser`) for ALL domain types. No domain type is defined in
  usermgmt anymore. usermgmt retains only: Service orchestration, HTTP
  handlers, SQL stores, CasbinProjection, read models, and decide/dispatch
  functions.

- **Casbin as first-class dependency** of identity-model (see ADR-0044).

- **Upcaster registry** moved to identity-model so fold functions can use it
  directly. usermgmt re-exports the types via aliases.

- **Constants** exported from identity-model (e.g. `EventUserRegistered`,
  `CmdRegisterUser`) and aliased in usermgmt via `var` (Go has no const alias).

## Consequences

**Positive:**

- Domain types are accessible without importing infrastructure code
- identity-model has zero dependency on usermgmt, cqrs-htmx root, or SQL
- The authorization model (Authz) is usable standalone
- Fold functions and upcaster logic have a single implementation
- Test coverage of domain logic is isolated and fast (1s vs 20s for usermgmt)
- Constants have a single source of truth (no string-literal drift)

**Negative:**

- Go has no const alias, so constants become `var` in usermgmt (practically
  irrelevant: nobody reassigns event type strings)
- `unmarshalPayload` requires a thin wrapper in usermgmt (generic function
  cannot be aliased via var)
- The global upcaster registry now lives in identity-model, slightly increasing
  identity-model's coupling to event schema evolution (but this is a domain
  concern, so it belongs here)

## Alternatives Considered

1. **Keep domain types in usermgmt** — Rejected: the coupling problem would
   persist and worsen as more consumers are added.

2. **Shared types in root module** — Rejected: root is the HTTP/HTMX library,
   not a domain model. Mixing domain types with HTTP infrastructure creates the
   same coupling in a different place.

3. **Proto/IDL-generated types** — Deferred: adds build complexity with no
   current payoff for a single-language (Go) codebase.
