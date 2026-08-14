# Session Status: TODO P0/P1 Execution, Hermetic Build Repair, MySQL Completion — 2026-08-14 13:19

> Scope: this session only. Started from the TODO List (paste_1.txt snapshot, updated 2026-08-12).
> Worked P0 → P1 → quick P3 wins, then discovered and partially repaired a **hermetic (GOWORK=off) build regression** across 5 modules.
> All changes auto-committed by the git daemon (HEAD ~12615961).

---

## a) FULLY DONE

### 1. P0 — go-cqrs-lite upstream drift: VERIFIED ALREADY FIXED (no code needed)

The TODO header claimed the build was broken (go-cqrs-lite `af4b60841` reverted ADR-0111). **Stale claim.**
Commits `d7855047`→`8312ad4c` (2026-08-13, prior session) already re-integrated WithActor. Verified: all 12
production modules `go build` clean in workspace mode.

### 2. NEW upstream drift in TEST files — fixed

go-cqrs-lite master renamed the tombstone/listing API AGAIN (after the TODO was written). Production code
compiled, but test files did not (`go vet` caught it; plain `go build` does not compile tests — the drift
was invisible to the previous session's build checks):

| File                                      | Broken usage                                                                       | Fix                                                                                                                                                                                                                                      |
| ----------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `usermgmt/es_materialize_adapter_test.go` | `stack.Materialize.DeleteTypes` field (removed); `listing.DeleteInclude` (removed) | Removed `DeleteTypes`; delete-event now tombstone-marked via `event.MarkTombstone` (deprecated upstream, but the ONLY trigger for `OnTombstone` dispatch — `//nolint:staticcheck` with reason); `mat.List(ctx, stack.IncludeTombstoned)` |
| `dashboardui/handlers_index_test.go`      | `listing.StatusActive` (moved to event pkg)                                        | `listing.StreamStatus{Status: event.TombstoneActive}`                                                                                                                                                                                    |

Both suites pass (`usermgmt` 19.4s, `dashboardui` 0.46s).

### 3. P1.1 — MySQL event-store support: COMPLETE

- **`usermgmt/mysql_stores.go` (new, compiled)**: `NewMySQLCheckpointStore(db)` and
  `NewMySQLSnapshotStore(db)` — MySQL-dialect (`sqlpkg.MySQLDialect{}`) wrappers over
  `storage.NewSQL{Checkpoint,Snapshot}StoreWithDialect`, errorfamily-wrapped. The guide previously
  claimed these were "Manual" — false; upstream had the pieces all along.
- **`usermgmt/mysql_setup.go` (template) enhanced**: `MySQLSetupConfig.CreateSessionStore bool` creates a
  MySQL-backed `SQLSessionStore` (auto-migrating DDL) exposed as `MySQLEventSourcedSetup.SessionStore`;
  `CheckpointStore` now DEFAULTS to `NewMySQLCheckpointStore` (projection positions survive restarts).
- **`usermgmt/sql_readmodel_dialect.go` (new)**: `newSQLReadModelsForDialect(db, dialect)` — one factory,
  sqlite/mysql/postgres constructor families; unsupported dialect → Rejection error.
- **`ReadModelDialect` field added** to `EventSourcedConfig` AND `ServiceConfig` (default `""` = SQLite,
  backward compatible), threaded through `NewService` → `NewEventSourcedSetup`. Previously `ReadModelDB`
  HARDCODED SQLite read models — MySQL/Postgres consumers literally could not use the Service path.
- Tests: `sql_readmodel_dialect_test.go` (sqlite family incl. `""` default, unsupported-dialect Rejection).
  Full usermgmt suite passes. `check-templates.sh` passes.
- **Docs**: `docs/guides/mysql-setup.md` rewritten (support table now all ✅ with real constructor names;
  correct `ServiceConfig` wiring example with `ReadModelDialect: "mysql"`; manual-wiring path). README SQL
  bullet updated. NOTE: the original guide draft I first wrote had a WRONG ServiceConfig example
  (repository fields that don't exist on ServiceConfig) — caught by reading `service_core.go`, fixed.

### 4. P1.2 — Async startup integration test: COMPLETE

`integration_test/async_startup_test.go` — `TestAsyncStartupReadinessLifecycle`, the exact test the TODO
asked for: real HTTP server (`httptest.NewServer`), `/health` = `ReadinessHandler(ProjectionReadinessCheck(svc))`,
`AsyncStartup: true`:

1. Phase 1 seeds 25 users synchronously (journal backlog).
2. Phase 2 restarts on the same journal with a `slowJournal` wrapper (15ms sleep per `ReadFrom`) so the
   drain window is deterministic — without it the in-memory journal drains in µs and 503 is unobservable.
3. Asserts: server answers IMMEDIATELY (liveness decoupled), first `/health` = **503**, polls to **200**,
   `journal.reads > 0`, then **read-your-writes**: last seeded user queryable via `svc.GetUser` once ready,
   steady-state 200.
   Passes 5x consecutive runs (0.718s total). Stable.

### 5. P3 — `ActorIDAsUserID(actorID) (UserID, bool)` helper: COMPLETE

- Added to `identity-model/id.go` (kind-guard + zero-value-on-false semantics; preserves historical
  `NewUserID` parse-or-synthesize behavior, `//nolint:staticcheck` documented).
- Re-exported in `usermgmt/id.go` (with the standard Deprecated marker, matching sibling wrappers).
- Both real call sites migrated: `identity-model/session.go` `newSession`, `usermgmt/audit_log.go` `Handle`.
  (TODO said three sites incl. `authz_roles.go` — that file no longer exists; only two sites remain.)
- Test `TestActorIDAsUserID` (user roundtrip + bot/system/service/zero rejected) — passes.

### 6. P3 — ADR-0048 Liveness/Readiness Decoupling: COMPLETE

`docs/adr/0048-liveness-readiness-decoupling.md` (ACCEPTED 2026-08-14): context (startup outage from
blocking drain), decision (AsyncStartup flag + ProjectionReadinessCheck gate, deployment contract),
deliberate semantics (backward-compat default, failure surfacing via monitoring not constructor, liveness
≠ readiness), alternatives considered (longer DrainTimeout, SWR stale reads, per-projection readiness),
consequences, verification pointer to the new integration test. INDEX.md row added.

### 7. P3 — Doc cross-references: COMPLETE

- `docs/guides/projection-health-monitoring.md` See Also → `async-projection-startup.md` link added.
- `docs/guides/leveraging-httputil.md` → new **Recipe 9: RecommendedSecurityMiddleware** (zero-arg
  baseline, delegation by adminui/dashboardui, nonce access, ordering example).

---

## b) PARTIALLY DONE

### P1.3 — Full nix verification gates: ~40% done, and it found real breakage

`nix run .#build` + `.#test` (background) exposed that **root module's hermetic (GOWORK=off) build was
BROKEN** — invisible to workspace-mode builds:

- **Root `handler.go`**: used `basic.ApplyOptions(...)` from go-cqrs-lite commit `393c88b0c`
  (feat(command,query): add ApplyOptions) — published AFTER command/v4.6.0, i.e. unpublished.
  **FIXED**: `command.Option` is an exported `func(*BasicCommand)`, so options are now applied via a
  direct loop (`for _, opt := range CommandOptionsFromContext(ctx) { opt(basic) }`) — compiles against
  BOTH published tags and master. Same for query side. Root hermetic build now passes.

- **Hermetic scan of all 24 modules** found 4 more failures; fixed 3 of 4 by adding TEMPORARY local
  `replace` directives (the documented AGENTS.md pattern — "do NOT strip until tags published"):

| Module                               | Missing symbols in published tags                                                                                      | Fix                                                                                     | State    |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | -------- |
| root                                 | `ApplyOptions` (command/query v4.6.0 tag lacks it)                                                                     | direct option loop, no replace needed                                                   | ✅ FIXED |
| usermgmt                             | `identitymodel.ErrRegistrationClosed`, `ActorSystem`, `ActorService` (identity-model > v4.2.0)                         | `replace identity-model/v4 => ../identity-model`                                        | ✅ FIXED |
| adminui                              | `usermgmt.ParseActorID` 2-value signature (usermgmt > v4.7.2), same identity-model symbols                             | `replace usermgmt/v4 => ../usermgmt` + `replace identity-model/v4 => ../identity-model` | ✅ FIXED |
| setup                                | same identity-model symbols (replaces in go.work don't apply to hermetic builds — replace only applies in main module) | `replace identity-model/v4 => ../identity-model`                                        | ✅ FIXED |
| systemadapter + examples/system-demo | `projectionadapter.EventWithID` has no `OccurredAt` (metaengine/projectionadapter > v4.4.0)                            | **NOT FIXED — interrupted here**                                                        | ❌ OPEN  |

- Gates NOT yet re-run as a unified suite: `#test`, `#coverage-gate`, `#lint`, `#check-cqrs-lint`,
  `#test-fuzz`, `#test-flake`, `nix flake check --no-build`. The original background gate run FAILED on
  root; root is now fixed but the re-run hasn't happened.

---

## c) NOT STARTED (from the TODO, untouched this session)

- **P2 systemadapter lint remediation** (104 issues) + coverage-gate threshold + CI/isolation/dep-budget wiring.
- **P2 fullstack UI integration test expansion** (seeded-user admin render, dashboard projection-health
  assertions, login-page auth-button presence/absence per Service config).
- **P2 `health/v4` module** (go-health + go-health-dashboard bridge).
- **P2 `auditlog/v4` module** (samber-do-auditlog one-liner).
- **P2 remaining BuildFlow tools in devShell** (cspell, vitest, jest).
- **P2 wire `check-codegen`/`check-templates`/`check-cqrs-lint` into CI**.
- **P2 adminui → direct identity-model imports** (~26 files, 133 SA1019 suppressions).
- **P2 integration_test → direct identity-model imports** (22 SA1019).
- **P3 remaining items**: cqrs-lint strict CI gate; cross-module dep version drift (my replaces are a
  mitigation, the real fix is tagging); datastar/go-sse ADR; golines in `nix fmt`; Go markdown link
  checker; v4-branch blob rewrite (~27.7MB).
- **TODO_LIST.md / CHANGELOG.md updates for THIS session's completed items** — not yet done (interrupted
  before the docs pass).

## d) TOTALLY FUCKED UP / REGRETS

1. **I nearly shipped a wrong API example in the MySQL guide.** My first draft passed
   `UserRepository`/`ReadModel`/`Authz()` fields to `ServiceConfig` — those fields DON'T EXIST there
   (ServiceConfig takes stores; it builds its own repositories). Caught it by reading `service_core.go`
   before committing. Lesson re-learned: read the real struct before writing consumer-facing examples.
2. **Test drift was invisible for a session cycle.** The previous session verified with `go build`, which
   skips `_test.go`. Two test files were broken on master the whole time. The cheap fix (run `go vet`
   occasionally) wasn't in the verification habit. The nix `#test` gate WOULD have caught it — it just
   wasn't run.
3. **Release debt grew.** Three more modules (usermgmt, adminui, setup) now carry local replaces because
   identity-model v4.2.0 / usermgmt v4.7.2 tags lack master symbols. This makes every hermetic build
   machine-dependent until identity-model/v4.3.0+ and usermgmt/v4.7.3+ are tagged. It's the documented
   pattern, but the pile is getting bigger, not smaller.
4. **`go mod tidy` was run in 3 modules** as part of replace-adding — go.sum churn beyond the replace
   lines is possible (the daemon committed it; not individually reviewed).

## e) WHAT WE SHOULD IMPROVE

- **Make `go vet` (or test-compile) part of every verification pass**, not just `go build` — test files
  rot invisibly otherwise.
- **Run the full nix gate suite before declaring upstream-integration work done** — hermetic GOWORK=off
  failures are structurally invisible in workspace mode and have now bitten twice (root ApplyOptions,
  usermgmt identity-model).
- **Tag releases more aggressively.** identity-model and usermgmt have untagged master symbols blocking
  hermetic builds; each day untagged adds another replace.
- **Consider a `make`-free drift sentinel**: a tiny CI script that runs `GOWORK=off go build ./...` per
  module so unpublished-API usage fails loudly at PR time instead of at gate time.

## f) NEXT — up to 50 queued items (Pareto-ordered)

**P0/P1 finish line:**

1. Fix systemadapter hermetic build (`EventWithID.OccurredAt`) — add local replace for
   `metaengine/projectionadapter/v4` (same pattern) or pin to v4.4.0 API.
2. Fix examples/system-demo hermetic build (same root cause).
3. Re-run `nix run .#build` — expect all-green.
4. Re-run `nix run .#test` (14 suites).
5. Re-run `nix run .#coverage-gate` (12 gates; verify root still ≥90 after handler.go change).
6. Re-run `nix run .#lint` (12 modules at 0; verify my new files don't regress).
7. Re-run `nix run .#check-codegen`, `.#check-templates`, `.#check-cqrs-lint`.
8. Re-run `nix run .#test-fuzz`, `.#test-flake`.
9. `nix flake check --no-build`.
10. Update TODO_LIST.md: remove/mark P0, P1.1, P1.2, ADR-0048, ActorID helper, doc cross-refs items.
11. Append CHANGELOG [Unreleased] entries for everything in section (a).
12. Update AGENTS.md: new replaces in usermgmt/adminui/setup go.mod; listing/tombstone API renames
    (StatusActive→event.TombstoneActive, DeleteTypes removed, stack.IncludeTombstoned); MySQL stores;
    ReadModelDialect.

**P2 quality (TODO items):**
13. systemadapter lint remediation (43 SA1019 via direct identity-model imports, exhaustruct ×23,
contextcheck ×10, err113 ×4 → errorfamily, wsl_v5 ×7, wrapcheck ×2, goimports ×3, gci ×1, mnd ×4,
errcheck ×4, nlreturn ×3).
14. Add systemadapter coverage-gate threshold + CI + isolation/dep-budget scripts.
15. Fullstack UI test expansion: seeded user → GET /admin/ → assert email in HTML.
16. Fullstack UI test: dashboard shows projection names/health.
17. Fullstack UI test: login page buttons per auth config (TOTP on/off, WebAuthn on/off, OAuth2 on/off).
18. `health/v4` module: `health.NewProbe(svc, opts)` + `health.NewDashboard(probe, opts)`.
19. `auditlog/v4` module: `auditlog.WithAuditLog(opts) []do.HookProvider` + `MountReport`.
20. adminui → direct identity-model imports (26 files, 133 SA1019) — v5 prerequisite.
21. integration_test → direct identity-model imports (22 SA1019).
22. devShell: add cspell, vitest, jest.
23. CI: wire check-codegen (pin templ version), check-templates (workspace mode), check-cqrs-lint (blocked
on Nix-only binary — Go-installable distribution needed).

**P3 debt:**
24. Tag identity-model/v4.3.0 (ErrRegistrationClosed, ActorSystem/ActorService, ActorIDAsUserID) — then
strip usermgmt/setup/adminui identity-model replaces.
25. Tag usermgmt/v4.7.3 (ParseActorID 2-value, ReadModelDialect, MySQL stores) — then strip adminui
usermgmt replace.
26. Cross-module dep drift sweep before next release (adminui still requires usermgmt v4.6.1-era refs
pre-tag).
27. cqrs-lint strict CI gate.
28. ADR for datastar/go-sse exclusion (or migrate to go-sse KeyedLines).
29. golines alignment in `nix fmt` (treefmt wiring).
30. Go/goldmark-based markdown link checker.
31. v4 branch `git filter-repo` blob strip (~27.7MB binaries) + force-push (needs approval).
32. `mysql_integration_test.go` — verify it still passes with new dialect helpers (uses testcontainers;
may be skipped without Docker).

**Small polish found this session:**
33. Guide `mysql-setup.md`: mention `NewMySQLSnapshotStore` in the See Also / ADR-0041 snapshot doc.
34. Consider exporting `sqlReadModels` factory pattern for the SQLite/Postgres setup templates
(they still hand-roll four constructor calls each).
35. Add `AsyncStartup` mention to README feature list (guide + ADR exist; README bullet doesn't).
36. `slowJournal` test helper could move to a shared testutil if more drain-window tests appear.
37. Deprecation audit: `event.MarkTombstone` in our test is upstream-deprecated — when go-cqrs-lite ships
a domain-event path for Materialize OnTombstone, migrate the test.
38. setup/Config could expose `ReadModelDialect` passthrough (currently setup users get SQLite read models
only — same hardcode I just fixed in usermgmt).

## g) QUESTIONS (cannot figure out myself)

1. **Tag now or replace-and-wait?** Should I cut `identity-model/v4.3.0` + `usermgmt/v4.7.3` tags now
   (removes 4 local replaces, unblocks hermetic builds everywhere), or keep accumulating replaces until
   the next coordinated release? Tagging publishes to the module proxy — your call on release cadence.
2. **systemadapter hermetic fix direction:** local replace for `metaengine/projectionadapter/v4` (fast,
   more debt — my default), or refactor `declarations.go` to the published v4.4.0 `EventWithID` API
   (no debt, but fights the local-master replaces everywhere else in go.work)?
3. **Remaining P2 priority order:** systemadapter lint (104 issues, blocks gate integration) vs fullstack
   UI test expansion vs identity-model direct-import migrations (155 SA1019 total, v5 prerequisite)?
   Which first?

---

**Bottom line:** P0 verified-fixed, P1.1 + P1.2 fully done, 4/5 hermetic build breaks repaired, ADR-0048 +
ActorID helper + doc cross-refs shipped. The unified gate re-run and the systemadapter hermetic fix are the
two open threads before this session's work is fully verified.
