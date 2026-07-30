# dashboardui Roadmap

> Long-term direction and raw ideas not yet refined into actionable tasks.

## Templ + Tailwind v4 Migration

**Status:** Future — not started

The current rendering layer uses raw Go `strings.Builder` HTML with embedded CSS.
While functional, it is painful to maintain: format-string bugs (counting `%s`
placeholders), no compile-time HTML validation, no component reuse, and inline
styling that resists consistency.

### Target

Migrate all rendering to **templ** (`a-h/templ`) + **Tailwind v4** via the
[`templ-components`](https://github.com/larsartmann/templ-components) library:

- Type-safe HTML templates with compile-time checking
- Tailwind utility classes instead of hand-rolled CSS
- Component composition (cards, tables, empty states, page headers)
- Consistent dark mode via Tailwind's `dark:` variant
- Generated `_templ.go` files committed (no consumer codegen needed)

### Why Not Now

- **VERSCHLIMMBESSER risk:** The dashboard works. 64 tests pass. A mid-sprint
  migration touching all 28+ files risks breaking everything for zero
  customer-facing benefit.
- **Wrong scope:** The improvement sprint focused on features (mobile, a11y,
  observability), not rendering technology.
- **Multi-day effort:** ~5,500 lines of strings.Builder HTML need conversion.
  Each handler's `render*` method becomes a `.templ` file. CSS custom properties
  become Tailwind theme tokens. The embedded `dashboardCSS`/`dashboardJS` consts
  are replaced by Tailwind's build pipeline.

### Migration Path (When Started)

1. Add `templ` and `templ-components` as dependencies
2. Start with leaf components: `emptyState`, `statCard`, `metaRow`, `badge`
3. Move up to composite components: `pageHeader`, `filterBar`, `dataTable`
4. Convert each panel handler one at a time (overview, events, aggregates, etc.)
5. Replace `dashboardCSS` with Tailwind v4 theme configuration
6. Remove all inline styles in favor of utility classes
7. Update tests to match new HTML structure

### Decision Record

The user asked "Why are we not using templ?!?!" and "Tailwind v4?" during the
Session 3 improvement sprint (2026-07-30). The decision was to finish the feature
sprint on the current CSS approach and plan the migration as a dedicated future
project. See `docs/planning/2026-07-30_21-15_dashboardui-sprint-session3.md`.

## Not Planned

- **Full-text search across event payloads** — would require indexing infrastructure
  (SQLite FTS, Elasticsearch) that is out of scope for a zero-dependency dashboard.
- **Multi-tenant dashboard isolation** — consumers should wrap the dashboard with
  their own tenant-scoped middleware.
