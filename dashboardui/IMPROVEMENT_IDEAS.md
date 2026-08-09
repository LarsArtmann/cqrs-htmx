# DashboardUI Improvement Ideas

> **Pruned:** 2026-08-05 — The original 883-line file contained 350+ items, ~80% already resolved. This rewrite keeps only genuinely open work, cross-referenced against the current codebase.

---

## Open Work (by priority)

### Architecture

- **Templ migration evaluation** — The rendering layer is raw `strings.Builder` HTML. A migration to templ would improve type safety and maintainability. Write a decision document (pros/cons/risk) before starting. See Pareto plan T23.
- **Separate data loading from rendering** — Handlers mix data loading (journal/store calls) with HTML rendering. Splitting into `loadX(ctx) (data, error)` + `renderX(data) string` would improve testability.

### UX Polish

- **Sortable columns** — All tables are static order. Add clickable headers with `?sort=col&dir=asc`. See T12.
- **Page-size selector** — `?limit=` works but there's no dropdown UI. See T13.
- **Keyboard navigation for time-travel** — Arrow left/right to scrub versions on the slider. See T17.
- **Command/query status badge** — Show success/failure + duration if available from persisted metadata. See T18.

### Advanced Features

- **CSV export** — Export events/commands/DLQ tables as CSV. See T21.
- **JSON API mode** — `?format=json` returns JSON instead of HTML for programmatic access. See T22.
- **Demo with seeded data** — A runnable demo with events, projections, DLQ entries, commands. See T24.

### Resolved (not listed)

The following categories were fully resolved in prior sessions and are no longer open:

- XSS escaping (all handlers use `esc()`)
- Pagination (cursor-based, bidirectional)
- CSS class system (no inline styles)
- Dark mode (`prefers-color-scheme`)
- Accessibility (semantic HTML5, ARIA, skip-link, focus-visible, reduced-motion)
- HTMX projection-health polling
- Toast notifications
- Copy-to-clipboard for IDs
- Confirmation dialogs for destructive actions
- Relative time + human-readable bytes
- Styled 404 page
- SSE infrastructure (broadcaster, replay, heartbeat, reconnection)
- DLQ index with per-projection counts
- Overview health/DLQ stat cards and event linking
