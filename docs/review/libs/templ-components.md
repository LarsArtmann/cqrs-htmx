# templ-components Utilization Report

> Audit date: 2026-11-14 (refreshes the 2026-11-05 audit) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/templ-components` (+ `errorpage`, `htmx`, `icons`, `utils`, `datastar` submodules; single v1 tag stream) · Checkout: `/home/lars/projects/templ-components`

## Verdict at a glance

| Pin                          | Latest (verified 2026-11-14) | Prod import sites                                | Test import sites | Adoption                              | Score                                                |
| ---------------------------- | ---------------------------- | ------------------------------------------------ | ----------------- | ------------------------------------- | ---------------------------------------------------- |
| v1.27.0 (all submodules)     | v1.27.0 (= latest tag; 91 unreleased commits ahead, none load-bearing for us) | 44 files — 21 `.templ` + 23 hand-written `.go` (adminui, dashboardui, loginpage) | 14 files          | **Used — the UI component substrate** | **95/100 count · series-graded 97/94/95 per module** |

## What the library is

The family UI kit: 261 non-test `.templ` components across `display`, `forms`, `feedback`, `layout`, `htmx`, `icons`, `charts` (echarts), `datastar`, `navigation`, `errorpage`, `utils` — Tailwind v4 class-driven, CSP-safe, dark-mode-aware, with `BaseProps.ID` DOM-stable hooks and golden-tested rendering.

## How cqrs-htmx uses it today

Distinct adopted symbols per package (verify block below): **display 73** (Button family ×100+, Badge, Card, Table, DefinitionList, Grid, StatCard, EmptyState, CopyButton, RelativeTime, Avatar, Dropdown, Modal, StatusBadge, ListNote, **LineChart** — new 2026-11-06…), **icons 15**, **layout 9** (Base, ThemeScript, ThemeToggle), **forms 10** (Input/InputEmail/InputText, Select, **Checkbox**, FilterInput), **feedback 9** (Alert, Spinner, ToastContainer, **FeedbackSuccess/Warning**), **htmx 6** (GlobalErrorHandling, PolledRegion, CSRFToken, LoadingButton), **navigation 8** (**SidebarNav**, Breadcrumbs, Pagination — adminui), **utils 3** (Class, Ternary…), plus the `errorpage` submodule (ErrorPage/NotFound404). `utils.BaseProps` at 76 sites in `.templ` sources. All three UI modules depend on it.

## Adoption delta since the 2026-11-05 audit (symbol-set diff at `68525b76` → HEAD)

The 2026-11-06 **v1.27.0 templ-era sweep** (`96b3f129` green scope + `7e39deba` closeout — replace blocks dropped, all `_templ.go` regenerated with the canonical pinned generator **templ v0.3.1100** via `nix run .#templGen`, flake.nix:139-141; CSS bundles rebuilt same-change `49fd66a9`; go.work carries no templ-components replaces anymore) was followed by the **adminui M05-M17 wave** (2026-11-07..14, exit report [2026-11-14_16-21](../../status/2026-11-14_16-21_adminui-m05-m17-completion-status.md)) and dashboardui's throughput panel:

