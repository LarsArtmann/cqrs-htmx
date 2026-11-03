# ADR 0057: Authorization Contract — One Question Type, Durable Grants, Recognisable Denials

**Status:** Accepted
**Date:** 2026-11-02
**Related:** [ADR 0044](0044-casbin-first-class-dependency.md) (Casbin in identity-model), [ADR 0043](0043-identity-model-extraction.md) (domain source of truth), [ADR 0047](0047-re-export-layer-retirement-plan.md) (alias retirement), [ADR 0058](0058-defer-relationship-based-authorization.md) (ReBAC deferral), [2026-11-02 consumer audit](../status/archived/2026-11-02_10-13_consumer-usage-audit-and-authz-model_status.md)

## Context

A read-only audit of 17 downstream services established three facts.

1. **Nobody asks the policy engine anything.** Zero of twelve services in the
   dependency graph call `Enforce`, `EnforceEx`, `EnforceAny`,
   `Service.Authorize`, or the root HTTP authorization options. Zero write an
   object/action policy. Real usage is role membership, read back through
   `RolesForUser` / `ImplicitRolesForUser`.
2. **The grants that back that membership were not durable.** Two services wrote
   roles straight into the in-memory enforcer with `AddGroupPolicy`, which is not
   an event — the grant disappeared on restart. One of them compensated with a
   reconcile loop after every projection drain; another mounted an admin panel
   that no one could reach because the grant was behind a "users already exist"
   guard that fired on the second boot.
3. **A denial could not be recognised.** Three forbidden errors existed with
   three different codes (root `forbidden`, identity-model `forbidden`,
   adminui `adminui.forbidden`), so no classifier worked across modules. A
   three-argument check against the shipped four-field model was a request-size
   error, which surfaced as **503 Service Unavailable** rather than 403.

There were also four decision surfaces (`Enforce`, `EnforceEx`, `EnforceAny`,
`Service.Authorize`), three "is this user an admin?" predicates, and three
decision shapes reaching audit with matched rules dropped on the floor.

## Decision

### 1. One question type, in the root package

```go
type Request struct {
	Subject  string
	Domain   string
	Resource Resource   // {Type, ID}
	Action   string
}

type Decision struct {
	Allowed      bool
	Reason       DecisionReason   // allowed | denied_by_policy | no_matching_policy | not_authenticated
	MatchedRules []string
}
```

`Reason` exists because a bare boolean cannot distinguish "you are not signed in"
(401) from "you are signed in and not allowed" (403) from "your grant is
missing" (which is what an operator actually needs to see).

`Resource{Type, ID}` is split now, not later: today's model flattens it to one
Casbin object string, but a graph engine needs the kind to traverse edges, and
changing the field shape after consumers depend on it is a breaking change.
`String()` renders `type:id`, `type`, or `id`, so every existing single-string
caller keeps its behaviour.

### 2. The seam is primitive-shaped, and it points inward

```go
type PermissionChecker interface {
	Check(subject, domain, resource, action string) (allowed bool, matchedRules []string, err error)
}
```

The interface lives in the root package and is expressed in primitives, so
`identitymodel.Authz` satisfies it structurally. Two import rules survive intact:

- the root package still never imports Casbin (framework-agnostic, no draggable
  policy engine), and
- identity-model still never imports the HTTP layer (pure domain).

`usermgmt` is the only module that can see both, so the compile-time assertion
lives there.

### 3. Grants are events

`Service.GrantRoles(ctx, actor, tenant, roles)` dispatches the existing
`AddMember` command and falls back to `UpdateMemberRoles` when the membership
already exists, so it converges instead of erroring on a second call. Role
values are validated before dispatch — an invalid role would otherwise become a
policy row that no check ever asks about.

No new event type. The projection already handles membership events, so replay
reconstructs every grant on restart, rebuild, and second instance.

`Authz.AddGroupPolicy` is deprecated in v4 and removed in v5. Its replacement for
projections is `Authz.ApplyGroupPolicy`: the same write, without a durability
warning, because a replayer derives the grant from the journal by definition.

### 4. Denials share one code

`IsForbidden` / `IsUnauthorized` match on `forbidden` / `unauthorized` and walk
the whole unwrap chain, so a module that re-codes a denial is still recognised by
its cause. adminui's private sentinel now aliases identity-model's.

Matching on the code rather than sentinel identity is deliberate: `errors.Is`
against three separate sentinel values can never match two of them.

### 5. The arity is explicit, with a documented fallback

`EnforceIn(enforcer, subject, domain, resource, action)` is the shipped
four-field shape. `Enforce` keeps its three-argument signature (consumers depend
on it) and now asks in the wildcard domain, falling back to three arguments when
the enforcer reports a request-size mismatch — the only signal a duck-typed
`Enforce(rvals ...any)` interface can give about the model behind it.

A broken engine still answers 503. A denial answers 403. The regression is
covered end to end.

## Consequences

- Consumers can keep their role-membership model and gain durability for free.
- A 403 now means "policy denied"; a 503 means "engine failed". Those were the
  same answer before.
- The wildcard domain must be explicit. `Authorize` answers global policies only;
  tenant-scoped checks need `AuthorizeIn` or `AuthorizeRequest`.
- `Authz.GroupPolicies()` returns two kinds of row in the same section — grants
  `(subject, role, domain)` and mirrored hierarchy edges
  `(parentRole, childRole, domain)`. Anything enumerating grants must skip rows
  whose subject is itself a role name. This bit the first implementation of the
  admin-grant diagnostic and is pinned by a test.
- The static gate `check-permission-surface` fails on the four anti-patterns
  this ADR retires.

## Rejected alternatives

- **Put the contract in identity-model.** It would drag HTTP request types into
  the domain module, or the domain into every root consumer's import graph.
- **Make `PermissionChecker` take `Request`/`Decision`.** Structural typing then
  requires identity-model to import the root package — a dependency inversion.
- **Add a `GrantRole` event.** The membership aggregate already expresses the
  intent; a second event would fork the write path and the projection.
- **Keep `AddGroupPolicy` as the documented way.** It cannot be made durable
  without a journal write, which is exactly what `GrantRoles` does.

## Delivery slices

| Slice | Content | Gate |
| --- | --- | --- |
| 1 (landed) | `Request`/`Decision`/`PermissionChecker`/`Decide`, `IsForbidden`, `EnforceIn` + 4-arg fix, `GrantRoles`/`RevokeRoles`/`HasAdminGrant`, adminui sentinel, projection seam, `check-permission-surface` | contract tests, durability restart tests, scanner self-test |
| 2 | route-posture manifest helper, proof kit wired into consumer suites | 401/403/200 matrix per service |
| 3 | maturity ratchet promoted from advisory to blocking | two green waves |
