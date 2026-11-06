# ADR 0063: Authorization at the Dispatch Boundary — Postures Declared at Registration

**Status:** Accepted
**Date:** 2026-11-06
**Related:** [ADR 0057](0057-authorization-contract.md) (the contract this reuses), [ADR 0058](0058-defer-relationship-based-authorization.md) (why params stay closed), [ADR 0044](0044-casbin-first-class-dependency.md) (Casbin in identity-model), [ADR 0055](0055-owner-session-gated-credential-ceremonies.md) (library-enforced route posture), plan of record [`2026-11-06_07-30_SUPERB-command-permission-gate-dispatch-boundary-pareto-plan.md`](../planning/2026-11-06_07-30_SUPERB-command-permission-gate-dispatch-boundary-pareto-plan.md), predecessor report [`2026-11-06_07-24_..._status.md`](../status/2026-11-06_07-24_command-permission-middleware-design-triage_status.md)

## Context

ADR 0057 shipped one question type and one denial vocabulary, and ADR 0055 made a
route's posture a property the library enforces. Both work. Neither reaches a
dispatch that has no route.

Authorization today is a property of the **handler**: `Authorize`,
`AuthorizeIn` and `AuthorizeRequest` write `authMode`/`resource`/`action` into a
`handlerConfig`, and `executeAuthorization` reads it before the body is decoded.
The dispatcher never sees any of it. Three consequences follow, all verified in
this repo:

1. **A command dispatched without a route is authorized by nobody.** Service-level
   `Dispatch` — the call shape of `Service.GrantRoles`, seeds, bootstraps,
   scheduled jobs and admin CLIs — takes a context and a command. There is no
   gate in that path. `TestDispatchWithoutRouteIsUngated` pins this, including the
   sharper form: the *same* command refused on a gated route is admitted when the
   dispatcher is called directly.
2. **A route gate cannot be moved down.** `handlerConfig` is a per-endpoint value;
   the dispatcher has no registry of it. Copying the option into the middleware
   would create a second declaration of the same fact, which is exactly the drift
   guardrail 11 forbids.
3. **The repository can be at L4 and still ship this hole.** The maturity ladder
   measures route postures. A repo can declare every route perfectly and still
   dispatch ungated from a job.

The missing half is therefore not another HTTP option. It is a **dispatch-level
gate with a declaration surface**, so that "this command is gated" is a fact about
the command rather than about one of its callers.

Two upstream facts shaped the design:

- `command/v4@v4.14.1/handler.go:10` defines `Handler func(ctx, cmd Command) error`
  and `dispatcher/v4@v4.5.2/middleware.go:12` defines `Middleware[H any] func(H) H`,
  so a middleware receives the concrete command value and the actor is already in
  `ctx` via the audit chain. No upstream change is required.
- `dispatcher/v4@v4.5.2/dispatcher.go:62..206` exposes no enumeration API — only
  `Use`, `Register`, `Dispatch`, `Close`. A gate therefore cannot discover what is
  registered. If registration is not the declaration, nothing is.

## Decision

### 1. The gate lives in the root package, and the object is spelled with a slash

`AuthorizeCommand(checker)` and `AuthorizeQuery(checker)` are
`command.Middleware`/`dispatcher.Middleware[query.Handler]` values built from the
already-accepted contract: they derive a `Request` from the command and call
`Decide`. No new decision type, no new denial code, no new dependency — root still
imports no Casbin and no identity model (guardrail 4 of ADR 0057 survives intact).

The object string is `command/<Name>` and `query/<Name>`. The delimiter is a
**slash**, and that is load-bearing:

- Under the shipped `keyMatch` (`casbin/v3@v3.11.0/util/builtin_operators.go:85`)
  only `*` is special, so a slash is a literal and `command/*` is a prefix pattern.
- Under `keyMatch2` — plausible in a consumer model — `:[^/]+` is rewritten to
  `[^/]+` (`builtin_operators.go:34,142`) and the result is fed to a regexp. A row
  spelled `command:CreateItem` therefore stops being a literal and becomes
  `^command[^/]+$`: it would grant every object starting with `command` that
  contains no slash. A colon spelling is a silent wildcard, not a typo.
- Still under `keyMatch2`, `.` reaches the regexp engine as an any-character
  match, so `command.CreateItem` would match `commandXCreateItem`.

Both hazards are pinned as table rows in
`identity-model/authz_matcher_test.go`, so the spelling cannot be "simplified"
later without a red test.

### 2. Posture is declared at registration, and undeclared means gated

Because the dispatcher cannot be enumerated (above), the declaration surface is the
registration call. `RegisterCommand(disp, typ, posture, handler)` and
`RegisterQuery` wrap `command.RegisterTyped`/`query.RegisterTyped` and record the
posture in the App's registry — extending the `commandRegistry` pattern ADR 0056
already introduced for sync replay rather than adding a parallel one.

Three postures, matching the route vocabulary so the two cannot drift:

| Posture | Meaning at dispatch |
| --- | --- |
| `PosturePublic` | no identity required, engine never consulted |
| `PostureAuthenticated` | an identity is required; no object permission is asked |
| `PostureRoleGated` | the identity must satisfy `command/<Name>` (or `query/<Name>`) |

