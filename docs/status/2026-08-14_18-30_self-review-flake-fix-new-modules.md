# Session Status: Brutal Self-Review + Full Report — 2026-08-14 18:30

> Session started ~15:30 from `docs/status/2026-08-14_15-22_hermetic-repair-complete-gates-green.md` backlog.
> User instruction: keep executing autonomously, one verified step at a time.
> Interim report written at 16:33: `2026-08-14_16-33_flake-rootcause-health-auditlog-modules.md` (its "fuzz/flake running"
> note is now resolved — both PASS; the usermgmt replace-strip attempt below happened after it was written).
> Tagging/releases untouched — the 3 open questions from 15:22 §g remain, restated in g) below.

---

## a) FULLY DONE (all verified this session)

### 1. The "unreproducible" dashboardui flake: ROOT-CAUSED as a real data race, fixed

- Ran `-race -count=10` on dashboardui → failed in `TestDashboard_SSEHeartbeatEmission` (NOT the test the prior session
  suspected). Full race report: `dashboardui/sse.go:115` spawned `go stream.Heartbeat(stream.Context(), interval)` and
  the handler returned **without joining the goroutine** — heartbeat could write to the ResponseWriter after handler
  return (net/http violation; races `stream.Close()` on real connections).
- Fix: derived cancellable context (`r.Context()`, satisfies contextcheck) + join via `defer` before any return path.
  Verified 50× single-test + 10× full package under `-race`; before the fix it failed within 10 runs.
- The prior session's nil-deref hypothesis was ALSO real: `TestDashboard_Close` now checks `New`'s error, asserts double
  `Close()`, and `Dashboard.Close()` gained a nil-receiver guard (documented no-op; matches "safe to call" contract).

### 2. New module: `health/v4` — go-health + go-health-dashboard bridge (100% coverage)

- `NewProbe(provider, opts...)`: `gohealth.Probe` with one named check per projection; semantics mirror
  `ProjectionReadinessCheck` (live/stopped pass, drain transient, failed infrastructure + LastError).
- `Recorder(provider)`: `HealthRecorder` MERGING projection checks with a samber/do injector's own service checks
  (for `gohealth.New(injector, gohealth.WithHealthRecorder(health.Recorder(svc)))`).
- `NewDashboard(probe, opts...)`: go-health-dashboard UI (HTML/SSE/JSON via Accept header); test asserts the projection
  check renders through the probe's refresh cache.
- **Born hermetic**: root v4.7.0 (needed types are in the published tag) + go-health v0.0.2 + go-health-dashboard
  v0.3.0 from the proxy — zero local replaces, zero replace-pile growth.
- 100.0% coverage (gate 90), 0 golangci issues, cqrs-lint clean. README + `.golangci.yml` + `.cqrs-lint.json`.
  Wired into go.work, `.#lint`, `.#coverage-gate`, `.#check-cqrs-lint`, dep budgets (5 deps / budget 6), CI
  (build/test/coverage/lint/mod-tidy).

### 3. New module: `auditlog/v4` — samber-do-auditlog bridge (100% coverage)

- `WithAuditLog(auditCfg, viewerCfg) (*Setup, error)`: `Setup.Opts` (`*do.InjectorOpts` for `do.NewWithOpts`),
  `Setup.Plugin` (reports/exports), `Setup.Viewer` (`*live.Server`, an `http.Handler` with SSE updates).
- Upstream reality differed from the TODO sketch (`Opts() *InjectorOpts` exists; no `MountReport`; `live.Server`
  implements `ServeHTTP`) — API designed around what exists, README documents the mount.
- Tests: container wiring records invocations, viewer serves dashboard + report JSON, invalid config → Orchestration
  errorfamily. 100.0% coverage (gate 90). Same full wiring as health. Resolves v0.9.2 tag only.

### 4. Fullstack UI integration test expansion (integration_test)

- Harness: `setupFullstackUI` now also returns the `*usermgmt.Service`.
- `TestFullstackUI_AdminRendersSeededUser`: register → super_admin via real authz path → `GET /admin/users` with session
  cookie (bounded poll) → seeded email + name in HTML.
- `TestFullstackUI_DashboardShowsProjectionHealth`: `/dashboard/projections` renders real projection names
  (user-read-model / casbin-projection / tenant-read-model) from the Service's projection host.
