# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-07-28 | **Version:** v4.6.1 (15 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root ~93.5% (gate 90%), usermgmt ~81% (gate 74%), identity-model ~41% (no gate) — recompute via `nix run .#coverage-gate` | **Lint:** `nix run .#lint` fails on root (~565, varnamelen-dominated), usermgmt (~330, SA1019-dominated), and dashboardui (~154) on pre-existing style nits + `id.*AggregateID` SA1019 deprecation; the other 7 modules pass clean. Non-release-blocking. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — Release Hygiene

- [ ] **Auth sub-module CHANGELOGs behind by one version.** totp/webauthn/oauth2 CHANGELOGs are at `[v4.6.0]`; need `[v4.6.1]` entries (dependency bumps, lockstep alignment). Source: verified 2026-07-28 — `head usermgmt/{totp,webauthn,oauth2}/CHANGELOG.md` shows v4.6.0 as latest.

---

## P2 — Quality Gates

- [~] **identity-model test coverage.** ~41% coverage (2 test files for 25 source files). No coverage-gate threshold defined for this module in `flake.nix`. Needs tests for the Authz engine, command constructors, event payload round-trips, crypto helpers, and the remaining fold functions (`foldMembership`, `foldBot`). Source: `docs/status/2026-07-23_21-27_identity-model-consolidation-brutal-review.md`.
- [~] **dashboardui test coverage.** SSE reconnect replay, initial backfill, heartbeat emission, and Close lifecycle now tested (`sse_replay_test.go`, 4 tests; 16 tests total across 2 files). Remaining: handler-level tests and payload-rendering tests.
- [ ] **dashboardui `handlers.go` split.** Single file at **1179 lines** — split per domain (overview, events, aggregates, projections, DLQ, audit, snapshots, time-travel).
- [ ] **Triage lint nits.** Root ~565 (varnamelen ~405, exhaustruct 61, staticcheck 37, errcheck 27, canonicalheader 24), usermgmt ~330 (staticcheck SA1019 271 — `id.NewAggregateID` deprecation), dashboardui ~154. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module. Non-release-blocking.

---

## P3 — Technical Debt & Future

- [ ] **Migrate `id.NewAggregateID` → `id.NewStreamID`** across usermgmt and dashboardui. go-cqrs-lite v4 renamed the API; backward-compatible aliases exist, but the migration aligns with upstream naming. Examples were migrated (commit `a7c09ab`), but **usermgmt and dashboardui remain** — 2 production call sites in usermgmt (`import_export.go`, `service_oauth2.go`), ~24 non-test files and ~51 test files using the deprecated APIs. Source: `docs/status/2026-07-28_15-06_govalid-transient-failure-and-partial-aggregateid-migration.md`.
- [ ] **CorrelationID gap in panic recovery.** `writePanicResponse` recovers RequestID from the `X-Request-ID` response header but does NOT recover CorrelationID from `X-Correlation-ID`. Same root cause — recovery runs outside `ContextEnrichmentMiddleware`. Small, bounded fix: read `X-Correlation-ID` and enrich context. Source: `docs/status/2026-07-28_14-51_panic-recovery-request-id-fix.md` §b.1.
- [ ] **Error swallowing in `Close()` methods.** All 6 `Close`/`GracefulClose` methods across `Service`, `EventSourcedSetup`, and dashboardui silently discard errors from store/bus closure instead of logging. Should use `slog.Warn`. Source: `docs/status/2026-07-22_11-38_projectionhost-adoption-self-review.md` §b.
- [ ] **MySQL event-store support.** Currently Postgres + SQLite only (dropped in v3.0.0 when `SQLEventStore` was delegated to `go-cqrs-lite/storage`, which has no MySQL dialect). `SQLSessionStore` already supports MySQL.
- [ ] **Offline sync E2E browser testing.** The SharedWorker IndexedDB queue (`sync/sync-worker.js`) is unit-verified but never exercised in a real browser. Needs a Playwright E2E test verifying the queue → retry → ACK cycle and the cross-session `rebuildAndRetry` path. The #1 deferred item across every sync status report since ADR-0029.
- [ ] **Unit tests for dedup helpers.** 10 helper functions extracted during the 2026-07-28 dedup pass lack direct unit test coverage (tested only indirectly via existing tests). Source: `docs/status/2026-07-28_10-16_deduplication-pass-self-critique.md` §c.7.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
