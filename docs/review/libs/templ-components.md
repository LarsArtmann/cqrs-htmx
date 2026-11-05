# templ-components Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/templ-components` (+ `errorpage`, `htmx`, `icons`, `utils` submodules; single v1 tag stream) · Checkout: `/home/lars/projects/templ-components`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Test import sites | Adoption | Score |
| --- | --- | --- | --- | --- | --- |
| v1.27.0 (all submodules) | v1.27.0 (= latest) | 39 files (adminui, dashboardui, loginpage) | 10 files | **Used — the UI component substrate** | **89/100 count · series-graded 97/94/95 per module** |

## What the library is

The family UI kit: 261 non-test `.templ` components across `display`, `forms`, `feedback`, `layout`, `htmx`, `icons`, `charts` (echarts), `datastar`, `navigation`, `errorpage`, `utils` — Tailwind v4 class-driven, CSP-safe, dark-mode-aware, with `BaseProps.ID` DOM-stable hooks and golden-tested rendering.

## How cqrs-htmx uses it today

Distinct adopted symbols per package: **display 70** (Button family ×100+, Badge, Card, Table, DefinitionList, Grid, StatCard, EmptyState, CopyButton, RelativeTime, PageHeader-divergent…), **icons 13**, **layout 9** (Base, ThemeScript, ThemeToggle), **forms 8** (Input, Select, CSRFToken), **feedback 7** (Alert, ToastContainer), **htmx 6** (GlobalErrorHandling, PolledRegion, CSRFToken), **utils 3** (Class, Ternary…), plus the `errorpage` submodule (ErrorPage/NotFound404). `BaseProps` appears at 102 sites. All three UI modules (adminui, dashboardui, loginpage) depend on it; loginpage's 2026-10-04 adoption consumed the former opportunities list entirely.

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | `display` primitives (badges, buttons, cards, tables, lists, states) | 🟢 Fully leveraged | 70 distinct symbols; golden-pinned rendering |
| 2 | `forms` (Input, Select, CSRFToken) | 🟢 Fully leveraged | filter bars, page-size selects, DLQ/snapshot forms |
| 3 | `feedback` (Alert, ToastContainer) | 🟢 Fully leveraged | loginpage + dashboardui layout |
| 4 | `layout` (Base, ThemeScript/ThemeToggle) | 🟢 Fully leveraged | loginpage composes `layout.Base`; class-driven dark mode in adminui+dashboardui |
| 5 | `htmx` helpers (GlobalErrorHandling, PolledRegion) | 🟢 Fully leveraged | PolledRegion with self-pinned `hx-target="this" hx-select="unset"` (the 2026-10-18 boost-inheritance defense) |
| 6 | `icons` | 🟢 Fully leveraged | 13 symbols; nav + page icons |
| 7 | `errorpage` module | 🟢 Fully leveraged | built-in shells for error/404 pages |
| 8 | `utils` | 🟢 Fully leveraged | Class/Ternary composition |
| 9 | `navigation` (Pagination, SidebarNav) / `layout.AppShell` | ⚪ N/A (documented divergence) | cursor pagination + custom shells — [ADR-0053](../../adr/); the grep-able adoption table in [dashboardui/README.md § Styling](../../../dashboardui/README.md) records every row with rationale |
| 10 | `charts` | 🔴 Missed (low) | no charting surface today; candidate: dashboardui event-throughput/timeline panels |
| 11 | `datastar` package | ⚪ N/A | the HTMX UIs; the repo's DataStar adapter module is server-side only |

## Score derivation

11 rows − 3 N/A = **8 applicable**; 7 full ×1 = **7 → 88-89/100**. The independently graded series ([2026-10-04 report](../../research/2026-10-04_templ-components-deep-dive.html), episode 5) scores **adminui 97 / dashboardui 94 / loginpage 95** — this report defers to those finer-grained audits and adds only the `charts` row.

## Version currency

v1.27.0 pinned = v1.27.0 latest (verified `git ls-remote --tags origin`, 2026-11-05; single tag stream covers all submodules). Bundle hygiene is mechanized: `check-css-bundles` + `check-css-bundle-classes` gate every family bump, and the same-commit CSS-rebuild rule is a documented gotcha.

## Should cqrs-htmx use it (more)?

Essentially at max. The only untapped package is `charts` — a dashboardui event-rate panel would be the natural first consumer, when that panel is ever wanted.

## How to verify

```bash
for p in display forms feedback layout htmx icons utils charts datastar; do
  rg -oN "\b$p\.[A-Z][A-Za-z0-9]*" adminui dashboardui loginpage -g '*.templ' | sed "s/.*$p\.//" | sort -u | wc -l
done
rg --files ~/projects/templ-components -g '*.templ' | grep -v _test | wc -l   # 261
git -C ~/projects/templ-components ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- Series entry: [2026-10-04 repo-wide deep dive](../../research/2026-10-04_templ-components-deep-dive.html) (97/94/95) · dashboardui episode: [2026-09-17](../../research/2026-09-17_templ-components-dashboardui-deep-dive.html) (14/100 pre-adoption era) · adoption inventory: [dashboardui/README.md](../../../dashboardui/README.md).
