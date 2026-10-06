# SUPERB hardening execution — M11+M12 landed (GCL), M13–M15 not started

> **SUPERSEDED 2026-10-06 21:35:** M13–M15 are DONE, tested, and receipted; the
> three open questions below were answered by execution (proceeded gated on
> preflight; took the signature break with a Changed receipt; disabled both
> **ANNOTATED 2026-10-06 (docs-health round 18)** — fully superseded and closed by the 21:35 completion report: M13, M14, M15 all shipped, tested, and receipted (`44bc4d48`, `7d3eabb6`, `99be51de`); the three open questions below were answered by execution exactly as predicted (proceeded gated on preflight; `core.ListStreamsPaged` shipped as a CHANGELOG-Changed breaking signature; both prev and next disabled in sorted mode). Struck below: §b2, §c1–c3.
> prev and next in sorted mode). Live picture:
> [`2026-10-06_21-35_SUPERB-hardening-m11-m15-complete.md`](2026-10-06_21-35_SUPERB-hardening-m11-m15-complete.md).

**Created:** 2026-10-06 19:31 · **Scope:** execution status of
[`docs/planning/2026-10-06_14-49_SUPERB-dashboardui-metaengine-system-hardening-pareto-plan.md`](../../planning/2026-10-06_14-49_SUPERB-dashboardui-metaengine-system-hardening-pareto-plan.md)
— this session was assigned **M11–M15 only** (F35–F53). M01–M10 were NOT part of
this session and their state is unknown here (not researched, per instruction).
**Repo key:** GCL = go-cqrs-lite, CH = cqrs-htmx.

## a) Fully done

| Item | Evidence |
|------|----------|
| ~~**M11/F35 — cache ordering (GCL): invalidate-before-write + generation guard**~~ done (this session, GCL `system/cache.go` + `cache_test.go`, absorbed by daemon auto-commit ~19:05, CHANGELOG receipt authored `7d77afa33`) | ~~`Save`/`AppendBatch` invalidate before AND after the store write; `Load` repopulates only when the per-stream write generation is unchanged across its store read. Generation entries deliberately never pruned (absent must mean "never written" — pruning reintroduced the captured-0 ambiguity, caught live by the race test mid-session).~~ |
| ~~**M11/F36 — race test**~~ done (`TestCachedEventStore_ConcurrentLoadSaveNeverServesPreSaveSnapshot`, `slowLoadStore` fixture; writer asserts post-Save read contract while 4 readers straddle commits) | ~~Proven failing-first against the old code (`served pre-save snapshot: got 1 events, want 2`), green ×5 under `-race` after the fix. Full `system` suite + vet green.~~ |
| ~~**M12/F37 — reifyTo returns structured error**~~ done (GCL `system/evolutions.go` + `metaengine/store_folds.go`, absorbed by daemon ~19:17–19:30) | ~~`reifyTo(src, dst) error`; explicit-fold closure panics with `errorfamily` **Corruption** (`system.evolution.reify_failed`); metaengine's `applyFold` recover now wraps error-valued panics with `%w` so the family/code chain survives into poison + DLQ. Non-error panics keep the legacy message shape.~~ |
| ~~**M12/F38 — poison-event test**~~ done (`system/evolution_poison_test.go`: `poisonPrevEngine` test driver registered via `metaengine.RegisterDriver`, journal-seeded 3-event scenario, `WithDeadLetterStore`) | ~~Pins: DLQ entry for `poison.renamed` with `ErrorFamily=corruption` + `ErrorCode=system.evolution.reify_failed`, `IsPoisoned("poison_views")`, no `WorkerFailed`, healthy collection still projects and reads post-poison.~~ |
| ~~**M12/F39 — regression: normal evolutions unaffected**~~ done (`system/evolutions_internal_test.go` direct-assign/JSON-round-trip/nil unit tests + full existing `TestSystem_Evolution_*` suite green; full `system` + `metaengine` suites green standalone rc=0) | ~~—~~ |

GCL verification actually run this session: `system` full suite + `-race` (green),
`metaengine` full suite standalone (green, ~98s), `go vet` both modules, gofmt clean.
One metaengine FAIL seen earlier was a load flake from running two suites
concurrently — re-ran standalone, rc=0.

## b) Partially done

