# SUPERB Hardening M11–M15 — COMPLETE (status report)

**Date:** 2026-10-06 21:35 · **Scope:** M11–M15 (findings F35–F53) of [`docs/planning/2026-10-06_14-49_SUPERB-dashboardui-metaengine-system-hardening-pareto-plan.md`](../../planning/2026-10-06_14-49_SUPERB-dashboardui-metaengine-system-hardening-pareto-plan.md) · **Supersedes:** [`2026-10-06_19-31_SUPERB-hardening-m11-m12-done-m13-m15-pending.md`](2026-10-06_19-31_SUPERB-hardening-m11-m12-done-m13-m15-pending.md)

**Verdict: M11, M12, M13, M14, M15 all DONE, tested, committed, and receipted.** The session scope (M11–M15) is complete. GCL = go-cqrs-lite repo; CH = cqrs-htmx repo.

> **ANNOTATED 2026-10-06 (docs-health round 18)** — the "not done" column closed the same night: **M01–M10 all shipped** by the sibling lane (22:34 report §a: M01–M03 CH + M04/M05/M09 GCL; 23:20 report: M05–M10), verified + receipted both repos. Struck below: §2-M01–M10, §5 items 1–10, §6-Q1. STILL OPEN: M16–M27 unscheduled (TODO_LIST P1 progress note), the flightrecorder v0.2.1 lag train (the standing push blocker, TODO_LIST), §5.22–24 gotcha-12 amendment + quarantine cleanup + forensics (owner §6-Q3), §6-Q2 release-train versioning call (rides the next train).

---

## 1. Done (evidence per milestone)

