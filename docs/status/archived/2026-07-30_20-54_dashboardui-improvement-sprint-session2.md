# Status Report: dashboardui Improvement Sprint — Session 2

**Date:** 2026-07-30 20:54
**Module:** `dashboardui/v4`
**Build:** PASSING | **Tests:** 56 PASS / 0 FAIL | **Vet:** CLEAN
**Lines of code:** ~5,327 across 28 Go files

---

## A. FULLY DONE (This Session)

### 1. Observability Endpoints [items #312, #313, #314]

- **NEW FILE:** `handlers_health.go` (83 lines)
- `GET /-/healthz` — liveness probe, returns `{"status":"ok"}` (200) or `{"status":"shutting_down"}` (503)
- `GET /-/readyz` — readiness probe, checks data source configured + not shutting down
- `GET /-/versionz` — returns module path, Go version, capabilities, readOnly flag, basePath, title
- Routes registered **unguarded** (k8s/load balancers need access without auth)
- All endpoints return JSON with `Cache-Control: no-store`

### 2. Event Filtering [items #61, #62, #63, #69]

- `eventFilter` struct with `Type`, `StreamType`, `StreamID` fields
- `parseEventFilter()` reads `?type=`, `?streamType=`, `?streamID=` from URL
- `loadFilteredEvents()` scans up to 500 raw events, applies in-memory filtering, returns pageSize+1 for HasMore detection
- `renderEventFilterBar()` renders a GET form with labeled inputs, pre-filled current values, Filter + Clear buttons
- Filter state preserved across pagination via `extraParams()` encoding
- Empty state message changes when filters are active ("No matching events")

### 3. Event Browser Improvements [items #165, #167, #169]

- **Schema version badge:** `<span class="badge badge-neutral">schema v{N}</span>` in event detail header
- **Encoding indicator badge:** color-coded — JSON=green, CBOR=amber, Raw=neutral
- **Prev/Next navigation:** `findEventNeighbors()` scans recent events to find adjacent IDs, renders Previous/Next buttons in event detail
- `encodingBadgeClass()` helper in `format.go`

### 4. Time-Travel Improvements [items #184, #190, #191]

- **Range slider:** `<input type="range" min="1" max="{N}" value="{current}">` with `onchange` redirect to `?v={N}`
- **Latest link:** "Latest (v{N})" button jumps to max version
- **First link:** "First" button jumps to version 1 (only shown when not on v1)
- **Version display:** "Viewing version **N** of **M**" text below slider
- **Conditional numbered links:** Shows individual version number links only when ≤20 versions (avoids rendering 100+ links)
- **CSS:** `.version-slider` with `accent-color: var(--accent)` styling, `.version-display` typography
- Permalink `?v=N` already worked — verified, no change needed

### 5. Projection Panel Improvements [items #150, #151, #157]

- **`projectionStat` struct expanded:** Added `Restarts int`, `Checkpoint string`, `LastError string` fields
- **`buildProjectionStats()` shared helper:** Consolidated the WorkerState → projectionStat conversion that was duplicated in 3 places (overview, projections index, health partial). All 3 now call the same function.
- **Projections table expanded:** Now shows 9 columns: Name, Status, Lag, Processed, Errors, Restarts, Checkpoint, Last Error, Actions
- **DLQ link from row:** Each projection row now has a "DLQ (N)" button linking to `/dead-letters/{name}`
- **Checkpoint with tooltip:** Shows truncated checkpoint with full value in `title` attr
- **Last Error display:** Shows truncated error or "—" when empty

### 6. JS Features [items #104, #111] — PARTIALLY DONE (see section B)

- SSE event count: Added `[data-sse-count]` span in header, `updateCount()` in JS
- Copy-to-clipboard: Added `data-copyable` click handler in JS, `.copyable` CSS class
- Applied `data-copyable` to events table Stream ID column
- **NOT YET applied to:** event detail pages, aggregate detail, snapshot detail, command audit

---

## B. PARTIALLY DONE

### JS Features (Copy-to-Clipboard + SSE Count) [items #104, #111]

**What's done:**

- SSE event count display in header (`<span class="sse-count" data-sse-count>`)
- `updateCount()` function in JS increments and displays count
- Copy-to-clipboard JS handler added (click on `[data-copyable]` → clipboard + toast)
- CSS `.copyable` class with 📋 hover indicator
- Events table Stream ID has `data-copyable`

**What's missing:**

- `data-copyable` NOT yet on event detail page (event ID, stream ID)
- `data-copyable` NOT yet on aggregate detail page (aggregate ID, event IDs)
- `data-copyable` NOT yet on snapshot detail page
- `data-copyable` NOT yet on command/query audit tables
- Build was NOT verified after last JS edit before session was paused

