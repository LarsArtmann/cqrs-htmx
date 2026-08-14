# Session Status: Hermetic Build Repair Complete, Full Gates Green, cqrs-lint + golangci Remediation — 2026-08-14 15:22

> Continuation of the TODO-execution session (previous report: `2026-08-14_13-19_todo-execution-hermetic-build-repair.md`).
> User instruction this session: keep executing the TODO autonomously, one verified step at a time.
> All changes auto-committed by the git daemon (HEAD `f5e20540`).

---

## a) FULLY DONE

### 1. systemadapter + examples/system-demo hermetic build: FIXED

`systemadapter/declarations.go` uses `projectionadapter.EventWithID[P].OccurredAt` — a field go-cqrs-lite master
added AFTER the published `metaengine/projectionadapter/v4 v4.4.0` tag. No dual-compatible API exists (field access),
so per the documented repo pattern I added a **temporary local replace** (with removal condition) to
`systemadapter/go.mod` and `examples/system-demo/go.mod`:
`metaengine/projectionadapter/v4 => /home/lars/projects/go-cqrs-lite/metaengine/projectionadapter`.
Verified: `GOWORK=off go build/vet/test` all pass in both modules. system-demo needs only this one (it uses the
`memory` driver, not sqlite).

### 2. Two more hermetic failures found and fixed (my full-workspace scan)

- **examples/setup-demo**: `go mod tidy` failed — local `usermgmt` (via its own replace) requires
  `identity-model/v4 v4.7.0` which is NOT tagged. Added `replace identity-model/v4 => ../../identity-model`.
- **integration_test**: same root cause, same fix (`replace identity-model/v4 => ../identity-model`). This one was
  caught by `nix run .#test` (the gate), not by my build scan — `go build ./...` doesn't compile `_test.go`, and
  test-only imports dragged in the untagged ref.

### 3. systemadapter hermetic TEST failure: FIXED

`TestDomainConfig_SQLiteDeployment` failed hermetically with `unknown driver "sqlite"`. Root cause: go-cqrs-lite
master's `sqliteengine` gained `register.go` (self-registers the driver via `init`, `metaengine.RegisterDriver`) after
v4.0.1. Added the third temporary replace: `metaengine/sqliteengine/v4 => /home/lars/projects/go-cqrs-lite/metaengine/sqliteengine`.
All systemadapter tests pass hermetically now.

### 4. cqrs-lint strict gate: 28 findings → 0 (real fixes + reasoned suppressions)

`nix run .#check-cqrs-lint` was failing on root + systemadapter. Triage:

**Real fixes:**
- `systemadapter/projections.go`: `errors.New`/`fmt.Errorf` → `errorfamily` constructors (Rejection for wiring
  failures with `systemadapter.*` codes; Transient for drain timeout) — errors now classify into HTTP statuses.
- `systemadapter/projections.go`: added `WithBatchSize(256)` (P008) and **new `WithCheckpointStore` /
  `WithDeadLetterStore` functional options** on `NewProjectionLayer` — the C017 "in-memory store with persistent
  event store" ERROR findings are now a documented, overridable default instead of a hard smell. New test
  `TestProjectionLayer_CustomStores` covers dispatch → drain → read-model end-to-end with injected stores.
- `examples/dashboard-demo/main.go`: `WithBatchSize(256)` (P008).
- README.md/AGENTS.md: stale `go-cqrs-lite v4.2.0`/`v4.4.0` → `v4.6.0`; `go-sse v0.3.0/v0.4.0` → `v0.5.0` (D005).
- Root `go.mod`: bumped `storage/memory/v4` v4.2.0 → v4.3.0 (V006 version drift — dashboardui needed a tidy after).

**Reasoned suppressions (each with a comment; verified the claim first):**
- `must()` panic (C009): `system.DomainConfig.Commands` is `func(*System)` — no error channel exists; registration
  is static init-time wiring.
- `views.go` `time.Time` fields (C013): metaengine serializes views as JSON; RFC3339 keeps timezone — detector
  assumes raw SQL column mapping.
