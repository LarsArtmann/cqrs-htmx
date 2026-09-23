# ADR 0053: dashboardui's justified divergences from templ-components defaults

**Date:** 2026-09-23
**Status:** Accepted
**Related:** adoption audits `docs/research/2026-09-17_templ-components-dashboardui-deep-dive.html` (Run 1, 14/100) and `docs/research/2026-09-23_templ-components-deep-dive.html` (Run 4, §06 rubric + §07 evaluations), `dashboardui/README.md` adoption table

## Context

The 2026-09-23 adoption audit found dashboardui at ~94/100 library adoption
(rubric v1) with three structural divergences from templ-components defaults.
Each had accumulated rationale across sessions; this ADR consolidates the
verdicts so future sessions stop re-litigating them and audits stop re-counting
them as gaps.

## Decision

**Three divergences are deliberate and maintained.** A revisit needs NEW
capability on the library side, not effort on the consumer side.

| Divergence | Library path not taken | Why the divergence wins |
| --- | --- | --- |
| **Hand-rolled page shell** (not `layout.Base`/`AppShell`) | `Base` auto-injects CDN htmx + public SEO surface; `AppShell` cannot express `hx-boost` partial mode, the `#main-content` swap target, the skip-link, or the per-instance `--accent` var | The dashboard is a self-hosted, noindex, session-gated panel whose boost/partial rendering contract is pinned by goldens and CSP tests. `AppShell` lacks the hooks (verified against `AppShellProps`: sidebar/header/content/container only). |
| **Cursor + history pagination** (not `navigation.Pagination`) | `Pagination` renders numbered pages with an ellipsis window | Journal streams are append-only and cursor-addressed; page numbers are meaningless and unstable under concurrent appends. `LoadMore` solves a different UX. No library component addresses cursor semantics. |
| **Custom dark sidebar** (not `navigation.SidebarNav`) | `SidebarNav` + `AppShell` sidebar | Same shell constraints as row 1, plus theming: the library emits Tailwind gray literals and an inline `--tc-sidebar-w` style attribute that nonce CSP drops; matching the dark token theme requires the adminui-class scope pin-backs. TODO_LIST P2 carries the three revisit criteria (dark-token shell, zero-JS drawer, adminui production proof) — none met yet. |

Same-day evaluation batch (audit report §07) additionally rejected
`forms.FilterInput`/`FilterDropdown` (single-field form semantics vs the
3-field AND filter bar), `forms.Slider` (labeled wrapper vs the compact
scrubber row), `display.DataTable` (data-driven shape fights inline templ
cells), `htmx.ConfirmDelete` (per-form pairing vs one delegated listener),
and `display.Scrollback` (log-stream semantics vs a payload blob viewer).

## Consequences

- Adoption audits score these as **justified divergences** (excluded from the
  rubric denominator), not gaps.
- A library-side change that invalidates a row (e.g. `AppShell` gaining
  boost/partial hooks, `Pagination` gaining cursor mode, `SidebarNav`
  dark-token theming) reopens that row; the audit series tracks the triggers.
- The shell's boost/partial contract remains pinned by goldens; any shell
  rewrite must regenerate them in the same change.
