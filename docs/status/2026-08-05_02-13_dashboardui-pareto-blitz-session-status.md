# DashboardUI Pareto Blitz — Session Status

> **Date:** 2026-08-05 02:13
> **Scope:** `dashboardui/` module — executing the Pareto plan from `docs/planning/2026-08-05_01-49_dashboardui-pareto-execution-plan.md`
> **Session goal:** Execute the ENTIRE 24-task TODO list. Got through 8 tasks before user halted for status.

---

## a) FULLY DONE (committed, build verified at commit time)

### T01: Remove dead `listStreams()` ✅
- Deleted the dead `listStreams()` method from `handlers_aggregates.go`.
- Updated `TestListStreams_NilReader` and `TestListStreams_WithReader` to test `listStreamsPaged` instead.
- Committed in `48951a7`.

### T02: Prune `IMPROVEMENT_IDEAS.md` ✅
- Rewrote from 883 lines (350+ items, ~80% stale) to ~60 lines of genuinely open work.
- Cross-referenced against current codebase state.
- Committed in `0dd494e` (part of planning doc commit).

### T03-T05: HTMX Partial Rendering ✅
- Added `isHTMXRequest(r)` helper in `render.go` — checks `HX-Request: true` header.
- Added `HTMX bool` field to `pageData` struct, populated in `dashboard.go` `page()` method.
- Modified `renderLayout` in `layout.go` — when `p.HTMX` is true, returns only `<title>` + `<main id="main-content">` (no full HTML shell, no sidebar, no DOCTYPE).
- Updated `data-hx-boost="true"` div with `data-hx-target="#main-content" data-hx-select="#main-content" data-hx-swap="outerHTML"` for proper content extraction on boost.
- Added 5 tests: `TestIsHTMXRequest_NormalRequest`, `TestIsHTMXRequest_BoostedRequest`, `TestRenderLayout_HTMXPartial`, `TestRenderLayout_FullPage`, `TestOverviewHandler_HTMXReturnsPartial`.
- Committed in `a956132`.

### T06: SSE `dashboard:event` Listener ✅
- Added JS `addEventListener("dashboard:event")` in `dashboardJS` constant in `layout.go`.
- On event: triggers `htmx.trigger("#projection-health", "refresh")` to reload projection health panel.
- On event: prepends a new `<tr>` row to `#main-content .data-table tbody` with event data (type link, stream ID, version).
- Caps table at 50 rows (removes oldest beyond that).
- Updated projection health panel `hx-trigger` from `"every 10s"` to `"every 10s, refresh"` so the SSE listener can trigger immediate refresh.
- Added `@keyframes newRowHighlight` CSS animation for `.new-row` class.
- Committed in `a018136`.

### T07: Command Detail View ✅
- Route: `GET /commands/{id}` registered in `handler.go`.
- `commandDetailHandler` in `handlers_audit.go` — parses command ID, loads via `loadCommandByID`.
- `loadCommandByID` — scans SeekableCommandJournal or falls back to ReadAll, mirrors `loadEventByID` pattern.
- `renderCommandDetail` — metadata table (type, stream type, stream ID, received at, command ID, correlation/causation/user IDs) + payload `<pre>` block with `prettyJSON`.
- Added "View" link column to command list rows.
- Committed in `941fa8a`.

### T08: DLQ Entry Detail View ✅
- Route: `GET /dead-letters/{projection}/{eventID}` registered in `handler.go`.
- `dlqEntryDetailHandler` in `handlers_dlq.go` — loads entry by scanning DeadLetterStore.List + filtering by eventID.
- `renderDLQEntryDetail` — error details (type, ID, stream ID, failed at, error family badge, error code) + error message `<pre>` + original event payload + delete action.
- Added "View" link to DLQ table rows (event type cell is now a link).
- Committed in `941fa8a`.