- **display 70→73**: +`DropdownPositionLeft` — the M09 header **tenant switcher** reuses the library Dropdown (`adminui/layout.templ:121-135`, feat `44e6187f` + tests `03e6adff`; same component as the user menu at `layout.templ:163-197`, `DropdownPositionRight`); +`LineChart`/`LineChartProps`/`LineChartSeries` — `dashboardui/throughput.templ:31-33` (2026-11-06, fleet-plan A21: the repo's **first charts consumer**, pure SVG, zero client JS so strict CSP holds; README adoption-table row added same-change); −`BadgePrimary` (refactored away).
- **forms 8→10**: +`Checkbox`/`CheckboxProps` — M06 bulk-import options (`adminui/users.templ:558`, `components.templ:56`). NOTE: the import form's `type="file"` (`users.templ:43`) is hand-rolled; upstream `forms.InputFile` (`forms/file_input.templ`) exists but was not taken — the one obvious remaining forms gap.
- **feedback 7→9**: +`FeedbackSuccess`/`FeedbackWarning` — the M06 import-result panel (`adminui/users.templ:73,79`), composed with `htmx.LoadingButton` + `feedback.Spinner` in the submit button (`users.templ:58`).
- **icons 13→15**: +`Chart` (`dashboardui/throughput.templ:21`), +`ExclamationTriangle` (`dashboardui/commandfates.templ:17`).
- **navigation (8, not new — a correction to this report, not a delta)**: the 2026-11-05 audit marked the package ⚪ N/A, but adminui already imported `templ-components/navigation` then: `SidebarNav` (`adminui/layout.templ:50,174`), `Pagination` (list footers), `Breadcrumbs` (detail pages). [ADR-0053](../../adr/0053-dashboardui-templ-components-divergences.md) is **dashboardui-scoped** (cursor pagination, custom shell/sidebar); its row-3 revisit criterion "adminui production proof" for `SidebarNav` is now met by adminui's shell.
- Member attribution M-wave bits (`e45de744`) and tenant chips (`2bdd0bea`, CSS bundle rebuilt same-change) reused existing symbols (Avatar, Badge variants) — no new symbols.
- **loginpage** adoption unchanged; its news is version-train, below.

**e2e battery** (context for the dropdown/keyboard paths): final M05 verdict — Chromium (blocking engine) **green: 117 passed, 0 failures, 112 listed specs** (completion report §b M05); the mid-wave snapshot `e2e/pw-chromium-report.json` (2026-11-14T13:23Z) had shown 110 expected / 2 unexpected before the triaged fixes. The library Dropdown's keyboard-nav/light-dismiss scripts are exactly what gotcha 57 protects: element-bound JS dies with the pre-swap DOM under `hx-boost` — document-delegated listeners, synthetic events `bubbles: true` (AGENTS.md:155). `e2e/server` consumes the family (`templ-components`, `datastar`, `errorpage`, `htmx`, `icons`, `utils`) all at v1.27.0.

## Capability inventory

| #   | Capability                                                           | Status                         | Evidence / reason                                                                                                                                                                              |
| --- | -------------------------------------------------------------------- | ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `display` primitives (badges, buttons, cards, tables, lists, states, dropdowns, modals, avatars) | 🟢 Fully leveraged | 73 distinct symbols; golden-pinned rendering; M-wave reused Dropdown/Avatar/Badge rather than forking                                                                                          |
| 2   | `forms` (Input, Select, Checkbox, FilterInput)                       | 🟢 Fully leveraged             | filter bars (adminui adopted `FilterInput` — the ADR-0053 rejection was dashboardui's 3-field bar only), page-size selects, import options; `InputFile` deliberately not taken (`users.templ:43` hand-rolled) |
| 3   | `feedback` (Alert, Spinner, ToastContainer)                          | 🟢 Fully leveraged             | loginpage + dashboardui layout; import-result panel variants                                                                                                                                   |
| 4   | `layout` (Base, ThemeScript/ThemeToggle)                             | 🟢 Fully leveraged             | loginpage composes `layout.Base`; class-driven dark mode in adminui+dashboardui                                                                                                                |
| 5   | `htmx` helpers (GlobalErrorHandling, PolledRegion, LoadingButton)    | 🟢 Fully leveraged             | PolledRegion with self-pinned `hx-target="this" hx-select="unset"` (the 2026-10-18 boost-inheritance defense)                                                                                  |
| 6   | `icons`                                                              | 🟢 Fully leveraged             | 15 symbols; nav + page icons                                                                                                                                                                   |
| 7   | `errorpage` module                                                   | 🟢 Fully leveraged             | built-in shells for error/404 pages (`adminui/errorpage.go`, `dashboardui/handler.go`)                                                                                                         |
| 8   | `utils`                                                              | 🟢 Fully leveraged             | Class/Ternary/BaseProps composition                                                                                                                                                            |
| 9   | `navigation` (Pagination, Breadcrumbs, SidebarNav)                   | 🟢 Fully leveraged (adminui)   | 8 symbols — sidebar, list-footers, breadcrumbs; **corrected from N/A this audit** (see delta section). `layout.AppShell` + cursor pagination remain documented dashboardui divergences — [ADR-0053](../../adr/) and the adoption table in [dashboardui/README.md § Styling](../../../dashboardui/README.md) |
| 10  | charting (`display` SVG charts; `charts`/echarts submodule)          | 🟡 Partially used              | `display.LineChart` adopted 2026-11-06 (`throughput.templ`, first charts consumer, zero-JS CSP); Bar/Sparkline/Area unconsumed with no current demand; the echarts submodule stays untaken — it needs script exceptions the zero-JS CSP policy forbids |
| 11  | `datastar` package                                                   | ⚪ N/A                         | the HTMX UIs; the repo's DataStar adapter module is server-side only                                                                                                                           |

## Score derivation

11 rows − 1 N/A = **10 applicable**; 9 full ×1 + 1 partial ×0.5 = **9.5/10 → 95/100** (up from 89: navigation re-verdicted 🟢 after correcting the 2026-11-05 blind spot, and charting moved 🔴→🟡 with the throughput panel). The independently graded series ([2026-10-04 report](../../research/2026-10-04_templ-components-deep-dive.html), episode 5) scores **adminui 97 / dashboardui 94 / loginpage 95** — kept as the series of record; the M-wave only raises adminui's floor (its rubric was not re-run; the count score above carries the delta).

## Version currency

**v1.27.0 pinned = v1.27.0 latest** (verified `git ls-remote --tags origin`, 2026-11-14; re-verified 2026-11-15 plan B10.1 — single tag stream covers all submodules, so `templ-components/datastar` rides the same v1.27.0 = latest; e2e/server's `datastar v1.27.0 // indirect` pin is current; adminui/dashboardui/loginpage + e2e/server go.mod all v1.27.0 — loginpage root+utils direct, htmx/icons indirect). The 2026-11-06 sweep (commits `96b3f129`/`7e39deba`, CSS rebuild `49fd66a9`) resolved the templ-era split: one generator era (v0.3.1100) across every committed `_templ.go`. loginpage rode it as **v4.17.0** (2026-11-07, train alignment — `loginpage/CHANGELOG.md:17-25`), and its **v4.16.0** section was cut retroactively the same day (`CHANGELOG.md:27-51`: `Routes()`, `Config.NoWebAuthn` — the W14/S36 follow-through — `AuthPrefix` validation). Upstream carries **91 unreleased commits** past v1.27.0; the ones relevant to us: StatCard focus fix (`7eeafb8e`), CopyButton script-mount fix for fragments (`7fccc348`), BarChart fix (`0bf45923`), Sparkline responsive (`f6891f9f`), Stacked form fields (`2b7a05f7`), id/ARIA/keyboard fixes (`2e16bcaf`). Bundle hygiene unchanged: `check-css-bundles` + `check-css-bundle-classes` gate every family bump; same-commit CSS rebuild is a documented gotcha.

## Popover anchoring — upstream ask still open (2026-11-14)

Root-caused 2026-11-05: `tcPopoverPosition` (upstream `display/shared.go:329-336`) defers to the CSS anchor layer whenever `CSS.supports("anchor-name: --x")` (Chromium ≥125), but the library's compound `--tc-popover-area` values (`bottom span-inline-start` etc., upstream `templates/custom.css:152-164`) are **rejected by Chromium 156** — declaration drops, popover parks at the viewport corner — and flush-edge alignment needs `anchor()`, which cannot see the implicit popovertarget invoker. adminui vendors a **corrected layer** in `adminui/tailwind.css:300-370`: single-keyword position-area bands (:325-355), plus explicit `anchor-name` on `#admin-user-menu-button` (:360-362) and `position-area:none` + `position-anchor` + `anchor()` edges on `#admin-user-menu-menu` (:364-369); anchor-names live in the stylesheet, not inline styles, so strict CSP holds. **Upstream status: unfixed** — none of the 91 commits since v1.27.0 touch popover positioning (the three custom.css/shared.go commits are CopyButton/tokens/ARIA); the upstream ask (fix the compound values or emit explicit anchor-names) remains open.

## htmx 4.0.1 embed — no interplay change for this library

The root repo now embeds **htmx 4.0.1 alongside 2.0.10** (commit `c6bc1f13`, 2026-11-07): `htmx/embed4.go:9` (`HTMX4Version = "4.0.1"`), `htmx/htmx4.min.js`, `htmx/serve4.go` (same caching/ETag contract as the 2.x default), sha-pinned in `htmx/asset_pins_test.go:39`. The **NV4 static verdict** (commit `107a1aed`, recorded in the QUERY row + ROADMAP q35(a) as a 4.x-default-flip precondition): 2.x extensions cannot register under htmx 4 — `htmx.defineExtension`/`htmx.createEventSource` are absent from the 4.0.1 bundle (extensions moved to ES modules). templ-components' `htmx` submodule targets the 2.x surface and the UI modules still serve 2.x by default, so nothing in this report's adoption claims changes; 4.x stays opt-in per-page.

## Should cqrs-htmx use it (more)?

One honest gap and one policy line: `forms.InputFile` would replace the hand-rolled upload input (`adminui/users.templ:43-49`) if it ever gains drag-drop/progress the hand-rolled version lacks; the echarts `charts` submodule stays untaken by the zero-JS CSP policy, and `display`'s Bar/Sparkline/Area charts are the natural next consumers if more panels are ever wanted. Otherwise at max.

## How to verify

```bash
for p in display forms feedback layout htmx icons utils charts datastar navigation; do
  rg -oN "\b$p\.[A-Z][A-Za-z0-9]*" adminui dashboardui loginpage -g '*.templ' | sed "s/.*$p\.//" | sort -u | wc -l
done   # 73 10 9 9 6 15 3 0 0 8
rg --files ~/projects/templ-components -g '*.templ' | grep -v _test | wc -l   # 261
git -C ~/projects/templ-components ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1   # v1.27.0
rg -o 'utils\.BaseProps' adminui dashboardui loginpage -g '*.templ' | wc -l   # 76
rg -l 'larsartmann/templ-components' adminui dashboardui loginpage -g '*.templ' | wc -l                    # 21 .templ sources
rg -l 'larsartmann/templ-components' adminui dashboardui loginpage --type go -g '!*_test.go' -g '!*_templ.go' | wc -l   # 23 hand-written .go
rg -l 'larsartmann/templ-components' adminui dashboardui loginpage --type go -g '*_test.go' | wc -l        # 14 test files
rg -N --no-filename 'templ: version' adminui dashboardui loginpage -g '*_templ.go' | sort -u               # // templ: version: v0.3.1100
git -C ~/projects/templ-components log v1.27.0..HEAD --oneline -- templates/custom.css display/shared.go   # popover layer untouched
```

## Prior reports

- Series entry: [2026-10-04 repo-wide deep dive](../../research/2026-10-04_templ-components-deep-dive.html) (97/94/95) · dashboardui episode: [2026-09-17](../../research/2026-09-17_templ-components-dashboardui-deep-dive.html) (14/100 pre-adoption era) · adoption inventory: [dashboardui/README.md](../../../dashboardui/README.md) · M-wave evidence: [2026-11-14 M05-M17 completion status](../../status/2026-11-14_16-21_adminui-m05-m17-completion-status.md).
