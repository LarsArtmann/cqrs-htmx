# Status: SUPERB hardening phase 2 complete — M05–M10 executed (dashboardui/metaengine/system wave)

**Date:** 2026-10-06 ~23:20 · **Plan:** [`docs/planning/2026-10-06_14-49_SUPERB-dashboardui-metaengine-system-hardening-pareto-plan.md`](../planning/2026-10-06_14-49_SUPERB-dashboardui-metaengine-system-hardening-pareto-plan.md) · **Predecessor:** [22-34 phase-1-complete report](2026-10-06_22-34_SUPERB-phase1-complete-phase2-m06-midedit.md)

**Lane (M01–M10): DONE.** M01–M03 (CH) + M04/M09 (GCL) receipted in the predecessor. This session finished M05–M10. The foreign session completed M11–M15 earlier (21-35 report). M16–M27 remain unscheduled.

> **ANNOTATED 2026-10-06 23:42:** the lane claim was INCOMPLETE — F22 (adttest
> cross-engine pagination conformance matrix) belonged to M06 and was never
> executed (caught in the 23-23 brutal review §f). Closed the same night:
> new `metaengine/adttest/pagination_conformance.go` (`RunPaginationConformance`
> + per-engine `CursorKey*` derivations) wired over memory, sqlite, bbolt,
> pebble, badger in each module's matrix tests — and it caught a REAL bug:
> sqlite `MapScan` tiebreaked on stored value bytes, which json v2's
> non-canonical map-key order makes impossible to rebuild from a returned
> item (the walk showed drops+dupes). Fixed: sqlite `MapScan` now SELECTs the
> key column (mirrors pgengine); the F21 sqlite ties test's json-round-trip
> cursor derivation was replaced with the raw map key. Everything else on
> this page stands.

## Executed this session (all GCL, all verified + receipted)

| Task | Change | Evidence |
| --- | --- | --- |
| M06 | `SortKeyCursor{Sort, Key}` compound keyset cursor in `SortPaginate` — AND the discovery that 6 of 9 engine `MapScan`s (memory, sqlite, pg, mysql, duckdb, dgraph) carried their own inlined copies of the tie-lossy filter: all delegated to the one shared core (semantics-preserving: byte-key tiebreak ≡ `strings.Compare`; memory/sqlite keep deterministic key order when no sortFn) | Tie-heavy tests at 3 levels: unit (limits 1–30 incl. tie-straddling), memory-engine `MapScan`, sqliteengine `MapScan` ~~(cursor key = canonical JSON bytes)~~ (cursor key = raw map key — the JSON-bytes derivation was unsound per json v2's non-canonical map order; superseded same night by the F22 fix, see annotation above); legacy value-cursor semantics PINNED as documented-lossy. Commits `7584b7a84`+`8a7a1f19a`+`1c36bb6dc` (daemon), CHANGELOG receipt |
| M07 | `fail(err)` teardown helper — all 15 `New` error returns after engine creation now close projHost + engines + closers (order mirrors `Close`; teardown errors join AFTER the cause); event bus registers as closer BEFORE `buildPublisher` (stranded-bus window closed) | `TestSystem_NewErrorPath_ClosesCreatedEngines` (goleak-bait heartbeat engines) + `...ClosesEventBus`; suite-wide goleak. Commit `77076688b`, receipt `1336857e8` |
| M08 | `Close` stops timers (was: never touched them — only GracefulClose did); `stopTimers` now actually WAITS (WaitGroup; was: cancel-and-hope despite its doc comment); doc comments made honest; split `stopTimersLocked` for Close's critical section | `TestSystem_CloseStopsTimersAndWait`; system suite green under `-race`. Commits `9ce1367c4`/`143cbd589`, receipt `9477b865e` |
| M10 | `ErrUnknownInstanceRole` (typo'd roles were SILENTLY ignored — no store bound, no error); `sot.implicit_memory` ADVISORY on the implicit memory event-store fallback; `ErrUnknownEngine` messages list configured engine names + registered drivers (both paths) | 4 tests in `role_validation_test.go`. Authored commit `5688d848f` (clean, no race) |
| on-sight | `metaengine.RegisteredDrivers()` now sorted — the M07 tests' global driver registrations exposed `Explain()` topology nondeterminism (`TestSystem_WiringDeterministic` red under `-race`) | Race suite green after |

## The 3 open questions — resolved by documented autonomous default

1. **flightrecorder v0.2.1 bump train + push:** NOT run — push is the owner's call; the lag-train blocker stands as documented in the 22-34 report.
2. **M06 issuance scope:** plan-faithful (F20 comparator + F21 tests only). The full `NextCursor` protocol (engines fill it, `ScanPage` prefers it, `ParseCursor` normalizes the round-tripped JSON form) is queued in **GCL's TODO_LIST** ("Compound-cursor issuance (M06 tail)") with the design notes. Rationale: unsolicited wire-protocol change in a library vs. a bounded plan contract.
3. **Resume order:** tree state inspected first — the mid-edit M06 content HAD survived (daemon committed it as `7584b7a84` while the report was being written).

## Final battery (all green)

- GCL: metaengine, system, sqliteengine, systemtest suites (`-count=1`); system also `-race`; all 8 engine modules build+vet (workspace mode).
- CH: `preflight-tree-check` OK → workspace build rc=0; tests green on `.`, `systemadapter`, `dashboardui`, `setup`, `integration_test` (workspace mode = local GCL replaces active).
- api_surface: 7553 → **7557** exports (M06 `SortKeyCursor` + fields; M10 `ErrUnknownInstanceRole`), regenerated + verified by pre-commit.

## Honest-failure record

- Daemon commit races ×2 this session (M06 content, M07 files) — content safe in heuristic commits; one authored commit (`5687...`→ actually `5688d848f` M10) landed clean between daemon polls.
- The M07 test's first draft had a never-stopping heartbeat goroutine (would have failed goleak on SUCCESS) and a wrapper that hid the memory engine's backend methods (interface embedding promotes only the interface's own methods) — both caught by running the tests, both fixed.
- sqliteengine hermetic (`GOWORK=off`) build is red on PRE-EXISTING family-pin drift (claiming v4.0.2 / record v4.6.2 / dedup v4.2.4 / id v4.7.1 vs pinned) — same class as the CH alignment wave; belongs to the next release train, not this lane. Flagged, not fixed.

## Next steps (owner or next session)

1. flightrecorder v0.2.1 lag train → push both repos (CH ~31 ahead, GCL ~40 ahead, all committed).
2. M16–M27 of the plan (unscheduled).
3. GCL TODO_LIST: compound-cursor issuance; the engine-module hermetic pin drift.