### T11: Query Detail View ✅
- Route: `GET /queries/{id}` registered in `handler.go`.
- `queryDetailHandler` in `handlers_audit.go` — parses query ID, loads via `loadQueryByID`.
- `loadQueryByID` — scans SeekableQueryJournal or falls back to ReadAllQueries.
- `renderQueryDetail` — metadata table (type, received at, request ID, correlation/causation/user IDs) + payload `<pre>` block.
- Added "View" link column to query list rows.
- Added `prettyJSON` helper in `payload.go` for command/query payload pretty-printing.
- Committed in `941fa8a`.

---

## b) PARTIALLY DONE (code written, NOT verified, BUILD BROKEN)

### T09: Aggregate Event Timeline Pagination ⚠️ BROKEN BUILD
- **What's written:** `aggregateDetailHandler` now calls `aggregateTimelinePagination(r)` and passes `paginationState` to `renderAggregateDetail`. The renderer paginates events in-memory using version numbers as cursors (`?after=3`). Added `paginateEventsByVersion` helper. Added `renderPagination` call with aggregate detail path.
- **What's wrong:** The `strconv` import is missing from `handlers_aggregates.go`. Build fails with `undefined: strconv` at lines 131 and 230.
- **Fix needed:** Add `"strconv"` to the import block, then verify build + tests.

---

## c) NOT STARTED

| Task | Description | Why it matters |
|------|-------------|----------------|
| T10 | Total count display ("Showing 1–N of M") | Users don't know dataset size |
| T12 | Sortable column headers (`?sort=col&dir=asc`) | Can't find things in large datasets |
| T13 | Page-size selector dropdown (25/50/100/200) | Users stuck at default page size |
| T14 | HTMX-powered filter form (partial swap) | Filter form does full page reload |
| T15 | Event payload copy button + JSON download | One-click copy of payload |
| T16 | Projection detail view (`/projections/{name}`) | Show checkpoint, restarts, last error, DLQ link |
| T17 | Keyboard navigation for time-travel slider | Arrow left/right to scrub versions |
| T18 | Command/query success/failure badge + duration | Need to check if PersistedCommand has these fields |
| Doc.go fix | Has false "templ-components" claim | Pre-existing bug, identified but not fixed |
| CHANGELOG.md | Needs entry for all session improvements | Convention: done work goes to CHANGELOG |
| README update | Document `?prev=` param, HTMX partials, SSE listener | Consumer-facing docs |
| Detail view tests | No tests written for command/query/DLQ detail handlers | Coverage will drop |

---

## d) TOTALLY FUCKED UP

### 1. Left the build broken on T09
I wrote the aggregate pagination code, referenced `strconv.ParseUint`, but forgot to add `"strconv"` to the import block. The build was broken when the user halted. I should have run `go build` immediately after the edit and fixed it before moving on.

### 2. Didn't write tests for any of the 3 detail views (T07/T08/T11)
The Pareto plan explicitly called for tests (`TestCommandDetailHandler_Renders`, `TestCommandDetailHandler_NotFound`, `TestDLQEntryDetailHandler_Renders`, `TestDLQEntryDetailHandler_NotFound`, `TestQueryDetailHandler_Renders`, `TestQueryDetailHandler_NotFound`). None were written. The code was committed without tests. Coverage will have dropped.

### 3. Auto-commit daemon committed mid-work
The auto-commit daemon committed the detail views (`941fa8a`) before tests were written. This means the committed code has no test coverage for the new handlers. The commits are dirty from a quality perspective.

### 4. Didn't fix `doc.go` while in the module
The `doc.go` file still claims "composes templ-components for the UI" which is false (it uses raw `strings.Builder` HTML). This was identified as a pre-existing bug in the self-critique but still not fixed despite being a 2-second fix.

### 5. Didn't update CHANGELOG.md
The convention says completed work goes to CHANGELOG.md. Three commits shipped new features and no CHANGELOG entry was written.

### 6. `Dashboard` struct field rename (`cfg` → `config`) discovered mid-edit
The auto-commit daemon renamed the struct field from `cfg` to `config` between sessions. I discovered this when the build failed on `d.cfg` references. I bulk-replaced with `sed` without understanding why the rename happened or whether it was intentional. This could mask a problem.

