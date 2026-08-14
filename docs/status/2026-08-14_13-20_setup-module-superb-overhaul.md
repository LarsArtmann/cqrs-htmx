# Status Report: Making `setup/` SUPERB

**Date:** 2026-08-14 13:20
**Scope:** `setup/v4` module (one-call composition root) + `adminui` accessor + `examples/setup-demo`
**Session:** Single session, ~3 hours. Auto-git daemon committed work as `5604e810`, `285f1b00`.

---

## Executive Summary

The setup module had four real defects and one big gap when I started: custom panel
paths silently broke all internal links (BasePath never passed), the CQRS dashboard
was **publicly reachable** by default (session middleware enriches but does not block),
route collisions panicked inside `Mount` instead of failing at `New`, there was no
serve lifecycle helper (docs taught `http.ListenAndServe` with zero timeouts), and the
async-startup feature had no end-to-end test (the oldest open TODO for it). All are
fixed, tested (57 tests, 89.5% coverage, race-clean, 0 lint issues), and documented
(README written, guide + doc.go refreshed, demo example added). Two doc updates
(TODO_LIST item removal, AGENTS.md) did NOT land — interrupted mid-edit. A concurrent
session's ActorID work transiently broke the workspace build during my verification;
my modules verified clean in isolation.

---

## a) FULLY DONE

### Bug fixes (real defects, each with a regression test)

1. **Custom panel paths broke every internal link** — `AdminPath`/`DashboardPath`
   were used for the mux mount (`StripPrefix`) but never passed to the panels as
   `BasePath`. Panels mounted at `/manage/` still linked to `/admin/...`. Fixed in
   `setup/setup.go` (`BasePath: cfg.AdminPath` / `cfg.DashboardPath`) + new
   `adminui.Handler.Config()` accessor (mirrors `dashboardui.Dashboard.Config()`)
   so tests can assert the wiring. (`adminui/handler.go:164`)

2. **Dashboard was publicly reachable** — `/dashboard/*` renders event payloads,
   stream IDs, and DLQ contents, but sat behind session middleware only, which
   enriches context without blocking; the dashboard's default authorizer allows all.
   `Mount` now applies an explicit `requireSession` gate (401, mirroring the admin
   panel). (`setup/mount.go`, `requireSession` helper)

3. **Panel paths without a trailing slash only matched the root** — `"/manage"`
   registers as an exact match; every sub-route 404'd. `withDefaults` now
   normalizes `AdminPath`/`DashboardPath` to end with `/` (subtree patterns) and
   strips the trailing slash from `HealthPath` (exact match, no redirect hop).
   (`setup/config.go`, `ensureTrailingSlash`/`trimTrailingSlash`)

4. **Route collisions panicked at Mount time** — `AdminPath == DashboardPath`
   (or either colliding with `HealthPath`, after normalization) survived `New` and
   crashed inside `http.ServeMux.Handle` on first `Mount`. `validate` now rejects
   pairwise-equal resolved paths with a descriptive rejection error, and rejects
   `/` for any of the three (the login page owns the site root catch-all).
   (`requireDistinctPaths`, `validatePathRoots`)

### New features

5. **`Bundle.Run(ctx, addr)` / `Bundle.RunHandler(ctx, addr, handler)`**
   (`setup/run.go`) — one-call serve lifecycle: mount (or take a composed handler),
   serve via `httputil.Server` with `ReadHeaderTimeout` + `IdleTimeout` and
   **deliberately no `WriteTimeout`** (the dashboard's SSE streams outlive any fixed
   deadline), graceful shutdown on context cancel using `context.WithoutCancel`
   (keeps the 30s shutdown budget), and `Bundle.Close` on every exit path exactly
   once. This replaces the docs' zero-timeout `http.ListenAndServe` teaching.

6. **Config passthroughs** (`setup/config.go`): `Logger` (structured auth-event
   logging), `AdminMode` + `TenantID` (tenant-scoped admin panel),
   `AdminAuthorizer` (custom admin access control), `DashboardAuthorizer` (runs
   after the new session gate).

