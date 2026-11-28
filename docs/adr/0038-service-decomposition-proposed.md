# ADR-0038: Decompose usermgmt.Service (Proposed)

## Status

PROPOSED — 2026-07-19 (deferred to a coordinated v5 major bump)

AMENDED — 2026-11-02: method-count ceiling raised once, 80 → 85, for the
ADR-0057 durable-grant surface. See "Ceiling history" below.

AMENDED — 2026-11-28: **EXECUTION STARTED in v4.** The ceiling was crossed
(95 by 2026-11-24, 96 on 2026-11-27), which (per the 2026-11-02 rule) starts
the decomposition instead of another raise. The first two increments,
`BotService` and `TenantService`, are extracted and embedded; `*Service` is
back at 75 declared methods. See "Execution log" below for what moved, the
three decisions made while moving it, and the remaining increments.

This ADR records the decision to decompose the `usermgmt.Service` god-object
and the rationale for deferring full execution to v5. The parts already
shipped in v4 are listed in the execution log.

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

## Execution log

### 2026-11-28 — first increments: `BotService` (6 methods) and `TenantService` (15)

These two went first because they are the aggregates with the least inbound
coupling: nothing outside their own files calls them except two bot helpers
(deleting or merging a user re-owns its bots), so each extraction is a
receiver change plus the fields that aggregate owns. `*Service` went from 96
declared methods to 75.

Three decisions made while executing refine the proposal above:

1. **Promotion is the delegating facade.** The decision says `Service`
   "embeds all six and retains its current method set as thin delegating
   facades". In code, hand-written delegates (the shape of the
   `OAuth2Service` prototype: `BeginOAuthLogin` forwarding to
   `s.oauth2Svc.BeginLogin`) still count against the guard and are one more
   place to get a receiver wrong. A sub-service embedded **by value** gives
   the same call sites (`svc.RegisterBot(...)`) through a method set the
   compiler writes, and a value (not a pointer) keeps "a Service without its
   sub-service" unrepresentable.
2. **Owned dependencies, no back-pointer.** Each sub-service holds the
   fields of its aggregate (`BotService`: the bot read model and the token
   pepper; `TenantService`: the tenant read model, the tenant repository and
   the unique-names switch) plus the shared dependencies it dispatches
   through. None holds a `*Service`: a back-pointer would let every
   sub-service reach everything, which is the god-object under a new name.
   Cross-aggregate cascades go through the embedded field and read as one
   (`s.BotService.deleteBotsForUserBestEffort(...)` in `DeleteUser`).
3. **The facade is pinned, not asserted.** `TestService_ExportedAPISurface`
   (`usermgmt/testdata/service_api_surface.golden`) renders the exported
   method set of `*Service` through reflection: 68 signatures, captured
   before the move and byte-identical after it. The risk named under "Why
   deferred" (wrong receiver, missed delegation, that the test suite may not
   fully catch) is now a failing test, because a method that drops off
   `*Service` changes that file.

The dead `botRepo` field left `Service` with the move: it was written at
construction and never read (`RegisterBotCommands` takes the repository
directly).

### Remaining increments, cleanest seam first

| Increment | What moves | What it waits for |
| --- | --- | --- |
| `OAuth2Service` | the three hand-written delegates (`BeginOAuthLogin`, `FinishOAuthLogin`, `UnlinkExternalAccount`) | the sub-service becomes non-optional: always embedded, a nil provider answering `ErrOAuthNotConfigured` inside it |
| `SessionService` | authentication: `totp.go`, `webauthn_service.go`, `service_login.go`, `service_impersonation.go`, `service_auth_methods.go` | nothing; it owns the session store, the lockout store and the auth audit that the others borrow |
| `UserService` | `service_misc.go`, `import_export.go`, `email_verification.go`, `service_merge.go` | `SessionService`, because `DeleteUser` and `MergeUsers` revoke sessions |
| `MembershipService` | `service_membership.go`, `service_grants.go` | nothing; `TenantService` reads the membership read model directly today and keeps doing so until then |
| What stays on `Service` | construction, lifecycle (`Close`, `GracefulClose`, `Stop`), the infrastructure accessors, projection rebuild | nothing; this is the end state, about 16 methods of shared infrastructure |

Every increment takes the same three steps: the golden must not change, the
methods and their fields move, the golden and the guard are re-run.

## Ceiling history

The `check-service-methods.sh` guard keeps this ADR honest while execution is
deferred: it fails CI when `*Service` reaches the ceiling so the decomposition
decision is re-made consciously instead of by drift.

| Date       | Ceiling | Count at decision | Reason                                                              |
| ---------- | ------- | ----------------- | ------------------------------------------------------------------- |
| 2026-07-19 | 80      | ~52                | ADR accepted; tripwire set with headroom above the then-current count |
| 2026-11-02 | 85      | 81 → 80            | ADR-0057 durable-grant surface added 3 methods (`GrantRoles`, `RevokeRoles`, `HasAdminGrant`); one receiverless costume method (`classifyDispatchError`) reclaimed as a package function the same day |
| 2026-11-28 | 85      | 96 → 75         | The crossing: 95 by 2026-11-24 (the identity-merge and tenant waves), 96 on 2026-11-27. It surfaced only on 2026-11-28, when the earlier failures in the same CI job had been healed. Per the rule below the decomposition started instead of a raise: `BotService` and `TenantService` extracted. The guard also gained a per-sub-service ceiling (30), discovered from the receivers, so the god-object cannot be renamed |

The 2026-11-02 raise was a one-time accommodation for a shipped, documented
contract, not a precedent. The crossing it reserved the decision for happened
on 2026-11-27 and started the decomposition (see the execution log). Raising
the ceiling again without executing this ADR requires a new ADR that
supersedes this one.
