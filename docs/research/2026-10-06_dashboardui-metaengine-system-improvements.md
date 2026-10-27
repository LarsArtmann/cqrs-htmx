# Improvement Ideas: dashboardui + metaengine + system — 2026-10-06

Cross-module study of three codebases — **note the scope split: 139 of the 308 ideas target `go-cqrs-lite`** (Parts B/C), the rest `cqrs-htmx`; the report lives here because the study was commissioned from this repo:

- `cqrs-htmx/dashboardui` (observability dashboard, templ+HTMX)
- `go-cqrs-lite/metaengine` (cost-based storage planner, 12 engines)
- `go-cqrs-lite/system` (ADR-0123 composition root)
- plus the integration seams between them (`cqrs-htmx/systemadapter`, `dashboardui.Autodetect`, docs)

**308 ideas, globally numbered** so they are referenceable (`idea 217`). Evidence cites `file:line` where the finding is location-specific. Prior art consulted: `2026-09-17_templ-components-dashboardui-deep-dive.html` (UI only), `storage-first-principles-analysis.md`, ADR-0149 (durable checkpoints shipped upstream). Not duplicated: dashboardui `IMPROVEMENT_IDEAS.md`/`ROADMAP.md` open items (noted as stale where CHANGELOG contradicts them).

| Part | Scope                                                                                            | Ideas   |
| ---- | ------------------------------------------------------------------------------------------------ | ------- |
| A    | dashboardui — security, perf, pagination, features, a11y/i18n, SSE, testing, theming             | 1–82    |
| B    | metaengine — API, planner/cost model, engine duplication, correctness, vector/temporal/graph/SSE | 83–175  |
| C    | system — config, validation, defaults, lifecycle, health, bus/cache/snapshots, observability     | 176–262 |
| D    | Integration seams — telemetry panels, systemadapter, missing bridges, docs drift                 | 263–308 |
| —    | Priority shortlist (Top 30, impact × urgency, effort-sized)                                      | end     |

## Verification status

Findings were produced by four parallel deep-dive sub-agents; the headline claims were then spot-verified directly against source (2026-10-06) — all passed:

| Idea | Claim                                                                   | Verified at                                                             |
| ---- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| 1    | SSE rows built via `innerHTML` without escaping                         | `dashboardui/layout.go:379-381`                                         |
| 2    | Accent injected via `templ.Raw("<style>…")` with only HTML-escaping     | `dashboardui/layout.templ:76-78`                                        |
| 6    | Hardcoded ETag `dashboardui-v4.9.0` (module is v4.13.x)                 | `dashboardui/assets.go:43`                                              |
| 9    | Active sort resets cursor, loads `filterScanLimit` (500)                | `dashboardui/handlers_events.go:48-51`                                  |
| 114  | Vector ops delegate straight to the index, bypassing `m.mu`             | `metaengine/memory_engine.go:248-256`                                   |
| 123  | `WithFilter` appends `FilterSpec` without validating `op`               | `metaengine/scan_options.go:31-35`                                      |
| 175  | `Doctor(ctx) string` is text-only                                       | `metaengine/explain.go:255`                                             |
| 187  | Error returns after the engine-creation loop never close engines        | `system/constructor.go:128-136` (1 of 12+ paths)                        |
| 210  | `stopTimers` has exactly one call site (GracefulClose), none in `Close` | `system/timers.go:77-78`, `system/system.go:310`                        |
| 230  | Fan-out built only when `len(inst.Publish) > 1`                         | `system/bus.go:69`                                                      |
| 288  | systemadapter compiles against deprecated `usermgmt.*` re-exports       | live gopls: 65 deprecation hints in `domain_config.go`/`projections.go` |

Everything else is agent-sourced: re-verify at the cited `file:line` before acting on it, and treat unverified line numbers as approximate.

---

## Part A — dashboardui (ideas 1–82)

### A.1 Security (1–8)

1. **[security] XSS in SSE row injection** — `dashboardJS` builds live event rows via `row.innerHTML = ... data.type ... data.streamId ...` with zero escaping (`layout.go:377-381`). A malicious event type/payload executes on inject. Fix: `document.createElement`/`textContent`.
2. **[security] CSS injection via `AccentColor`** — HTML-escaped but injected into a raw `<style>` tag (`layout.templ:76-78`); `html.EscapeString` does not neutralize `;{}`. `AccentColor: "red;} body{background:url(//evil)"` breaks out. Validate as strict CSS color in `withDefaults` (`config.go:134`).
3. **[security] No Origin/Referer defense-in-depth on write POSTs** — `guard` (`dashboard.go:128-142`) only calls `Authorizer`; cheap `Origin` check when `!ReadOnly`.
4. **[security] CSV formula injection in exports** — event/command types flow unneutralized into CSV (`export.go:79-88`); values starting with `=`,`+`,`-`,`@` execute in Excel. Prefix `'`.
5. **[security] `versionz` leaks config unauthenticated** (Title, BasePath, capabilities, Go version — `handlers_health.go:49-58`); guard or opt-in.
6. **[security] Stale hardcoded ETag** — `assets.go:43` pins `dashboardui-v4.9.0` while the module is v4.13.x (already caused the v4.10.1 stale-CSS incident). Derive from content hash at `New`.
7. **[security] `dashboard.css`/`dashboard.js` served `max-age=86400` with no ETag/versioned URL** (`layout.go:62-77`) — 24h stale assets after upgrades; only the Tailwind bundle got the asset-handler treatment.
8. **[security] Destructive ops use only native `confirm()`** (`dashboardJS:435-440`) — no typed confirmation for Replay All/Purge All, no step-up auth hook.

### A.2 Performance (9–16)

9. **[perf] Sorting silently kills pagination** — sorted view resets cursor and loads 500 events (`handlers_events.go:48-51`); page 2 of a sorted view is unreachable, truncation not surfaced.
10. **[perf] `FetchOverview` fallback `ReadAll` loads whole journal to count events** (`core/overview.go:174-193`); same in `LoadRecentEvents` (`core/events.go:100-112`). Cap or TTL-cache.
11. **[perf] `FindEventNeighbors` scans 500 events per detail render** (`core/events.go:270-307`); prev/next silently vanish past position 500.
12. **[perf] Missing-event detail = full journal scan** — `loadEventByJournalScan` pages the entire journal (`core/events.go:212-243`) to render a 404.
13. **[perf] `DLQProjectionLinks` materializes every projection's DLQ entries just to count them** (`core/dlq.go:28-32`); errors silently yield 0. Add a `Count`/`Len` interface.
14. **[perf] Export not streaming** — rows fully materialized (`export.go:77-105`), reader errors dropped (`handlers_audit.go:28-31` `cmds, _ =`), `writer.Write` errors ignored (`export.go:47-53`); broken store exports an empty 200 CSV.
15. **[perf] Non-seekable command/query fallback truncates `ReadAll` in memory per page view** (`handlers_audit.go:59-63,132-137`).
16. **[perf] `dlqEntryDetailHandler` lists all entries then linear-scans for one** (`handlers_dlq.go:47-62`).

### A.3 Error handling (17–21)

17. **[errors] Infrastructure failures rendered as 404** — `eventDetailHandler` (`handlers_events.go:93-98`), `commandDetailHandler` (`handlers_audit.go:174-179`) map every load error to "not found"; use errorfamily kind to pick 404 vs 500.
18. **[errors] `ListStreamsPaged` swallows `StreamReader.List` errors** — renders empty table indistinguishable from "no data" (`core/events.go:329-332`).
19. **[errors] `writeJSON` marshals after `WriteHeader`** (`handlers_health.go:71-84`) — marshal failure appends to an already-sent 200. Marshal first.
20. **[errors] Invalid `?after=` cursor silently ignored** (`handlers_events.go:40`, `handlers_audit.go:47,120` — `afterID, _ := id.ParseEventID`) — corrupted cursor quietly resets to page 1. 400 or log.
21. **[errors] Error-path audit logging at Info level, error detail dropped** (`handlers_dlq.go:87-96`, `handlers_snapshots.go:59-70`).

### A.4 Pagination / sorting (22–27)

22. **[pagination] `prev` cursor stack grows unboundedly in the URL** (`core/pagination.go:24-49`) — deep paging busts URL limits; opaque encoded state instead.
23. **[pagination] `ComputePageStart` assumes uniform page size through history** (`core/pagination.go:75-91`) — changing `?limit=` mid-traversal corrupts "Showing X–Y of Z"; unguarded.
24. **[pagination] Filter/sort params concatenated without `url.QueryEscape`** — `EventFilter.ExtraParams` (`core/events.go:49-68`), `sortState.extraParams` (`sort.go:27-33`), `PaginationQuery` (`core/pagination.go:53-71`), `pageSizeOptionsFor` (`pagination.go:50-73`): stream types containing `&`, `=`, `#` break links. Real correctness bug.
25. **[pagination] Commands/queries fallback truncates with `config.PageSize` while `hasNext` uses parsed `pageSize`** (`handlers_audit.go:134-136` vs `61-63`) — `?limit=200` miscounts pages.
26. **[sorting] Commands/queries/aggregates/snapshots tables have no sort/filter** (`audit.templ:25-31`, `aggregates.templ`) — generalize `sortState`/`eventSortHeader` (`sort.go:99-119`) into a reusable column spec.
27. **[pagination] `pageSizeOptions` {25,50,100,200} and `maxPageSize=200` duplicated in three places** (`pagination.go:9`, `config.go:21-22`, `core/capabilities.go:28-29`) — expose via Config, single-source.

