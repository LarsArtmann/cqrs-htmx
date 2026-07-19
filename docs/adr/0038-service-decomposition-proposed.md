# ADR-0038: Decompose usermgmt.Service (Proposed)

## Status

PROPOSED — 2026-07-19 (deferred to a coordinated v5 major bump)

This ADR records the decision to decompose the `usermgmt.Service` god-object
and the rationale for deferring execution. It is a decision trail, not yet
shipped code.

## Context

`usermgmt.Service` has grown to ~52 methods and ~30 fields spanning six
domains: User, Membership, Tenant, Bot, OAuth2, and Session. It is the single
biggest piece of structural debt in the module:

- Adding a User concern risks touching Tenant/Bot code in the same file.
- The struct's field list mixes concerns (event store, auth providers, session
  store, lockout, OAuth2 state, TOTP pending secrets, ...).
- Test setup must construct the whole Service even when testing one domain.

The service methods are grouped by file already (`service_core.go`,
`service_register.go`, `service_login.go`, `service_tenant.go`,
`service_membership.go`, `service_bot.go`, `service_oauth2.go`), so the
extraction seams are visible.

## Decision (proposed)

**Extract six focused sub-services and keep `Service` as a backward-compatible facade.**

- `UserService`, `TenantService`, `MembershipService`, `BotService`,
  `OAuth2Service`, `SessionService` — each owns its methods and the fields
  relevant to its domain.
- `Service` embeds all six and retains its current method set as thin
  delegating facades, so the public API is unchanged (non-breaking).
- New code is encouraged to use the focused services directly; `Service`
  remains for existing consumers and one-call setup.

## Why deferred

Executing this safely requires:

1. Mapping all 52 methods to their owning sub-service (some cross domains).
2. Splitting the 30-field struct without breaking the `NewService` constructor.
3. Re-running the full usermgmt + integration_test suites (slowest module).
4. Coordinating with the other breaking changes (T16 renames, T17 tristate,
   T23 TOTPSecret, T24 ActorID) so they ship together in a single v5 bump
   rather than fragmenting the API across multiple minor releases.

Doing it piecemeal in one session on a published v4 library risks subtle
facade-wiring bugs (wrong receiver, missed delegation) that the test suite may
not fully catch. The facade-preserving design is correct but large (~240 min).
This ADR locks the decision so execution can proceed deliberately.

## Consequences (when shipped)

- **Positive:** Compiler-enforced domain boundaries; each sub-service testable
  in isolation; clearer ownership.
- **Positive:** Non-breaking via the facade — existing `svc.Register(...)`,
  `svc.CreateTenant(...)`, etc. keep working.
- **Negative:** More types in the public API surface (the six sub-services).
- **Negative:** One-time migration cost for the facade wiring.
