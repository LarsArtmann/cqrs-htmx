# DashboardUI Improvement Ideas

> **Purpose:** A comprehensive, prioritized catalog of everything that can and should be improved in the `dashboardui/` module. Each idea is specific, actionable, and tagged with a severity (P0=critical, P1=high, P2=medium, P3=polish) and category.
>
> **Generated:** 2026-07-30 | **Source:** Full codebase audit of all 28 Go files (~4,080 lines).

---

## Table of Contents

1. [Critical Bugs & Correctness (P0)](#1-critical-bugs--correctness-p0)
2. [Architecture & Rendering (P1)](#2-architecture--rendering-p1)
3. [HTMX Integration (P1)](#3-htmx-integration-p1)
4. [Pagination & Data Loading (P1)](#4-pagination--data-loading-p1)
5. [Filtering & Search (P1)](#5-filtering--search-p1)
6. [Sorting & Table UX (P2)](#6-sorting--table-ux-p2)
7. [CSS & Styling (P1)](#7-css--styling-p1)
8. [JavaScript & SSE (P1)](#8-javascript--sse-p1)
9. [Security (P1)](#9-security-p1)
10. [DLQ Panel Improvements (P2)](#10-dlq-panel-improvements-p2)
11. [Projection Panel Improvements (P2)](#11-projection-panel-improvements-p2)
12. [Event Browser Improvements (P2)](#12-event-browser-improvements-p2)
13. [Aggregate Browser Improvements (P2)](#13-aggregate-browser-improvements-p2)
14. [Time-Travel Improvements (P2)](#14-time-travel-improvements-p2)
15. [Snapshot Inspector Improvements (P2)](#15-snapshot-inspector-improvements-p2)
16. [Command/Query Audit Improvements (P2)](#16-commandquery-audit-improvements-p2)
17. [Overview Page Improvements (P2)](#17-overview-page-improvements-p2)
18. [New Panels & Features (P3)](#18-new-panels--features-p3)
19. [API & Export (P2)](#19-api--export-p2)
20. [Testing & Quality (P1)](#20-testing--quality-p1)
21. [Documentation (P2)](#21-documentation-p2)
22. [Demo & Examples (P3)](#22-demo--examples-p3)
23. [Observability & Metrics (P3)](#23-observability--metrics-p3)
24. [Accessibility (P2)](#24-accessibility-p2)
25. [Mobile & Responsive (P2)](#25-mobile--responsive-p2)

---

## 1. Critical Bugs & Correctness (P0)

1. **Fix overview TotalAggregates stat** — `overviewStats` reads StreamReader with `Limit: 1` and reports `"1+"` when `HasMore` is true. This always shows "1+" regardless of actual count. Should either use a `Count()` method if available, or show a meaningful approximation like "50+" using PageSize.

2. **Fix overview TotalEvents stat** — Uses `recentEventsLimit` (5) as the read limit, so it always shows `"5+"`. Should read a larger batch or use a dedicated count method to produce an accurate-ish number.

3. **Fix `Close()` false warning** — `dashboard.go:133` logs `slog.Warn("no broadcaster configured")` when EventBus was never configured. This is expected behavior, not a warning. Should only warn if `caps.EventBus` is true but broadcaster is nil (which would indicate an actual bug).

4. **Fix SSE JS reconnection bug** — `layout.go:182`: `es.onerror` creates a new `EventSource` but **never re-attaches the `event` and `onerror` listeners** to the new instance. After the first disconnect, the dashboard goes permanently silent. The reconnect creates a zombie connection.

5. **Fix dead code import hacks** — `handlers.go:51-53` has `var _ = id.NewStreamID` and `var _ = event.Type("")` to suppress unused import errors. These are code smells indicating the imports are not actually needed and should be removed along with the unused imports.

6. **Fix `doc.go` false documentation** — Claims "composes templ-components for the UI" but the code uses raw Go string-builder HTML. The README contradicts this saying "Future iterations will migrate to templ-components." The doc.go should describe what IS, not what might be.

7. **Fix unused `pageData.LogoutURL`** — Always set to `""` in `dashboard.go:88`. Either implement logout support or remove the field.

8. **Fix unused `navItem.Icon` field** — Icons are assigned (`"chart"`, `"queue"`, `"cube"`, etc.) in `buildNav()` but **never rendered** in the sidebar. The sidebar shows only text labels. Either render icons (SVG) or remove the field.

9. **Fix `csrfMeta` always returns empty string** — `payload.go:83` is a stub that returns `""`. The layout never renders a CSRF meta tag. If CSRF is intended, this needs implementation; if not, remove the dead code.

10. **Fix `csrfToken` reads only form value** — `payload.go:79` reads `r.FormValue("_csrf")` which requires form parsing and body consumption. Should also check headers and nosurf context for the actual token.

11. **Fix event table does not escape HTML** — `handlers_events.go:242-244`: `evt.Type()`, `evt.StreamType()`, `evt.Version().String()` are written to HTML **without `esc()`**. This is an XSS vector if event types contain HTML special characters. (Stream ID in `handlers_events.go:243` is also unescaped.)

12. **Fix overview projection table does not escape** — `handler_overview.go:206`: projection name and status are written without escaping.

13. **Fix `rowsSbNNN` naming pattern** — Variables like `rowsSb72`, `rowsSb131`, `rowsSb209`, `rowsSb326` appear to be auto-generated names from a refactoring tool. These should be named meaningfully (e.g., `rowsBuilder` or just `rows`).

14. **Fix `rows` variable anti-pattern** — Several handlers declare `var rows string` then append a builder's output: `rows += rowsSbNNN.String()`. The intermediate `rows` variable is redundant — just use the builder directly.

15. **Fix CHANGELOG inaccuracy** — v4.0.0 entry mentions `templ_render.go` which doesn't exist in the codebase. The v4.0.0 entry also mentions "Templ rendering" and "DTOs" that don't match the actual implementation.

16. **Fix no 404 handling** — Unknown routes under the dashboard prefix return the stdlib default 404, not a branded error page. Should serve a styled "page not found" within the dashboard layout.

17. **Fix no method-not-allowed handling** — POST to a GET-only route returns stdlib 405. Should be styled consistently.

18. **Fix htmx.js served unconditionally but never used** — `layout.go:26` loads htmx.js on every page, but **no HTMX attributes exist anywhere in the generated HTML**. This is wasted bandwidth (htmx is ~14KB gzipped) and misleading to consumers.

---

## 2. Architecture & Rendering (P1)

19. **Migrate to templ-components** — The entire rendering layer is raw Go `strings.Builder` HTML with inline styles. This is unmaintainable, untestable, and error-prone (see XSS issues above). adminui already uses templ successfully. Migrate all `render*` methods to `.templ` files. This is the single highest-leverage improvement.

20. **Extract shared table renderer** — Every panel reimplements the same `<table>` HTML with the same inline styles. Extract a reusable `renderTable(headers, rows)` helper (or templ component) that handles consistent styling, sorting indicators, and responsive overflow.

21. **Extract shared card/stat component** — `statCard` is defined in `handler_overview.go` but every panel duplicates the card styling inline. Create a reusable card component.

22. **Extract shared empty-state component** — Every panel has its own `<div style="padding:40px;text-align:center;color:...">No X found</div>` block. Extract to a single `emptyState(message)` function.

23. **Extract shared metadata table component** — `metaRow` is shared but the surrounding `<table>` boilerplate is duplicated in event detail and snapshot detail. Extract a `metadataTable(rows)` component.

24. **Separate layout from content** — `renderLayout` takes a `func() string` closure for content. This forces all content to be fully rendered as a string before layout wraps it. With templ, the layout should be a wrapper component that accepts `templ.Component` children.

25. **Create a consistent page header component** — Every page starts with a different `<div style="margin-bottom:24px"><h2>` block with inconsistent fields shown. Create a `pageHeader(title, subtitle, breadcrumb)` component.

26. **Remove inline styles in favor of CSS classes** — The HTML is 90% inline `style=""` attributes. This bloats the HTML, prevents consistency, and makes dark mode harder. Move all styles to the CSS file and use class names.

27. **Consolidate CSS into a proper stylesheet** — `dashboardCSS` is only ~25 lines of actual CSS. The rest is inline. Consolidate all styling into `dashboard.css` with proper class names, utility classes, and component classes.

28. **Add CSS custom properties for all themeable values** — Currently only `--accent` is themeable. Add properties for sidebar width, border radius, spacing scale, font sizes, etc.

29. **Create a rendering test helper** — Each test manually constructs a Dashboard, mounts it, fires a request, and checks `strings.Contains`. Extract a `testRender(t, dash, path) *httptest.ResponseRecorder` helper.

30. **Separate data loading from rendering** — Handlers mix data loading (calling journal/store methods) with HTML rendering in the same function. Separate into `loadX(ctx) (data, error)` and `renderX(data) string` for testability.

31. **Add a `pageData.RequestPath` field** — Currently handlers pass raw strings for active nav matching. Include the request path in pageData for consistent nav highlighting and breadcrumb generation.

32. **Create a response writer wrapper** — Centralize Content-Type, Cache-Control, and error handling in a single `writeResponse(w, r, html)` instead of the current `writeHTML` + `renderPage` + direct `http.Error` mix.

33. **Standardize error responses** — Some handlers use `http.Error(w, "msg: "+err.Error(), 500)` (leaking internal errors), others use `triggerToast`. Create a unified `renderError(w, r, statusCode, message)` that renders within the layout for HTMX requests and plain text otherwise.

34. **Don't leak internal errors to clients** — `http.Error(w, "failed to load events: "+err.Error(), ...)` exposes internal error details. Log the full error, show a generic message to the user.

35. **Add request-scoped logging** — Pass `r.Context()` to `slog` calls consistently. Currently `writeHTML` logs with context but handler error paths don't log at all.

---

## 3. HTMX Integration (P1)

36. **Actually use HTMX for partial rendering** — The dashboard loads htmx.js but never uses it. Add `hx-get` attributes to pagination links so they swap only the table content, not the full page.

37. **Add HTMX polling for live data** — Overview page should use `hx-get="/-/partial/overview-stats" hx-trigger="every 5s"` to auto-refresh stats without full page reload.

38. **Add HTMX polling for projection health** — Projection table should auto-refresh via `hx-trigger="every 10s"` on the table element.

39. **Add HTMX boost to sidebar links** — Use `hx-boost="true"` on the `<nav>` element so all navigation is HTMX-powered (faster page transitions, no full reload).

40. **Add HTMX indicators for write operations** — DLQ replay, projection reset, snapshot delete should show an `hx-indicator` spinner during the request.

41. **Add OOB swaps for toast notifications** — Write operations should use `hx-swap-oob` to update a toast container instead of relying on `Hx-Trigger` events that have no corresponding JS listener.

42. **Implement toast rendering** — `triggerToast` sets the `Hx-Trigger` header with toast data, but **there is no HTML element or JS to display toasts**. Add a toast container to the layout and JS to render `showToast` events.

43. **Use HTMX for time-travel version slider** — Instead of full-page links, use `hx-get` with `hx-target="#timeline"` so switching versions only swaps the timeline content.

44. **Use HTMX for event detail loading** — In aggregate event timelines, clicking an event should load its detail in a side panel via HTMX, not navigate away.

45. **Add `hx-push-url` to filtered views** — When filtering/searching via HTMX, push the URL so browser back button works.

46. **Use HTMX extensions** — The SSE extension is already available in the root module. Use `hx-ext="sse"` with `sse-connect` for real-time table updates instead of the custom JS EventSource code.

47. **Add loading states** — HTMX requests should show a skeleton/spinner in the target element via `htmx:afterRequest`/`htmx:beforeRequest` event listeners.

48. **Add HTMX history restoration support** — Currently no `hx-history` or `HX-History-Restore-Request` handling. Ensure partial vs full rendering works correctly with browser history.

---

## 4. Pagination & Data Loading (P1)

49. **Implement actual pagination for events** — Events page loads `PageSize` items from the beginning of the journal and stops. No next/prev buttons, no page numbers. Add cursor-based pagination using `SeekableJournal.ReadFrom(after, limit)`.

50. **Implement pagination for aggregates** — Aggregate browser loads one page and stops. Use `StreamReader.List()` with cursor/offset to paginate.

51. **Implement pagination for commands** — Command audit loads one page and stops. Use `SeekableCommandJournal.ReadFrom` for pagination.

52. **Implement pagination for queries** — Query audit loads one page and stops. Use `SeekableQueryJournal.ReadQueriesFrom` for pagination.

53. **Implement pagination for DLQ** — Dead letter list loads all entries with no pagination. Use `DeadLetterStoreAdmin.ListPaged` when available.

54. **Add page-size selector** — Let users choose 25/50/100/200 rows per page. Currently fixed at config time.

55. **Add "load more" infinite scroll option** — Alternative to pagination: append rows via HTMX `hx-get` with `hx-trigger="revealed"` on a sentinel element.

56. **Add total count display** — Show "Showing 1-50 of 1,247" when the total is knowable.

57. **Handle `HasMore` properly** — `StreamReader.List` returns `HasMore` but the dashboard ignores it. Use it to show/hide the "Next" button.

58. **Support reverse chronological order** — Events are read from the beginning (oldest first). Add a toggle for newest-first ordering.

59. **Add deep-linking for page state** — `?page=2&cursor=abc123` should restore the exact view on reload.

60. **Lazy-load aggregate detail events** — Aggregates with thousands of events load ALL events at once. Paginate the event timeline within the detail view.

---

## 5. Filtering & Search (P1)

61. **Add event type filter** — Events page should have a filter dropdown/input to show only events of a specific type (e.g., `user.created`).

62. **Add stream type filter** — Events page should filter by stream type (e.g., `User`, `Order`).

63. **Add stream ID filter** — Events page should filter by specific stream/aggregate ID.

64. **Add free-text search** — Global search box that searches event types, stream IDs, and event IDs.

65. **Add aggregate type filter** — Aggregate browser should filter by stream type.

66. **Add date range filter** — Events page should filter by occurred-at date range.

67. **Add command type filter** — Command audit should filter by command type.

68. **Add projection filter** — DLQ should show a dropdown of projections that have dead letters.

69. **Add URL-synced filters** — Filter state should be encoded in URL query params so filters survive page reload and are shareable.

70. **Add HTMX-powered filter updates** — Filters should update the table via HTMX partial swap, not full page reload.

71. **Add filter presets** — Quick-filter buttons like "Errors only", "Last hour", "User events".

72. **Add correlation/causation ID search** — Search for all events in a causal chain by correlation ID.

---

## 6. Sorting & Table UX (P2)

73. **Add sortable columns** — All tables should have clickable headers that sort ascending/descending.

74. **Add sort indicators** — Show `▲`/`▼` icons on the active sort column.

75. **Preserve sort in URL** — Sort state in query params (`?sort=time&dir=desc`).

76. **Add row hover highlighting** — Tables have no hover state. Add `tr:hover` background change.

77. **Add zebra striping** — Alternating row colors for readability in large tables.

78. **Add sticky table headers** — When scrolling a long table, the header row should stick to the top.

79. **Add column visibility toggle** — Let users hide/show columns (e.g., hide Event ID when not needed).

80. **Add clickable rows** — Event rows in aggregate timeline should be clickable to open event detail, not just the type link.

81. **Add right-aligned numeric columns** — Version, event count, processed, errors should be right-aligned.

82. **Add relative time display** — "2 minutes ago" instead of raw RFC3339 timestamps, with full timestamp in tooltip.

83. **Add human-readable byte sizes** — Snapshot state size should show "12.3 KB" not "12648 bytes".

---

## 7. CSS & Styling (P1)

84. **Fix stat card dark background** — `statCard` uses hardcoded `#1d3557` and `#16a34a` backgrounds that clash with the CSS variables system. Should use `var(--accent)` or dedicated CSS variables.

85. **Fix overview hardcoded colors** — Overview table uses hardcoded `#e6e8ec`, `#64748b`, `#16a34a`, `#d97706`, `#dc2626` instead of CSS variables. These won't adapt to dark mode.

86. **Fix projection status colors hardcoded** — Same issue in `handlers_projections.go:87-96` — hardcoded colors that ignore the CSS variable system.

87. **Fix empty-state hardcoded colors** — Multiple panels use `color:#64748b` instead of `var(--muted)` for empty states.

88. **Add proper dark mode support** — CSS has `@media (prefers-color-scheme: dark)` but most styling is inline with hardcoded light-mode colors. Dark mode is broken.

89. **Add a dark/light mode toggle** — User should be able to toggle theme regardless of OS preference. Store in localStorage.

90. **Fix sidebar always dark** — Sidebar uses hardcoded `#0f172a` background regardless of theme. In dark mode this blends with the page background.

91. **Add proper focus styles** — No `:focus` or `:focus-visible` styles on links or buttons. Keyboard navigation is invisible.

92. **Add transition animations** — No CSS transitions anywhere. Hover states, theme changes, and content swaps are instant and jarring.

93. **Fix header transparency** — Header uses `color-mix(in srgb, white 86%, transparent)` which is broken in dark mode (white on dark).

94. **Add print stylesheet** — Dashboard should be printable for audit reports.

95. **Add consistent button styles** — Buttons are styled inline per-use. Create `.btn`, `.btn-danger`, `.btn-secondary` classes.

96. **Add badge/pill styles** — Projection status, error family, etc. should use consistent badge components.

97. **Add code/monospace styles** — `<code>` has basic styling but inline monospace spans (`font-family:monospace;font-size:0.85em`) are repeated everywhere. Use a `.mono` class.

98. **Add proper pre/code block styling** — Payload `<pre><code>` blocks need syntax highlighting, line numbers, and copy button.

99. **Add CSS for forms** — Filter inputs, select dropdowns, and buttons need consistent form styling.

100.  **Reduce CSS specificity issues** — Inline styles have highest specificity, making overrides impossible. Moving to classes fixes this.

---

## 8. JavaScript & SSE (P1)

101. **Remove `console.log` from production JS** — `layout.go:184`: `console.log("dashboardui loaded with live updates")` should not be in production.

102. **Fix SSE reconnection properly** — Rewrite the dashboard JS to use a proper reconnection manager that re-attaches all listeners on each reconnect.

103. **Add SSE connection status indicator** — Show "Connected" / "Reconnecting..." / "Disconnected" in the header, not just a blinking dot.

104. **Add SSE event count** — Show how many events have been received in the current session.

105. **Use `dashboard:event` for live table updates** — The custom event is dispatched but nothing listens. Add listeners that update projection health and recent events tables in place.

106. **Add client-side event filtering** — Let users filter which SSE events trigger UI updates (e.g., only `user.*` events).

107. **Add exponential backoff for reconnection** — Fixed 5-second delay is too aggressive. Use exponential backoff with jitter.

108. **Add visibility-aware SSE** — Close SSE connection when tab is hidden (Page Visibility API), reconnect on focus. Saves server resources.

109. **Handle SSE max reconnect attempts** — Give up after N failed reconnections and show a "connection lost" banner.

110. **Add JSON payload syntax highlighting** — Payload `<pre>` blocks should have JSON syntax highlighting (client-side, lightweight library or custom highlighter).

111. **Add copy-to-clipboard for IDs** — Click any ID to copy it to clipboard. Show a tooltip "Copied!".

112. **Add keyboard shortcuts** — `g o` for overview, `g e` for events, `g a` for aggregates (vim-style navigation), `/` to focus search.

113. **Add command palette** — `Cmd+K` opens a fuzzy-search command palette for navigating panels.

114. **Minify dashboard JS** — The JS is served raw. Minify for production.

115. **Remove inline script** — Dashboard JS is embedded in a Go string constant. Consider using `embed` for a proper `.js` file.

116. **Add CSP-friendly JS** — Current inline script execution may be blocked by strict CSP. Move to external file or add `nonce`.

117. **Add `Last-Event-ID` header reading on client** — The server supports it but the JS doesn't explicitly set it (EventSource handles it natively, but verify it works through proxies).

118. **Add window beforeunload for SSE cleanup** — Ensure SSE connections are closed cleanly on page navigation.

---

## 9. Security (P1)

119. **Fix XSS in event rendering** — Event types, stream types, stream IDs, and versions are rendered without HTML escaping in multiple handlers (see P0 items).

120. **Add Content-Security-Policy** — No CSP is set. The dashboard serves inline scripts and styles. Add a nonce-based CSP.

121. **Sanitize error messages** — `http.Error(w, "msg: "+err.Error(), ...)` can leak internal paths, SQL queries, or stack info. Sanitize or genericize all user-facing error messages.

122. **Add rate limiting** — No rate limiting on dashboard endpoints. A malicious user could DoS the event journal scanning.

123. **Add authentication guidance** — `Authorizer` is optional and nil by default. The README says "consumer MUST wrap with auth middleware" but there's no runtime warning. Add a startup log warning when `ReadOnly: false` and `Authorizer == nil`.

124. **Audit all write operations** — DLQ replay, projection reset, snapshot delete should be logged with the actor's identity (from auth middleware context).

125. **Add CSRF protection** — Forms include `_csrf` hidden inputs but there's no CSRF validation middleware. The `csrfToken` helper reads the value but never validates it.

126. **Add confirmation dialogs for destructive actions** — Snapshot delete, DLQ purge, projection reset have no JavaScript confirm dialog. One-click destructive actions are dangerous.

127. **Add audit trail** — Write operations (reset, replay, delete, purge) should be logged to a dedicated audit log with timestamp, actor, target, and result.

128. **Validate all path parameters** — Stream type and ID from path values are used in store queries. Ensure they're validated before use (some are, some aren't).

129. **Add `X-Content-Type-Options: nosniff`** — Already applied via SecurityHeadersMiddleware, but verify CSS/JS responses also get it.

130. **Prevent clickjacking** — Add `X-Frame-Options: DENY` or CSP `frame-ancestors 'none'` since the dashboard should never be embedded.

131. **Add session timeout warning** — If wrapped with session auth, the dashboard should detect session expiry and prompt re-authentication.

---

## 10. DLQ Panel Improvements (P2)

132. **Add DLQ index with projection list** — The DLQ index page is a placeholder that says "Select a projection." It should **list all projections that have dead letters** with counts, so users know where to look.

133. **Add DLQ entry detail view** — Clicking a dead letter should show the full event payload, error stack trace, and retry history.

134. **Add DLQ batch operations** — Select multiple entries for bulk delete or replay.

135. **Add DLQ error grouping** — Group entries by error code or error family to identify patterns.

136. **Add DLQ age column** — Show how long ago each entry was dead-lettered.

137. **Add DLQ retry count** — Show how many times each event was retried before being dead-lettered.

138. **Add DLQ export** — Export dead letters as JSON for offline analysis.

139. **Use `DeadLetterStoreAdmin` when available** — Type-assert to access `Count()`, `ListPaged()`, and `PurgeBefore()` for better UX.

140. **Add DLQ search by event ID** — Direct lookup of a specific dead-lettered event.

141. **Add DLQ filter by error family** — Show only rejections, conflicts, etc.

142. **Add DLQ auto-refresh** — Poll for new dead letters when the page is open.

143. **Show original event in DLQ entry** — `DeadLetterEntry.Event` holds the original event. Display its payload and metadata.

144. **Add "select all failing" for replay** — After replay, offer to delete only successfully replayed entries.

---

## 11. Projection Panel Improvements (P2)

145. **Add projection detail view** — Clicking a projection name should show detailed status: checkpoint position, last processed event, error history, restart count.

146. **Add projection lag sparkline** — Mini chart showing lag over time.

147. **Add projection throughput metric** — Events processed per second/minute.

148. **Add projection error rate** — Percentage of events that caused errors.

149. **Add projection uptime** — How long since last restart.

150. **Show `WorkerState.Restarts`** — The WorkerState struct has a `Restarts` field that the dashboard doesn't display.

151. **Show `WorkerState.LastError`** — The WorkerState struct has a `LastError` field that the dashboard doesn't display.

152. **Show `WorkerState.Checkpoint`** — The checkpoint cursor position is available but not shown.

153. **Add projection reset confirmation** — Show which projection will be reset and what data will be lost.

154. **Add projection auto-refresh** — Poll projection status via HTMX.

155. **Color-code lag severity** — Green (0s), yellow (<10s), red (>60s) based on lag thresholds.

156. **Add projection health timeline** — Historical health data over time (requires storing snapshots).

157. **Link projection to its DLQ** — From projection detail, link directly to its dead letters.

158. **Add projection pause/resume** — If the host supports it, pause and resume individual projections.

---

## 12. Event Browser Improvements (P2)

159. **Add event detail deep-linking** — Event detail URLs should be shareable and survive journal replays (they use EventID, which is good).

160. **Add event payload diff** — When two events of the same type are adjacent, show a diff of their payloads.

161. **Add event metadata exploration** — Expand custom metadata with collapsible sections.

162. **Add event chain visualization** — Show the full event chain for a stream (correlation/causation graph).

163. **Add event export** — Download event payload as JSON file.

164. **Add event timeline visualization** — Visual timeline showing events on a horizontal axis by timestamp.

165. **Add event schema version display** — Show schema version prominently with a link to the schema if an EventCatalog is available.

166. **Add raw event view** — Toggle between pretty-printed and raw payload bytes.

167. **Add encoding indicator** — Show whether payload is JSON/CBOR/Raw with a badge.

168. **Add event ID copy button** — One-click copy of the full event ID.

169. **Add prev/next event navigation** — From event detail, navigate to the previous/next event in the journal.

170. **Add related events** — Show other events in the same stream or with the same correlation ID.

171. **Add event signing status** — If events are signed, show verification status.

---

## 13. Aggregate Browser Improvements (P2)

172. **Add aggregate search by ID** — Direct lookup of an aggregate by its stream ID.

173. **Add aggregate type grouping** — Group aggregates by stream type with counts.

174. **Add aggregate version column sorting** — Sort by version, event count, last event time.

175. **Add aggregate event count sparkline** — Mini chart of event frequency.

176. **Add aggregate state reconstruction** — Show the folded/computed current state from events (requires a decider/fold function).

177. **Add aggregate link to time-travel** — Already exists but should be more prominent.

178. **Add aggregate link to snapshots** — If snapshot store is configured, link from aggregate to its snapshot.

179. **Show `listing.StreamStatus`** — StreamReader supports `ListWithStatus` which returns tombstone status. Use it to show active/deleted streams.

180. **Add aggregate event timeline graph** — Visual representation of event versions on a number line.

181. **Add aggregate age** — How long since the aggregate was created (first event timestamp).

182. **Add aggregate last-modified relative time** — "Modified 3 minutes ago".

183. **Add aggregate event type distribution** — Pie chart or breakdown of event types within the aggregate.

---

## 14. Time-Travel Improvements (P2)

184. **Add actual range slider** — Version navigation uses clickable number links. Replace with an HTML `<input type="range">` for smooth scrubbing.

185. **Add keyboard navigation** — Arrow left/right to move between versions.

186. **Add state reconstruction at version** — Show the computed aggregate state at the selected version, not just the event list.

187. **Add version diff** — Show what changed between the selected version and the previous version.

188. **Add timestamp-based time travel** — `LoadToTimestamp` exists on EventSource. Add a date picker to travel to a point in time.

189. **Add event highlighting at current version** — In the timeline, visually distinguish events at/before the selected version from those after.

190. **Add "latest" quick link** — Jump back to the latest version.

191. **Add time-travel permalink** — `?v=42` URL should be shareable and survive reloads.

192. **Add parallel event/state view** — Show events on the left, computed state on the right, updating as the version slider moves.

193. **Add forward/backward browser-style navigation** — Track time-travel history within the session.

194. **Add version annotations** — Let users annotate specific versions (requires backend storage).

195. **Add snapshot markers on timeline** — If snapshots exist for this aggregate, mark their positions on the version slider.

196. **Add event compression for large histories** — For aggregates with 1000+ events, paginate or virtualize the timeline.

---

## 15. Snapshot Inspector Improvements (P2)

197. **Add snapshot comparison** — Compare snapshot state with current event-sourced state to detect drift.

198. **Add snapshot list by aggregate** — Show all snapshots for a specific aggregate, not just the latest.

199. **Add snapshot age** — How long since the snapshot was created.

200. **Add snapshot creation trigger** — Show why the snapshot was created (interval, manual, version threshold).

201. **Add snapshot restore** — Option to restore aggregate state from a snapshot (if supported by the store).

202. **Add snapshot size visualization** — Show state size with a visual indicator.

203. **Add snapshot state diff with events** — Show events that occurred since the snapshot version.

204. **Add snapshot creation** — Manual "create snapshot now" button for an aggregate.

205. **Show snapshot codec** — Display the encoding/codec used for the snapshot state.

206. **Add snapshot validation** — Verify the snapshot can be decoded by the expected type.

207. **Link snapshot to time-travel** — Jump to time-travel at the snapshot version.

208. **Add batch snapshot operations** — Delete all snapshots for aggregates that no longer exist.

---

## 16. Command/Query Audit Improvements (P2)

209. **Add command detail view** — Currently only a list view. Clicking a command should show full payload, metadata, and result.

210. **Add query detail view** — Same for queries — show full query payload and result.

211. **Add command success/failure status** — Show whether the command succeeded or failed.

212. **Add command duration** — Show how long the command took to execute.

213. **Add command-to-event tracing** — Show which events were produced by a specific command.

214. **Add query result display** — Show the query result alongside the query itself.

215. **Add command/query correlation** — Group commands and queries by correlation ID.

216. **Add command payload rendering** — Use the same PayloadRenderer for command/query payloads.

217. **Add command retry indicator** — Show if a command was retried (from middleware).

218. **Add command actor display** — Show who initiated the command (user ID from metadata).

219. **Add command stream link** — Link from command to the aggregate it targeted.

220. **Add query-to-command link** — Show commands that read the same aggregate.

221. **Add command/query timeline view** — Chronological interleaved view of commands and queries.

---

## 17. Overview Page Improvements (P2)

222. **Fix stat accuracy** — See P0 items. Stats are misleading.

223. **Add DLQ count stat** — Show total dead letters across all projections.

224. **Add command/query count stat** — Show total commands and queries processed.

225. **Add throughput sparkline** — Events per minute/hour chart.

226. **Add system health summary** — Overall health badge: green if all projections healthy, yellow if degraded, red if any failed.

227. **Add recent errors section** — Show recent projection errors or DLQ entries.

228. **Add event type distribution** — Breakdown of event types in the system.

229. **Add storage size estimate** — Approximate journal size.

230. **Add last-event timestamp** — When was the most recent event committed.

231. **Add uptime indicator** — How long the system has been running (requires tracking start time).

232. **Make overview configurable** — Let consumers add custom stat cards via a callback or interface.

233. **Add quick actions** — One-click access to common operations (reset worst projection, purge oldest DLQ entries).

234. **Add system info panel** — Go version, module version, config summary.

235. **Add recent activity feed** — Unified feed of recent events, commands, and projection status changes.

---

## 18. New Panels & Features (P3)

236. **Add Event Catalog panel** — Integrate with `cqrshtmx.EventCatalog` to show all registered event types with their schemas. Currently `usermgmt.DefaultEventCatalog()` exists but dashboard doesn't use it.

237. **Add Saga/Process manager panel** — If `deriver` module is used, show saga state and progression.

238. **Add Scheduler panel** — If `scheduling` module is used, show scheduled jobs and timers.

239. **Add Idempotency panel** — If `idempotency` module is used, show idempotency key statistics.

240. **Add OTel/Tracing integration** — Link events/commands to their distributed traces.

241. **Add Metrics panel** — Integrate with `prometheus` module to show throughput, latency, error rates.

242. **Add Replay/Simulation panel** — Replay events from one aggregate against a test decider to verify business logic.

243. **Add Schema migration panel** — Show event schema versions and upcaster registry status.

244. **Add Backup/Export panel** — Export the entire event journal as a downloadable file.

245. **Add Configuration viewer** — Show the dashboard's own configuration (which capabilities are active, settings).

246. **Add Health check page** — Dedicated page showing system health with dependency status.

247. **Add WebSocket panel** — Alternative to SSE, using the root module's WebSocket support.

248. **Add Datastar integration** — The examples have a `datastar-demo`. Consider supporting Datastar as an alternative to HTMX.

---

## 19. API & Export (P2)

249. **Add JSON API endpoints** — Every panel should have a corresponding `?format=json` or `/api/` endpoint for programmatic access.

250. **Add CSV export** — Export event lists, command logs, DLQ entries as CSV.

251. **Add JSON export** — Export any table as a downloadable JSON file.

252. **Add webhook configuration** — Let consumers register webhooks for dashboard events (new DLQ entry, projection failure).

253. **Add OpenAPI spec** — Generate or manually maintain an OpenAPI spec for the dashboard's API endpoints.

254. **Add GraphQL endpoint** — For flexible querying of events, aggregates, projections.

255. **Add streaming export** — Stream large event logs as NDJSON for backup/migration.

256. **Add clipboard integration** — Copy table data as TSV for pasting into spreadsheets.

257. **Add diff export** — Export version diffs as unified diff format.

258. **Add printable report generation** — Generate a PDF audit report from the dashboard data.

---

## 20. Testing & Quality (P1)

259. **Add table-driven rendering tests** — Current tests use `strings.Contains` which is fragile. Use HTML parsing to assert structure.

260. **Add XSS test** — Verify that event types containing `<script>` tags are escaped.

261. **Add pagination tests** — Test cursor-based pagination once implemented.

262. **Add filter tests** — Test all filter combinations.

263. **Add SSE integration test with real HTTP server** — Current SSE tests use `httptest.NewRecorder` which doesn't properly handle streaming. Use `httptest.NewServer`.

264. **Add concurrency test for SSE** — Multiple concurrent SSE clients should all receive events.

265. **Add SSE replay ordering test** — Verify replayed events arrive in order before live events.

266. **Add error path tests** — Every handler should have tests for store errors, invalid inputs, and capability-not-configured.

267. **Add CSS/JS serving tests** — Verify Content-Type and Cache-Control headers on static assets.

268. **Add authorizer tests** — Test that `Authorizer` blocks unauthorized requests.

269. **Add read-only mode tests** — Verify write endpoints return 404/405 in read-only mode.

270. **Add coverage gate** — dashboardui has no coverage gate in `.buildflow.yml`. Add one (target: 72.5% matching current level, ramp to 80%+).

271. **Add benchmark tests** — Benchmark rendering with 1000+ events to ensure performance.

272. **Add fuzz tests** — Fuzz event types, stream IDs, and payload rendering with malformed input.

273. **Add golden file tests** — Snapshot the rendered HTML for each page and compare on changes.

274. **Add race condition tests** — Run tests with `-race` flag (already done in CI, but ensure no data races in SSE bridge).

275. **Add test for handler routing** — Verify all routes are registered correctly for each capability combination.

276. **Add test for base path handling** — Test mounting under various prefixes (`/`, `/dashboard/`, `/a/b/c/`).

277. **Add test for `StreamRefFromID`** — Test with invalid types, invalid IDs, empty strings.

278. **Remove unused test stubs** — `fakeEventSource.LoadFromVersion` and `LoadToTimestamp` are stubbed but not tested.

---

## 21. Documentation (P2)

279. **Fix doc.go description** — Claims templ-components, should describe actual implementation.

280. **Add package-level examples** — `ExampleNew`, `ExampleMount`, `ExampleWithSSE` in test files.

281. **Add integration guide** — Step-by-step guide for wiring the dashboard with SQLite, Postgres, and memory stores.

282. **Add screenshots** — The README has no screenshots. Add them for each panel.

283. **Add configuration reference** — Document every Config field with examples.

284. **Add capability matrix** — Clear table showing which interfaces activate which panels.

285. **Add troubleshooting guide** — Common issues: blank pages (missing interfaces), no SSE (missing EventBus), auth errors.

286. **Add migration guide** — From v4.0.x to v4.1.x and future versions.

287. **Add contributing guide** — How to add a new panel, modify rendering, run tests.

288. **Add API documentation** — Document the exported types: `Config`, `Dashboard`, `PayloadRenderer`, `EventByIDLoader`.

289. **Update README SSE section** — Current SSE section doesn't mention reconnect replay, backfill, or heartbeat.

290. **Add architecture diagram** — Show how event bus → broadcaster → SSE → browser works.

291. **Add security guide** — Document auth, CSRF, read-only mode, and recommended middleware.

292. **Add performance guide** — Expected performance characteristics, tuning PageSize, SSE limits.

293. **Fix README demo section** — Says "requires the module to be tagged and published" which is outdated.

294. **Document `StreamRefFromID`** — Exported function with no documentation about when consumers would use it.

295. **Add CHANGELOG entry for templ migration** — When it happens.

---

## 22. Demo & Examples (P3)

296. **Fix demo README** — Says module needs to be tagged; verify and update.

297. **Add EventBus to demo** — Demo doesn't configure EventBus so SSE doesn't work in the demo. Add it.

298. **Add projections to demo** — Demo has no projection host so the projection panel is empty. Add a simple projection.

299. **Add DLQ entries to demo** — No dead letters seeded in the demo.

300. **Add interactive demo actions** — Buttons to create events, trigger errors, generate DLQ entries on demand.

301. **Add demo authentication** — Show how to wrap the dashboard with session middleware.

302. **Add multi-module demo** — Show dashboard alongside adminui and custom endpoints.

303. **Add Docker demo** — Dockerfile + docker-compose for one-command demo.

304. **Add demo data generator** — Continuously generate events to show live SSE updates.

305. **Add demo with PostgreSQL** — Show persistent storage configuration.

306. **Add demo with real WebAuthn** — Show the full auth stack with the dashboard.

307. **Add comparison example** — Side-by-side adminui and dashboardui to show the difference.

---

## 23. Observability & Metrics (P3)

308. **Add request logging** — Log every dashboard request with method, path, status, and duration.

309. **Add SSE subscriber count metric** — Expose current SSE subscriber count.

310. **Add render duration metric** — Track how long each page takes to render.

311. **Add error counter** — Count and categorize errors by handler and type.

312. **Add `/-/healthz` endpoint** — Return 200 if the dashboard is functional, 503 if dependencies are down.

313. **Add `/-/versionz` endpoint** — Return module version, Go version, build info.

314. **Add `/-/readyz` endpoint** — Return readiness based on store connectivity.

315. **Add Prometheus metrics** — Expose dashboard metrics in Prometheus format.

316. **Add request tracing** — Add OpenTelemetry spans for dashboard requests.

317. **Add slow query logging** — Log when a data loading operation takes longer than a threshold.

318. **Add memory usage tracking** — Track and display dashboard memory usage (goroutines, allocs).

---

## 24. Accessibility (P2)

319. **Add ARIA landmarks** — `<aside>` for sidebar, `<main>` for content, `<nav>` for navigation. Currently using bare `<div>`.

320. **Add proper heading hierarchy** — Pages jump from `<h2>` to `<h4>` skipping `<h3>`. Fix heading levels.

321. **Add `aria-label` to icon-only elements** — Live indicator, brand badge, etc. need labels.

322. **Add `role="table"` semantics** — Tables should have proper `caption`, `scope="col"` on headers.

323. **Add keyboard navigation for tables** — Arrow keys to move between cells.

324. **Add screen reader announcements** — Use `aria-live` for SSE event notifications and toast messages.

325. **Add skip-to-content link** — Hidden link to jump past the sidebar for keyboard users.

326. **Add high contrast mode** — CSS media query for `prefers-contrast: high`.

327. **Add reduced motion support** — Respect `prefers-reduced-motion` for any animations.

328. **Add focus trap for modals** — If confirmation dialogs are added, trap focus within them.

329. **Add descriptive link text** — "View" and "Inspect" links should include context ("View snapshot for Order/abc123").

330. **Add `lang` attribute to code blocks** — `<code lang="json">` for proper screen reader pronunciation.

331. **Add color-blind safe status indicators** — Don't rely on color alone for projection status (add icons/text).

332. **Add form label associations** — Filter inputs need `<label>` elements with `for` attributes.

---

## 25. Mobile & Responsive (P2)

333. **Add responsive sidebar** — Fixed 248px sidebar breaks on mobile. Add hamburger menu toggle (like adminui has).

334. **Add responsive tables** — Tables overflow on narrow screens. Add horizontal scroll wrapper or card-based layout on mobile.

335. **Add touch-friendly tap targets** — Links and buttons are too small for touch (8px padding). Minimum 44x44px.

336. **Add responsive grid for stat cards** — `grid-template-columns:repeat(auto-fit,minmax(200px,1fr))` is good but needs testing on small screens.

337. **Add viewport meta verification** — `<meta name="viewport">` is present but layout doesn't adapt.

338. **Add responsive font sizes** — Use `clamp()` or media queries for readable text on all screen sizes.

339. **Add swipe gestures** — Swipe left/right to navigate between time-travel versions on mobile.

340. **Add mobile-friendly payload view** — `<pre>` blocks overflow on mobile. Add word wrap or horizontal scroll indicator.

341. **Add responsive pagination** — Pagination buttons should wrap or use "Prev/Next" on mobile.

342. **Test with real mobile devices** — Ensure SSE works on iOS Safari and Android Chrome (proxy timeout issues).

---

## Summary

| Category                         | Ideas   | Priority |
| -------------------------------- | ------- | -------- |
| Critical Bugs & Correctness      | 18      | P0       |
| Architecture & Rendering         | 17      | P1       |
| HTMX Integration                 | 13      | P1       |
| Pagination & Data Loading        | 12      | P1       |
| Filtering & Search               | 12      | P1       |
| Sorting & Table UX               | 11      | P2       |
| CSS & Styling                    | 17      | P1       |
| JavaScript & SSE                 | 18      | P1       |
| Security                         | 13      | P1       |
| DLQ Panel Improvements           | 13      | P2       |
| Projection Panel Improvements    | 14      | P2       |
| Event Browser Improvements       | 13      | P2       |
| Aggregate Browser Improvements   | 12      | P2       |
| Time-Travel Improvements         | 13      | P2       |
| Snapshot Inspector Improvements  | 12      | P2       |
| Command/Query Audit Improvements | 13      | P2       |
| Overview Page Improvements       | 14      | P2       |
| New Panels & Features            | 13      | P3       |
| API & Export                     | 10      | P2       |
| Testing & Quality                | 20      | P1       |
| Documentation                    | 17      | P2       |
| Demo & Examples                  | 12      | P3       |
| Observability & Metrics          | 11      | P3       |
| Accessibility                    | 14      | P2       |
| Mobile & Responsive              | 10      | P2       |
| **Total**                        | **342** |          |

---

## Recommended Execution Order (Pareto)

### Phase 1: Critical Fixes (do first, highest impact)

1. Fix XSS vulnerabilities (items 11, 12, 119) — security critical
2. Fix SSE reconnection bug (items 4, 102) — core functionality broken
3. Fix overview stats (items 1, 2, 222) — user trust
4. Remove console.log and dead code (items 101, 5, 7, 8) — professionalism

### Phase 2: Templ Migration (enables everything else)

5. Migrate to templ-components (item 19) — eliminates an entire class of bugs
6. Extract shared components (items 20-25) — maintainability
7. Consolidate CSS (items 26-28) — dark mode, consistency

### Phase 3: Core UX

8. Add real HTMX integration (items 36-48) — the dashboard already loads htmx.js
9. Add pagination (items 49-60) — basic usability
10. Add filtering (items 61-72) — basic usability
11. Implement toast rendering (item 42) — write op feedback

### Phase 4: Polish

12. CSS overhaul and dark mode (items 84-100)
13. Accessibility (items 319-332)
14. Mobile responsive (items 333-342)
15. New panels and metrics (items 236-318)