### 7. No test for HTMX partial rendering on actual handlers
The `TestOverviewHandler_HTMXReturnsPartial` test exists but only covers the overview. No tests verify that events/commands/queries/aggregates/DLQ handlers return partials when `HX-Request: true`.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements
1. **Run `go build` after EVERY edit, not just after a group of edits.** The broken `strconv` import would have been caught immediately.
2. **Write tests BEFORE committing.** The auto-commit daemon makes this critical — any code without tests gets committed untested.
3. **Update CHANGELOG and doc.go as part of the task, not as an afterthought.** They keep getting forgotten.
4. **Verify coverage didn't drop after new code.** Three new handlers with zero tests will have moved coverage backward.

### Code quality improvements
5. **The `loadCommandByID` / `loadQueryByID` functions duplicate the scan pattern from `loadEventByID`.** Extract a generic `scanJournal[T]` helper.
6. **`prettyJSON` in `payload.go` duplicates logic from `DefaultPayloadRenderer.Render`.** Should reuse the renderer instead of re-implementing.
7. **The SSE JS listener creates rows with hardcoded column counts.** It doesn't adapt to different table structures (commands table has 5 columns, events table has 4). This will create malformed rows on non-events pages.
8. **The `aggregateTimelinePagination` function doesn't set `HasNext`** — it's computed later in `renderAggregateDetail`. This split logic is confusing.
9. **`paginateEventsByVersion` has a subtle bug risk:** if `page.PageSize` is 0 (shouldn't happen but could from a misconfigured default), it returns 1 item (0+1=1, then sliced to [:0]). Should guard against zero PageSize.

### Architecture improvements
10. **All detail handlers mix data loading and HTML rendering.** The plan identified this as architectural debt. Each `loadXByID` + `renderXDetail` pair should eventually become `loadX(ctx) (data, error)` + `renderX(data) string`.
11. **The DLQ entry detail loads ALL entries then filters.** For projections with thousands of dead letters, this is O(N). Should add a `Get(ctx, projection, eventID)` method to the store interface (but that's an upstream change).
12. **HTMX partial mode returns `<title>` tag alongside `<main>`.** This is correct for HTMX boost, but the `<title>` is outside `<main>`, meaning the Hx-Boost title extraction relies on HTMX's built-in `<title>` sniffing. This should be verified to work with the pinned HTMX version.

---

## f) Up to 50 Things to Get Done Next

### Immediate fixes (must do before any more feature work)
1. **Fix the broken build** — add `"strconv"` to `handlers_aggregates.go` imports
2. **Run `go test ./... -race`** to verify aggregate pagination doesn't break existing tests
3. **Write tests for command detail handler** — `TestCommandDetailHandler_Renders`, `TestCommandDetailHandler_NotFound`, `TestCommandDetailHandler_InvalidID`
4. **Write tests for query detail handler** — `TestQueryDetailHandler_Renders`, `TestQueryDetailHandler_NotFound`
5. **Write tests for DLQ entry detail handler** — `TestDLQEntryDetailHandler_Renders`, `TestDLQEntryDetailHandler_NotFound`
6. **Write test for aggregate timeline pagination** — `TestAggregateDetail_Pagination` with >50 events
7. **Fix `doc.go`** — remove false "templ-components" claim, describe actual rendering approach
8. **Update `CHANGELOG.md`** with all improvements from this session

### Remaining Pareto plan tasks
9. **T10: Total count display** — add `"Showing 1–N of M"` to events, aggregates, commands, queries pages
10. **T12: Sortable column headers** — `sortState` struct + `parseSort(r)` helper + clickable headers on events table
11. **T13: Page-size selector** — `<select>` dropdown with 25/50/100/200 options wired to `?limit=`
12. **T14: HTMX filter form** — add `hx-get` + `hx-target` + `hx-push-url` to event filter form for partial swap
13. **T15: Payload copy button + JSON download** — copy button next to `<pre>` blocks, download as `.json`
14. **T16: Projection detail view** — `GET /projections/{name}` route + handler + renderer
15. **T17: Keyboard nav for time-travel** — `keydown` listener for arrow keys on focused slider
16. **T18: Command/query status badge** — check if `PersistedCommand` has success/duration fields

### Test coverage gaps
17. **Test HTMX partial on all index handlers** — events, aggregates, commands, queries, DLQ, projections, time-travel, snapshots
18. **Test SSE `dashboard:event` listener** — verify JS exists in served `dashboard.js`
19. **Test `paginateEventsByVersion`** — edge cases: 0 events, exactly PageSize events, PageSize+1 events
20. **Test `prettyJSON`** — valid JSON, invalid JSON, empty bytes
21. **Test `loadCommandByID` with SeekableCommandJournal** — verify scan loop terminates
22. **Test `loadQueryByID` with SeekableQueryJournal** — verify scan loop terminates
23. **Test DLQ entry detail with `entry.Event != nil`** — verify payload rendering
24. **Run coverage report** — verify it didn't drop below 84.8%

### Polish and hardening
25. **Fix SSE JS row prepend** — adapt column count to the active page's table structure
26. **Guard `paginateEventsByVersion` against zero PageSize**
27. **Extract shared `scanJournal[T]` generic helper** — DRY up loadXByID functions
28. **Reuse `DefaultPayloadRenderer` in `prettyJSON`** instead of duplicating
29. **Add `HX-Request` header to 404 handler** — so 404s during boost also return partials
30. **Update `README.md`** — document `?prev=`, HTMX partials, SSE live updates, new detail routes
31. **Lint check** — `golangci-lint run --max-issues-per-linter 0 --max-same-issues 0`
32. **Coverage gate** — `nix run .#coverage` to verify module gate passes

### Future sessions (lower priority)
33. **CSV export** (`?format=csv`) for events/commands/DLQ tables
34. **JSON API mode** (`?format=json`) for programmatic access
35. **Demo with seeded data** — events, projections, DLQ, commands, queries
36. **Templ migration evaluation document** — pros/cons/risk/decision
37. **Correlation/causation ID search** — find all events in a causal chain
38. **Date range filter** on events page
39. **Aggregate search by ID**
40. **DLQ batch operations** — select + bulk delete/replay
41. **Dark/light mode toggle** — localStorage persistence
42. **Keyboard shortcuts** — `g o`, `g e`, `/` for search
43. **Command palette** — `Cmd+K` fuzzy nav
44. **Column visibility toggle** — hide/show columns
45. **Sticky table headers** on scroll
46. **Zebra striping** for readability
47. **Right-aligned numeric columns**
48. **JSON syntax highlighting** in payload `<pre>` blocks
49. **Minify `dashboardJS`** for production
50. **Add CSP-friendly external JS file** via `embed` instead of string constant

---

## g) Questions I Cannot Answer Myself

### Q1: Should the `Dashboard` struct field be `cfg` or `config`?
The auto-commit daemon renamed `cfg` to `config` between sessions (commit `a956132` or earlier). I don't know if this was an intentional rename by a previous agent session or an accidental side effect of the adminui templ-components migration that was committed together. The rest of the codebase (adminui, loginpage) uses `cfg` — should I rename it back for consistency, or keep `config`?

### Q2: Should I continue executing the full Pareto plan, or pause for a review?
The user said "GET SHIT DONE! THE WHOLE TODO LIST!" but then halted twice for status. Should I continue grinding through T10-T18, or does the user want to review/test what's been built so far before adding more? The broken build (missing `strconv` import) needs fixing first regardless.

### Q3: The auto-commit daemon keeps committing my work before I've written tests. Should I disable it, or write tests first before any implementation code?
Three feature commits went out without tests because the daemon committed between my implementation and test-writing steps. This means HEAD is dirty (untested code in history). Options: (a) disable auto-commit, (b) always write tests first (TDD), (c) accept it and add tests in follow-up commits. What's the preferred approach?