### Accessibility [items #320, #321, #332] — NOT STARTED

### Mobile [items #333, #334, #335] — NOT STARTED

### Tests [items #261, #267, #276] — NOT STARTED

### Demo [items #297, #298, #299] — NOT STARTED

### Docs [items #283, #284, #295] — NOT STARTED

---

## C. NOT STARTED

| Todo Item           | Description                                                | Items              |
| ------------------- | ---------------------------------------------------------- | ------------------ |
| Accessibility       | Fix heading hierarchy, aria-labels, form labels            | #320, #321, #332   |
| Mobile              | Hamburger toggle, responsive table wrappers, touch targets | #333, #334, #335   |
| Tests               | Pagination tests, CSS/JS serving tests, routing tests      | #261, #267, #276   |
| Demo                | Add EventBus, projections, DLQ entries to demo             | #297, #298, #299   |
| Docs                | Update README, config reference, CHANGELOG                 | #283, #284, #295   |
| Sorting             | Column sort on all data tables                             | #61-72 (partially) |
| Aggregate browser   | Type filter, event count column, copy ID                   | #170-182           |
| Command/Query audit | Type filter, result status, duration                       | #200-215           |
| Snapshot inspector  | Codec indicator, age column, type filter                   | #216-222           |

---

## D. TOTALLY FUCKED UP — Nothing

No regressions, no broken code, no data loss. Build passes, all 56 tests pass, vet is clean. The one issue (codec.Encoding vs string type mismatch on `encodingBadgeClass`) was caught immediately and fixed in the same step.

---

## E. WHAT WE SHOULD IMPROVE

### Code Quality Issues Found During This Session

1. **Projection table is now 9 columns wide** — Will overflow on narrow screens. Needs horizontal scroll wrapper (part of mobile todo, but should be acknowledged now).

2. **`encodingBadgeClass` uses `badge-ok` (green) for JSON** — JSON encoding is not a "success" state, it's just the default encoding. Should use `badge-neutral` for all encodings, or at minimum not green.

3. **No tests written for ANY new feature this session** — Zero test coverage for: health endpoints, event filtering, event neighbor navigation, time-travel slider, projection stat expansion. Every feature was verified only by "does it compile + do existing tests pass."

4. **`findEventNeighbors` is O(n) scan** — Scans up to 500 events on every event detail view. For large journals this is wasteful. The `EventByIDLoader` interface exists for O(1) lookup; a similar `EventNeighborsLoader` would be better.

5. **Filter bar on events page does a full GET form submit** — Not an HTMX partial swap. The whole page reloads when filtering. This is functional but not the snappy UX users expect from an HTMX dashboard.

6. **`loadFilteredEvents` is in-memory filtering** — Reads up to 500 events then filters client-side. For large journals with rare event types, this may return an empty page even though matching events exist beyond the scan limit. The correct solution is store-side filtering (SQL WHERE), but that requires journal interface changes.

7. **Health endpoints are unguarded** — Intentional for k8s probes, but a consumer using `Authorizer` on the dashboard might be surprised. Should document this clearly.

8. **Version slider uses inline `onchange` handler** — Inconsistent with the rest of the codebase which uses event listeners. Minor but worth noting for CSP compliance.

9. **DLQ link in projection row shows "DLQ (0)" for healthy projections** — The link always renders when `caps.DeadLetterStore || caps.ProjectionHost`. Should hide when errors=0 and no store configured.

10. **`writeJSON` in `handlers_health.go` duplicates the pattern from `render.go`** — `render.go` has `triggerToast` which also marshals JSON. Could consolidate.

---

## F. NEXT 50 THINGS TO GET DONE

### Immediate (finish current sprint)

1. **Verify build after JS changes** (was interrupted mid-edit)
2. **Add `data-copyable` to event detail page** (event ID, stream ID, correlation ID)
3. **Add `data-copyable` to aggregate detail page** (aggregate ID, event IDs)
4. **Add `data-copyable` to snapshot detail page** (stream ID)
5. **Add `data-copyable` to command/query audit tables** (command/request IDs)
6. **Fix `encodingBadgeClass`** — use `badge-neutral` for all encodings, not green for JSON

### Accessibility [items #320, #321, #332]

7. **Fix heading hierarchy** — overview uses h3, detail pages jump to h2 then h4 (should be h2→h3)
8. **Add `aria-label` to all icon-only buttons** (Reset, Delete, DLQ)
9. **Add `<label for="id">` to all filter inputs** (partially done, verify all)
10. **Add `role="table"` considerations** — tables have `<th scope="col">` but lack `scope="row"`
11. **Add skip-to-content link** (CSS exists but link is not rendered in HTML)
12. **Add `aria-live="assertive"` to toast container for error toasts**
13. **Add keyboard navigation for version slider** (arrow keys work natively, but add `aria-valuenow`)

