# DashboardUI Improvement Ideas

> **Pruned:** 2026-08-05 — original 883-line file, 350+ items, ~80% resolved.
> **Pruned again:** 2026-10-07 — every open T-item was re-verified against
> CHANGELOG + code; shipped ones are struck inline. Stale-item check: before
> adding work here, grep `CHANGELOG.md` and the README for the T-number —
> an item listed as open here while the CHANGELOG receipts it is the exact
> split-brain this file exists to prevent.

---

## Open Work (by priority)

### UX Polish

- **Command/query status badge** — Show success/failure + duration if available from persisted metadata. See T18. (Verified still open 2026-10-07: no status/duration column in `audit.templ`.)

### Shipped while listed as open (struck 2026-10-07)

- ~~**Separate data loading from rendering**~~ **DONE** — the `core/` pure-data package owns every fetch; handlers render templ components on top.
- ~~**Sortable columns** (T12)~~ **DONE** — sortable headers via `?sort=`/`?dir=` (README § Sorting).
- ~~**Page-size selector** (T13)~~ **DONE** — dropdown UI over `pageSizeOptions`.
- ~~**Keyboard navigation for time-travel** (T17)~~ **DONE** — slider scrubs with the arrow keys.
- ~~**CSV export** (T21)~~ **DONE** — README § CSV and JSON Export.
- ~~**JSON API mode** (T22)~~ **DONE** — same section.
- ~~**Demo with seeded data** (T24)~~ **DONE** — `examples/dashboard-demo/` (8 users, orders, projections, 5s live-event publisher).

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