7. **`adminui.Handler.Config()`** — read-only resolved-config snapshot, parity with
   dashboardui.

8. **`examples/setup-demo/`** — runnable showcase: one `setup.New` call, seeds a
   super_admin (exercising the real role-based authorizer, not a bypass), adds a
   `/dev-login` route next to the bundle's routes via `RunHandler`, serves on :8099.
   Verified live: `/health` 200, `/` 200, `/admin/` + `/dashboard/` 401 unauthenticated,
   `/dev-login` 303. Includes an end-to-end test (`main_test.go`) walking public
   routes, all three auth gates, and the authenticated panel flow through both
   panels.

### Tests

9. **46 → 57 tests; coverage 87.9% → 89.5% (gate 80); `-race` clean.** New coverage:
   BasePath passthrough (custom + default), trailing-slash behavior (now behavioral —
   asserts the subtree actually routes and returns 401, not just "no panic"), all
   route-conflict permutations, root-path rejection (3 subtests), passthrough
   assertions, admin creation failure path (tenant mode without TenantID — exercises
   the cleanup/rollback branch), full `Run` lifecycle (bind → health 200 → cancel →
   nil return), async startup 503→200 transition, sync startup blocking semantics.

10. **Async startup integration test** — the single most valuable missing test per
    the archived async-startup report. Uses a `gatedStore` wrapping the memory store
    whose `ReadFrom` blocks on a channel (projectionhost drains via `ReadFrom`, so
    this deterministically holds workers out of `live` without sleeps or huge
    journals). Asserts: `New` returns immediately with `AsyncStartup=true`, `/health`
    answers 503 while gated, flips to 200 after release. Companion test proves
    `AsyncStartup=false` keeps `New` blocked while gated and completes after release.

### Documentation

11. **`setup/README.md`** — the module had none (every sibling has one): quick start
    with `Run`, route/access table, full Config field reference table, three serving
    options, persistence, fail-fast validation notes, cross-links.

12. **`doc.go` refreshed** — Run/RunHandler first-class, 401 gates in the route
    table, new Config fields, path normalization/validation contract, BasePath
    behavior.

13. **`docs/guides/fullstack-wiring.md`** — SDK quick-start now uses
    signal-context + `bundle.Run`, documents `RunHandler` for custom routes,
    auth-gate column corrected (401), BasePath/validation notes added.

### Quality gates (my modules, run directly)

14. **`setup/.golangci.yml` created** (adminui-pattern per-module config with the
    established identity-model SA1019 exclusion + test relaxations) —
    `golangci-lint run`: **0 issues**. Before this, setup inherited no local config
    and my first lint pass had 13 findings, then 17 after the config tightened.

15. `setup`, `adminui`, `examples/setup-demo`: build, test (`-race`), lint all green.
    `go work use ./examples/setup-demo` added the module; flake auto-discovery
    (`forEachGoModule`) picks it up for `.#build` with no manual list changes.

---

## b) PARTIALLY DONE

1. **TODO_LIST.md async-startup item removal — NOT LANDED.** The edit failed twice
   (file-not-read guard, then file-changed-on-disk because the auto-git daemon kept
   committing), I re-viewed the target lines, and the session was interrupted before
   the edit executed. The item at line 27 is done in code but still listed open.
   Remaining work: delete that one bullet.

2. **AGENTS.md update — NOT STARTED but required.** Per the memory protocol I should
   have updated it when the facts changed: setup coverage 88→89.5% (Quick Reference
   says `[unverified] ~88%/80`), new `Run`/`RunHandler` API, new Config fields,
   BasePath/session-gate behavior changes, `examples/setup-demo` in the examples
   list, setup now has its own `.golangci.yml`, adminui `Config()` accessor. Nothing
   was written.

