# authztest

Proof kit for an HTTP surface's access posture, the dispatch boundary behind it,
plus the grant semantics a posture matrix cannot see.

Three suites, three questions:

| Suite          | Question it answers                                                                      |
| -------------- | ---------------------------------------------------------------------------------------- |
| `Suite`        | Does every declared route refuse the callers it says it refuses?                         |
| `CommandSuite` | Does every declared command/query refuse them when dispatched with no route at all?      |
| `GrantSuite`   | Does a grant stay in its tenant, survive a revoke, and outlive the process that made it? |

Both are fail-closed: a declaration set that cannot be tested is an error, never
a skipped case. A skipped case reports green, which is the failure mode this
package exists to prevent.

## Installation

The package lives in the root module and is test-only in practice — it imports
`net/http/httptest` and `testing`, so it never reaches a production binary.

```go
import authztest "github.com/larsartmann/cqrs-htmx/v4/authztest"
```

## 1. Proving route postures

```go
func TestRoutes(t *testing.T) {
	authztest.Suite{
		Handler: app.Handler(), // fully wired — the gates live in middleware
		Routes: []authztest.Route{
			{Method: "GET", Path: "/healthz", Posture: authztest.Public,
				Why: "readiness probe; unauthenticated by design"},
			{Method: "GET", Path: "/orders", Posture: authztest.Authenticated,
				Why: "any signed-in user lists their own orders"},
			{Method: "POST", Path: "/admin/roles", Posture: authztest.RoleGated,
				Why: "only admin may change role grants"},
		},
		Anonymous:    func(r *http.Request) *http.Request { return r },
		Unprivileged: withSession("user-1"),
		Privileged:   withSession("admin-1"),
	}.Run(t)
}
```

`Posture` has three values — `Public`, `Authenticated`, `RoleGated` — and the
centrepiece case is the third: an authenticated caller **without** the role must
get `403`. A `200` there is the missing-gate bug.

`Why` is required. A posture without a reason is a guess, and a guess reviewed
later is indistinguishable from a decision.

### The other direction: undeclared routes

The matrix only proves what you declared. `Undeclared` closes the gap that
matters more — a route nobody declared is invisible to the matrix:

```go
registered := []authztest.RouteID{
	{Method: "GET", Path: "/healthz"},
	{Method: "GET", Path: "/orders"},
	{Method: "POST", Path: "/admin/roles"},
}

if missing := authztest.Undeclared(declared, registered); len(missing) > 0 {
	t.Fatalf("routes with no declared posture: %v", missing)
}
```

Feed it the surface's own route table, so adding a route without declaring its
posture fails the build.

## 2. Proving grant semantics

`GrantSuite` answers the questions the 2026-11-02 consumer audit found nobody
was asking: a grant that evaporated on restart, and a grant that only worked in
whichever domain the check happened to use.

```go
suite := authztest.GrantSuite{
	Checker:       service.Authz(),          // cqrshtmx.PermissionChecker
	Governor:      governor{svc: service},   // grant + revoke
	Subject:       "user-1",
	Tenant:        "acme",
	ForeignTenant: "globex",                 // required: isolation needs a comparison
	Roles:         []string{"staff"},
	Resource:      "orders",
	Action:        "read",
	Fresh: func(t *testing.T) cqrshtmx.PermissionChecker {
		// A checker built from DURABLE state alone: reopen the store, rebuild
		// the projection, drain it, and return the new engine.
		return reopen(t).Authz()
	},
}

suite.ProveTenantIsolation(t) // allowed in acme, denied in globex
suite.ProveRevocation(t)      // grant → allowed → revoke → denied
suite.ProveDurability(t)      // grant → fresh process → still allowed
```

Each case is self-contained: it grants before asserting, so any one of them can
run alone and `Governor.GrantRoles` is expected to converge on repeat calls.

`Governor` is string-shaped so this package never imports an identity model. The
adapter for this repository's own stack:

```go
type governor struct{ svc *usermgmt.Service }

func (g governor) GrantRoles(ctx context.Context, subject, tenant string, roles []string) error {
	return g.svc.GrantRoles(ctx, usermgmt.ActorID(subject), usermgmt.TenantID(tenant), toRoles(roles))
}

func (g governor) RevokeRoles(ctx context.Context, subject, tenant string) error {
	return g.svc.RevokeRoles(ctx, usermgmt.ActorID(subject), usermgmt.TenantID(tenant))
}
```

### Run it after the projections have drained

Both suites read whatever the engine has materialized so far. Under
`AsyncStartup` that can be nothing yet — poll readiness (or
`cqrshtmx.ProjectionReadinessCheck`) before running either suite, or a green
result means "the replay had not reached the projection yet".

## 3. Proving the dispatch boundary

`Suite` proves routes. A route gate lives in a handler the dispatcher never sees,
so a command can be refused on its route and admitted when a job, a seed, or a
service method dispatches it directly. `CommandSuite` proves the second surface.

```go
suite := authztest.CommandSuite{
    Cases: []authztest.DispatchCase{
        {
            Kind:    cqrshtmx.DispatchCommandKind,
            Name:    "CreateItem",
            Posture: authztest.RoleGated,
            Why:     "creates an item in the caller's tenant",
            Dispatch: func(ctx context.Context) error {
                return cmdDisp.Dispatch(ctx, &createItemCmd{})
            },
        },
    },
    Anonymous:    authztest.NoIdentity(),          // context untouched
    Unprivileged: authztest.AsUser(userNoRole),
    Privileged:   authztest.AsUser(userWithRole),
}

suite.Run(t)
```

The matrix per case:

| Posture declared | anonymous | unprivileged | privileged |
| --- | --- | --- | --- |
| `Public` | admitted | — | admitted |
| `Authenticated` | 401 class | — | admitted |
| `RoleGated` | 401 class | 403 class | admitted |

Assertions use `cqrshtmx.IsUnauthorized` / `IsForbidden`, not strings, so a
re-coded denial fails the case. `Verify` refuses a set it cannot prove — no
anonymous identity, no unprivileged identity for a `RoleGated` case, a case with
no reason, or a posture the matrix cannot probe — so a skipped case is impossible.

`UndeclaredDispatches(declared, registered)` is the coverage helper for wiring
that can name its own types. There is no dispatcher-side equivalent of
`Undeclared`: a dispatcher exposes no registered-type listing, which is why the
gate refuses an undeclared type at dispatch time instead.

## What this does not cover

- **Authentication mechanics** — login flows, credential storage, session
  fixation. Those belong to the auth-strategy modules' own tests.
- **Policy authoring** — whether a policy says the right thing. The kit proves
  the plumbing carries the decision; the policy content is a review question.
- **Load and concurrency** — the suites are single-request, deterministic.
- **Policy content** — a `CommandSuite` case proves the gate asks `command/Name`
  and honours the answer; whether the row exists, and for whom, is a policy
  review. `check-policy-rows` catches rows that match no declared type.

## Related

- `docs/guides/authorization-system.md` — grant → declare → decide → prove.
- `docs/PERMISSIONS.md` — the maturity ladder these suites move a service up.
- ADR-0057 — the authorization contract (why `Request`/`Decision` look like
  this).
- ADR-0063 — the dispatch boundary (why `CommandSuite` exists).