~~- **M12 CHANGELOG receipt (GCL):** the code+tests are committed, but the~~ done — receipt `a938279f8` (21:35 session)
  planned `[Unreleased] → Fixed` entry for the reify/DLQ fix has NOT been added
  (the commit carrying it raced the daemon twice; only M11's receipt landed).
  Content is ready to re-add.
- **M11 authored commit message:** the fix landed inside a daemon
  `chore: auto-commit` (my authored commit lost the HEAD-lock race twice);
  the CHANGELOG receipt carries the description instead. Amending during
  active concurrent work was judged unsafe.

## c) Not started (assigned to this session)

| Item | What remains |
|------|--------------|
~~| **M13/F40–F45 — escaping sweep (CH dashboardui)** | `url.QueryEscape` in `core.EventFilter.ExtraParams`, `sortState.extraParams`, `core.PaginationQuery`, `pageSizeOptionsFor`; invalid `?after=` cursor → 400+log (`handlers_events.go:40`, `handlers_audit.go:47,120`); fallback truncation/hasNext on parsed pageSize (`handlers_audit.go:134-136` — also note the early-truncate makes fallback hasNext always false); hostile-param round-trip test suite |~~ done — `44bc4d48` (21:35 session)
~~| **M14/F46–F50 — error honesty (CH dashboardui)** | errorfamily kind → 404 vs 500 in event/command/query detail handlers; `ListStreamsPaged` error propagation + error panel instead of empty table; `writeJSON` marshal-before-WriteHeader; audit failure logs at Error with detail (`handlers_dlq.go`, `handlers_snapshots.go`); tests |~~ done — `7d3eabb6` (21:35 session)
~~| **M15/F51–F53 — sorted-view notice (CH dashboardui)** | truncation chip in `events.templ`; disable next (and prev) pagination in sorted mode; README note that sorted mode is windowed (idea 9 full fix out of scope) |~~ done — `99be51de` (21:35 session)

## d) Totally fucked up

- Nothing broken. One self-inflicted detour (M11 first attempt pruned
  generation entries; my own race test immediately caught the hole and the
  design was corrected before commit) — net effect: stronger final design and
  a test that demonstrably bites.
- Environment friction, not damage: GCL's auto-commit daemon raced every
  authored commit (3 HEAD-lock failures); content is all committed, only
  authored-message readability suffered. cqrs-htmx has a concurrent session
  actively editing `usermgmt/` (staged: `es_authz_injection_test.go`,
  `es_setup.go`, `service_core.go`) — untouched by me, M13+ must not touch it
  and should run `nix run .#preflight-tree-check` before the first tree-mutating step.

## e) Suggested improvements

1. GCL daemon-race mitigation for authored commits: retry-commit wrapper
   (sleep 2–5s, re-add, commit -F file) as a documented pattern, or
   `nix run .#wait-tree-quiet`-equivalent before authored commits.
2. metaengine could expose the "fold panics with error → poison chains it"
   contract as a first-class documented pattern (currently only the system
   module uses it).
3. `makeExplicitFold` does not consume `EvolveKey(...)` — explicit folds on
   events with multiple same-typed fields fail key inference at declaration
   time (fail-fast, acceptable, but undocumented; hit while writing F38's test).
4. Add the M12 CHANGELOG receipt (b) before the next GCL release train.

## f) Next 50 (ordered, from this session's scope)

1. M12 CHANGELOG receipt (GCL `[Unreleased]`).
2. F40: `QueryEscape` values in `core.EventFilter.ExtraParams` (keys stay literal).
3. F41: escape `sort`/`dir` in `sortState.extraParams` + after/prev in `core.PaginationQuery`.
4. F42: escape extraParams inside `pageSizeOptionsFor` link building (it concatenates the same fragment).
5. F43: `?after=` parse failure → 400 + `slog` in events + commands + queries index handlers.
6. F44: fallback (non-seekable journal) path — drop the early truncation, use parsed pageSize for hasNext+truncate (commands AND queries).
7. F45: hostile-param round-trip test table (`&`, `=`, `#`, space, unicode) across filter/sort/pagination builders (core tests + handler-level link assertions).
8. F46: `errorfamily.Classify` → Rejection ⇒ 404, else 500 + error detail in event/command/query detail handlers.
9. F47: `core.ListStreamsPaged` returns error (breaking core-API change — CHANGELOG Changed entry); `renderStreamIndex` renders the error panel.
10. F48: `writeJSON` marshals to a buffer before `WriteHeader`.
11. F49: DLQ/snapshot/projection audit failures at `slog.ErrorContext` with `"error", err` (keep success at Info).
12. F50: tests — infra error ⇒ 500 body with detail; store error ⇒ visible error panel, not an empty table.
13. F51: sorted-view chip in `events.templ` (use existing `@badge` helper so the CSS bundle is untouched).
14. F52: force `HasNext`/`HasPrev` false when `sortBy.Active()` (sorted mode ignores cursors); strip bogus `TotalCount` in sorted mode.
15. F53: README note — sorted mode is a windowed view of the most recent 500 events; full fix is idea 9.
16. Regenerate `events_templ.go` via `nix run .#gen` (module-dir canonical form) after F51.
17. Run dashboardui test suite + `nix run .#lint` scoped + affected gates (`check-css-bundle-classes` only if any class changes).
18. Commit per milestone (M13, M14, M15) with CHANGELOG receipts (consumer-visible changes).
9–50. The remaining plan items beyond this session's assignment: M16–M26 per the
Pareto plan's own ordering (M16 asset caching → M17 versionz → M18 authorizer
actor → M19 Config.Validate → M20 variadic Autodetect → M21 Routes() → M22
systemadapter checkpoints/DLQ → M23 RecommendedDeployment → M24 docs truth pass
→ M25 FromSystem bridge → M26 telemetry phase 1 → M27 full battery + dual-repo
release trains, which must also carry M11/M12's GCL tags wave-ordered).

## g) Up to 3 questions

1. **Resume M13–M15 now?** The assignment is unambiguous and the research is
   done; the only hazard is the concurrent usermgmt session in this tree —
   proceed with preflight-tree-check + dashboardui-only scope, or wait?
2. **F47 signature change:** `core.ListStreamsPaged` gains an error return
   (breaking for direct `dashboardui/core` importers). OK to ship as a
   CHANGELOG-Changed breaking change, or prefer a new `ListStreamsPagedE`-style
   twin to keep the old signature stable until v5?
3. **Sorted-mode prev-link (F52):** plan says disable *next*; I intend to also
   disable *prev* (sorted mode ignores cursors entirely, so prev is equally a
   lie) — confirm or keep prev enabled?