- go.mod V003/V006: **verified false positives** — `sqliteengine v4.0.1` IS the latest tag (detector compares
  against the core module's version); `commandlifecycle/projections v4.0.0` IS the latest.
- A018 (adapter doesn't dispatch), E014 (read-your-writes owned by consumers via `WaitForDrain`), F019 (metaengine
  has no volume-hint option — checked v4.10.0), F001 (`DeleteBot` DOES emit `EventBotDeleted` via the decider —
  detector can't trace through dispatch), C023/C015 (void or secondary `Close()` on error paths), B024 (bus wraps
  handlers with recovery internally — same suppression usermgmt already carries).

Gate result: **All modules pass cqrs-lint strict.**

### 5. golangci-lint gate: all remaining failures → 0 issues in all 13 modules

The lint gate had ALSO picked up systemadapter (the concurrent session integrated it — TODO item 13 was live).
Fixes:

- **`OnTyped` → `OnRecordTyped` migration (31 sites, `systemadapter/declarations.go`)**: `OnTyped` is deprecated
  for v5 removal. `OnRecordTyped` handlers must take `(record.Record, E)` as first param — did the migration via
  sed (rename + param prepend) + manual import + golines/wrap for 3 long signatures. Tests pass both modes.
- **Root `middleware.go` gocognit refactor**: `ContextEnrichmentMiddleware` complexity 33 (>30) → extracted
  `enrichRequestID`, `enrichCorrelationID`, `enrichUserAndActor` helpers. Identical behavior.
- **dashboardui godoclint**: `core/capabilities.go` comment I added earlier created a duplicate package godoc
  (doc.go already had one). Trashed doc.go, moved the canonical package comment into capabilities.go (with the
  E014 suppression line inside it).
- **integration_test**: errcheck (`defer func() { _ = resp.Body.Close() }()`), wrapcheck nolint on the
  `slowJournal` delay-injection delegator, golines on the long line.
- **usermgmt**: `newSQLReadModelsForDialect` cyclop 17→~5 via table-driven `readModelConstructors` struct
  (exhaustruct nolint on the deliberate zero-value false signal); `runTestOAuth2Login` now takes/forwards
  `context.Context` (contextcheck), all 10 call sites updated.

Gate result: `nix run .#lint` exit 0, **13 modules × 0 issues**.

### 6. Full verification gate suite: ALL GREEN (verified 2026-08-14)

| Gate | Result |
|---|---|
| `.#build` | 24/24 modules (hermetic GOWORK=off) |
| `.#test` | 15/15 suites ok |
| `.#coverage-gate` | 13 gates pass (root 93.4%/90, systemadapter 89.6%/70, dashboardui/core 86.1%/80, …) |
| `.#lint` | 0 issues / 13 modules |
| `.#check-cqrs-lint` | all module configs pass strict |
| `.#check-codegen` / `.#check-templates` | pass |
| `.#test-fuzz` | PASS (7.1M execs) |
| `.#test-flake` | 45 suite-runs ok — **one `TestDashboard_Close` failure in dashboardui, see d)** |
| `nix flake check --no-build` | pass |

### 7. Project docs updated (TODO/CHANGELOG/AGENTS)

- **CHANGELOG.md `[Unreleased]`**: Added (MySQL completion, `ReadModelDialect`, `ActorIDAsUserID`, ADR-0048,
  systemadapter store options + custom-stores test, async startup integration test), Fixed (hermetic build
  regression across modules with the full replace inventory, upstream test drift, cqrs-lint 28 findings,
  systemadapter lint remediation, middleware refactor, oauth2 test context plumbing), Changed
  (NewProjectionLayer errorfamily classification).
- **TODO_LIST.md**: removed completed items (stale P0 build-broken claim, MySQL, gates run, ADR-0048, doc
  cross-refs, ActorID helper, httputil recipe), replaced P0 section with a new **P1: tag pending module releases
  and strip temporary local replaces**, systemadapter item downgraded to `[~]` (only isolation/dep-budget/CI
  wiring left), header block refreshed with verified coverage/lint numbers and the pending-replace warning.