### A.5 Missing features (28–38)

28. **[feature] No global "jump to ID" box** — paste an event/command/stream ID in the header to deep-link; every panel already has a detail route.
29. **[feature] No time-window/date-range filter on events** — `EventFilter` supports only exact type/streamType/streamID (`core/events.go:19-23`); `OccurredAt` range is the most natural ops query.
30. **[feature] Filter inputs free-text with placeholder hints** (`events.templ:77-79`) — populate `<datalist>` from distinct observed types (the 500-event scan already happens).
31. **[feature] Overview stat cards never live-update** — stable ValueIDs exist "so live-updating scripts survive refactors" (CHANGELOG v4.10.0) but `dashboardJS` only refreshes `#projection-health` (`layout.go:364-366`); wire `dashboard:event` to the cards.
32. **[feature] No single dead-letter replay** — DLQ entry detail (`handlers_dlq.go:41-77`) offers no replay-one; only Replay All (`handler.go:101`).
33. **[feature] No export for DLQ/snapshots/projections** — export.go covers events/commands/queries only; ops need a DLQ dump most.
34. **[feature] No NDJSON/streaming export variant** — `?format=ndjson` suits `SeekableJournal` without the 10k cap (`export.go:15`).
35. **[feature] No projection lag sparkline** — ROADMAP blesses `display.Sparkline` as the Tier-4 exception; lag history is ring-bufferable in `projectionStat` (`core/overview.go:34-44`).
36. **[feature] Aggregate browser lacks search** — `/aggregates` is pure cursor listing (`render.go:146-156`); stream-type filter + ID prefix search are cheap.
37. **[feature] Event payload viewer is flat `<pre>`** (`events.templ:125`) — collapsible JSON tree / key-path copy; gjson is already in the module graph.
~~38. **[feature] No route manifest** — `Routes() []Route` would let consumers print active panels and aid auth-middleware allowlisting.~~ CONSUMED — M21 shipped `Routes()` (route manifest ≡ registration pinned by a `mux.Handler` drift test); `docs/status/2026-10-07_02-57_SUPERB-m20-m27-session-m20-m26-done-battery-80pct-foreign-interference.md`.

### A.6 API design / DX (39–46)

39. **[API] Triple-maintenance Config seam** — every new data source must be added to `Config` (`config.go:29-116`), `core.Config` (`core/capabilities.go:43-57`), `coreConfig()` mapping (`core_bridge.go:115-131`), and `Autodetect` (`autodetect.go:42-84`). Embed `core.Config` to collapse three of four.
40. **[API] `Authorizer` returns only `error`** (`config.go:101`) — audit logs record op/projection but never WHO (`handlers_dlq.go:103-116`). Change to return an actor, or read actor from context.
41. **[API] No exported `Config.Validate()`** — consumers (setup/v4) can't pre-validate without constructing a Dashboard; export a wrapper over `withDefaults` (`config.go:118`).
~~42. **[API] `Autodetect` can't combine sources** — single `store any` can't express "SQL store + separate projection host + bus"; accept variadic sources merging probes (`autodetect.go:39`), make the probe table-driven to drop the `//nolint:cyclop`.~~ CONSUMED — M20 shipped `Autodetect(sources ...any)` first-wins merge + table-driven `fieldProbe[T]` probes (cyclop chain deleted); 02-57 report.
43. **[API] `PageMeta.HTMX` is dead weight** — documented as "normally never sees HTMX=true" (`layoutfunc.go:80-82`); populate honestly or remove.
44. **[DX] `modulePath` hardcoded** (`handlers_health.go:69`) — forks lie in versionz; use `runtime/debug.ReadBuildInfo()`.
45. **[DX] No `Example*` godoc functions** for `New`/`Autodetect`/`core.FetchOverview` — `core` is advertised as a public headless API (`core/capabilities.go:1-12`).
46. **[DX] Write-op redirect target ignores pagination/filter state** — `dlqDeleteHandler` redirects to page 1 (`handlers_dlq.go:168`) even from page 3.

### A.7 i18n / a11y (47–53)

47. **[i18n] Hardcoded English, no localization seam** — `lang="en"` (`layout.templ:25`), "Showing X–Y of Z" (`pagination.go:31-45`), "just now"/"ago" (`core/format.go:11-21`), nav labels (`config.go:172-200`). Add `Config.Labels`/Localizer.
48. **[i18n] `truncate` byte-slices** (`handler_overview.go:52-57`) and `initials` byte-indexes words (`layoutfunc.go:48-59`) — mid-rune splits = mojibake for non-ASCII.
49. **[a11y] Timestamps inconsistent** — events/audit lists are raw text (`events.templ:54`, `audit.templ:38`), overview uses `title=` (`overview.templ:97`); stray `</time>` artifact pinned in `testdata/golden/snapshot_detail_page.golden:4` (from `snapshots.templ:47`).
50. **[a11y] `aria-live` on both `sse-status` and `sse-count`** (`layout.templ:126-127`) — announces every event, spamming screen readers; count should be `aria-live="off"` or debounced.
51. **[a11y] Mobile drawer has no focus trap/ESC** (`dashboardJS:386-406`) — tab order escapes the "modal"; `aria-modal`/focus return absent.
52. **[a11y] Unlabeled "Actions" column** — `plainHeaders(..., "")` renders an empty `<th>` (`tables.go:9-22`, `projections.templ:35`, `audit.templ:26,62`); use a visually-hidden label.
53. **[a11y] Sort affordance inconsistent** — commands/queries/aggregates headers `Sortable:false` with no explanation.

### A.8 SSE / realtime (54–58)

54. **[SSE] Injected live row has wrong column count** — JS builds 2–4 `<td>`s against the 5-column events table, omitting Stream Type (`layout.go:379-381`); table misaligns after the first live event.
55. **[SSE] `eventCount` never resets** (`dashboardJS:294,307-309`) — the "N events" badge counts session-wide, misleading after navigation.
56. **[SSE] `handleEvent` swallows JSON parse errors silently** (`dashboardJS:318`); no client logging of dropped frames.
57. **[SSE] No client cap/backpressure config** — `Broadcaster` created bare (`dashboard.go:68`); expose buffer size/max subscribers to prevent fan-out amplification from N tabs.
58. **[SSE] Embedded (`Layout`) mode docs don't cover the toast-container/live-region requirements** for consumer shells (`layoutfunc.go:30-31` is the only hint).

### A.9 Testing (59–64)

59. **[testing] Assertion style is string-containment almost everywhere** (`a11y_test.go:43-53`, `handlers_security_test.go:37-43`) — misses attribute-order/nesting regressions; add DOM-parsing assertions (goquery) alongside goldens.
60. **[testing] The `dashboardJS` innerHTML path has zero test coverage** — no JS execution test at all; a jsdom/playwright smoke would have caught ideas 1 and 54.
61. **[testing] No tests for URL-escaping of filter values** (idea 24) — only happy-path params covered.
62. **[testing] Page-level goldens cover only 3 pages** (`testdata/golden/`) — projections/DLQ/time-travel markup unpinned; harness ready, unused.
63. **[testing] No negative-capability matrix** — one table test over capability combinations × expected routes would pin `routes()` (`handler.go:43-145`) exhaustively.
64. **[testing] No context-cancellation tests** — `New` bridge, SSE replay (`sse_replay_test.go`), scans (`core/events.go:212-243`).

### A.10 Code quality (65–70)

65. **[quality] `layout.go` is 487 lines carrying ~200 CSS + ~200 JS as string constants** (`layout.go:79-486`) while the Tailwind bundle already uses `embed` (`assets.go:12-13`) — move to `assets/` for linting + content-hash ETags.
66. **[quality] `monoSpan()` builds raw HTML by string concat** (`detail_items.go:39-41`) — post-templ foot-gun; every caller must remember `esc()`.
67. **[quality] String-typed icon names mapped through `mapNavIconName`** (`config.go:161`, `layoutfunc.go:23-46`) — use `icons.Name` directly; the string enum is pre-templ archaeology.
68. **[quality] Commands/queries index handlers ~150 lines of near-duplicate code** (`handlers_audit.go:18-160`), both `//nolint:cyclop` — a generic `journalIndex[T]` deletes half.
69. **[quality] Duplicated constants + 6 pure pass-through wrappers** — `defaultPageSize`/`maxPageSize` (`config.go:21-22` vs `core/capabilities.go:28-29`), `core_bridge.go:59-105`.
70. **[quality] Three parallel kind→display mappings** — `statusKindToStatus` + `statusKindToBadgeType` + `healthKindToTone` (`handler_overview.go:39-50`, `badges.go:9-20`, `stats.go:8-19`) — unify into one table.

### A.11 Self-observability (71–74)

