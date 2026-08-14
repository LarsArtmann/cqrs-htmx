# Session Status: Flakes Root-Caused, 2 New Bridge Modules, Gates Green — 2026-08-14 16:33

> Continuation of the TODO-execution session (previous report: `2026-08-14_15-22_hermetic-repair-complete-gates-green.md`).
> User instruction: keep executing autonomously, one verified step at a time.
> Tagging/releases intentionally NOT touched — the 3 questions from the previous report §g remain open with the user.

---

## a) FULLY DONE

### 1. The "unreproducible" dashboardui flake: ROOT-CAUSED as a real data race, fixed

The prior session observed one `TestDashboard_Close` failure and hypothesized a nil-deref. Running the module under
`-race -count=10` reproduced a failure immediately — but in `TestDashboard_SSEHeartbeatEmission` (a different test),
exposing the actual bug:

**`dashboardui/sse.go` spawned `go stream.Heartbeat(stream.Context(), interval)` and returned without joining it.**
After the request context cancelled, the heartbeat goroutine could still be mid-write to the ResponseWriter while the
handler had already returned (net/http forbids touching the writer past handler return; on real connections this races
`stream.Close()`). Fixed by deriving a cancellable context and joining the goroutine via `defer` before any return path.
Reproduced under `-race -count=50` before the fix (writer = `go-sse WriteHeartbeat` via `sseHandler gowrap2`),
green 50× + full package 10× after. The hypothesized nil-deref was ALSO real and also fixed: `TestDashboard_Close`
now checks `New`'s error, and `Dashboard.Close()` gained a nil-receiver guard (documented no-op).

### 2. New module: `health/v4` (go-health + go-health-dashboard bridge)

- `health.NewProbe(provider, opts...)` — a `gohealth.Probe` with one named check per projection. Semantics mirror
  `cqrshtmx.ProjectionReadinessCheck`: live/stopped pass; drain states (idle/running/backoff/draining) transient;
  failed infrastructure (carries LastError).
- `health.Recorder(provider)` — a `HealthRecorder` that MERGES projection checks with a samber/do injector's own
  service checks (do users compose: `gohealth.New(injector, gohealth.WithHealthRecorder(health.Recorder(svc)))`).
- `health.NewDashboard(probe, opts...)` — the go-health-dashboard UI (HTML/SSE/JSON by Accept header).
- **Born hermetic**: resolves root v4.7.0 (the two needed types are in the published tag), go-health v0.0.2,
  go-health-dashboard v0.3.0 from the proxy — ZERO local replaces, zero replace-pile growth.
- 100.0% coverage (gate 90). Own README, `.golangci.yml` (setup pattern), `.cqrs-lint.json`. Wired into `go.work`,
  lint/coverage/cqrs-lint gates, dep budgets (5 deps, budget 6), and CI (build/test/coverage/lint/mod-tidy).

### 3. New module: `auditlog/v4` (samber-do-auditlog bridge)

- `auditlog.WithAuditLog(auditCfg, viewerCfg)` returns a `Setup`: `do.InjectorOpts` (for `do.NewWithOpts`), the plugin
  handle (reports/exports), and the live SSE viewer (an `http.Handler`) — the one-call integration the TODO sketched.
  The actual upstream API differed from the sketch (`Opts() *InjectorOpts`, `live.Server` implements `ServeHTTP`), so
  `MountReport(mux, report)` became "mount setup.Viewer" — documented in the README.
- 100.0% coverage (gate 90). Same full wiring as health. Also resolves published tags only (v0.9.2).

### 4. Fullstack UI test expansion (integration_test)

Three new tests + harness change (`setupFullstackUI` now returns the `*usermgmt.Service`):

- `TestFullstackUI_AdminRendersSeededUser` — register → super_admin via real authz path (`AddGroupPolicy`) →
  `GET /admin/users` with session cookie (bounded poll) → seeded email + display name in HTML.
- `TestFullstackUI_DashboardShowsProjectionHealth` — `/dashboard/projections` renders real projection names
  (user-read-model, casbin-projection, tenant-read-model) from the Service's projection host.
- `TestFullstackUI_LoginButtonsMatchAuthConfig` — 3 subtests: no providers (setup hint, no buttons), WebAuthn
  (passkey button), OAuth2 (provider button). **Finding:** the original TODO spec wanted a TOTP-button assertion, but
  loginpage has NO TOTP UI by design (second-factor API flow) — noted in the test comment.

### 5. systemadapter gate integration completed (TODO item closed)

- `check-module-isolation.sh`: hardcoded module list → **go.work auto-discovery** (same e2e/examples exclusion as
  flake apps) — this WAS the root cause of systemadapter being missing; the sentinel can no longer drift.