- **AGENTS.md**: lint gotcha now 13 modules incl. systemadapter (with the OnRecordTyped `(record.Record, E)`
  requirement), gates-passed date 2026-08-14, MySQL gotcha extended with the full-support state (stores, template
  defaults, ReadModelDialect), **new gotcha: module-level temporary replaces for hermetic builds** (why go.work
  replaces don't apply per-module; the full current pile + strip conditions), **new gotcha: `go build ./...` skips
  `_test.go`** — make `go vet` + per-module GOWORK=off builds part of every verification pass.

---

## b) PARTIALLY DONE

- **TestDashboard_Close flake**: `nix run .#test-flake` observed ONE failure of `dashboardui TestDashboard_Close`
  (run 1 of 3 in that module). Could not reproduce afterwards: 20× plain, 5× `-race`, full suite — all green.
  Root-cause hypothesis (confirmed by reading the test, fix NOT applied — interrupted): the test does
  `d, _ := New(Config{...})` — if `New` ever fails, `d` is nil and `d.Close()` panics. The correct fix is to
  `t.Fatalf` on the New error (and possibly assert Close idempotency via recover if a panic path exists).
- **P2 systemadapter gate integration**: lint/coverage/cqrs-lint/CI integration DONE (this session + concurrent
  session). Remaining per updated TODO: `check-module-isolation.sh` + `check-dep-budgets.sh` wiring.

## c) NOT STARTED (this session, from the TODO backlog)

- P2 fullstack UI integration test expansion (seeded-user admin render, dashboard projection health, login-page
  auth buttons per config).
- P2 `health/v4` module (go-health bridge) and `auditlog/v4` module (samber-do-auditlog).
- P2 adminui / integration_test → direct identity-model imports (155 SA1019 total).
- P2 devShell: cspell, vitest, jest. P2 CI wiring for check-codegen/check-templates/check-cqrs-lint.
- P3 items: cqrs-lint CI gate, golines in `nix fmt`, markdown link checker, v4-branch blob rewrite,
  datastar/go-sse ADR, mysql_integration_test with new helpers.
- Tagging releases to strip the replace pile (now the #1 P1 TODO item).

## d) TOTALLY FUCKED UP / REGRETS

1. **Piped commands masked failures twice.** My first setup-demo verification printed `SETUP-DEMO-OK` while
   `go mod tidy` had actually FAILED (the `&&`-chain short-circuit was hidden by `tail` in a pipeline). I caught
   it by reading the output text, but I should pipe to files / check `PIPESTATUS` or use `set -o pipefail`.
2. **`go build` in example dirs writes tracked binaries.** `examples/setup-demo/setup-demo` (27MB) got rebuilt by
   my hermetic scan — restored via `git restore` (twice this session). Lesson recorded: never run bare `go build`
   in example modules without checking `git status` after.
3. **CHANGELOG edit structural mess.** A `multiedit` "1 of 3 edits failed" left duplicated Added/Fixed/Changed
   blocks mid-file; I then "fixed" it with sed line-deletes and had to re-check three times (grep counts, section
   listing) before it was actually clean. Should have used python/sed deterministically from the start or restored
   and re-applied once.
4. **TODO_LIST multiedit failed on exact whitespace** — I retried the same long edit verbatim instead of switching
   strategy, wasted a round trip, then did it surgically with python (which worked first try).
5. **cwd drift in one verification**: ran `go mod tidy` for integration_test from the repo root (command worked on
   the wrong module — root was untouched, verified, but only by luck of `git diff --stat`).
6. **Left the TestDashboard_Close investigation unfinished** — interrupted mid-edit, so the observed flake has no
   fix and no closing diagnosis in code.

## e) WHAT WE SHOULD IMPROVE

- **Always `set -o pipefail` (or avoid pipes around `go` commands)** — masked exit codes caused both false-OKs
  this session.
- **A drift sentinel script** (per-module `GOWORK=off go vet ./...` + `go build`) as a flake.nix app — the nix
  gates do this, but a fast pre-commit version would catch unpublished-API usage before gate time. This was
  suggested last session and is now even more justified.
- **Tag more aggressively.** The replace pile grew to 8 module-level entries (identity-model ×5, projectionadapter
  ×2, sqliteengine ×1). Every day untagged adds machine-dependence. TODO_LIST now has this as P1 #1.
- **Read the flake-gate output line by line, not `grep -c ok`** — I nearly reported flake green while a FAIL line
  existed in the same output.
- **For lint remediation: fix root causes first (errorfamily, options, migrations), suppress only verified
  false-positives or API-constrained cases** — this order kept the suppression count low and every suppression
  honest (I verified V003/V006 were wrong by listing actual git tags).

## f) NEXT — up to 50 queued items (Pareto-ordered)

1. Fix `TestDashboard_Close` latent nil-deref: check New error, then Close twice (5-minute fix; unblocks flake
   gate confidence).
2. Tag `identity-model/v4 v4.7.0` (or bump refs to whatever gets tagged) → strip replaces in usermgmt, adminui,
   setup, integration_test, examples/setup-demo; verify each with `GOWORK=off go build ./...`.
3. Tag/go-proxy-publish `metaengine/projectionadapter/v4 v4.5.0+` → strip 2 replaces.
4. Tag `metaengine/sqliteengine/v4 v4.0.2+` → strip 1 replace.
5. Tag usermgmt (ParseActorID 2-value, ReadModelDialect, MySQL stores, ActorIDAsUserID re-export) + root v4.7.1+
   → strip the older adminui/dashboardui/setup/integration_test root+usermgmt replaces (see AGENTS.md gotcha).
6. Wire `check-module-isolation.sh` + `check-dep-budgets.sh` for systemadapter (last piece of that TODO).
7. Add the hermetic drift sentinel as a flake app + pre-commit (per e).
8. Fullstack UI test: seeded user → GET /admin/ → assert email in HTML.
9. Fullstack UI test: dashboard shows projection names/health in HTML.
10. Fullstack UI test: login page buttons per auth config (TOTP/WebAuthn/OAuth2 on/off).
11. `health/v4` module: `health.NewProbe(svc, opts)` + `health.NewDashboard(probe, opts)`.
12. `auditlog/v4` module: `auditlog.WithAuditLog(opts) []do.HookProvider` + `MountReport`.
13. adminui → direct identity-model imports (~26 files, 133 SA1019) — v5 prerequisite.
14. integration_test → direct identity-model imports (22 SA1019).
15. devShell: add cspell, vitest, jest.
16. CI: wire check-codegen (pin templ version), check-templates (workspace mode), check-cqrs-lint (blocked on
    Go-installable distribution).
17. `mysql_integration_test.go` verification with new dialect helpers (testcontainers; may skip without Docker).
18. mysql-setup guide: mention `NewMySQLSnapshotStore` in See Also / ADR-0041.
19. Consider exporting `sqlReadModels` factory pattern for SQLite/Postgres setup templates.
20. README feature list: add `AsyncStartup` bullet (guide + ADR exist; README bullet doesn't).
21. Move `slowJournal` to a shared testutil if more drain-window tests appear.
22. Migrate `event.MarkTombstone` in `es_materialize_adapter_test.go` when upstream ships a domain-event
    OnTombstone path.
23. setup/Config `ReadModelDialect` passthrough (setup users currently get SQLite read models only).
24. Dep-drift sweep across all module go.mod files before the next release tagging session.
25. cqrs-lint strict CI gate (blocked: Nix-only binary).
26. ADR for datastar/go-sse exclusion (or migrate to go-sse KeyedLines).
27. golines alignment in `nix fmt` (treefmt wiring).
28. Go/goldmark markdown link checker.
29. v4 branch `git filter-repo` blob strip (~27.7MB) + force-push (needs approval).
30. Root `middleware.go`: consider unit tests pinning the three new enrich* helpers' behavior (currently only
    covered via existing middleware tests).
31. systemadapter: persistent-checkpoint documentation in `docs/guides/leveraging-system-metaengine.md`
    (WithCheckpointStore/WithDeadLetterStore are new and undocumented in the guide).
32. Audit the 8 new cqrs-lint suppressions when cqrs-lint versions change (stale-detector will flag).

## g) QUESTIONS (cannot figure out myself)

1. **Tag now or wait?** Cutting `identity-model/v4 v4.7.0` + usermgmt + root tags (and asking you to tag
   go-cqrs-lite `metaengine/projectionadapter` v4.5.0 + `sqliteengine` v4.0.2) would strip all 8 temporary
   replaces and un-break hermetic builds on any machine. But it publishes to the module proxy and commits the
   release cadence — your call. If yes: tag cqrs-htmx side myself or do you want to review first?
2. **TestDashboard_Close fix scope:** one observed flake, unreproducible in 25+ runs; the test ignores `New`'s
   error (nil-deref if New ever fails). Minimal fix (assert the error) or also harden `Dashboard.Close()` docs/
   behavior for nil-receiver safety?
3. **Next P2 order:** fullstack UI test expansion (8-10), health/auditlog modules (11-12), or the direct
   identity-model migrations (13-14, v5 prerequisite)? All are independent; which do you want first?

---

**Bottom line:** hermetic builds fully repaired across all 24 modules; every nix gate green (build 24/24, tests
15/15, coverage 13/13, lint 13×0, cqrs-lint strict, fuzz, codegen, templates, flake-check) — with one observed,
unreproduced `TestDashboard_Close` flake left unfixed; cqrs-lint (28 findings) and golangci (systemadapter
integration + 5 modules) fully remediated with root-cause fixes first; docs (CHANGELOG/TODO/AGENTS) current.
Open threads: replace-pile stripping (needs tags), the flake fix, and the P2 backlog.
