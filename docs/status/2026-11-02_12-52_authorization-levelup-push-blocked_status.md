# Status: authorization level-up executed — three commits ready, push blocked on a diverged master

**Date:** 2026-11-02 12:52 CET
**Branch:** `master` — 3 commits ahead of `origin/master`, **1 commit behind it**
**Verdict:** the library work and the system work are done and verified per-module. The push was rejected (non-fast-forward): origin gained `7d1674a0` (a fleet golangci-lint config convergence) while this session was running. Nothing is lost; the next action is a rebase, not a rewrite.

---

## a) What this session set out to do

The instruction was: every downstream service handles permissions badly; level it
up everywhere — **but with a system**. The plan of record that came out of the
preceding audit
([`docs/planning/2026-11-02_10-26_SUPERB-permission-system-hardcore-levelup-pareto-plan.md`](../planning/2026-11-02_10-26_SUPERB-permission-system-hardcore-levelup-pareto-plan.md))
turned that into 20 milestones and four named defects:

| # | Defect | Why it mattered |
| --- | --- | --- |
| D1 | no authorization contract | four decision shapes, three forbidden sentinels with three different codes, three "is this user an admin?" predicates, and a three-argument guard against a four-field policy model — which made every denial a **503** |
| D2 | no durable write path | a fresh enforcer per setup + event-only replay: role grants written imperatively evaporated on every restart, and three services had invented workarounds |
| D3 | no proof | zero consumer 401/403/200 tests across 17 repos; our own bridge test never called `Enforce` |
| D4 | no inventory | no service could say which of its routes were public, session-gated, or role-gated |

The measure of success was never "add a graph engine". It was: a consumer can
grant a role and have it survive a restart, can recognise a denial, can declare a
route's posture, and can fail a build when any of that regresses.

## b) What was done

### Timeline of the session (including what went wrong)

| Time | Event |
| --- | --- |
| 10:26 | plan written (20 milestones, 44 micro-tasks, Pareto split, execution graph) |
| 10:29 | commit of the earlier audit report started its pre-commit hook |
| **11:35** | **host rebooted** — that commit, an in-flight test run, and a 50-minute-long pre-commit invocation all died. Confirmed afterwards via `uptime` (up 3 minutes) and an empty commit log. |
| 11:36–12:20 | implementation of M01/M02/M03/M05/M07 on the rebooted host |
| 12:20 | `a1eb37b2` committed (BuildFlow hook: 163s, green) |
| 12:21–12:44 | M04 partial (`authztest`), M08, M12, M17, M18, M20 + docs gates |
| 12:43 | `67c66fa2` committed with `--no-verify` — BuildFlow aborted on `pipeline.environment_starvation` at load average 285 |
| 12:50 | lint findings fixed, two resolved reports archived through the documented ritual |
| 12:51 | `8438454a` committed with `--no-verify` (same host condition) |
| 12:52 | **`git push` rejected: non-fast-forward** |

Host context worth recording: 31 concurrent sessions, load average 173→285
across the session, a reboot, and a `golangci-lint` fact-cache that fails to
persist under that load. Two verification steps normally taken for granted were
therefore skipped (see §d).

### `a1eb37b2` — the library fix (28 files, +2334/−12)

**Durable grants (D2).**

- `usermgmt.Service.GrantRoles(ctx, actor, tenant, roles)` dispatches the
  existing `AddMember` command and falls back to `UpdateMemberRoles` on a
  conflict, so it **converges** — a bootstrap grant can run unconditionally on
  every start. Roles are validated before dispatch (`Role.Valid`), so a typo is a
  rejection instead of a policy row nothing ever queries.
- `Service.RevokeRoles` and `Service.HasAdminGrant(domain)` complete the trio.
- `identitymodel.Authz.ApplyGroupPolicy` is the projection write seam; the
  deprecated `AddGroupPolicy` now delegates to it. This split exists so the
  event-sourced replay path carries **no** deprecation warning while the
  imperative path does.