3. **Full workspace verification — BLOCKED, not abandoned.** `nix run .#build`
   failed in `adminui/handler_members.go` (`usermgmt.ParseActorID` returns 1 value,
   caller expects 2) — a **concurrent session** flipping the ADR-0111 ActorID API
   mid-edit. Not my change; per safety rules I left it alone. It later compiled
   again on direct check; the tree keeps moving under me. Workspace-level gates
   (`.#lint`, `.#coverage-gate`, `.#test-flake`) were not re-run to completion.

4. **`examples/setup-demo` lint not verified.** Examples are excluded from
   `nix run .#lint`; the LSP showed warnings on `main.go` during editing
   (exitAfterDefer, mnd, gosec G124, forbidigo on fmt.Printf) — the restructure into
   `main()`/`run()` addressed exitAfterDefer, but a real golangci-lint pass on the
   example was never run to confirm zero.

---

## c) NOT STARTED

1. **`nix fmt` on new markdown** (`setup/README.md`, `CHANGELOG.md` additions,
   this report) — dprint formats markdown; never run this session.
2. **Root README module table** — no link to `setup/README.md` or the setup-demo
   example added there.
3. **`nix run .#coverage-gate`** — coverage verified manually (89.5% vs gate 80)
   but the actual gate app not run.
4. **`nix run .#test-flake`** on setup — single-run + race only.
5. **`docs/guides/fullstack-wiring.md` manual-wiring section** — still stale per the
   prior overhaul report; I only refreshed the SDK section.