- `check-dep-budgets.sh`: + systemadapter (13 deps, budget 16).
- **Bug found & fixed in both scripts**: `head -1 go.mod | awk '{print $2}'` parsed comment words ("adapter",
  "dashboard") as module names for go.mod files starting with `//cqrs-lint:ignore` lines → now `grep -m1 '^module '`.
- The drift-sentinel flake-app item is thereby satisfied: `nix run .#check-modules` auto-covers every module.
  (`check-version-drift --strict` inside that app still fails on known cross-tag drift — the documented P3 blocked on
  the next release; NOT touched.)

### 6. Docs

- CHANGELOG `[Unreleased]`: 4 Added, 4 Fixed, 1 Changed entries for this session.
- TODO_LIST: removed 4 completed items (fullstack UI, health, auditlog, systemadapter wiring); header now 26 modules /
  15 lint / 15 coverage gates with health+auditlog at 100%; CI-wiring item updated (check-codegen was already wired by
  a concurrent session; check-templates documented as blocked on go.work absolute replaces; check-cqrs-lint blocked on
  distribution).
- ROADMAP "Not Planned": devShell cspell/vitest/jest item — STALE ask (codespell already present; BuildFlow skips
  jest/vitest because the repo has no JS tests).
- AGENTS.md: module list + dependency direction + coverage/lint/gate numbers updated (26 modules, 15 lint-checked,
  17 suites, 15 coverage gates); health + auditlog module bullets.
- README.md: AsyncStartup feature bullet added. mysql-setup.md: See Also now links ADR-0041 for `NewMySQLSnapshotStore`.
- systemadapter go.mod: `record/v4` + `go-error-family` promoted to direct (missed tidy after OnRecordTyped).

## b) VERIFICATION (all green 2026-08-14)

| Gate                                    | Result                                                                                                              |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `.#build`                               | 26/26 modules hermetic                                                                                              |
| `.#test`                                | 17/17 suites ok                                                                                                     |
| `.#coverage-gate`                       | 15/15 (health 100%/90, auditlog 100%/90)                                                                            |
| `.#lint`                                | 15 modules × 0 issues                                                                                               |
| `.#check-cqrs-lint`                     | all pass strict                                                                                                     |
| `.#check-codegen` / `.#check-templates` | pass                                                                                                                |
| `.#check-modules` scripts               | isolation + dep-budgets pass                                                                                        |
| `check-docs-links.sh`                   | pass                                                                                                                |
| `nix flake check --no-build`            | pass                                                                                                                |
| `.#test-fuzz` / `.#test-flake`          | running at report time — dashboardui verified separately: heartbeat test 50× + full package 10× green under `-race` |

## c) PARTIALLY DONE / NOT STARTED

- **Tag pending releases** (P1 #1): still blocked on user answers — 8 local replaces remain (identity-model ×5,
  projectionadapter ×2, sqliteengine ×1). NOTE: usermgmt v4.7.2 IS now tagged (observed this session), so the
  integration_test usermgmt replace may be strippable — needs verification before stripping.
- adminui/integration_test direct identity-model migrations (155 SA1019) — untouched, P2.
- CI check-templates (blocked: go.work absolute replaces don't exist on runners) / check-cqrs-lint (blocked: Nix-only
  binary) — documented in TODO.

## d) MISTAKES / LESSONS

1. **I launched two nix gate sweeps in parallel and hit the documented GOCACHE race** — auditlog showed a transient
   `[setup failed]` in `.#test` while `.#build` ran concurrently (cold cache on the brand-new module). Passes
   deterministically when run sequentially (re-ran `.test` alone: 17/17). The AGENTS.govalid gotcha already warned
   about concurrent GOCACHE writers; I created a new instance of the same class. Gates must run sequentially.
2. My first multiedit on the new test file left a stray `}` (goconst edit boundary) and a placeholder import hack —
   caught by build/lint immediately. Test-authoring haste; two round trips wasted.
3. CHANGELOG multiedit "1 of 3 failed" on ambiguous section anchors (`### Fixed` exists in old version sections too) —
   re-anchored with unique first-bullet context.
4. tailed-away failure detail AGAIN on the first race run (`| tail -5` hid the failing test name) — re-ran with full
   output before concluding anything.

## e) OPEN QUESTIONS (unchanged from previous report §g)

1. Tag identity-model v4.7.0 (+ root v4.7.1+, and ask for go-cqrs-lite projectionadapter v4.5.0+ / sqliteengine
   v4.0.2+) to strip the replace pile? usermgmt v4.7.2 now exists — should I verify + strip its replace?
2. (Answered autonomously this session: flake fix = both the test error-check AND nil-safe Close; plus the real race fix.)
3. Next P2 priority — remaining candidates: adminui/integration_test identity-model migrations, `slowJournal` testutil,
   golines-in-`nix-fmt`, v4-branch blob rewrite.