A registered type with **no** declaration is treated as `RoleGated` and fails
**construction** with an error listing every offender. Fail-closed is the only
default that cannot silently ship an open door; failing at construction rather
than at first request turns a permissions hole into a compile-fix loop of
seconds. This is a hard change only for an app that adopts the gate — the gate is
opt-in, so no consumer's upgrade breaks.

### 3. Capability is enforced at dispatch; invariants stay in the decider

The gate answers "may this actor run this command at all". It does not answer "may
this actor run it against *this* aggregate". Ownership and tenancy rules stay in
the decider, for two reasons that point the same way:

- **Replay does not re-run deciders.** If the decider were the only gate, an
  event-sourced rebuild would be the only thing standing between an attacker and
  history. Dispatch-time capability is the part that composes with replay.
- **The domain must not own policy rows.** A decider that consults Casbin has
  become the policy engine, and its rows then live in two places (guardrail 12).

### 4. Actions come from a closed vocabulary, and parameters do not flow into policy

Commands ask for `execute`, queries for `read` — the verbs `identity-model`
already defines. A consumer that needs a domain verb overlays it through the
`PolicyContext` seam rather than by widening the ask.

`PolicyContext` is a **closed interface** (`PolicyContext(ctx) Request`), not a
map of command fields. ADR 0058 defers attribute- and relationship-based
authorization; a free-form map would let the policy file's shape drift with the
payload schema and quietly become ABAC without the engine that makes ABAC sound.
A command that cannot describe itself as a `Request` gets no extra reach — it
simply keeps the derived `command/<Name>` question.

### 5. Audit mode is a migration tool with a one-wave expiry

`AuditOnly()` turns a denial into a structured log plus a metric and lets the
dispatch through. It exists because a fleet-wide default-deny flip in one commit
would red every consumer at once. It is bounded: a module may sit in audit mode
for at most one wave, and the maturity gate records that it is there, so it cannot
become a permanent posture.

### 6. The repo gains a fifth maturity level

`docs/PERMISSIONS.md` gains **L5 — dispatch-gated**, whose artifact is the command
posture manifest. L4 remains route-scoped and does not imply L5: a repo can gate
every route and still dispatch ungated, which is precisely the hole this ADR
names.

## Consequences

- **Positive.** Non-HTTP dispatch becomes authorizable for the first time.
  Forgotten postures become construction errors. The route vocabulary and the
  dispatch vocabulary are one vocabulary, so documentation and gates derive from
  one declaration rather than two.
- **Positive.** Denials reuse the ADR 0057 codes, so `IsForbidden`/
  `IsUnauthorized` and every existing classifier keep working, and logs from both
  layers are comparable.
- **Negative.** Adoption is a real edit: an app must switch its
  `Register`-family calls to the declaring helpers before the gate can be
  installed. Audit mode exists to make that edit incremental.
- **Negative.** The gate depends on a spellable object name, so it depends on the
  `keyMatch` change in `DefaultRBACModel`. A consumer who replaces the model with
  an exact-equality matcher loses prefix permissions and must write one row per
  command on the *policy* side; they keep working, they just pay more rows.
- **Neutral.** A second gate stacked above an already-authorizing layer (route or
  sync push) is harmless but must not double-count denials. The outer layer's
  denial is the one that reaches the client; the gate is kept outermost so retry
  and circuit-breaker middleware never see an authorization decision.

## Rejected alternatives

- **Ship it in go-cqrs-lite upstream.** Upstream has no authorization contract
  (`middleware/v4@v4.7.2` has no authz file) and root already owns `Request`/
  `Decision`. Building there means either duplicating the contract or moving it
  inward, which inverts ADR 0057's dependency rule. Instead: file an upstream
  *ask* offering the shape, and have this module delegate if upstream ever ships
  one — one implementation, no split brain.
- **Derive the question from free-form command fields.** Rejected under ADR 0058;
  see decision 4.
- **Put policy rows in Go source.** Rejected. Postures are code, policies are
  data. A middleware that hard-codes subject→object rows has become the policy
  engine.
- **Keep gating at the route only, and forbid service-level dispatch.** Rejected:
  seeds, jobs and CLIs are legitimate, and forbidding them is not a security
  control — it just makes the gate bypassable by writing a small `main`.
- **Flip every consumer to default-deny in one release.** Rejected: it would red
  the fleet in a single commit and teach people to reach for the escape hatch.
  Opt-in plus audit mode plus the L5 ratchet instead.

## Delivery slices

| Wave | Milestones | Outcome |
| --- | --- | --- |
| 1 — decide | hole proof, matcher proof, this ADR | the problem and the mechanism are stated and gated |
| 2 — enforce | `AuthorizeCommand`/`AuthorizeQuery`, declare-on-register, `keyMatch`, audit mode | a consumer can gate a dispatcher |
| 3 — prove | `authztest.CommandSuite`, gate rule 5, docs, skill/README, `usermgmt`+`setup` wiring | the guarantee is testable and documented where consumers look |
| 4 — harden | bench, policy-row linter, release posture | cost is known, orphan rows cannot accumulate |

## Upstream ask

Filed as a go-cqrs-lite issue: "Authorization middleware for command/query
dispatchers" — describes the `Middleware[H]` seam, this module's `Request`
derivation, and the declare-on-register constraint imposed by the dispatcher's
missing enumeration API. If upstream ships an equivalent, this ADR is superseded
and the middleware becomes a thin adapter.

**Status of the ask:** see the plan's execution receipt for the issue link; the
ADR stays authoritative for behaviour either way.