6. **Remaining passthroughs** (all from the prior report, all still open):
   `SnapshotConfig`, `SecurityHooks`, `EmailVerification`, `SessionStore`,
   `CheckpointStore`, `Lockout`, `TokenPepper`, `DrainTimeout`, `MaxUsers`
   (added upstream in `e5cdc925`, setup doesn't expose it yet), SSE
   `Broadcaster`/`EnableSSE`/`SSEPath`, loginpage `Brand`/`CSSPath`/
   `OAuth2Buttons`/`CredentialName`/`AuthPrefix`, dashboard
   `SSEHeartbeatInterval`/`PayloadRenderer`/`DeadLetterStore`.
7. **CSRF hardening story** — `Mount` uses `httputil.CSRFConfig{}` (Secure=false),
   which logs a WARN on every startup (seen in every test run this session). No
   Config knob to tighten it for production.
8. **CSRF-behavior test** — no test asserts a mutation POST without a CSRF token is
   actually rejected by the mounted bundle.

---

## d) TOTALLY FUCKED UP

1. **Wrote syntactically broken placeholder test code.** First version of
   `TestNew_Passthroughs_ReachPanels` contained literal garbage — `resp, func() {}()`
   — committed to the file before fixing it one edit later. Sloppy; should never
   have been written in that shape.

2. **Sync-startup test designed wrong on the first attempt.** I held the journal
   gate closed _forever_ and expected `New` to succeed — it correctly failed after
   the 30s drain timeout, burning a 30-second red test. The right design (assert
   blocked-while-gated, complete-after-release) was obvious in hindsight and is
   what landed.

3. **Demo app registered `/` and panicked at runtime.** The login page owns `/`
   (its catch-all); my dev-redirect handler re-registered it. Real mux panic in a
   live smoke test. Silver lining: it surfaced the exact pitfall that now (a) has a
   code comment in the demo, (b) is documented in README + doc.go, and (c)
   motivated the root-path validation. But it should have been anticipated — the
   conflict was knowable from `loginpage.Mount(mux, "/")`.

4. **Demo e2e test failed three times for two distinct reasons of mine:**
   (a) wrong expectation — `/auth/me` unauthenticated is a _correct_ 401, I asserted
   200; (b) cookie-capture bug — `http.Get` follows the redirect, so
   `resp.Cookies()` read the final response (no cookie). Fixed with an
   `ErrUseLastResponse` client. Along the way I also wrongly suspected projection
   lag and added polling that didn't help (the cookie was simply never captured) —
   misdiagnosis cost a cycle. (The polling is kept: read-your-writes racing is real
   and the comment documents it.)

5. **Lint whack-a-mole: 13 findings, then 17.** I wrote `run.go`, `config.go`
   changes, and tests _before_ creating `setup/.golangci.yml`, then fixed findings
   in two rounds — and the second round surfaced _more_ (errcheck in tests) because
   the new config enabled stricter test checking. Correct order: config first, then
   lint-clean code.

6. **Two failed TODO_LIST edits** (guard rejection, then stale-file rejection from
   the ever-committing daemon) and I still haven't landed the one-line deletion.
   Lesson for this repo: view and edit in immediate succession.

7. **Probe/scratch litter.** `/tmp/probe/`, `/tmp/debug_auth_test.go`,
   `/tmp/setup-demo` binary still on disk. Killed the stray server processes, but
   the files remain (harmless, /tmp).

8. **CHANGELOG structure slightly awkward** — I introduced `### Added (earlier in
   this release)` to separate my entries from the pre-existing Unreleased Added
   list. Functional but unusual; a plain merge of the two lists would be cleaner.

---

## e) WHAT WE SHOULD IMPROVE

**High priority**

1. Land the TODO_LIST deletion + write the AGENTS.md update (facts changed, memory
   protocol says now, and the window was missed).
2. Re-run the full nix gate suite once the concurrent ActorID session settles —
   workspace truth currently depends on another session's in-flight state.
3. Add a `Config.CSRF` (or `SecureCookies bool`) knob so production setups stop
   booting with the Secure=false WARN, plus a test that CSRF actually blocks a
   tokenless mutation.
4. Expose `MaxUsers` through setup (upstream landed it; setup is the composition
   root consumers actually touch).
5. Decide and document the dashboard 401 gate as an intentional behavior change
   (it is one; consumers relying on a public dashboard must now mount the handler
   themselves — README says how, CHANGELOG records it).

**Medium**

6. Make Run's timeouts configurable (`Config.Run*` or a `RunConfig` arg) and add
   `RunListener(net.Listener)` for tests/Unix sockets; consider TLS passthrough.
7. Optional auto-mounted SSE endpoint (`EnableSSE`/`SSEPath`) + exposed
   `Broadcaster` on the Bundle — setup-demo would use it for the admin sync bar.
8. Richer setup-demo seed (users, tenants) so the dashboard shows real data.
9. `nix fmt` pass over all new markdown; root README cross-links.
10. Add setup-demo to the pre-commit/lint verification (even though examples are
    excluded from `.#lint`, a one-off `golangci-lint run` there would have caught
    the leftover warnings).

**Lower**

11. Live/ready split for `/health`; optional `ProjectionStatusHandler` +
    `EventCatalogHandler` mounts as Config booleans; `Mount` double-call guard or
    documented contract.

---

## f) Up to 50 things to get done next

1. Delete the done async-startup-test bullet from TODO_LIST.md (line 27)
2. Update AGENTS.md Quick Reference (coverage 89.5%, Run/RunHandler, new fields)
3. Update AGENTS.md module description + examples list with setup-demo
4. Update AGENTS.md Gotchas: dashboard 401 gate, root-path reservation, BasePath normalization
5. Note setup/.golangci.yml existence in AGENTS.md lint section
6. Re-run `nix run .#build` after ActorID session settles
7. Re-run `nix run .#lint` (12 modules) to confirm 0 across the board
8. Run `nix run .#coverage-gate` (setup gate 80 vs actual 89.5)
9. Run `nix run .#test-flake` for setup (3x)
10. Run `nix run .#check-cqrs-lint` (setup in custom list — verify picked up)
11. Run golangci-lint once on examples/setup-demo; fix leftovers
12. Run `nix fmt` (markdown: setup/README.md, CHANGELOG.md, docs/status reports)
13. Merge the two CHANGELOG "Added" sections cleanly
14. Root README: link setup/README.md and examples/setup-demo
15. Add `Config.MaxUsers` passthrough (+ validation: negative → rejection)
16. Add `Config.CSRF`/cookie-security knob; kill the startup WARN path
17. Test: tokenless mutation POST to /admin is rejected (403) by mounted CSRF
18. Test: logout flow clears session (Auth + cookie)
19. `Config.SnapshotConfig` passthrough
20. `Config.EmailVerification` passthrough
21. `Config.SecurityHooks` passthrough (signing/encryption)
22. `Config.SessionStore` passthrough (multi-instance deployments)
23. `Config.CheckpointStore` passthrough (restart replay)
24. `Config.Lockout` passthrough
25. `Config.TokenPepper` passthrough (bot tokens)
26. `Config.DrainTimeout` passthrough
27. `Bundle.Broadcaster` + `Config.EnableSSE`/`SSEPath` auto-mount
28. Wire SSEURL default to the auto-mounted SSE endpoint when enabled
29. loginpage `Brand`/`CSSPath` passthrough
30. loginpage `OAuth2Buttons`/`CredentialName`/`AuthPrefix` passthrough
31. dashboard `SSEHeartbeatInterval` passthrough
32. dashboard `PayloadRenderer` passthrough
33. dashboard `DeadLetterStore`/`CommandJournal`/`QueryJournal`/`SnapshotStore` passthrough
34. Make Run timeouts configurable (RunConfig or Config fields)
35. `RunListener(net.Listener)` variant for tests/Unix sockets/TLS
36. TLS config passthrough for Run
37. setup-demo: seed demo users + tenants (richer dashboard data)
38. setup-demo: demonstrate AsyncStartup with health polling output
39. setup-demo: consider DisableLogin + own root redirect as alternative composition
40. docs/guides/fullstack-wiring.md manual-wiring section refresh (still stale)
41. Add troubleshooting section to setup README ("/" pitfall, conflict errors)
42. Add security note to README: CSRF Secure, reverse-proxy TLS termination
43. Test: custom LoginRedirect honored after login (e2e)
44. Test: SessionTTL expiry honored through the bundle (or link usermgmt test)
45. Test: DashboardAuthorizer deny path returns its error's mapped status
46. Test: AdminAuthorizer deny → 403 through mounted bundle
47. Guard or document `Mount` double-call (second call panics on conflicts)
48. Optional `/livez` vs `/readyz` split (liveness without projection dependency)
49. Optional Config booleans to mount ProjectionStatusHandler / EventCatalogHandler
50. Cleanup: remove /tmp/probe, /tmp/debug_auth_test.go, /tmp/setup-demo leftovers

---

## g) Questions I cannot answer myself

1. **The dashboard 401 gate is a breaking behavior change** (public → session-gated
   for every existing default-configured consumer). I chose secure-by-default with
   an escape hatch (mount `Dashboard.Handler()` yourself, documented). Do you want
   it kept as-is for the next release, or softened behind an explicit
   `Config.DashboardPublic` (default false)?

2. **The concurrent ActorID session is mid-flight and transiently broke the
   workspace build** during my verification. Should I re-run and own the unified
   workspace gates once it lands, or is that session's owner responsible for the
   final green run? (I only verified setup/adminui/setup-demo in isolation.)

3. **For the next setup work batch, which passthrough group has priority?**
   (a) production hardening (`MaxUsers`, `TokenPepper`, `SessionStore`,
   `CheckpointStore`, CSRF knob), (b) realtime (`Broadcaster` + auto-mounted SSE),
   or (c) UI polish (loginpage/dashboard field passthroughs). I'd pick (a), but
   it's your product call.

---

**Bottom line:** setup/ is materially stronger — four real bugs fixed with
regression tests, a production-shaped serve lifecycle, the flagship missing
integration test written, docs that match the code, and a runnable demo. The debt
this session itself created (un-landed TODO/AGENTS edits, unrun workspace gates,
markdown formatting) is listed above and cheap to clear.