| Milestone              | What shipped                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | Verification                                                                                                              | Commits                                                                                        |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| **M11** (F35–36) — GCL | `system/cache.go`: invalidate-before-write + never-pruned per-stream generation guard on `CachedEventStore`; race test (`slowLoadStore`, 4 readers × 100 saves, `-race`) proven failing-first; `Save`-error contract change receipted                                                                                                                                                                                                                                                                                                                                                                                                                                              | `-race -count=5` green; full system suite green                                                                           | `7d77afa33` (receipt) + daemon commits                                                         |
| **M12** (F37–39) — GCL | `system/evolutions.go`: `reifyTo` returns errors; explicit-fold closure panics with an `errorfamily` Corruption VALUE (`system.evolution.reify_failed`, names evolution+event); `metaengine/store_folds.go` recover preserves the chain (`%w`) → DLQ entries carry real family/code; poison test (mis-typed stored state → DLQ `corruption`, worker survives, healthy collections keep projecting) + `reifyTo` unit tests                                                                                                                                                                                                                                                          | system + metaengine suites green standalone (re-verified 21:33 on the repaired cache)                                     | `a938279f8` (receipt) + daemon commits                                                         |
| **M13** (F40–45) — CH  | URL-escaping at value construction: `core.EventFilter.ExtraParams`, `sortState.extraParams`, sort-header hrefs, `core.PaginationQuery` (after/prev; comma-safe). Malformed `after` cursor → 400 + warn-log (was silent page reset) on events/commands/queries. ReadAll-fallback: truncate-after-hasNext with the PARSED page size (was: hidden Next link; queries used config default). Hostile-value round-trip tests (`&`,`=`,`#`, space, unicode) at builder + handler level (events whose TYPE is the hostile value)                                                                                                                                                           | dashboardui suites green ×3; one stale core test expectation updated (`prev=abc%2Cdef`)                                   | `44bc4d48` (receipt) + daemon commits                                                          |
| **M14** (F46–50) — CH  | `renderLoadError`: family-mapped 404 (Rejection = not found) vs 500 (infra, cause logged) for event/command/query detail. **Root cause fixed:** `core.loadEventFromAll` returned Infrastructure + "no event source available" for not-found. `core.ListStreamsPaged` → error-returning signature (breaking; family-preserving wrap — Rejection→400 at stream pages, else 500 panel; nil page = error). `writeJSON` marshals before WriteHeader (500+`marshal_failed` on failure). Audit failures (dlq replay/delete/purge, snapshot delete, projection reset) → `slog.ErrorContext` + `error` attached. Tests: writeJSON 500, infra→500 bodies, rejection→400, streams error panel | dashboardui suites green ×3                                                                                               | `7d3eabb6` (receipt; also re-applied M13 receipts lost to a concurrent stale-buffer overwrite) |
| **M15** (F51–53) — CH  | Sorted-only events view = honest window: shows the ENTIRE `FilterScanLimit` (500) window with a "sorted view: first 500 events" badge; pagination bar + page-size selector + "of Z" total hidden (cursors can't follow a re-sorted order — the old Next link reloaded the same window forever, a real bug, not just honesty). Filter+sort keeps cursor pagination. README § Sorting documents window semantics + idea-9 link. `nix run .#gen` regenerated `events_templ.go`                                                                                                                                                                                                        | `sorted_view_test.go` green (window, no cursor links, no pagination bar; filter+sort still paginates; unsorted paginates) | `99be51de` (receipt)                                                                           |

**Final battery (21:33–21:36):**

- CH `nix run .#test` (full workspace, every go.work member): **rc=0, all 28 modules green** on the repaired shared cache.
- GCL `system` + `metaengine` suites standalone: **rc=0**.
- CH dashboardui `go build` + `go vet` (GOWORK=off): green.
- Scoped treefmt on every touched file: 0 changes.
- `nix run .#lint`: running at report time (background).

## 2. Not done / out of scope (this session)

- **M01–M10** (SSE XSS, AccentColor validation, metaengine FilterOp/jsonPath injection, bus targets, mutex coverage, keyset pagination, engine cleanup, timers, InstanceRole): NOT STARTED by this session. A concurrent session appears active on the M01-class work (`dashboardui/layout.go` SSE/event-count changes + untracked `layout_sse_xss_test.go` appeared mid-session).
- **M16–M27** (asset ETags, versionz guard, audit actor, Config.Validate, Autodetect variadic, Routes manifest, systemadapter options/presets, docs truth pass, telemetry panels): NOT STARTED. The plan's Execution-status table carries the live picture.

## 3. Fucked up / incidents hit and resolved this session

1. **Shared Go cache corruption (infrastructure, not code):** `/mnt/buildcache` carried 323 zero-byte download-cache entries + 8,487 zero-byte extracted `.go` files + poisoned `go-build` action-cache entries (the disk-full era per AGENTS gotcha 12; every "checksum mismatch → same bogus hash" and "expected 'package', found 'EOF'" was this). Repaired: quarantined corrupt entries to `/mnt/buildcache/go-mod/.quarantine-corrupt-20261006/` (recoverable, ~215 module dirs; go re-extracts on demand), then `go clean -cache` on the action cache. Verified: full-workspace test battery green against the repaired cache. **Note for the fleet: the quarantine dir can be deleted once confidence holds.**
2. **Lost-update on receipts:** a concurrent session's stale editor buffers overwrote my in-flight `dashboardui/CHANGELOG.md` + plan-file edits (daemon commit `a0ba8067` carried the reversion). Detected by assertion, re-applied my authored content in `7d3eabb6`, committed fast to close the window.
3. **BuildFlow pre-commit failed (51 steps) in bare shells** — the documented deterministic environment class (AGENTS gotcha 8). Used the sanctioned `--no-verify` fallback with step names + independent verification (tests green, fmt clean) in every receipted commit message.
4. **Commit `44bc4d48` swept in one foreign file** (`dashboardui/layout.go`, concurrent session's coherent SSE/event-count work) staged by the failed first commit attempt. Left in place per the never-revert-foreign-work rule; noted here for attribution.
5. **Test-writing missteps (self-caught):** two fallback tests initially panicked on `MustNew` validation (needed a Journal); one core test pinned the pre-escaping `prev=abc,def`; M14's invalid-cursor tests assumed strict `ParseStreamID` (it is lenient — only empty fails) — replaced with a rejection-family reader test that pins the actual 400 mapping.

## 4. Improvements made beyond the strict plan text

- `core.loadEventFromAll` not-found family fix (root cause behind F46, one layer deeper than the plan looked).
- `ListStreamsPaged` family-preserving wrap (a reader's own classification survives to the HTTP layer) — the plan only asked for "propagates errors".
- M15 shows the whole window instead of pageSize-of-the-window (the plan's badge text only makes sense that way; also kills the truncate-then-Next-loop bug).
- Cache-repair runbook knowledge captured in this report (§3.1) — candidate for AGENTS.md gotcha 12 amendment.

## 5. Top of the queue (next ~50, ranked; first 10 actionable now)

**Immediate (high, exploit/correctness class — the plan's own 1% tier):**
~~1. M01 (idea 1+54): SSE row-injection XSS → `textContent` row builder (concurrent session may be mid-flight — coordinate via tree state).~~ done — 22:34 session (`textContent` row builder + Stream Type cell + count reset + hostile-payload smoke)
~~2. M02 (idea 2): AccentColor strict CSS-color validation (style-tag breakout).~~ done — 22:34 session (`accent_color.go` closed-grammar validator, `fbd044f0`)
~~3. M03 (ideas 123+124): metaengine FilterOp enum + jsonPath identifier validation at construction AND SQL-build (GCL).~~ done — 22:34 session (`ValidateFilterSpecs` + `ValidateIdentifier` at Scan entry + all builders)
~~4. M04 (idea 230+192): honor single `publish: [x]` + validate bus targets (GCL).~~ done — 22:34 session (publish fan-out + `ErrUnknownPublishTarget`)
~~5. M05 (idea 4): CSV formula neutralization (`=`,`+`,`-`,`@` prefixes) in dashboardui exports.~~ done — 22:34 session (OWASP neutralizer in `writeCSV`)
~~6. M06 (idea 114+137): memory-engine mutex coverage + concurrent conformance (GCL).~~ done — 23:20 session (RWMutexes + `AssertConcurrent*` helpers)
~~7. M07 (idea 115): keyset pagination compound `(sortValue,key)` cursor comparator (GCL).~~ done — 23:20 session (`SortKeyCursor` compound filter + 6 engine delegations; issuance queued in GCL TODO)
~~8. M08 (idea 187+242): `system.New` engine cleanup on all error paths + goleak test (GCL).~~ done — 23:20 session (`fail(err)` teardown + goleak tests)
~~9. M09 (idea 210+211): `Close()` stops timers; `stopTimers` waits (GCL).~~ done — 22:34 session (Close stops timers + waits)
~~10. M10 (idea 179+197): reject unknown InstanceRole; ADVISORY on silent memory fallback (GCL).~~ done — 23:20 session (`ErrUnknownInstanceRole` + `sot.implicit_memory` ADVISORY + engine-name hints)

**Next wave (consumer trust, M16–M24 as planned):**
11. M16: content-hash ETags for dashboard.css/js (kill hardcoded v4.9.0). 12. M17: versionz opt-in guard + buildinfo. 13. M18: Authorizer returns actor; audit entries carry actor+correlation. 14. M19: export `Config.Validate()` + single-source page-size constants. 15. M20: variadic `Autodetect`. 16. M21: `Routes()` manifest. 17. M22: systemadapter checkpoint/DLQ wiring. 18. M23: `RecommendedDeployment` presets. 19. M24: docs truth pass (deprecated-path guides, README fixes, IMPROVEMENT_IDEAS prune).
20. Sorted-view full fix (research idea 9): server-side sorted pagination — retires the M15 window workaround.
21. Idea 251 follow-up: DLQ replay of poison entries after evolution-schema repair (reifies cleanly now that errors are structured).

**Hygiene/fleet:**
22. Amend AGENTS gotcha 12 with the zero-byte-cache signature + repair runbook (quarantine → `go clean -cache` → verify). 23. Delete `.quarantine-corrupt-20261006` once stable. 24. Investigate the truncating writer (disk-full era or crashed session) — 8.5k empty files don't appear from nowhere. 25. Pin the go-flightrecorder v0.2.1 train-lag alignment (16 requires, flagged at commit time). 26–50: the plan's M25–M27 + backlog buckets (twin extraction ADR, testing matrices, error-message DX sweep, telemetry panels phase 1: Topology/HealthCheckDetailed/plan-table/engine-stats cards).

## 6. Three questions

~~1. **M01 ownership:** the concurrent session's `layout_sse_xss_test.go` suggests it took M01 — confirm before anyone else starts it (two sessions editing `layout.go`/`dashboardJS` will collide exactly like §3.2).~~ resolved — the sibling lane took M01–M03 + M04/M05/M09 (22:34) and M05–M10 (23:20); no collision
2. **Release train:** M13–M15 touch published dashboardui module code incl. a breaking `core.ListStreamsPaged` signature — should the next train be **v4.14.0** (minor, breaking-within-v4 per house style), and does M11/M12 GCL code ride the same wave?
3. **Cache forensics:** do you want a root-cause hunt for the mass-truncation event (§5.24), or is disk-full-era attribution sufficient to close it?

---

> Point-in-time snapshot per `docs/status/README.md`. The plan file's Execution-status table is the live tracker.