71. **[observability] No request logging/metrics of its own** — an observability tool that doesn't observe itself; add slog request log + optional Prometheus endpoint.
72. **[observability] `readyz` never probes the data source** (`handlers_health.go:23-46`) — a dead store still reports ready; add `ReadinessProbe func(ctx) error` to Config.
73. **[observability] `versionz` lacks build commit/time** (`handlers_health.go:60-67`) — `debug.ReadBuildInfo` VCS info is free.
74. **[observability] Audit entries lack request/correlation IDs and actor** (see idea 40) — can't correlate `dashboardui.audit` events to consumer request logs.

### A.12 Docs (75–79)

~~75. **[docs] README references a nonexistent demo** — `examples/dashboard-demo/main.go` (`README.md:359`, CHANGELOG v4.2.0) does not exist; IMPROVEMENT_IDEAS T24 claims it's open while CHANGELOG says shipped.~~ OBSOLETE-ON-VERIFICATION — the demo EXISTS (8 users Alice..Henry, orders, projection host, 5s publisher); README demo section kept; 02-57 report §c.
~~76. **[docs] README documents the deleted `data-copyable` protocol** (`README.md:310`) — removed in v4.10.0 for library CopyButtons.~~ CONSUMED — dead Copy-to-Clipboard README section removed after verifying `data-copyable` exists only in docs; 02-57 report §c.
~~77. **[docs] IMPROVEMENT_IDEAS.md contradicts CHANGELOG** — lists shipped features (T12/T13/T17/T21/T22/T24) as open; prune + add a CI check striking items whose T-numbers appear in CHANGELOG.~~ CONSUMED — pruned with per-item verification (six T-items struck as SHIPPED with evidence; T18 verified genuinely open and kept; stale-item check note added to the header; CI check not added); 02-57 report §c.
~~78. **[docs] `core/capabilities.go:9-11` package doc still says "fmt.Fprintf-based handlers"** — strings.Builder layer deleted 2026-09-22.~~ CONSUMED — the stale claim is dropped from the package doc; 02-57 report §c.
79. **[docs] No SECURITY.md for write mode** — ReadOnly=false checklist scattered across `config.go:91-93`, `dashboard.go:51-57`, README.

### A.13 Theming (80–82)

80. **[theming] Two parallel styling systems** — semantic custom CSS (`dashboard.css`) + Tailwind utilities (`dashboard-tw.css`); finish the migration or document which layer owns what.
81. **[theming] No `forced-colors`/high-contrast handling** — only reduced-motion + dark class handled (`layout.go:267-269`).
82. **[theming] Density is fixed** (`--sidebar-width`, radii, gaps at `layout.go:100-105`) — even a `Config.Compact` toggle would help ops-dense tables.

---

## Part B — metaengine (ideas 83–175)

### B.1 API design (83–90)

83. **[API] `Plan(engines []Engine, args ...any)` mixes queries and options in one untyped variadic** (`planner.go:48-67`) — a stray non-query arg fails late with `errNotQueryMeta`. Split into typed params.
84. **[API] `Query[Q,R](name, args...)` panics** (`query.go:78,87,96,100`) — no error-returning variant for dynamic registration (config/YAML). Add `QueryE`/`MustQuery`.
85. **[API] Input-type dispatch silently shadows colliding queries** — "resolves to the most recently registered query" (`execute.go:31-35`); fail at `Plan()` instead of last-write-wins.
86. **[API] `Cursor.String()` swallows marshal errors** silently resetting pagination (`cursor.go:20-24`) — make the footgun observable (debug hook or metric).
87. **[API] Vector distance metric is a raw `string`** ("cosine"/"dot"/"euclidean") with silent default fallback (`vector_search.go:66-74,178-180`) — typed `VectorMetric` enum; typos currently degrade to euclidean silently.
88. **[API] `WithSortColumns` silently applies only the first column on pushdown engines** (`scan_options.go:68-70`) — reject or WARN.
89. **[API] Watcher deletes signaled by zero value of V** (README:532-536, COOKBOOK:169-180) — indistinguishable from a legit zero update; deliver an `Op` field in `SeqValue`.
90. **[API] Engine constructor naming is inconsistent** — `NewSQLiteEngineFromDSN` / `pgengine.New` / `NewPebbleEngine` / `duckdbengine.New` across 12 modules; standardize.

### B.2 Planner / cost model (91–101)

91. **[planner] `ComplexityODegree` is a constant 100 regardless of volume/depth** (`cost.go:95-98`, constants 27-30) — parameterize from declared traversal depth; a 1M-node graph and a 50-node graph cost identically.
92. **[planner] Volume defaults to magic 1000 when unset** (`cost.go:77-80`) — live row counts already exist (`ListPlannedTables`); auto-populate from engine-reported counts.
93. **[planner] Ranking is read-latency argmin only; `NsPerWrite` is "observability only"** (`engine.go:36-44`) — add a write-weighted term so write-heavy projections don't route to read-optimized engines.
94. **[planner] `planConfig.stats map[string]WorkloadStats` is vestigial** (`planner.go:32`) — wire `workloadMeter` observations into ranking (close the adaptive loop) or delete.
95. **[planner] Hysteresis is global** (`routingHysteresis`/`routingMinDelta`, `store.go:35-36`) — make per-query.
96. **[planner] `CostEstimate` carries no confidence flag** (prior vs calibrated vs live) (`cost.go:32-47`); `LiveLatency.Fresh` exists (`reliability.go:106-112`) but doesn't reach the estimate.
97. **[planner] `defaultNsPerOp = 100` fallback silently prices uncalibrated engines at fantasy 100ns** (`cost.go:61,103-105`) — require engines to declare or emit SCREAM.
98. **[planner] `filterSelectivity` (0.1^n) computed but unused for routing** (`cost.go:117-136`) — integrate for the FilterOnField-vs-closure decision or remove.
99. **[planner] No plan caching/memoization** — every `Replan` re-ranks from scratch (`store.go:158-168`); fingerprinted plan cache + incremental replan.
100. **[planner] No statistics/NDV collection** — selectivity hardcoded 0.1ⁿ (`cost.go:126`); no per-column histograms or distinct counts feeding the model.
101. **[planner] No per-query engine pinning escape hatch** (`WithQueryEngine("q","sqlite")`) for the operator who knows better — `override.go` handles folds, not engines.

### B.3 Engine parity / duplication (102–107)

102. **[duplication] `stream_log.go` near-verbatim twin across SQL engines** (sqlite:13-150 vs pg:14-155 vs mysql; only `?` vs `$N` + dialect differ) — extract a shared dialect-keyed query-set builder (the `sqliteQuerySet` pattern already proves it works).
103. **[duplication] Vector backends are dialect twins** across bbolt/badger/pebble/duckdb (`bboltengine/vector.go:26-64` vs `badgerengine/vector.go:26-66`, self-labeled `//art-dupl:accept`) — define an internal KV adapter (Get/Put/PrefixIter), implement vector once.
104. **[duplication] Test twins copied into 6–8 engine modules** — `calibration_bench_test.go`, `calibration_constants_dump_test.go`, `record_stamp_test.go`, `register_durability_test.go`, `durability_report.go` — migrate to `enginetest`/`adttest` parameterized harnesses.
105. **[duplication] `//art-dupl:accept` markers carry no tracking ID** (sqliteengine/dueclaim.go:25, pgengine/stream_log.go:65, badgerengine/vector.go:31) — require an ADR/backlog reference + a lint that counts accepted twins so the debt is measurable (est. 3–5k lines).
106. **[duplication] Graph recursive-CTE logic duplicated** (pgengine/graph.go, mysqlengine/graph_undirected.go, sqliteengine/graph.go) — share the CTE template parameterized on table+placeholder style.
107. **[parity] pgengine inlines SQL strings per call while sqliteengine has a `stmtCache`** (`sqliteengine/engine.go:28`, `stmt_cache.go`) — close the gap.

### B.4 Performance (108–113)

108. **[perf] SQLite `StreamAppend` inserts one row per call in a loop** (`sqliteengine/stream_log.go:13-22`) while pg has batched insert/COPY (`pgengine/stream_log.go:14-25`, `stream_copy.go`) — multi-row VALUES batching.
109. **[perf] Memory engine has no secondary index for `FilterOnField`** — filtered scans stay O(N) Go-side (COOKBOOK:200), forcing SQL in prod; a sorted secondary map for declared filter fields closes the dev/prod gap.
110. **[perf] SSE plain path drops oldest events with no `id:` field** — clients cannot detect loss (`sse.go:71-76,133-148`); always write ids.
111. **[perf] `SortPaginate` sorts the entire slice before cursor filter + limit** (`sort_paginate.go:29-51`) — partial top-K heap (limit+1) beats full sort for large KV scans.
112. **[perf] Pebble `CounterGet` is an O(N) prefix scan** (pebbleengine/README:28) — materialized per-key counters or count cache.
113. **[perf] No ANN index anywhere** — every engine is brute-force O(N·D) except probe-gated libSQL (sqliteengine/README:42-48); ship an HNSW memory engine option or a documented sqlite-vec/turso production path.

### B.5 Correctness / consistency (114–125)