- `TestFullstackUI_LoginButtonsMatchAuthConfig` (3 subtests): no providers → setup hint; WebAuthn → passkey button;
  OAuth2 → provider button. **Spec correction:** the original TODO wanted a TOTP-button assertion, but loginpage has NO
  TOTP UI by design (second-factor API flow) — documented in the test.

### 5. systemadapter gate integration completed (closed the `[~]` TODO item)

- `check-module-isolation.sh`: hardcoded module list → **go.work auto-discovery** (root cause of systemadapter being
  missing; the sentinel can no longer drift when modules are added).
- `check-dep-budgets.sh`: + systemadapter (13 deps, budget 16).
- **Pre-existing bug found in BOTH scripts**: `head -1 go.mod | awk '{print $2}'` parsed `//cqrs-lint:ignore` comment
  words ("adapter", "dashboard") as module names → `grep -m1 '^module '` now. Verified both run green.
- The "hermetic drift sentinel flake app" backlog item is thereby satisfied by `nix run .#check-modules`
  (isolation + budgets pass; its `check-version-drift --strict` leg still fails on known cross-tag drift — documented
  P3, blocked on the next release, untouched).

### 6. Small backlog items

- `docs/guides/mysql-setup.md`: See Also links ADR-0041 for `NewMySQLSnapshotStore` (item 18).
- `README.md`: AsyncStartup feature bullet (item 20).
- devShell cspell/vitest/jest item: verified STALE (codespell already in devShell; BuildFlow skips jest/vitest — no JS
  tests) → moved to ROADMAP "Not Planned" with reasoning (item 15).
- `systemadapter/go.mod`: `record/v4` + `go-error-family` promoted to direct (missed tidy after OnRecordTyped).

### 7. Docs

- CHANGELOG `[Unreleased]`: +4 Added (health, auditlog, fullstack tests, script auto-discovery), +4 Fixed (heartbeat
  race, nil-deref, go.mod comment parsing, systemadapter tidy), +1 Changed (nil-safe Close).
- TODO_LIST: 4 items removed as done; header = 26 modules / 15 lint-checked / 15 coverage gates; CI item rewritten
  (check-codegen was already wired by a concurrent session; check-templates + check-cqrs-lint documented as blocked).
- AGENTS.md: 26 modules, health + auditlog bullets, dependency direction, gate numbers (17 suites, 15 gates).
- `nix fmt`: 0 changed (formatting verified clean).

### 8. Verification (final state, all green 2026-08-14)

| Gate                                                                                   | Result                                                                       |
| -------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `.#build`                                                                              | 26/26 modules hermetic                                                       |
| `.#test`                                                                               | 17/17 suites ok (re-ran sequentially after the race in d).1)                 |
| `.#coverage-gate`                                                                      | 15/15 gates (health 100%/90, auditlog 100%/90; usermgmt drifted up to 81.5%) |
| `.#lint`                                                                               | 15 modules × 0 issues                                                        |
| `.#check-cqrs-lint`                                                                    | all module configs pass strict                                               |
| `.#check-codegen` / `.#check-templates` / `check-docs-links` / dep-budgets / isolation | pass                                                                         |
| `.#test-fuzz`                                                                          | PASS (~3.7M execs)                                                           |
| `.#test-flake`                                                                         | 51 suite-runs ok (3×17), zero FAIL lines                                     |
| `nix flake check --no-build` / `nix fmt`                                               | pass / 0 changed                                                             |

---

## b) PARTIALLY DONE