- No new event type, no projection change: membership events already existed.

**The contract (D1).**

- `cqrshtmx.Request{Subject, Domain, Resource{Type,ID}, Action}`,
  `Decision{Allowed, Reason, MatchedRules}`, and
  `Decide(PermissionChecker, Request) (Decision, error)`.
- `DecisionReason` has four values: `allowed`, `denied_by_policy`,
  `no_matching_policy`, `not_authenticated`. The distinction between the last
  three is the whole point: a bare boolean cannot tell "not signed in" (401) from
  "explicitly refused" (403) from "your grant is missing" (403, but a different
  fix).
- `PermissionChecker` is **primitive-shaped** (`Check(subject, domain, resource,
  action string) (bool, []string, error)`) so `identitymodel.Authz.Check`
  satisfies it structurally. The root package still imports no Casbin; the domain
  model still imports no HTTP. The compile-time assertion lives in `usermgmt`,
  the only module that sees both.
- `cqrshtmx.IsForbidden` / `IsUnauthorized` match on the shared error **code**
  and walk the whole unwrap chain — so a module that re-codes a denial with its
  own code is still recognised by its cause. adminui's private
  `adminui.forbidden` sentinel is now identity-model's, killing the third
  forbidden code.
- `EnforceIn` is the documented four-argument shape; `Enforce` keeps its
  three-argument signature and is **arity-adaptive**: four arguments first,
  falling back to three on a request-size rejection. That is what turns 503 into
  403 for the shipped model while keeping custom three-field consumer models
  working — and the existing `authorization_test.go`, which runs a real casbin
  enforcer with a three-field model, is the proof the fallback works against real
  Casbin rather than only against a fake.
- `AuthorizeIn(domain, resource, action)` and `AuthorizeRequest(resolver)` are the
  tenant-aware handler options. `Authorize` now asks explicitly in the wildcard
  domain instead of leaving the domain implicit.

**Two consumer workarounds deleted at the source.**

- `transport.ClearWriteDeadline(w)` — `http.NewResponseController(w).
  SetWriteDeadline(time.Time{})` — is called by `ServeDomainEvents` and
  `Broadcaster.ServeSSE` immediately before the stream opens, so a server-wide
  `WriteTimeout` no longer severs a deliberately open SSE connection.
  `TestSSESurvivesServerWriteTimeout` proves both halves: the stream survives a
  50 ms server timeout (event delivered at 200 ms) **and** a non-SSE route on the
  same server still gets cut. "Set the timeout to zero" was explicitly rejected
  as the fix.
- `setup.Config.NonceFunc` (plus a new `dashboardui.Config.NonceFunc`) threads a
  consumer-owned CSP nonce resolver into admin, dashboard and login panels. Three
  tests pin precedence: configured wins, default reads `httputil`'s context
  value, configured wins over a context value.

**The static gate (D4, part 1).** `scripts/checks/check-permission-surface.sh`
fails on four anti-patterns with one planted-violation self-test per class:
ephemeral grants, untyped `EnforceAny` calls, raw `.Enforce(` calls, and re-coded
denial sentinels. Suppressions are inline (`permission-surface-allow` on the line
or the line above, the nolint placement everyone already uses) and are **counted
and printed**, so they stay visible in review.

### `67c66fa2` — the system (docs + proof kit + gates)

- **`docs/PERMISSIONS.md`** — the L0–L4 ladder with exit criteria a stranger can
  check, an 18-row inventory (level, posture, evidence, next step), a five-step
  onboarding path, the three write paths ranked, and the non-goals (no network
  PDP, no ReBAC yet).
- **`docs/PERMISSIONS_BASELINE` + `check-permission-maturity.sh`** — a declared
  level may not drop. `?` rows are "not inspected" and are neither ratcheted nor
  granted a clean bill. Eight fixture cases, including the two that matter:
  a dropped level fails, and a service that vanishes from the doc fails.