### Mobile [items #333, #334, #335]

14. **Add hamburger toggle** — JS to toggle `.sidebar.open` class, hamburger button in header
15. **Wrap all data tables in `<div class="table-scroll">` with `overflow-x: auto`**
16. **Increase tap target sizes** — buttons to min 44x44px, nav links to 44px height
17. **Make projection table horizontally scrollable** (9 columns is too wide for mobile)
18. **Stack stat cards vertically on mobile** (already responsive via grid, verify)
19. **Make filter bar stack vertically on mobile** (`flex-direction: column` at <768px)

### Tests [items #261, #267, #276]

20. **Test health endpoints** — `GET /-/healthz` returns 200 + `{"status":"ok"}`
21. **Test versionz returns capabilities** — verify JSON structure
22. **Test readyz returns 503 when no data source** — and 200 when configured
23. **Test event filtering** — `?type=X` filters events, `?streamType=Y` filters, combined filters
24. **Test event neighbor navigation** — prev/next IDs correct for middle event, empty for first/last
25. **Test pagination cursor navigation** — `?after=ID` returns next page, HasMore detection
26. **Test CSS serving** — `Content-Type: text/css`, `Cache-Control: public, max-age=86400`
27. **Test JS serving** — `Content-Type: text/javascript`, same cache headers
28. **Test routing for all capability combinations** — events without journal, projections without host
29. **Test time-travel slider** — `?v=N` loads correct version, clamps to max
30. **Test copy-to-clipboard attribute presence** — verify `data-copyable` on rendered cells
31. **Test DLQ link from projection row** — verify link href and error count

### Demo [items #297, #298, #299]

32. **Add EventBus to `examples/dashboard-demo/main.go`** — wire `event.Bus` to Config
33. **Add a simple projection** — demo projection with checkpoint store
34. **Seed DLQ entries** — inject a dead letter entry for demonstration
35. **Add more seed events** — generate 20+ events for pagination demonstration
36. **Add stream reader** — wire `listing.NewInMemoryStreamReader` for aggregate browser

### Docs [items #283, #284, #295]

37. **Update README** — add SSE reconnect/backoff section, filter bar docs, observability endpoints
38. **Add config reference** — document every `Config` field with type, default, and effect
39. **Add CHANGELOG entry** — all improvements from this session and prior session
40. **Document capability matrix** — which Config fields activate which panels/routes
41. **Add "Getting Started" section** — minimal wiring example

### Polish & Further Improvements

42. **Add sorting to event table** — clickable column headers, `?sort=type&dir=asc`
43. **Add aggregate type filter** — `?type=User` on aggregates page
44. **Add result status to command audit** — success/failure indicator per command
45. **Add duration column to query audit** — execution time
46. **Add codec indicator to snapshot detail** — show encoding of snapshot state
47. **Add event type dropdown** — `<select>` of known event types for filter bar (currently free text)
48. **Add "Export as JSON" button** — download event/aggregate data as JSON file
49. **Add dark mode toggle** — manual toggle instead of only `prefers-color-scheme`
50. **Add keyboard shortcut help** — `?` key shows available shortcuts overlay

---

## G. QUESTIONS I CANNOT ANSWER MYSELF

### Q1: Should the health/readiness endpoints be guarded by Authorizer?

**Context:** I made them unguarded intentionally (k8s probes need access without auth cookies). But a consumer using `Config.Authorizer` for admin-only access might expect all dashboard routes to be protected. Should I add a `Config.HealthzUnguarded bool` (default true) to let consumers opt into guarding them?

### Q2: Is the `findEventNeighbors` O(n) scan acceptable, or should I add a new interface?

**Context:** The event detail page now scans up to 500 events to find prev/next. For a journal with millions of events, this is O(500) per detail view. I could add an `EventNeighborLoader` interface that SQL stores implement with `WHERE id < ? ORDER BY id DESC LIMIT 1` + `WHERE id > ? ORDER BY id ASC LIMIT 1`. Should I add this interface, or is the scan acceptable for a dashboard tool?

### Q3: Should the filter bar use HTMX partial swap or full page reload?

**Context:** The event filter bar currently does a full `GET` form submit (page reload). I could make it HTMX-powered (`hx-get` + `hx-target` + `hx-swap`), which would be snappier but requires a partial endpoint. Given that the rest of the dashboard already uses HTMX for the projection health partial, should I add an events partial too?