- **Replace-pile reduction (P1 #1)**: attempted the FIRST strippable replace — `integration_test`'s usermgmt replace —
  because `usermgmt/v4.7.2` is now tagged and contains `ProjectionHost()` with a clean go.mod (identity-model v4.2.0).
  Strip + hermetic `go build` PASSED, but `go vet` failed: the published tag's `id.go` calls the 2-value `ParseActorID`
  from UNRELEASED identity-model v4.7.0 while the local identity-model replace compiles the 1-value signature.
  **Restored immediately** (hermetic build+vet+test re-verified green); the replace comment now records the exact
  interop failure so the next attempt knows the real condition: _tag exists AND interoperates with the replace graph_.
  Net result: pile still 8 entries, but with one entry's removal condition corrected and a proven verification recipe.

## c) NOT STARTED (this session, from the backlog)

- Tagging identity-model v4.7.0 / usermgmt v4.7.3 / root v4.7.1+ and stripping the 8 replaces (needs user answers — g).
- adminui → direct identity-model imports (~26 files, 133 SA1019); integration_test → same (22 SA1019).
- CI wiring for check-templates (blocked: go.work absolute local replaces to `/home/lars/projects/...` don't exist on
  runners) and check-cqrs-lint (blocked: Nix-only binary).
- Remaining P3s: golines in `nix fmt`, Go markdown link checker, v4-branch blob rewrite, datastar/go-sse ADR,
  mysql_integration_test with testcontainers, `slowJournal` testutil extraction, cross-tag dep drift cleanup.

---

## d) TOTALLY FUCKED UP / REGRETS (honest ledger)

1. **I ran two nix gate sweeps in parallel and hit the documented GOCACHE race myself.** `.#test` showed auditlog
   `[setup failed]` while `.#build` ran concurrently (cold cache on the brand-new module). AGENTS.md documents
   `max_concurrency: 1` in `.buildflow.yml` for EXACTLY this class; I re-created it at the shell level. Sequential
   re-run: 17/17 green. Gates from now on: strictly sequential.
2. **Tailed away failure detail AGAIN on the first race run** (`| tail -5` hid which test failed; I briefly believed my
   target test `TestDashboard_Close` had failed when it was `TestDashboard_SSEHeartbeatEmission`). Same mistake class
   as last session's lesson #4. Full output before conclusions — I did recover correctly, but only after the re-run.
3. **Sloppy first writes in the new test files**: a stray `}` from a goconst multiedit, a `var _ = strings.TrimSpace`
   placeholder hack, and an unused `pollUntilOK` helper — all caught by build/lint, all cleaned, but 2–3 wasted round
   trips from writing faster than thinking.
4. **Assumed upstream APIs without checking, twice**: `errorfamily.FamilyOf` (real: `Classify`) and
   `do.ProvideValue` returning a value (it returns nothing). Both compile errors, both fixable in seconds — both
   preventable by reading the source first (which I did do for go-sse/go-health afterward; order was wrong).
5. **Wrong mental model of go-health check semantics**: asserted per-check `fail` for non-critical failures (actual:
   `warn`; only critical → `fail`) and uppercase `<!DOCTYPE html>` (actual: lowercase), and `CachedResponse` does not
   reflect `Evaluate` (had to add `probe.Start` + poll in the dashboard test). Two failed runs before I read
   `buildChecks`/`classify`. Research-first would have saved both.
6. **The replace-strip attempt (b))**: I verified "symbol exists in tag" and "tag's go.mod is clean" but NOT
   "tag interoperates with the rest of the replace graph" — `go vet` caught the `ParseActorID` arity mismatch that
   `go build`… actually `go build` PASSED (test-only import path), and only `vet` (which compiles tests) caught it.
   This is the exact "`go build` skips `_test.go`" trap from AGENTS.md, and I nearly fell in because my verification
   recipe used build+tidy before vet. Reverted cleanly; lesson encoded in the replace comment.
7. **Two git round trips wasted on tag inspection** (`git show usermgmt/v4.7.2:go.mod` instead of `:usermgmt/go.mod`
   — tags are repo commits, modules live in subdirs) and CHANGELOG multiedits failing on non-unique `### Fixed`
   anchors that exist in historical version sections too. Both self-inflicted, both recovered on the second attempt.
8. **Wrote the 16:33 report before fuzz/flake gates finished** (noted as "running at report time" there). They passed
   (fuzz PASS, flake 51/51), so nothing was misreported — but a report should close its own loops.

---

## e) WHAT WE SHOULD IMPROVE

- **Run gates strictly sequentially, ever.** The GOCACHE-writer race is now a twice-documented failure mode (buildflow
  level + my shell level). One gate at a time; no parallel nix app runs.
- **Never pipe a failing candidate through `tail`/`grep -c` on first read** — full output first, filter second.
- **Pre-flight API checklist for new modules**: read upstream source for (a) exact constructor/option signatures,
  (b) status/classification semantics, (c) cache-vs-live evaluation, BEFORE writing tests. This session proved the
  5-minute read beats 2 failed test runs.