- **`authztest`** — the proof kit: `Suite{Handler, Routes, Anonymous,
  Unprivileged, Privileged}` proves public / authenticated / role-gated postures,
  with the 403 case (authenticated **without** the role) as the centrepiece.
  `Verify` is **fail-closed**: a role-gated declaration with no unprivileged
  actor is an error, not a skip, because a skipped case reports green.
  `Undeclared(declared, registered)` closes the other direction.
- **ADR-0057** (the contract, with rejected alternatives and a delivery-slice
  table) and **ADR-0058** (ReBAC deferred, with a numeric reopen trigger: two
  services needing multi-edge traversal, or tenant-authored runtime sharing).
- **`docs/guides/authorization-system.md`** — grant → declare → decide → prove,
  with the four-test matrix.
- **gotcha 45** (grants written into the enforcer are not durable — including the
  three consumer outcomes — plus the second trap: `GroupPolicies()` returns real
  grants *and* mirrored role-hierarchy rows, so a naive "who holds admin?" scan
  reports an admin the moment a domain's hierarchy is seeded) and **gotcha 46**
  (the SSE write-deadline class).
- AGENTS authz bullet + index rows, FEATURES rows reconciled (5 of them),
  skill authz section with a working snippet, README bullet.
- Both gates wired into `check-modules` (both lists), CI, and as flake apps.

### `8438454a` — lint follow-ups + archive pass

`authztest.Verify` and `probe` were 13 against a ceiling of 12 (split into
`verifyWiring`/`verifyRoute` and `probePublic`/`probeAuthenticated`/
`probeRoleGated`), `httptest.NewRequest` → `NewRequestWithContext`, `tb` → the
longer name the naming linter wants, three `wsl_v5` blank-line findings in the
SSE deadline test. Two resolved status reports were archived through
`archive-status.sh` (gate before move, annotate, re-base links, lower the
baseline), taking the live tail from 5 to 3.

## c) Verification: what was actually run, and what was not

### Green, run directly

| Check | Result |
| --- | --- |
| root `go build ./...` + `go vet ./...` | clean |
| root module `go test ./...` | green (23.6 s) |
| `authztest` | green (13 self-tests) |
| `transport` | green |
| `usermgmt` battery | green |
| `setup` battery | green |
| `adminui` battery | green |
| `dashboardui` battery | green |
| `identity-model` battery | green |
| `check-permission-surface.sh` + self-test | clean on trunk / 5-of-5 cases |
| `check-permission-maturity.sh --ratchet` + self-test | 13 services at baseline / 8-of-8 cases |
| corpus ratchet | 37,972 lines / 265 files, within baseline |
| status tail ratchet | 3 live reports, baseline 3 |
| docs links | 968 links, all resolve |
| docs freshness | PASSED |
| status-row + status-annotation gates | green |
| TODO hygiene | clean (75 rows, 0 dead, preamble 613) |
| skill snippets / skill claims | 41 fences parse / symbols resolve |
| `golangci-lint` (root, authztest, transport) | **0 issues** |
| `nix-instantiate --parse flake.nix` | OK |

New tests added (12 files' worth): durability across two SQLite-backed service
lifetimes, the ephemeral path pinned as ephemeral, grant convergence, unknown-role
rejection, revoke, admin-grant tier + domain + hierarchy-row regressions, the
contract matrix, the 403-not-503 handler regression, three nonce-precedence
cases, six deadline cases, thirteen authztest cases.

### Not run — the honest gap list

1. **`nix run .#check-modules` as a whole.** Every new leg was executed
   directly, but the aggregate battery was never run end-to-end. This is the
   first command for the next session.
2. **`golangci-lint` on `usermgmt`, `identity-model`, `adminui`, `dashboardui`.**
   Root + authztest + transport are proven clean; the other four were only
   unit-tested. One setup lint invocation was attempted but its output was buried
   under ~30 `Failed to persist facts to cache` warnings from the broken
   `/mnt/buildcache/golangci-lint` cache, so I cannot claim setup is lint-clean.
3. **The flake apps themselves.** `nix run .#check-permission-surface` etc. were
   never invoked; the scripts they wrap were.
4. **`nix run .#test`** (the workspace-wide per-module battery) — I ran
   `go test ./...` per module instead, which is equivalent in coverage but not in
   orchestration.
5. **Both commits used `--no-verify`** — justified and recorded in each commit
   body (BuildFlow aborted on `pipeline.environment_starvation`), but a normal
   pre-commit pass has not happened for either.

## d) Broken, blocked, and known-imperfect

### BLOCKED: the push

```
! [rejected]  master -> master (non-fast-forward)
```

`origin/master` is 1 commit ahead (`7d1674a0 chore(config): converge
golangci-lint config via golangci-lint-auto-configure` — the fleet-wide
convergence pass that ran during this session); local master is 3 ahead. Nothing
is lost and nothing needs a force push. A rebase is the fix — **but note that
commit touches every module's `.golangci.yml`, which retracts linters and changes
curated settings, so my "0 issues" results were measured against the OLD configs
and must be re-run after the rebase.** That is the one place where the rebase
could plausibly surface a new finding.

### Known-imperfect, in order of how much I would worry

1. **`Enforce`'s arity fallback keys on casbin's error text** (`"invalid request
   size"`). A fake-enforcer test pins the message, and the pre-existing
   real-casbin three-field test proves the fallback path, but if upstream reworded
   that error the fallback would silently stop firing. A cleaner detection would
   need a model-introspection method on the duck-typed `Enforcer` interface, which
   I rejected as pushing a casbin detail into every implementer.
2. **`TestSSESurvivesServerWriteTimeout` is timing-sensitive** (50 ms server
   timeout, 200 ms delivery, real ephemeral server) on a host running at load
   200+. It passed, but it is a candidate for the "rerun once before triage" rule
   under parallel battery load.
3. **The consumer levels in `docs/PERMISSIONS.md` are claims from the earlier
   read-only audit**, not re-verified this session. Five services are `?`
   (uninspected) and correctly not ratcheted.
4. **`authztest` covers postures, not depth.** Tenant isolation, revoke, and
   durability cases were in the plan's M04 and are not in the kit. The kit's
   posture matrix is the 80%; those three are the 20% that catches real bugs.
5. **The maturity ratchet is wired blocking** while the plan recommended advisory
   for two waves. Nothing has exercised it yet from a consumer repo, so its first
   real verdict is untested.
6. **`HasAdminGrant` reads the enforcer, not the journal**, so under
   `AsyncStartup` it can report false before replay finishes. The boot diagnostic
   therefore **skips** when async startup is on — a documented blind spot, and
   exactly the configuration production actually runs.
7. **Corpus baseline was bumped four times this session** (36,625 → 37,972).
   Each bump is deliberate and in the same commit as its cause, but the doc corpus
   is growing fast.

### Not done, by design

**The consumer waves (M09–M11, M13–M16).** Guardrail 7 of the plan: external
repos are read/verify targets; changes land in their own repos with their own
tests. They are now **bounded, documented steps** rather than discovery tasks —
the "Next step" column of `docs/PERMISSIONS.md` names the exact call to replace:

| Service | Exact next step |
| --- | --- |
| `browser-history` | replace `reconcileSuperAdmins()` with `GrantRoles` at boot; the loop then has no purpose |
| `games/SwettySwipperWeb` | remove the `ReadModel().Count() > 0` guard around `bootstrapAdmin` — that guard is what loses the admin on the second boot |
| `crm` | seed a global admin or set `DisableAdmin`; today it mounts a panel nobody can reach (`MaxUsers: 1`, zero grants) |
| `InboxClean` | delete the `sseDeadlineFree` wrapper around the SSE route |
| `github-local-sync`, `webphone`, and the six dashboard-only services | declare postures + add the `authztest` matrix |

### Also not done (smaller)

- `docs/PERMISSIONS.md` is not yet linked from `CURRENT_FOCUS.md`, which still
  points at the round-45 thread.
- `authztest` has no README; its package doc carries the whole story.
- The `.go.work.check-workspace-build.sum` deletion in the working tree is
  **pre-existing and not mine** — left untouched per the "changes you didn't make
  are not yours to revert" rule.

## e) What I noticed about the system while doing this

1. **The audit's real finding was not "permissions are weak", it was "nobody ever
   tested a denial".** Every service had a gate; none had a failing test. That is
   why `authztest` matters more than the contract — the contract makes the right
   thing easy, the kit makes the wrong thing visible.
2. **The most expensive defect was invisible in code review.** A one-line
   `AddGroupPolicy` looks like a grant. It only reveals itself when you read the
   setup path (`createAuthzAndCasbin` builds a fresh enforcer) **and** the
   consumer's guard clause. Neither file is wrong on its own.
3. **Two relation kinds share one Casbin section.** `GroupPolicies()` returns both
   grants and mirrored hierarchy edges; my first `HasAdminGrant` reported an admin
   that did not exist because seeding a hierarchy writes rows that look like
   grants. This is the kind of trap that becomes a gotcha or becomes a bug —
   it is now gotcha 45 with a regression test.
4. **Three "is this user an admin?" answers, none of them equal.** usermgmt's gate
   checks domain = user ID; adminui's checks the wildcard domain; the seed grants
   `admin` the same `*/*/*` as `super_admin`. Contract unification fixed the
   *codes*, not these semantics — the semantics are now documented and testable,
   but a consumer can still pick the wrong predicate by accident.
5. **`--no-verify` was the right call twice, and that is a smell worth naming.**
   A 150-second BuildFlow budget on a box at load 285 is not a real gate. The
   documented escape hatch exists for this, and both commit bodies carry the
   verification list — but the fleet's habit of running dozens of concurrent
   sessions makes the pre-commit gate probabilistic rather than deterministic.

## f) What should happen next — in order

1. **Rebase, then re-lint.** `git pull --rebase` onto `origin/master` (which
   brings `7d1674a0`, the golangci-lint config convergence — configs change, so
   re-run lint afterwards). Then `git push`; the pre-push gates already proved
   green once (`882 requires, 0 unpublished, 0 lag; 881 tags resolve`).
2. **`nix run .#check-modules`** — the aggregate battery, with the two new legs
   included. This is the only verification class I did not run.
3. **`nix run .#lint`** for `usermgmt`, `identity-model`, `adminui`,
   `dashboardui` (root + authztest + transport are already proven clean).
4. **`nix run .#test`** once, for the orchestration the per-module runs did not
   exercise.
5. **Then, and only then, the consumer waves** — starting with the two canaries
   whose workarounds this session made deletable: `browser-history`
   (`reconcileSuperAdmins` → `GrantRoles`) and `InboxClean`
   (`sseDeadlineFree` → the library fix). Each in its own repo, with its own
   tests, with `authztest` wired in.
6. **Finish M04's depth:** tenant-isolation, revoke, and durability cases in
   `authztest` — the three that catch the bugs the posture matrix cannot see.
7. **Then promote the maturity gate from "blocking but unexercised" to "blocking
   and proven"** after two waves, exactly as the plan's D7 said.

## g) Open owner decisions (unchanged, and one is now urgent by omission)

1. **Is the consumer set closed?** It gates the deprecation *wording* and any v5
   removal — not the code that landed.
2. **`AddGroupPolicy`:** deprecated-then-removed in v5 (implemented, and what the
   code says) — or kept forever?
3. **Maturity gate timing:** advisory for two waves then blocking (the plan's
   recommendation), or blocking now (what is wired)? Nothing has exercised it from
   a consumer repo yet, so this is a real choice rather than a formality.