114. **[correctness] Data race: memory engine's Vector/Search/Spatial backends bypass `m.mu` entirely** (`memory_engine.go:248-307`; index structs unlocked `vector_memory.go:26-59`, `spatial.go:51-58`) — concurrent inserts into different collections race on the shared parent map. Race tests only cover Map/Counter (`concurrent_map_race_test.go`).
115. **[correctness] Keyset pagination drops rows on sort-value ties** — cursor filter uses only `sortFn(valueOf(p), cursor) <= 0` (`sort_paginate.go:39-51`); the compound (sortValue,key) comparator must apply to the cursor filter too.
116. **[correctness] Planned tables: rows written before registration stay invisible to planned reads** (pgengine/README:84-89) — `ApplyLayoutPlan` should refuse or auto-backfill when meta_map already holds rows.
117. **[correctness] `ExecuteAsOf` stringifies keys with `fmt.Sprint`** (`temporal.go:101`) — `int(1)` and `"1"` collide; type-aware key encoding.
118. **[correctness] `WithDemoteForce`/`WithBackfillForce` bypass the non-idempotent-fold guard silently** (`demote.go:24-31`) — SCREAM diagnostic + Doctor marking when force is used.
119. **[correctness] SSE replay dedup treats `Seq == 0` as "no seq" sentinel** (`sse.go:194-196`) — explicit `HasSeq` bool.
120. **[correctness] TieredStore fan-out has no documented failure policy** (README:623-640) — define all-or-nothing vs best-effort, surface per-replica errors.
121. **[correctness] `ExecuteAsOf` reuses `errNoQueryForInputType` for a collection-name miss** (`temporal.go:68-71`) — wrong sentinel, misleading message.
122. **[correctness] `Verify` checks row-count drift only** (`errors.go:87-88`) — no content parity/checksum between primary and shadow before `PromoteEngine`; stale engine passes on counts alone.
123. **[correctness] `WithFilter` performs no FilterOp validation** (`scan_options.go:31-35`) and the op is string-interpolated into SQL (`sqliteengine/filter_clause.go:43`) — unvalidated op = SQL injection vector; validate the enum at construction AND at SQL build.
124. **[correctness] `jsonPath` allows arbitrary path content** (`sqliteengine/encoding.go:53-57`) — apply the strict identifier allowlist already written for matviews (`materialized_view.go:97-109`) to filter/sort columns everywhere.
125. **[correctness] Enum `Valid()` checks run only at `Plan()`** (`planner.go:137-160`) — QueryBuilder/TypedReader paths skip validation entirely; validate at every entry point.

### B.6 Error handling (126–129)

126. **[errors] Two error taxonomies coexist** — ~50 sentinels (`errors.go:12-98`) vs `errorfamily.NewRejection` used exactly once (`errors_planned.go:21-24`); migrate or drop the dependency.
127. **[errors] `errADTNotSupported` spliced mid-sentence** (`planner.go:258-264`: `"...requires ADT %s but %w — add a Memory engine..."`) — awkward rendered message.
128. **[errors] `ApplyError` structures fold failures but scan/execute errors are bare `fmt.Errorf`** (`errors.go:108-120`) — extend structure to read paths (query, engine, pattern).
129. **[errors] `writePlainSSEEvent` silently skips unmarshalable values** (`sse.go:154-158`, `//nolint:nilerr`) — hook/metric so "my events vanish" is debuggable.

### B.7 Observability (130–135)

130. **[observability] `PlanAuditEntry` history is in-memory only and bounded** (`store.go:39`) — export as JSON for post-mortems; `plan_diff.go` already computes diffs, make `PlanDiff` public-facing.
131. **[observability] No continuous estimated-vs-actual feed** — `CostAccuracyReporter` (README:727) needs manual wiring; auto-pair with `OnExecute` and surface drift in Doctor.
132. **[observability] `LogPlan` only logs via slog** (`dsl.go:27-51`) — add `PlanJSON()`/`PlanText()` renderers.
133. **[observability] Quarantine/catch-up hooks have no Prometheus bridge** (`failover.go:63-71`) — otelobserver exists (`otelobserver/observer.go`); metrics-only users get nothing.
134. **[observability] Engine stats (RTT EWMA/p95, sample count, stale flag) exist with no consumer** (`engine_stats.go:13-55,137-163`) — document + expose for dashboards.
135. **[observability] No `metaengine-calibrate` CLI** — per-engine calibration benches exist but there's no tool that runs them on target hardware and emits an importable `CalibrationCosts` JSON (`reliability.go:23-29` expects manual construction). Serious planners ship a calibrator.

### B.8 Testing / conformance (136–140)

136. **[testing] Mixed paradigms: Ginkgo/Gomega (cursor_test.go:8-9) vs stdlib everywhere else** — pick one.
137. **[testing] `adttest` covers ADT semantics but not concurrency semantics** — add `AssertConcurrentVectorInsert`, `AssertConcurrentScanDuringWrite` (would have caught idea 114).
138. **[testing] DuckDB's whole suite is `*_cgo_test.go`** — parity silently skipped without CGO; document the CI matrix + add a non-CGO smoke test.
139. **[testing] `RunCapabilityConformance` skips optional interfaces** (`PushdownScan`, `VersionedStorage`, `RawValueReader`) (`adttest/conformance.go:38-47`) — extend the audit to optionals.
140. **[testing] No cost-model regression CI** — `NewCostAccuracyReporter` exists but nothing asserts predicted-vs-actual error bounds; drift detected only if someone looks.

### B.9 Docs (141–145)