- **Replace-strip verification recipe (encode in AGENTS.md next time it's touched)**: for each affected module run
  `GOWORK=off go mod tidy && go build ./... && go vet ./...` — vet is load-bearing because tags break in `_test.go`
  imports invisible to build. Condition = symbol exists AND tag graph interoperates.
- **The replace pile is now provably ordered**: identity-model v4.7.0 must be tagged BEFORE usermgmt can interoperate
  (published usermgmt already calls its 2-value ParseActorID). Tagging in dependency order (identity-model → usermgmt
  → root) collapses 6 of 8 replaces in one coordinated pass.
- **Status reports close their loops**: don't write the report while a gate is still running.

---

## f) NEXT — queued items (Pareto-ordered)

1. **Tag identity-model/v4 v4.7.0** → strip 5 identity-model replaces (usermgmt, adminui, setup, integration_test,
   examples/setup-demo), per-module `GOWORK=off tidy+build+vet`.
2. **Tag usermgmt/v4 v4.7.3** (post-identity-model, incl. ReadModelDialect/MySQL stores/ParseActorID interop) →
   re-attempt the integration_test usermgmt replace strip (recipe + comment already in place).
3. **Tag root v4.7.1+** (RecommendedSecurityMiddleware, ProjectionStatus/Readiness types) → strip root replaces in
   adminui/dashboardui/setup/integration_test/examples.
4. go-cqrs-lite side: tag `metaengine/projectionadapter/v4 v4.5.0+` + `metaengine/sqliteengine/v4 v4.0.2+` → strip
   3 remaining replaces (systemadapter ×2, examples/system-demo ×1).
5. Cross-module dep version drift cleanup after the tagging pass (`check-version-drift --strict` leg of check-modules).
6. adminui → direct identity-model imports (~26 files, 133 SA1019; v5 prerequisite, ADR-0047).
7. integration_test → direct identity-model imports (22 SA1019).
8. CI: wire `check-templates` once go.work absolute replaces are gone.
9. cqrs-lint: Go-installable distribution → CI strict gate (P3.1).
10. Update `docs/guides/leveraging-samber-do.md` + `fullstack-wiring.md` to reference the new `auditlog/v4` +
    `health/v4` modules (guides predate them).
11. Extend `examples/samber-do-demo` or `setup-demo` to mount `auditlog.WithAuditLog` + `health.NewProbe` end-to-end.
12. integration_test: `health.NewProbe` against a REAL `*usermgmt.Service` (module tests use a fake provider only).
13. integration_test: auditlog viewer mounted next to the fullstack UI (cross-module proof).
14. Encode the replace-strip recipe + tag-order lesson into AGENTS.md (e).4/e).5).
15. `slowJournal` → shared testutil if more drain-window tests appear.
16. golines into `nix fmt`/treefmt (P3).
17. Go-based markdown link checker (P3).
18. v4-branch blob rewrite (~27.7MB, destructive — needs explicit approval).
19. datastar/go-sse architecture decision ADR (P3).
20. `mysql_integration_test.go` with testcontainers (P3, Docker-gated).
21. Consider exporting `sqlReadModels` factory pattern for SQLite/Postgres setup templates.
22. dashboardui templ migration (large; ROADMAP-tracked).
23. Websites for health + auditlog modules (website-launch pattern) once tagged.

---

## g) QUESTIONS (cannot be answered from the repo)

1. **Tagging authority + sequence:** may I cut the cqrs-htmx-side tags myself in dependency order —
   `identity-model/v4 v4.7.0` first, then `usermgmt/v4 v4.7.3`, then root `v4.7.1+` — and strip the 8 replaces as each
   condition is met? (I will never push; tags stay local for your review. The failed strip attempt proved published
   usermgmt already depends on unreleased identity-model, so order is mandatory.)
2. **go-cqrs-lite tags:** `metaengine/projectionadapter v4.5.0+` and `metaengine/sqliteengine v4.0.2+` live in your
   other repo — do you want to tag/push those yourself, or should I prepare the exact tag-to-commit mapping for you?
3. **Next major work item after tagging:** adminui identity-model migration (6: biggest SA1019 win, v5 prerequisite),
   the guide/example updates for health+auditlog (10/11: consumer visibility), or the v4-branch blob rewrite
   (18: destructive, shrinks the repo)?

— Reporting stops here. Awaiting instructions.
