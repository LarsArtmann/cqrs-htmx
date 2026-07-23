# ADR-0044: Casbin as First-Class Dependency of identity-model

## Status

ACCEPTED — 2026-07-23

## Context

The Authz engine (wrapping Casbin's enforcer with type-safe Policy/GroupPolicy
wrappers, role queries, and policy CRUD) lived in usermgmt alongside SQL
stores, HTTP handlers, and CQRS infrastructure. Casbin was a dependency of
usermgmt only — not of the domain model.

When identity-model was extracted (ADR-0043), the question arose: where does
the Authz engine live? Two options:

1. Keep Authz in usermgmt, alias it from identity-model
2. Move Authz to identity-model as a first-class dependency

## Decision

**Casbin is a first-class dependency of identity-model. The Authz engine lives
in identity-model.**

Rationale:

- Casbin IS the authorization model. It is not infrastructure like SQL or HTTP.
  identity-model is an identity/auth domain module — the authorization engine
  belongs here.

- Policy, GroupPolicy, Role, Action, Effect are domain types. They describe
  what actions are permitted, not how they're stored or transmitted. Keeping
  them in usermgmt would force every consumer to import the full infrastructure
  stack just to check permissions.

- The Authz engine is usable standalone: `NewAuthz()`, `Authorize()`,
  `Enforce()`, policy CRUD — none of these require an event store, database, or
  HTTP server. Moving it to identity-model makes this explicit.

- The `CasbinProjection` (which syncs Casbin policies from events) stays in
  usermgmt because it is infrastructure (it reads from an event stream and
  writes to Casbin). The projection uses `identitymodel.Authz` through the type
  alias, maintaining clean separation.

## Consequences

**Positive:**
- Authorization logic is testable without any infrastructure
- Consumers can use the Authz engine without usermgmt
- The domain model is self-contained: types + logic + authorization
- Casbin version is pinned once in identity-model/go.mod

**Negative:**
- identity-model depends on casbin/v3, adding ~2MB to its dependency tree
- A consumer who only needs UserID/Email types also pulls in Casbin (mitigated
  by Go's lazy module loading — unused Casbin code is never compiled)

## Alternatives Considered

1. **Keep Authz in usermgmt** — Rejected: would require identity-model to
   depend on usermgmt for authorization types, creating a circular dependency
   or forcing all Authz types to live in root.

2. **Put Authz in a separate authz module** — Rejected: over-modularization.
   Authz is inherently part of the identity domain; splitting it adds a module
   boundary with no benefit.

3. **Use an interface instead of concrete Casbin** — Rejected: identity-model
   already provides the `Enforcer` duck-type interface for consumers who want
   to swap Casbin implementations. The concrete engine is Casbin because that's
   the default RBAC model; making the default abstract would add complexity
   with no payoff.