141. **[docs] MIGRATION.md:96 references nonexistent `metaengine.NewSQLiteEngine`** (it's `sqliteengine.NewSQLiteEngine`) — extend the README doc-validation test (`readme_quickexample_verify_test.go`) to COOKBOOK/MIGRATION.
142. **[docs] COOKBOOK teaches deprecated `On` throughout** (COOKBOOK:16-24, 62-73) while README mandates `OnRecord` (README:33-40).
143. **[docs] adttest doc comment claims "all 10 ADTs"** (`adttest/harness.go:1-4`) but the family is larger (DueClaim, Dedup, StreamLog, SortedMap...); regenerate from the registry.
144. **[docs] Capability tables hand-maintained per engine** (README:588-601) — generate from `CapabilityAudit` so they can't drift.
145. **[docs] README quick-example mixes `On`/`OnRecord`/`OnTyped` in one file** (README:36, 507-509) — converge on one constructor per doc version.

### B.10 Vector / temporal / graph / SSE features (146–158)

146. **[vector] `VectorBackend` has Insert/Search but no VectorDelete** (`vector_search.go:62-75`) — stale embeddings unremovable on every engine; deleted entities pollute k-NN forever.
147. **[vector] Filtered k-NN is AND-only** (`vector_search.go:35-43`) — add OR groups mirroring `WithOr` (`scan_options.go:80`).
148. **[vector] No API to reset a collection's dimension lock** (`vector_memory.go:50-56`) — operator must drop the engine; expose via `EngineResetter`.
149. **[temporal] `VersionedStorage` implemented only by memory/sqlite/bigtable** (pgengine/README:103-111) — a PG history-table is low-hanging; `rule_temporal_asof` currently just WARNs.
150. **[temporal] `AsOfSignal` is a dead public marker "retained for documentation"** (`temporal.go:29-44`) — deprecate explicitly or delete.
151. **[temporal] Version chains unbounded unless retention configured** (`memory_engine.go:20-21`, "nil = keep all") — default cap or WARN when retention unset and volume large.
152. **[graph] Pebble refuses graph entirely while core has a working multimap fallback** (`graph_fallback.go:14-67`, pebbleengine/README:50-54) — declare `DegradedADTs: ADTGraph` on KV engines so the planner routes with a warning.
153. **[graph] Fallback traversal is O(N·degree^depth) with per-node MultiGet** (`graph_fallback.go:32-37`) — batch level expansion (one prefix scan per level).
154. **[graph] No weighted-edge support in the `Edge` model** (README:199-205) — traversal cost and shortest-path out of reach; extend before more engines hard-code the 2-field shape.
155. **[SSE] No per-tenant/key filtering or auth hook in `ServeSSE`** (`sse.go:103-131`) — every client sees every mutation; filter-fn option needed before production HTTP use.
156. **[SSE] Replay default is unbounded** (`ReplayLimit` zero = replay ALL, `sse.go:38-41`) — invert to a sane cap with explicit opt-out.
157. **[migration] No `go fix` codemod for `On`→`OnRecord`** — the v5 removal lands as a big manual diff otherwise.
158. **[migration] `SwapEngine` does NOT close the old engine** (README:642-655) — add `WithCloseOldEngine()`; leaked engines are the predictable outcome.

### B.11 Config / naming / deprecation (159–166)

159. **[config] `DriverConfig` has no unknown-field rejection or schema versioning** (`registry.go:13-36`) — YAML typos silently no-op; strict decode + validation helper.
160. **[naming] Physical schema names hardcoded** — `meta_stream_log`, `meta_map`, `meta_planned_<col>` per engine (`sqliteengine/engine.go:95-121`) — no table-prefix option for shared/multi-tenant databases.
161. **[naming] Error prefixes vary** — `"metaengine: sqlite claimkit claims"` (sqliteengine/dueclaim.go:22) vs `"pgengine: claimkit claims"` (pgengine/dueclaim.go:24) — standardize.
162. **[naming] `dsl.go` holds `PlanFromMemory`/`LogPlan`** — nothing DSL about it; rename.
163. **[naming] ADT constants split between registry and ad-hoc consts** (`vector_search.go:12` vs registry lists) — consolidate.
164. **[deprecation] Deprecated `NsPerRead` remains the primary fallback** for engines lacking `ReadCosts` (`engine.go:26-34`) — publish the migration deadline.
165. **[deprecation] README shows `OnTyped` after the deprecation note** (README:94-98 vs 114) — examples should use the blessed constructor exclusively.
166. **[naming] `Store` is a god object** — 30+ fields, four mutexes with hand-documented lock-ordering invariants (`store.go:49-64`) — decompose into subsystems (health, routing, audit, replication).

### B.12 "Serious planner" gaps (167–175)

167. **[gap] Admission control / load shedding** — latency budgets gate planning only (`WithinBudget`, `cost.go:51-57`); nothing protects an overloaded engine at runtime (per-engine concurrency limits; circuit breaker is health-threshold only).
168. **[gap] Backup/restore as first-class** — no snapshot/restore contract per engine, no cross-engine consistency point for TieredStore.
169. **[gap] Schema evolution management** — pg/mysql `evolve.go` handle planned-table columns ad hoc; generalize `materialized_view_versions.go` into versioned LayoutPlan migrations with diff-driven DDL.
170. **[gap] Multi-tenancy** — collection names global; no namespace isolation, quotas, or prefix-scoped engines (compounds idea 160).
171. **[gap] Distributed engines & placement** — `Replication` enum and `NetworkRTT` exist (`engine.go:57-80`) but every engine is `ReplicationNone`; no sharding/partitioning/quorum. iroh is CRDT-wrapped local, not a planner target.
172. **[gap] Formal concurrency contracts per backend** — interface docs don't state thread-safety expectations (idea 114 proves the cost); state them and test them in adttest.
173. **[gap] Failover catch-up replays the whole EventLog from offset 0 through ≤64 passes** (`failover.go:113-137`, `catchUpMaxPasses=64`) — no persistent replay cursor; O(log) per reprobe.
174. **[gap] Shadow-replication staleness requires manual recovery** (README:669-673) — automate drain via the same catch-up machinery.
175. **[observability] `Store.Doctor(ctx) string` is text-only** (`explain.go:255`; section writers append to a `strings.Builder` at `catchup_state.go:111`, `capability_audit.go:238`, `doctor_degraded.go:16`) — no machine-readable variant for CI gates or dashboards; add `DoctorJSON(ctx)` beside the human renderer.

---

## Part C — system (ideas 176–262)

### C.1 API design (176–186)

176. **[API] `DomainConfig.Commands/Queries/Timers` closures return no error** — `RegisterDecider` returns an error (`register.go:42-47`) consumers can only panic on or drop; change to `func(*System) error` and propagate (`constructor.go:317-327`).
177. **[API] Multiple `source-of-truth`/`events` instances silently overwrite** `sys.eventStore` (`constructor.go:106-111`); dedicated roles DO check (`roles.go:26-28`). Fail on duplicates.
178. **[API] `InstanceConfig.Engine` vs `Engines` "mutually exclusive" but never validated** (`config_types.go:340-346`, `roles.go:175-181`).
179. **[API] `InstanceRole` is a free string** — typo `source_of_truth` wires nothing and silently falls to memory default (`roles.go:24-34`, `constructor.go:166-170`). Reject unknown roles.
180. **[API] Three projection decoder fields with 3-way precedence** (`ProjectionTypeDecoder` > `ProjectionEventDecoder` > `ProjectionDecoder`, `config_types.go:63-86`, `constructor.go:238-261`) — collapse into one.
181. **[API] `Execute(_ context.Context, ...)` accepts and ignores ctx** (`system.go:45`) — drop it or use it (tracing).
182. **[API] `Find`'s `After` option stores `cursor any` then asserts `string`** (`runtime.go:63,95-97,135-139`) — type it as string.
183. **[API] Free functions `Get/Find/GetCount(ctx, sys, name, ...)`** (`runtime.go:20,110,159`) — make them methods on `*System`.
184. **[API] `Count(...).On(...)` duplicate (eventType→key) entries silently accumulate folds** (`query_constructors.go:286-295`) — validate.
185. **[API] Registration-time validation missing** — `ErrNoDecider` surfaces only at dispatch (`register.go:113-115`); offer `sys.ValidateRegistrations()` dry check.
186. **[API] Command/query name registry absent** — `DispatcherInfo` counts commands only, no names (`system.go:158-159`, `introspection.go:122-127`).

### C.2 Construction-time validation / error messages (187–196)

187. **[validation] `New` leaks opened engines on every error return** — 12+ `return nil, err` paths after the creation loop never close `sys.engines` (`constructor.go:91,108,131,151,161,189,214,231,264,279,293,303`). Cleanup-on-failure.
188. **[validation] `validateShutdownDependencies` runs after engines are created AND manifest saved** (`constructor.go:274-304`) — a typo'd edge fails boot but has already persisted a new plan manifest.
189. **[validation] SCREAM failure reports only `Diagnostics[0].Detail`** (`constructor.go:35,288-294`) — join all findings.
190. **[validation] `wireSourceOfTruth` swallows `buildSnapshotStore` errors via `if ...; err == nil`** (`roles.go:136-140`) — "not capable" vs "construction failed" indistinguishable.
191. **[validation] `ErrUnknownEngine` doesn't list configured names or registered drivers** (`constructor.go:130-135`, `roles.go:191-193`) — include both.
192. **[validation] `Publish` targets never cross-checked against `Buses`** — `publish: [nats]` with no `buses:` entry silently fans to a fresh gochannel (`bus.go:73-81`).
193. **[validation] Volatile-SOT rule string-matches `driver == "memory"` only** (`scream_store.go:110`) — alias/third-party volatile drivers slip; key off a declared volatility capability.
194. **[validation] Duplicate `Evolve[R]` result types silently overwrite** (`projection_builder.go:74-84`) — duplicate-key error.
195. **[DX] `"no samples and no matching evolution"` doesn't print the result `reflect.Type`** (`query_constructors.go:105-108,239-242`) — Lookup[TaskView] vs Evolve[TaskEntity] mismatch should be obvious.
196. **[DX] `ErrCommandTypeMismatch: got %T`** (`register.go:104`) — include registered name + expected type; `ErrCacheCapacityInvalid` (`cache.go:23`) — include instance/engine.

### C.3 Defaults (197–202)

197. **[defaults] Silent memory fallback when no SOT wired** (`constructor.go:165-170`) — a typo'd role yields a volatile store with NO diagnostic; emit ADVISORY or make opt-in.
198. **[defaults] Default projection engine is memory even when SOT is persistent** (`constructor.go:172-183`) — projections evaporate on restart while checkpoints survive; WARN.
199. **[defaults] Magic engine names `"timers"`/`"checkpoints"` with fallback to `engines[0]`** (`timers.go:23`, `checkpoint_engine.go:139`) — implicit, rename-sensitive; make them config fields or documented roles.
200. **[defaults] Cache with `Capacity: 0` silently skipped** (`roles.go:161`) — configured-but-broken cache invisible.
201. **[defaults] No auto-derived shutdown edge** "projections engine closes before source-of-truth" — README tells users to hand-write it (README:317-327); derive when both roles exist.
202. **[defaults] No named ProjectionHostOptions presets** (dev/prod) for batch size/DLQ/restart policy (`config_types.go:91-94`).

### C.4 Config loading (203–209)

203. **[config] No strict/unknown-key detection** — `durabiltiy: strict` silently dropped by koanf unmarshal (`config_loader.go:88-94`); add strict mode.
204. **[config] Only YAML; no `LoadConfigReader(io.Reader)`, `LoadConfigFS(fs.FS)`, JSON/TOML** (`config_loader.go:59-67`).
205. **[config] Indexed env overrides support only 3 fields** (`config_loader.go:187-200`) — `publish`, `cache.capacity`, `collections`, `engines` pool unreachable via env.
206. **[config] No `CQRS_CONFIG_PATH` env convention** (`config_loader.go:59` hard-codes one path).
207. **[config] Legacy `CQRS_DEFAULT_DRIVER/DSN` silently no-op when a `primary` engine exists** (`config_loader.go:136-138`) — warn on mixed worlds.
208. **[config] `BusConfig.Mode` and `CacheConfig.Engine` parse but do nothing** (`config_types.go:306-319`) — config surface lies; deprecation WARN at LoadConfig ahead of v5.
209. **[config] Fully-empty deployment silently hits the memory default** (see 197) — consider erroring unless explicitly opted in.

### C.5 Lifecycle / shutdown / drain (210–217)

210. **[lifecycle] `Close()` never stops managed timers** — `stopTimers` is only called from `GracefulClose` (`system.go:310`); `ManageTimers` doc claims "stopped on GracefulClose/Close" (`timers.go:43-45`). Timer goroutines outlive `Close()` — goleak-visible.
211. **[lifecycle] `stopTimers` comment says it "waits" but only cancels** (`timers.go:77-89`) — add a WaitGroup or fix the comment.
212. **[lifecycle] `Start()` sets `started = true` before `projHost.Start`** (`constructor.go:341-347`) — failed start leaves system marked started; retry returns `ErrAlreadyStarted`.
213. **[lifecycle] `GracefulClose` ctx-expiry abandons the `Close()` goroutine** with no handle to observe completion (`system.go:318-334`); `Close()` takes no context so a hung engine blocks forever (`system.go:265-297`) — add `CloseContext(ctx)`.
214. **[lifecycle] `drainAll` returns the first drainer error and skips the rest** (`shutdown.go:195-198`) — asymmetric with Close's `errors.Join`.
215. **[lifecycle] `RegisterDrainer`/`RegisterCloser` after `Close()` silently never run** (`shutdown.go:179-226`) — guard with state check.
216. **[lifecycle] No `System.Stop()`** despite the doc comment listing it (`system.go:110`) — add (stop host/timers, keep engines) or fix the doc.
217. **[lifecycle] `Drain` is one-way** (`shutdown.go:204-214`) — no un-drain for aborted rolling deploys; document or add `Resume`.

### C.6 Health / readiness (218–222)

218. **[health] `HealthCheck` holds `s.mu.RLock()` while pinging engines** (`introspection.go:164-191`) — a slow engine stalls Close/RegisterCloser/Snapshot; snapshot the list first, ping outside the lock.
219. **[health] `instanceHealth` returns "healthy" from config presence alone** (`introspection.go:57-79`) — Topology's HealthStatus is decorative; wire to `metaengine.HealthChecker`.
220. **[health] `HealthCheckDetailed` omits engines without HealthChecker** (`introspection_extended.go:64-72`) — include as "no healthcheck capability" so the inventory is complete.
221. **[health] `Health()` returns a prose string** `"ok projections:not-started"` (`introspection.go:139-154`) — not machine-readable; struct/enum or delete.
222. **[health] Liveness vs readiness conflated** — a failed projection worker fails `HealthCheck` (`introspection.go:182-189`), restarting pods that need a rebuild; split `Liveness()`/`Readiness()`.

### C.7 Snapshots / timers / bus / cache (223–233)

223. **[snapshots] `RegisterDecider` hardcodes `codec.JSONCodec{}`** (`register.go:64`) — allow codec option (CBOR for smaller snapshots).
224. **[snapshots] `WithSnapshotStrategy` on a non-SnapshotBackend engine silently produces nothing** (`register.go:60-70`) — warn when requested-but-absent.
225. **[snapshots] `SnapshotAdapter.Load` zeroes `CreatedAt`** (`snapshot_adapter.go:20,67,92`) — age-based GC/audit impossible; persist a timestamp.
226. **[snapshots] No System-level snapshot tooling** — list/delete-by-age/compact (only raw `SnapshotStore()` at `system.go:246`).
227. **[timers] Scheduler `Start` errors discarded** (`_ = sched.Start(ctx)`, `timers.go:68-74`) — an immediately-failing scheduler is invisible.
228. **[timers] `ManageTimers` after `Start` silently never starts** (`timers.go:44-45`).
229. **[timers] No timer introspection** — no equivalent of `LagPerProjection` for due/overdue timers.
230. **[bus] Single `publish: [x]` target silently ignored entirely** — `buildPublisher` only builds fan-outs when `len > 1` (`bus.go:69`); the most common config is the broken one.
231. **[bus] Fan-out targets always create new in-process gochannels regardless of name** (`bus.go:73-80`) — `publish: [nats]` yields another gochannel; fail until real drivers exist.
232. **[bus] `MultiBus.Publish` returns on first error; later publishers silently miss events** (`multi_bus.go:118-124`) — aggregate or publish concurrently with Join.
233. **[cache] Stale-read race: `Save` invalidates AFTER the store write** (`cache.go:36-40`) — a concurrent Load can re-populate the pre-save snapshot between write and invalidate and serve it forever; invalidate-before-write or version-stamp.

### C.8 Observability / introspection (234–240)

234. **[observability] No logger injection** — two direct `slog.Warn` calls bypass operator config (`coeffect_gate.go:89`, `evolutions.go:296`); add `WithLogger`.
235. **[observability] Zero OTel/metrics wiring** despite `go-cqrs-lite/otel/v4` in the module graph (`go.mod:55`) — no spans on Dispatch/Save, no counters (commands dispatched, events appended, DLQ depth), no lag gauge. A composition root is THE natural place to auto-wire.
236. **[observability] `Explain()` text-only; `Snapshot()` has no JSON/Mermaid rendering** (`introspection.go:195-241`) — add `ExplainJSON()`/`Topology.Dot()` for dashboards and PR-reviewable diffs.
237. **[observability] No lifecycle transition hooks** (started/drained/closed) — only polling methods.
238. **[observability] `Explain` prints `runtime.Version()` but not module version** — add `debug.ReadBuildInfo()`.
239. **[observability] `CacheTierInfo` with `HitRate` defined but never populated** (`introspection.go:50-55`) — dead introspection surface; otter exposes hit stats.
240. **[observability] `Topology.ProjectionHost.Workers` declared but never populated** (`introspection.go:45-48,129-133`).

### C.9 Testing / docs / smells (241–249)

241. **[testing] Mixed test packaging** — white-box `package system` (`lifecycle_test.go:1,22-35`) vs black-box `system_test` (`determinism_test.go`) in one directory; white-box struct literals break on any refactor.
242. **[testing] No test for the New-failure engine leak** (idea 187) — goleak suite already runs (`main_test.go:22-24`); add the case.
243. **[testing] No godoc `Example*` functions** — quickstart lives only in README.
244. **[testing] `TestSystem_EngineNamesDeterministic` loops `for range 10`** (`determinism_test.go:59`) — `rapid` is already a dep (`go.mod:32`); use property-based iteration.
245. **[docs] Quick Start swallows an unmarshal error** — `_ = json.Unmarshal(...)` (README:61) teaches ignoring errors in the flagship example.
246. **[docs] Built-in driver list will drift** from `metaengine.RegisteredDrivers()` (README:235-238) — generate or reference the function.
247. **[docs] No SIGTERM wiring helper** — README shows hand-rolled timeout (README:344-348); ship `system.RunUntilSignal(ctx, sys)`.
248. **[docs] Scream ACK grammar (`"rule:target"`) only implied by examples** (README:194,398-399) — enumerate all rule IDs from scream_store/scream_plan.
249. **[smells] Dead branch in `createEventBus`** — both paths return the identical value (`bus.go:43-48`).

### C.10 Remaining smells + registry/evolution (250–262)

250. **[smells] `InstanceConfig.Subscribe` parsed-and-ignored** (`config_types.go:363`) — config surface lies.
251. **[smells] `reifyTo` panics on schema mismatch** (`evolutions.go:196-218`) — a poison event panics a projection worker mid-replay; route to the host's DLQ path.
252. **[smells] `System.mu` guards wildly different things** (repos, engines, lifecycle flags, closers) — split mutexes.
253. **[smells] `constructor.go` mixes 6 wiring phases in one function** — extract `wireBus`, `wireProjectionHost`, `runPlanDriftCheck`.
254. **[smells] O(n) hidden costs** — `ReadFromAfter` cursor scans whole journal on non-SeqSeekable backends (`adapter_core.go:163-175`); `LoadToTimestamp` loads full stream then filters in Go (`adapter_event.go:273-289`); `QueryAdapter.LoadQueries` reads whole journal (`adapter_query.go:72-94`).
255. **[registry] `createEngineFromDriver` doesn't pass the engine NAME into `DriverConfig`** (`driver_registry.go:29-35`) — engine-internal errors can't cite the operator-facing name.
256. **[registry] No capability matrix introspection** — which engines are AtomicAppender/MapBackend/SnapshotBackend/DueClaimer decides silent fallbacks (checkpoints→memory, snapshots→none, timers→first engine); report it.
257. **[registry] `EngineConfig.Pragmas` documented as "SQLite pragmas" but forwarded to every driver** (`config_types.go:189`, `driver_registry.go:31`) — rename to `Options` or validate per driver.
258. **[registry] No bus driver registry while engines have one** (`bus.go:13-49` hardcodes gochannel; README:244-246 admits it) — mirror `metaengine.RegisterDriver`.
259. **[evolution] `Internal()` reserved-but-unenforced** (`evolutions.go:50-56`) — dead API surface; enforce or remove before v5.
260. **[evolution] Plan-drift detection sees ADT/engine changes but not fold-logic changes** (`scream_plan.go:96-117`) — fold a hash of the event-set per projection into the manifest.
261. **[evolution] Convention folding keys off struct-name suffixes Created/Updated/Deleted** (`evolutions.go:74-87`) — a misnamed sample (`TaskClosed`) produces no fold and no error; validate at declaration.
262. **[batteries] Missing ready-mades** — no `/healthz`/`/readyz` http.HandlerFunc wrappers, no pprof/expvar hook, no JSON Schema for `cqrs.yaml`, no `Validate(deployment)` dry-run mode (CheckPlanSafety WRITES the manifest — `scream_plan.go:73-78` — so "check" mode mutates disk), no ResetProjection/VerifyProjections CLI story.

---

## Part D — Cross-cutting integration (ideas 263–308)

### D.1 dashboard ↔ metaengine telemetry (263–273)

~~263. **[telemetry] Add a "Plan" panel** rendering `Store.ExplainPlan()`/`System.ProjectionExplain()` (`metaengine/explain.go:113`, `system/introspection.go:271`) — nothing in dashboardui reads it today.~~ CONSUMED — M26 telemetry: `WireTelemetry` maps `MetaEngine().Plan()` into the query-placement table (query → engine + ADT + volume + est. latency); RuleTrace/Diagnostics surface remains phase-2; 02-57 report.
264. **[telemetry] Render per-query placement/cost** (`QueryPlacement`: engine, ADT, volume, est. latency) as a table instead of string-parsing Explain (`metaengine/explain.go:196-200`).
~~265. **[telemetry] Engine stats cards** — RTT EWMA/p95, sample count, stale flag (`metaengine/engine_stats.go:13-55`); `GetEngineStats` is documented "for any operator dashboard" with no consumer.~~ CONSUMED — M26 engine-latency cards (RTT EWMA/P95/samples, 5-minute freshness badge, explicit no-tracker/no-samples states) now consume `GetEngineStats`; 02-57 report.
266. **[telemetry] Plan audit history timeline** from `PlanHistory()` (version, trigger, priority snapshot) (`metaengine/plan_audit.go:31-53`); stream replans over SSE.
267. **[telemetry] Quarantine/failover live feed** — surface `HealthSnapshot()`, `OnQuarantined/OnReactivated/OnProbe/OnCatchUp` hooks as dashboard events (`metaengine/engine_health.go:170`, `health_observer.go:20-42`).
268. **[telemetry] Layout observability panel** — `GetLayoutInfo`, `LayoutWarnings` incl. JOIN_AMPLIFICATION (`metaengine/layout_observability.go:21,88`).
269. **[telemetry] Cost-drift chart** from `CostAccuracyReporter` (estimated vs actual) (`metaengine/observability.go:184+`).
270. **[telemetry] Per-query SQL explain viewer** via `TypedReader.Explain`/`ExplainAggregate` (`metaengine/explain.go:38-102`).
271. **[telemetry] Plan graph rendering** — `PlanResult.DotGraph()` → embedded SVG topology view (`metaengine/observability.go:140-167`).
272. **[telemetry] Fold/execute metrics** — `WithMetrics`/`MetricsRecorder` → dashboard stats endpoint (hooks exist log-oriented only) (`metaengine/observability.go:108-133`).
273. **[telemetry] Operator action buttons** — `Replan`, `SetPriority`, `PromoteEngine`/`DemoteEngine`, `ReactivateEngine`, `CatchUpEngine` (`store.go:90`, `priority.go:173`, `roles.go:157`, `demote.go:63`, `engine_health.go:204`, `failover.go:60`) — all knobs, no UI.

### D.2 dashboard ↔ system introspection (274–280)

~~274. **[introspection] `Config.System`/`TopologyProvider` capability** — render `System.Snapshot(ctx)` Topology (instances, engines, drivers, buses, durability) (`system/introspection.go:17-48,85`); data exists, nothing consumes it.~~ CONSUMED — M26 `Config.Topology` provider + system-topology table (WireTelemetry maps `sys.Snapshot`); 02-57 report.
~~275. **[introspection] Wire `HealthCheckDetailed` into dashboard healthz/overview** (`system/introspection_extended.go:58-86` vs `dashboardui/handlers_health.go:12-58` — dashboard health is self-only).~~ CONSUMED — M26: `Config.EngineHealths` composes `HealthCheckDetailed` into `/-/healthz` (engines array) + `/-/readyz` (503 `engine_unhealthy`); 02-57 report.
276. **[introspection] System-level lag display** even when the ProjectionHost panel is off; dashboard currently recomputes lag from `host.LagPerProjection()` (`core/overview.go:107` vs `introspection_extended.go:90-113`).
277. **[introspection] ScreamReport panel** — persisted WARN/OVERRIDE findings (`system/scream_store.go:161`); only CheckSafety is documented in the cqrs-htmx guide, never surfaced at runtime.
278. **[introspection] Plan-drift diff view** — `ProjectionPlan()` (SerializablePlan) + `manifest_path` pinning, dashboard diffs live plan vs pinned manifest (`system/introspection.go:246-257`).
279. **[introspection] `VerifyProjections` as a consistency-check button** alongside existing Reset (`system/introspection.go:261-267` vs `dashboardui/handlers_projections.go:66-102`).
280. **[introspection] versionz fingerprint** — report system/metaengine module versions + `metaengine.RegisteredDrivers()` (`dashboardui/handlers_health.go:49-58`).

### D.3 systemadapter completeness (281–289)

~~281. **[adapter] Never sets `DomainConfig.CheckpointStore` or `ProjectionHostOptions`** — declarative path always defaults (`systemadapter/domain_config.go:48-54` vs `system/config_types.go:91-101`); add `WithCheckpointStore`/`WithHostOptions` variants (ADR-0149 shipped durable checkpoints upstream — wire them).~~ CONSUMED — M22 shipped `WithCheckpointStore`/`WithDeadLetterStore`/`WithHostOptions` (durable-checkpoint restart proof: `TestDeclarative_DurableCheckpointSurvivesRestart`); systemadapter CHANGELOG + 02-57 report.
282. **[adapter] No `Queries` registration** (only Get/Find helpers) — `system.RegisterQuery` would enable `DispatchQuery` + the query journal (`queries.go` vs system README:220-221).
283. **[adapter] No `ShutdownDependencies` preset or `Middleware`** (authz/validation) wired (`system/README.md:322-327,376`).
284. **[adapter] Zero coverage of system capabilities** — timers, snapshots, multi-bus, cache tier, evolutions, failover/reprobe: none wired or documented as identity-relevant (`system/timers.go:21`, `system.go:246,202`, `evolutions.go`).
285. **[adapter] Ship `RecommendedDeployment(driver, dsn)` presets** (memory dev / sqlite prod / split engines) — the guide duplicates 3 hand-rolled copies (`docs/guides/leveraging-system-metaengine.md:107-146`).
286. **[adapter] `Enforce` re-implements role hierarchy in Go** (`queries.go:184-200`) while the deprecated path used Casbin Authz — fold-maintained closure or documented divergence from `identitymodel.Role`.
287. **[adapter] No README** — only CHANGELOG; package doc lives in `domain_config.go:1-22`. Cover declarative path first, ProjectionLayer last.
288. **[adapter] 60+ gopls deprecation hints** — compiles against deprecated `usermgmt.*` re-exports (`usermgmt.UserState`, `usermgmt.NewAuthz` in `domain_config.go`/`projections.go`); v5 alias removal (v5-removal-inventory class 3) breaks this module unless migrated to direct identity-model imports.
289. **[adapter] Composite external-account index impossible via single-field keying** — worked around by full-scan over `users` (`queries.go:117-140`); needs a composite-key metaengine pattern upstream.

### D.4 Type-safety across seams (290–293)

290. **[types] `system.RawQuery(decl any)` fully type-erased** — the 13 `system.RawQuery(xLookup())` calls lose `[Q,R]` until runtime (`system/projection_builder.go:49-53`, `systemadapter/declarations.go:33-50`); make `RawQuery[Q,R](metaengine.QueryDecl[Q,R])`.
291. **[types] Magic query-name strings repeated** in `declarations.go` and `queries.go` with no compile-time link — export typed constants or `system.Query[T]("users")` handles.
292. **[types] `metaengine.FilterOnField[TenantView]("Name", ...)` stringly-typed against view fields** (`declarations.go:73-74`) — struct-tag-derived option catches renames.
~~293. **[types] `dashboardui.Autodetect` asserts concrete `*projectionhost.Host`** (`autodetect.go:62`) — decorators/wrapped hosts unprobeable; also `system.CommandStore()` vs dashboard's `command.CommandJournal` are different interfaces needing an adapter (`system/system.go:252`, `core/capabilities.go:51`).~~ CONSUMED — M20 (leaf-interface probes) + M25 (duck-typed `System` accessor: decorators stay probeable, no system/metaengine import); 02-57 report.

### D.5 Missing bridges (294–298)

~~294. **[bridge] `dashboardui.FromSystem(sys)` adapter** — map EventStore→Journal/Seekable/EventByIDLoader, Bus→EventBus, ProjectionHost, SnapshotStore, QueryStore (`Autodetect(sys)` can't work because System implements none of the probed leaf interfaces).~~ CONSUMED — M25 shipped `FromSystem(sys)` (first-wins mapping, journal-derived StreamReader fallback, `TestDashboard_FromSystem_PanelsLightUp`); 02-57 report.
295. **[bridge] Projection-delta SSE channel** — metaengine `Watcher`/`ServeSSE` (per-collection read-model deltas, `metaengine/sse.go:103`, `dx.go:111`) vs dashboard SSE (raw domain events, `dashboardui/sse.go:53-60`); a Watcher→broadcaster bridge would let dashboard panels live-update read models.
296. **[bridge] Metrics wiring** — go-cqrs-lite ships `prometheus/` and `otel/`; neither systemadapter deployments nor dashboardui reference them; `systemadapter.Metrics(sys)` wiring `metaengine.WithMetrics` + engine health gauges closes the loop.
297. **[bridge] Health aggregation** — `System.HealthCheckDetailed` ignores metaengine's own quarantine state (`MetaEngine().HealthSnapshot()`) — a quarantined-but-pingable engine reads healthy (`introspection_extended.go:58-86` vs `engine_health.go:170-180`).
298. **[bridge] Single aggregate `/healthz`** should compose process health + System.HealthCheck + engine quarantine + projection workers + DLQ threshold (dashboard healthz knows nothing about the CQRS system today, `handlers_health.go:12-19`).

### D.6 Projection lifecycle alignment (299–302)

299. **[lifecycle] Three projection lifecycles coexist** — system's internal host (declarative), deprecated ProjectionLayer host, raw `projectionhost.New`; dashboard reset calls `host.Reset` (`handlers_projections.go:71`) but the canonical declarative API is `System.ResetProjection(ctx, name)` — dashboard should prefer system-level when wired.
300. **[lifecycle] Declarative path has no `WaitForDrain` equivalent** — guide tells users to poll queries (`declarative-projections.md:63-85`); promote `System.LagDuration()==0`-based or a `DrainFor(d)` helper so read-your-writes semantics survive migration.
~~301. **[lifecycle] DLQ dark on the declarative path** — systemadapter never wires `WithDeadLetterStore` into `ProjectionHostOptions` (`systemadapter/projections.go:81-85` vs `domain_config.go:48-54`); dashboard DLQ panels go empty unless the consumer knows.~~ CONSUMED — M22 shipped `WithDeadLetterStore` (replaces the in-memory DLQ default on the declarative path); systemadapter CHANGELOG + 02-57 report.
302. **[lifecycle] projectionhost hosts are one-shot** (crash-restart = fresh host, per cqrs-htmx AGENTS) — system's rebuild story should reuse `RebuildProjection` semantics; align naming/behavior across both repos' docs.

### D.7 Docs / version alignment (303–308)

303. **[docs] `declarative-projections.md:93-99` claims no custom checkpoint/DLQ and full replay on restart — STALE**: system now ships `DomainConfig.CheckpointStore` + `ProjectionHostOptions` (`system/checkpoint.go:49-61`, ADR-0149); update guide + adapter wiring.
304. **[docs] `leveraging-system-metaengine.md` still teaches deprecated `NewProjectionLayer` as Quick Start steps 4–5 with no deprecation note** (lines 12-18, 48-54) — contradicts `declarative-projections.md` and the code marker.
305. **[docs] Same guide's "Advanced" section uses old untyped `metaengine.Query[any,any]`/`On` style** (lines 212-221) instead of the `OnRecordTyped`+`EventWithID` pattern systemadapter actually uses.
306. **[docs] ADR-0051's "durable-checkpoint upstream ask" listed as open in archived status reports but shipped** — reconcile the criterion chain so the declarative path can be re-evaluated.
307. **[versions] systemadapter pins `system/v4 v4.10.2` + `metaengine v4.16.1`** while the sibling tree's system depends on `metaengine v4.16.0` behind `replace ../metaengine` — track the release-wave gap so unpublished surface doesn't strand consumers.
~~308. **[versions] dashboardui pins no system/metaengine deps at all** — any future capability panel adds a new direct dependency; decide the seam now (interface in `core` vs direct import).~~ CONSUMED — seam decided: `systembridge` sub-package is the ONE system/metaengine-importing place (dashboardui proper stays leaf-typed); 02-57 report §c + AGENTS.md.

---

## Priority shortlist (Top 30 by impact × urgency)

Effort: **S** ≈ hours (same-day), **M** ≈ one to two days, **L** ≈ multi-day or needs design/ADR. The S rows form a same-day quick-win batch.

| #  | Idea                                                           | Effort | Why first                                          |
| -- | -------------------------------------------------------------- | ------ | -------------------------------------------------- |
| 1  | **1** XSS in SSE row injection                                 | S      | Exploitable, trivial fix                           |
| 2  | **123/124** metaengine FilterOp/jsonPath SQL-injection surface | S      | Unvalidated enum → SQL interpolation               |
| 3  | **114** Memory-engine data race (vector/spatial bypass mutex)  | M      | Correctness + race detector                        |
| 4  | **2** AccentColor CSS injection                                | S      | Trivial validation fix                             |
| 5  | **24** Unescaped query params in pagination links              | M      | Correctness bug, corrupts navigation               |
| 6  | **9** Sorting silently kills pagination                        | M      | Core UX of the events panel                        |
| 7  | **17/18** Infra errors rendered as 404/empty tables            | M      | Ops trust: failure looks like absence              |
| 8  | **187** `system.New` leaks engines on error paths              | M      | Resource leak on every failed boot                 |
| 9  | **210** `Close()` never stops timers                           | S      | Leak + doc lie                                     |
| 10 | **230** Single `publish: [x]` silently ignored                 | S      | Most common bus config is the broken one           |
| 11 | **197/179** Silent memory fallback on typo'd role              | S      | Volatile store by accident                         |
| 12 | **115** Keyset pagination drops tie rows                       | M      | Silent data loss in scans                          |
| 13 | **146** No VectorDelete anywhere                               | L      | Stale embeddings forever                           |
| 14 | **263–273** dashboard↔metaengine telemetry panels              | L      | The dashboard misses its own domain's richest data |
| 15 | **303–305** Stale/deprecated guide content                     | S      | Consumers following docs into deprecated paths     |
| 16 | **294** `dashboardui.FromSystem` bridge                        | M      | Kills the consumer assertion dance permanently     |
| 17 | **102/103/105** Cross-engine twin extraction + debt lint       | L      | 3–5k lines of institutionalized copy-paste         |
| 18 | **281/301** systemadapter wires checkpoints/DLQ                | M      | ADR-0149 shipped; adapter ignores it               |
| 19 | **233** Cache invalidate-after-write race                      | S      | Serves stale-forever snapshots                     |
| 20 | **6** Content-hash ETags for embedded assets                   | S      | Recurring stale-asset incident class               |
| 21 | **4** CSV formula injection                                    | S      | One-line neutralization                            |
| 22 | **251** `reifyTo` panic on poison event                        | M      | Panics a worker mid-replay                         |
| 23 | **222** Liveness/readiness split                               | M      | Bad pod restarts during rebuild                    |
| 24 | **135** `metaengine-calibrate` CLI                             | L      | Unblocks honest cost model everywhere              |
| 25 | **156** SSE unbounded replay default                           | S      | Dangerous default inverted                         |
| 26 | **295** Watcher→broadcaster bridge                             | M      | Live read-model panels become possible             |
| 27 | **68, 26** Generic journalIndex + reusable sort spec           | M      | Deletes duplication + unlocks features             |
| 28 | **113** ANN vector path                                        | L      | Vector search is brute-force everywhere            |
| 29 | **298** Aggregate healthz composition                          | M      | One endpoint telling the truth                     |
| 30 | **41/42** Config.Validate + variadic Autodetect                | M      | Consumer DX at the two hottest seams               |

_(Full list above: 308 ideas. Counts by module: dashboardui 82, metaengine 93, system 87, integration 46.)_

---

## Appendix A — Editorial assessment (what the author actually believes)

Recorded 2026-10-06 so the judgment travels with the document, not just the chat it was voiced in. The "100 to 1000" framing manufactured volume; the honest distribution:

| Tier                                             | Share      | Representative ideas                                                                                                     |
| ------------------------------------------------ | ---------- | ------------------------------------------------------------------------------------------------------------------------ |
| **Load-bearing — act on these**                  | ~60 (20%)  | 1, 2, 114, 115, 123/124, 187, 197, 210, 230, 233, 251, 263–273, 294, 303–305                                             |
| **Solid hygiene — do opportunistically**         | ~200 (65%) | testing matrices (59–64, 136–140), docs fixes (141–145, 245–248), twin extraction (102–107), DX error messages (191–196) |
| **Debatable — needs a product decision first**   | ~30 (10%)  | 37, 47, 71, 81, 82                                                                                                       |
| **Veto — strategy questions disguised as ideas** | ~15 (5%)   | 171 and parts of the 167–175 "serious planner gaps" range                                                                |

### Ideas to push back on (veto or demote)

- **171 (distributed engines & placement)** — fights the "lite" philosophy; ADR-0146 (no federated query engine) and ADR-0147 (mesh policy enforcement non-goal) already drew this line. Reopen only as an explicit strategy decision, never as a backlog item.
- **47 (i18n Localizer interface)** — speculative API surface on a mountable library panel. YAGNI until a consumer asks; a `Labels` map is cheaper than a Localizer abstraction.
- **166 (decompose the `Store` god object)** — the smell is real, but four mutexes with hand-documented lock-order invariants make this surgery requiring its own ADR plus a race-test harness, not a cleanup chore. The one-line idea undersells the risk.
- **37, 81, 82 (JSON tree viewer, forced-colors, density toggle)** — individually fine, collectively feature-sprawl gravity on a panel whose core value is cheap mountability. Bundle behind explicit consumer demand.
- **136 (unify test framework)** — churn-heavy, benefit is reviewer aesthetics; migrate file-by-file only when already touched.

### The ten to do first

1 (SSE XSS), 123/124 (injection surface), 2 (accent CSS), 114 (memory-engine race), 230 (single publish target), 210 (timers on Close), 187 (engine leak), 251 (reifyTo panic), 303–305 (docs telling the truth), 294 (`FromSystem` bridge). Everything else survives a release cycle of waiting.

_Append-only addition per the research-dir convention — no findings above were altered._
