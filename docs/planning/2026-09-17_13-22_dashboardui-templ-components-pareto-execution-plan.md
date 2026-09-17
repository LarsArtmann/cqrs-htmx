# Pareto Execution Plan — dashboardui × templ-components Adoption Program

**Generated:** 2026-09-17 13:22 CEST
**Scope:** Everything identified in the 2026-09-17 dashboardui × templ-components deep-dive audit (`docs/research/2026-09-17_templ-components-dashboardui-deep-dive.html`) and the session status/self-review (`docs/status/2026-09-17_13-12_dashboardui-templ-components-audit-status.md`). No new research — this plan operationalizes known findings only.
**Total program:** 28 medium tasks (30–100 min) ≈ **1,905 min ≈ 31.75 h**, broken into **215 micro tasks** (≤12 min each).

---

## 0. Situation (read this first)

The audit verdict: dashboardui uses **1 of 22** applicable templ-components capabilities (score 14/100); ~800 lines of hand-rolled HTML/CSS/JS duplicate ~15 library components; version currency is perfect (v1.17.0). Two facts reshape everything:

1. **The hybrid path is verified:** `templ.Component.Render(ctx, io.Writer)` accepts the existing `strings.Builder` — adoption is incremental, no `.templ` conversion required.
2. **The real gate is Tailwind v4:** components emit Tailwind utilities; dashboardui has no pipeline. adminui already runs the exact BuildFlow `tailwind-build` pattern to copy.

Therefore the program is **gate-first**: nothing lands until the gate is open, then every rung is an independent, low-risk, shippable swap.

## 0.1 VERSCHLIMMBESSER-Schutz (how this plan avoids making things worse)

- **No big bang.** Every task M# is independently shippable; the dashboard must stay fully functional after every single micro task.
- **Tests stay green per task.** Existing string-contains tests make component swaps low-risk; a task is not done until `go build && go vet && go test` pass for dashboardui.
- **CSS is deleted only after parity.** Hand-rolled CSS blocks (`.badge-*`, `.stat-card`, `.data-table`, …) are removed only in the task that confirms the last consumer is migrated (adminui precedent: verify class parity, keep the generated artifact canonical).
- **No templ codegen from repo root** (module-dir `FileName:` is canonical — AGENTS.md). This plan introduces zero `.templ` files anyway.
- **CSP is sacred.** Components with inline scripts (Toast, CopyButton, GlobalErrorHandling) are adopted only after the nonce-plumbing task lands; the security test task proves it end-to-end.
- **The dead findings gate stays respected:** commits use the documented `--no-verify` + justification fallback until the gate triage task (M3) lands; never bypass without checking the log.
- **Concurrent sessions share this tree:** commit at task boundaries; re-check `git status --short` before every `git add`.

---

## 1. Pareto Breakdown

> **Honesty note:** literal 1% of a 32.8 h program is ~20 min — smaller than any viable task. The tiers below are therefore *minimum decisive increments*, with actual effort shares stated plainly. The classic Pareto compression happens because this program has a hard sequential gate (Tailwind): the first increment carries disproportionate gate cost but converts 100% of the remaining program from blocked theory into mechanical execution.

| Tier | Spirit | Tasks (cumulative) | Cum. effort | Cum. result | Why this delivers |
| --- | --- | --- | --- | --- | --- |
| **1%** | The smallest increment that unlocks the majority | M4 + M5 (Tailwind gate + StatusBadge spike) | 180 min (9%) | **51%** | Opens the only gate (nothing was adoptable before it), proves the hybrid `Render(ctx, &b)` path with zero-regression evidence, lands the first duplicated-code deletion. All 10 remaining component rungs become mechanical. Risk drops from "unknown" to "scheduled". |
| **4%** | + user-facing core | M1, M2, M3, M6, M7, M8 (debt cleanup, errorpage, badges, stat cards) | 555 min (29%) | **64%** | Highest-traffic surfaces move to the library: overview stat cards, every status badge, and — every user's worst moment — errors and 404s become styled, navigable, family-aware pages. Session debt (broken links, split-brain reports, dead gate) cleared. |
| **20%** | + the sweep | M9–M16 (EmptyState/PageHeader, Buttons, Toast+nonce, GlobalErrorHandling, CopyButton, 3 table conversions) | 1,155 min (61%) | **80%** | The full leaf vocabulary adopted; silent HTMX failures get retry+toast; copy becomes accessible; the three highest-traffic tables get sortable typed headers. |
| **100%** | The long tail | M17–M28 (pagination trio, ThemeToggle, SidebarNav, DefinitionList, CSS cleanup, security/golden/bench/a11y/e2e hardening, complexity refactors, e2e fixes, re-scoring) | 1,905 min (100%) | **100%** | Hardening, measurement, repo hygiene, and the remaining structure components. |

**The other 80% of work, not forgotten:** tier 4 IS that work — explicitly enumerated in the tables below (hardening, refactors, docs, measurement), not hand-waved.

---

## 2. Comprehensive Plan — Medium Tasks (30–100 min each, ALL todos)

Sorted by importance/impact/effort/customer-value. Impact: 5 = every dashboard user benefits, 1 = internal hygiene. "Deps" = hard prerequisites.

| ID | Task | Min | Impact | Tier | Deps | Customer value |
| --- | --- | --- | --- | --- | --- | --- |
| M1 | Reconcile the two same-day deep-dive reports; fix broken report links in dashboardui docs | 45 | 4 | 0 | — | Docs tell one truth; links resolve |
| M2 | HARVEST: TODO_LIST ladder entries, ROADMAP "Not Planned" (datastar, echarts), AGENTS.md gotchas, CHANGELOG | 60 | 4 | 0 | — | Findings become tracked, not entombed |
| M3 | Triage/downgrade the dead findings gate (103 pre-existing errors) so hooks are meaningful | 45 | 4 | 0 | — | Every future commit verifiable again |
| M4 | Tailwind v4 entry point + build + serve for dashboardui (app.css, @source, @theme tokens, buildflow step) | 90 | 5 | 1 | M3 | The gate; visual parity foundation |
| M5 | StatusBadge spike: first library component end-to-end via hybrid `Render(ctx, &b)` | 90 | 5 | 1 | M4 | Proves the path; first code deleted |
| M6 | errorpage adoption: styled family-aware `renderError` + 404 pages (ErrorHandler, WriteNotFound404) | 90 | 5 | 2 | M4 | Every error/404 becomes navigable + styled |
| M7 | Badge/StatusBadge full swap; delete duplicated switches + `.badge-*` CSS | 45 | 4 | 2 | M5 | Consistent status visuals; less code |
| M8 | StatCard + ValueID + Grid on overview & projection detail; delete `.stat-card` CSS | 90 | 5 | 2 | M4 | Live-dashboard-ready stat hooks |
| M9 | EmptyState (icon/action/role=status) + PageHeader swap | 70 | 3 | 3 | M4 | Better empty/loading moments, a11y |
| M10 | Button swap (btn / btn-danger / btn-accent → display.Button); delete `.btn*` CSS | 60 | 3 | 3 | M4 | Consistent buttons, 9 variants |
| M11 | Nonce plumbing + ToastContainer (port adminui toastHost) + delete old toast JS/CSS | 75 | 4 | 3 | M6.5 | Toasts that match adminui; CSP-safe |
| M12 | htmx.GlobalErrorHandling mounted (retry, 401, family-aware toasts) | 45 | 4 | 3 | M11 | Silent HTMX failures become visible+recoverable |
| M13 | CopyButton for payloads/IDs; delete `data-copyable` JS + emoji CSS | 60 | 3 | 3 | M11 | Keyboard-accessible copy (a11y fix) |
| M14 | Table conversion: events listing + sortable typed headers (aria-sort) | 100 | 4 | 3 | M7 | Sortable events = real UX upgrade |
| M15 | Table conversion: audit commands + queries listings | 90 | 3 | 3 | M14 | Consistency + sorting |
| M16 | Table conversion: projections, DLQ, snapshots/aggregates (renderStreamIndex), time-travel | 100 | 3 | 3 | M14 | Full table parity |
| M17 | Pagination trio: navigation.Pagination + display.ListNote + forms.Select page-size | 75 | 3 | 4 | M14 | Numbered pager > Prev/Next |
| M18 | ThemeScript/ThemeToggle: manual dark-mode override, no FOUC | 45 | 2 | 4 | M4 | Dark mode on demand |
| M19 | SidebarNav structural swap (keep custom shell per adminui precedent) | 60 | 2 | 4 | M4 | Active-state logic from library |
| M20 | DefinitionList for metaRow/metaRowCopyable | 45 | 2 | 4 | M13 | Semantic key/value markup |
| M21 | Delete `.data-table` CSS; dashboardCSS dead-rule audit; token-layer-only shrink | 45 | 3 | 4 | M16 | CSS shrinks toward tokens |
| M22 | Security tests: nonce → CSP end-to-end for adopted inline-script components | 60 | 4 | 4 | M11, M12 | Proof the CSP posture held |
| M23 | Golden tests for all adopted component outputs | 60 | 3 | 4 | M16 | Regression net for swaps |
| M24 | Benchmark: hand-rolled vs hybrid Render (b.Loop + benchstat) | 90 | 2 | 4 | M16 | Numbers, not vibes |
| M25 | a11y pass (keyboard, aria) + e2e extension for new components | 90 | 3 | 4 | M16 | Accessible dashboard verified |
| M26 | Complexity refactors: renderOverview (28), LoadEventByID (27), renderEventDetail (26), FetchOverview (22) | 90 | 2 | 4 | — | Lint warnings → 0 |
| M27 | Repo noticed-fixes: e2e/server G114 timeouts, exhaustruct, param names; integration_test templ-components train note | 45 | 2 | 4 | — | Hook output noise → 0 |
| M28 | Re-score adoption post-rungs; update AGENTS.md adoption table; ANNOTATE old reports/plans | 45 | 3 | 4 | M5–M16 | Measured delta: 14/100 → N |

**Totals:** 28 tasks, 1,905 min ≈ 31.75 h. Tier 0 = 150 min · Tier 1 = 180 min · Tier 2 = 225 min · Tier 3 = 600 min · Tier 4 = 750 min.

---

## 3. Detailed Breakdown — Micro Tasks (≤12 min each, ALL todos)

### Tier 0 — Session debt (M1–M3)

| ID | Micro task | Min | Dep |
| --- | --- | --- | --- |
| M1.1 | Read sibling report (`2026-09-17_templ-components-deep-dive.html`): extract score + recommendations | 12 | — |
| M1.2 | Adjudicate conflicts; pick canonical report; annotate the other with a pointer banner | 12 | M1.1 |
| M1.3 | Fix report path in `dashboardui/ROADMAP.md` (currently resolves to nonexistent dir) | 5 | M1.2 |
| M1.4 | Fix report path in `templ-migration-evaluation.md` | 5 | M1.2 |
| M1.5 | Verify both links resolve from their file locations | 6 | M1.3 |
| M1.6 | Commit tier-0 docs fixes | 5 | M1.5 |
| M2.1 | HARVEST ladder rungs → `TODO_LIST.md` (open items, `[ ]` convention) | 12 | — |
| M2.2 | ROADMAP "Not Planned": datastar module, charts/echarts (with reasons) | 10 | — |
| M2.3 | AGENTS.md gotcha: findings gate fails docs-only commits; `--no-verify` + justification recipe | 12 | — |
| M2.4 | AGENTS.md dashboardui adoption section: add "planned ladder" pointer | 10 | M2.3 |
| M2.5 | CHANGELOG entries for the planning-doc corrections | 8 | — |
| M2.6 | Commit | 8 | M2.1 |
| M3.1 | Inventory the 103 findings by tool (go-structure-linter 51 / gomod-check 26 / go-mod-ignore-check 26) | 12 | — |
| M3.2 | Classify: false-positive config vs real debt; choose downgrade (baseline or `--fail-on=critical`) | 12 | M3.1 |
| M3.3a | Implement gate downgrade change (.buildflow.yml) | 9 | M3.2 |
| M3.3b | Verify docs-only commit passes hook without --no-verify | 6 | M3.2 |
| M3.4 | Document decision in AGENTS.md + commit | 6 | M3.3 |

### Tier 1 — The 1% gate + proof (M4–M5)

> **SPIKE VERDICT (M5.8, executed 2026-09-17): the hybrid `Render(ctx, &b)` path is PROVEN.**
> `display.StatusBadge` renders cleanly into the existing `strings.Builder` with zero templ
> conversion; the old badge switch in `renderProjectionRow` is deleted. Findings for the
> remaining rungs: (1) `StatusBadge` colors by the MAPPED WORD, so unknown status kinds must
> fall back to `display.Badge(BadgeProps{Type: BadgeNeutral, Text: raw})` to preserve text +
> neutral styling (implemented in `statusBadge`, single mapping point `statusKindToStatus`).
> (2) Badges must be wrapped in `<td>` by the caller (components render bare `<span>`).
> (3) `go mod tidy` resolves NEW imports at `@latest` — always `go get <family modules>@v1.17.0`
> BEFORE tidy or the family drifts (hit: tidy bumped the family to v1.18.0, re-pinned).
> (4) contextcheck demands real ctx — plumbed `r.Context()` through the 3 render funcs
> (4 signature changes, cheap). (5) The M4 false-green root cause was NOT the fallback
> location but the module-cache layout: the icons module extracts at
> `templ-components/icons@v1.17.0`; its parent is NOT the root module
> (`templ-components@v1.17.0` is a sibling entry). The flake app now derives the version from
> the icons dir, downloads the root module if missing, and canary-asserts the output
> (plus fails loudly on zero scanned .templ files).

| ID | Micro task | Min | Dep |
| --- | --- | --- | --- |
| M4.1 | Create `dashboardui/app.css`: `@import "tailwindcss";` | 8 | M3 |
| M4.2 | `@source` templ-components module dir (module-cache or vendor path, per adoption guide) | 12 | M4.1 |
| M4.3 | `@theme` mapping: `--accent/--ok/--warn/--err` → `--color-tc-primary/success/warning/danger` + dark strategy | 12 | M4.1 |
| M4.4 | Copy needed bits of `templates/custom.css` (only what adopted components require) | 10 | M4.1 |
| M4.5 | First build: `tailwindcss -i app.css -o styles.css --minify` (tailwindcss_4 in devShell) | 12 | M4.2 |
| M4.6 | Add BuildFlow `tailwind-build` step for dashboardui (adminui pattern) | 12 | M4.5 |
| M4.7 | Serve compiled stylesheet (extend `serveCSS` / asset mount, ETag + cache headers) | 12 | M4.5 |
| M4.8 | Visual parity check on overview page; commit | 12 | M4.7 |
| M5.1 | Add `templ-components` root dep to `dashboardui/go.mod`; hermetic tidy | 10 | M4 |
| M5.2a | Add StatusBadge props helper in renderProjectionRow | 8 | M5.1 |
| M5.2b | Render via .Render(ctx, &b) into builder | 7 | M5.1 |
| M5.3 | `statusKind → status string` helper (single mapping point) | 10 | M5.2 |
| M5.4 | Update string-contains tests | 10 | M5.2 |
| M5.5 | Verify emitted Tailwind classes exist in compiled styles.css | 10 | M4, M5.2 |
| M5.6 | Delete the badge switch in `renderProjectionRow` | 10 | M5.3 |
| M5.7a | go build + vet dashboardui | 8 | M5.4 |
| M5.7b | go test + lint + commit | 7 | M5.4 |
| M5.8 | Spike verdict → planning doc (hybrid path proven / surprises) | 10 | M5.7 |

### Tier 2 — The 4% user-facing core (M6–M8)

> **M8 VERDICT (executed 2026-09-17, closed after daemon-interrupt recovery):**
> (1) **StatCard adopted, Grid NOT.** `display.Grid` uses templ `{children...}`, which is
> EMPTY in the standalone hybrid `.Render` path — the container renders as a bare `<div>`
> with no children. Kept the `.stat-grid` div; only the leaf cards moved to
> `statCardHTML(ctx, valueID, value, label, tone)`. Same pre-check applies to `display.Table`
> in M14 (inspect its `Body` slot mechanics BEFORE converting).
> (2) **ValueID scheme landed:** `stat-total-events`, `stat-total-aggregates`,
> `stat-projections-active`, `stat-system-health`, `stat-dlq-count`, `stat-<metric>-<slug>`
> on projection detail — stable DOM hooks for live-updating scripts.
> **M8.7 SSE JS decision: no JS change needed.** The embedded dashboard JS targets only
> `data-*` hooks, `#projection-health`, and `#main-content .data-table tbody` — it never
> addressed `.stat-card-value`, so the markup swap cannot break it. Future scripts should
> address values through the ValueIDs.
> (3) **Both "failing M8 tests" were pre-existing test bugs EXPOSED, not M8 breakage:**
> `TestOverview_HealthStatCard` asserted `stat-card ok` — which matched the old
> PROJECTIONS card's variant class (emitted by the old `statCard(.., "ok")`), never the
> health card. Vacuously green since inception; the M8 swap removed the wrong card it was
> matching. Root cause of the real value ("Unhealthy"): journal-only projection hosts drain
> to `stopped` (no subscriber → `process()` returns nil → worker run loop exits
> `stopped`), which dashboardui classified as StatusBad — contradicting the root library's
> readiness semantics (`ProjectionReadinessCheck`: live/stopped = ready). FIXED at root
> cause: `ProjectionStatusKind("stopped")` → StatusGood (only `failed` is unhealthy), with
> both classification tables + `TestOverviewStats_ProjectionHostHealthClassification`
> updated to the corrected semantics. `TestOverviewStats_AccurateCount` now asserts the
> exact value ("10") scoped to the `stat-total-events` ValueID region via the new
> `statValueByHTMLID` test helper — the bare `>1<`/`>10<` page-wide Contains was the
> vacuous-assertion class this program is eliminating.
> (4) **Second false-green found and fixed (M8.5+):** the rebuilt bundle STILL lacked all
> amber classes — errorpage defines its runtime class strings in `styles.go` (Go source),
> not `.templ`, and the CSS build only scanned `*.templ`. The flake app now copies
> `styles.go` into the scan dir and the canary list pins `bg-amber-50/100`,
> `border-amber-200`, `bg-amber-900` (bundle 58,126 → 69,156 bytes). Lesson generalized:
> **a family module may carry runtime classes in ANY source file — scan the module's
> non-test `.templ` AND its styles source, and extend canaries per adopted component.**
> (5) Lint end-state: exactly the 4 pre-planned M26 complexity findings; the M8 WIP's
> `ctx := context.Background()` shadow (contextcheck) removed in favor of the passed ctx.

| ID | Micro task | Min | Dep |
| --- | --- | --- | --- |
| M6.1 | Add `templ-components/errorpage` dep; read ErrorHandlerConfig contract | 10 | M4 |
| M6.2a | Define HTMLShell bridging renderLayout head | 8 | M6.1 |
| M6.2b | Wire shell into ErrorHandler config | 7 | M6.1 |
| M6.3a | Map errorfamily family -> errorpage.Family | 8 | M6.2 |
| M6.3b | Route renderError through WriteError | 7 | M6.2 |
| M6.4a | NotFound handler for unknown streams/projections | 8 | M6.2 |
| M6.4b | Wire WriteNotFound404 + test shape | 7 | M6.2 |
| M6.5 | `pageData.Nonce` via `httputil.NonceFromRequest` (shared plumbing, reused by M11/M12) | 12 | M6.1 |
| M6.6 | Tests: family → status, links present, 404 shape | 12 | M6.3 |
| M6.7 | Commit | 11 | M6.6 |
| M7.1 | `BadgeType` mapping helper (statusGood/Warn/Bad/Neutral) | 10 | M5 |
| M7.2 | Swap remaining raw badge spans (`renderProjectionDetail` + others) | 12 | M7.1 |
| M7.3 | Delete `.badge-*` CSS rules | 5 | M7.2 |
| M7.4 | Tests + commit | 12 | M7.3 |
| M7.5 | Lint/gofmt pass | 6 | M7.4 |
| M8.1 | `StatCardProps` mapping helper (value/label/tone) | 12 | M4 |
| M8.2 | ValueID scheme: `stat-<name>-<projection>` (stable, documented) | 10 | M8.1 |
| M8.3a | Overview statCards loop -> Grid container | 8 | M8.2 |
| M8.3b | Overview statCards -> StatCard components | 7 | M8.2 |
| M8.4 | Projection detail: statGrid → Grid + StatCard | 12 | M8.2 |
| M8.5 | Delete `.stat-card`/`.stat-grid` CSS | 8 | M8.4 |
| M8.6 | Update tests | 12 | M8.4 |
| M8.7 | Note decision: SSE JS may later target ValueIDs (do not change JS yet) | 10 | M8.3 |
| M8.8 | Commit | 11 | M8.6 |

### Tier 3 — The 20% sweep (M9–M16)

| ID | Micro task | Min | Dep |
| --- | --- | --- | --- |
| M9.1 | EmptyStateProps helper w/ per-page icon | 12 | M4 |
| M9.2 | Swap `emptyState` call sites | 10 | M9.1 |
| M9.3 | Delete `.empty-state` CSS | 5 | M9.2 |
| M9.4a | PageHeader swap: overview + events + aggregates | 8 | M4 |
| M9.4b | PageHeader swap: projections + DLQ + audit + timetravel | 7 | M4 |
| M9.5 | Delete `.page-header` CSS | 5 | M9.4 |
| M9.6a | Update string-contains tests | 8 | M9.3 |
| M9.6b | Commit | 7 | M9.3 |
| M9.7 | Lint pass | 8 | M9.6 |
| M10.1 | Button variant mapping helper (default/danger/accent) | 12 | M4 |
| M10.2a | Swap .btn + .btn-accent call sites | 8 | M10.1 |
| M10.2b | Swap .btn-danger call sites | 7 | M10.1 |
| M10.3 | Delete `.btn*` CSS | 8 | M10.2 |
| M10.4a | Update tests | 8 | M10.3 |
| M10.4b | Commit | 7 | M10.3 |
| M10.5 | Lint pass | 10 | M10.4 |
| M11.1a | Copy adminui toastHost wrapper into dashboardui | 8 | M6.5 |
| M11.1b | Adapt event names + nonce param | 7 | M6.5 |
| M11.2 | Mount ToastContainer in renderLayout (nonce) | 10 | M11.1 |
| M11.3 | Align `triggerToast` Hx-Trigger body → `tcShowToast(message, type, title, duration)` | 12 | M11.1 |
| M11.4 | Delete old toast JS block + `.toast-*` CSS | 10 | M11.3 |
| M11.5a | Update tests | 8 | M11.4 |
| M11.5b | Visual toast check in browser | 7 | M11.4 |
| M11.6a | Commit tier-3 toast work | 6 | M11.5 |
| M11.6b | Verify git status clean post-commit | 7 | M11.5 |
| M12.1 | Add `htmx.GlobalErrorHandling` to layout head | 12 | M11 |
| M12.2 | Configure MaxRetries/RetryDelayMS/MaxErrorHistory | 10 | M12.1 |
| M12.3 | Verify error → toast path fires (kill server mid-swap) | 10 | M12.2 |
| M12.4a | Update tests | 7 | M12.3 |
| M12.4b | Commit | 6 | M12.3 |
| M13.1a | Add CopyButton props helper for payload | 8 | M11 |
| M13.1b | Replace window.copyPayload handler + button | 7 | M11 |
| M13.2a | CopyButton for command/query IDs | 8 | M11 |
| M13.2b | CopyButton for stream IDs (metaRowCopyable) | 7 | M11 |
| M13.3 | Delete `data-copyable` listener + `.copyable` CSS | 8 | M13.2 |
| M13.4 | Tests + commit | 12 | M13.3 |
| M13.5 | Lint pass | 10 | M13.4 |
| M14.1a | Map events listing -> Headers + Rows | 10 | M7 |
| M14.1b | Render via display.Table in events renderer | 10 | M7 |
| M14.2a | Add sort query param plumbing | 10 | M14.1 |
| M14.2b | TypedHeaders with SortDirection from request | 10 | M14.1 |
| M14.3 | `LazyRows` decision + implement for long listings | 12 | M14.1 |
| M14.4a | Decide live-row strategy (document decision) | 7 | M14.1 |
| M14.4b | Implement chosen strategy | 8 | M14.1 |
| M14.5a | Update events tests | 8 | M14.4 |
| M14.5b | Add sort-param test | 7 | M14.4 |
| M14.6a | go vet + lint | 9 | M14.5 |
| M14.6b | Commit | 9 | M14.5 |
| M15.1a | Map commands listing -> Table | 9 | M14 |
| M15.1b | Render + tests commands table | 9 | M14 |
| M15.2a | Map queries listing -> Table | 9 | M14 |
| M15.2b | Render + tests queries table | 9 | M14 |
| M15.3a | Extract shared sortable-header helper | 8 | M15.1 |
| M15.3b | Refactor both tables onto helper | 7 | M15.1 |
| M15.4a | Tests audit tables | 8 | M15.3 |
| M15.4b | Commit | 7 | M15.3 |
| M15.5 | Commit | 12 | M15.4 |
| M15.6 | Lint pass | 12 | M15.5 |
| M16.1a | Projections table -> Headers/Rows | 8 | M14 |
| M16.1b | Render + spot-check | 7 | M14 |
| M16.2a | DLQ table -> Headers/Rows | 8 | M14 |
| M16.2b | Render + spot-check | 7 | M14 |
| M16.3a | Snapshots + aggregates renderer -> Table | 8 | M14 |
| M16.3b | Verify shared renderStreamIndex path | 7 | M14 |
| M16.4a | Time-travel listing -> Table | 8 | M14 |
| M16.4b | Spot-check detail page | 7 | M14 |
| M16.5a | Update tests: projections + DLQ | 10 | M16.4 |
| M16.5b | Update tests: snapshots/aggregates/timetravel | 10 | M16.4 |
| M16.6a | go vet + lint full module | 10 | M16.5 |
| M16.6b | Commit table conversions | 10 | M16.5 |

### Tier 4 — The long tail to 100% (M17–M28)

> **RUN-2 FINAL VERDICT (2026-09-17): Tier 4 executed. 15 of 22 capabilities adopted; score re-estimated ~85/100.**
> Adopted this tier: forms.Select (page size), display.Table raw-body + data-row paths across
> all nine listing tables, display.DefinitionList across all six detail metadata surfaces,
> display.CopyButton for every copyable ID and the event payload. **Deliberate exclusions with
> reasons (not drift):** navigation.Pagination (cursor+history model, numbered pages meaningless
> on append-only journals), display.ListNote (count-only vs the dashboard's range semantics),
> navigation.SidebarNav/layout.AppShell (custom dark theme + mobile drawer, adminui precedent),
> ThemeScript/ThemeToggle (new feature, not an adoption swap — routed to TODO_LIST if wanted).
> Lint end-state: ZERO golangci-lint findings in dashboardui (all four planned M26 complexity
> findings refactored away: LoadEventByID split into per-source loaders, FetchOverview health
> classification extracted, renderOverview → renderStatGrid + renderRecentEventsTable,
> renderEventDetail → eventMetaItems). New guards: golden markup pins, CSP handler-attribute
> prohibition test, a11y live-region/landmark/aria-sort tests, hybrid-render benchmark
> (~2µs/component, docs/benchmarks/dashboardui-render-2026-09-17.md).

| ID | Micro task | Min | Dep |
| --- | --- | --- | --- |
| M17.1a | Pagination URL/param builder helper | 10 | M14 |
| M17.1b | Swap pager markup -> navigation.Pagination | 10 | M14 |
| M17.2 | `display.ListNote` replaces `renderPaginationInfo` | 10 | M17.1 |
| M17.3a | forms.Select props (options, selected) | 8 | M17.1 |
| M17.3b | Swap page-size selector + tests | 7 | M17.1 |
| M17.4 | Delete `.pagination` CSS | 5 | M17.3 |
| M17.5a | Update pagination tests | 9 | M17.4 |
| M17.5b | Visual check pager across pages | 8 | M17.4 |
| M17.5c | Commit | 8 | M17.4 |
| M18.1 | `ThemeScript(nonce)` in head | 10 | M6.5 |
| M18.2 | `ThemeToggle` in sidebar/header | 12 | M18.1 |
| M18.3 | `@custom-variant dark` strategy in app.css (class-based) | 8 | M18.1 |
| M18.4a | Tests (toggle persists, no FOUC) | 8 | M18.3 |
| M18.4b | Commit | 7 | M18.3 |
| M19.1a | Map p.Nav -> SidebarNavItem list | 8 | M4 |
| M19.1b | Render SidebarNav in renderSidebar | 7 | M4 |
| M19.2 | Keep custom dark-sidebar theming via `BaseProps.Class` override | 12 | M19.1 |
| M19.3 | Mobile hamburger decision (keep JS; document) | 10 | M19.1 |
| M19.4a | Tests (active state, logout link) | 8 | M19.2 |
| M19.4b | Commit | 7 | M19.2 |
| M19.5 | Lint pass | 8 | M19.4 |
| M20.1 | DefinitionList helper replacing `metaRow` | 12 | M4 |
| M20.2 | CopyButton integration in definition rows | 10 | M13 |
| M20.3 | Delete `.meta-table` CSS | 5 | M20.2 |
| M20.4a | Tests definition lists | 9 | M20.3 |
| M20.4b | Commit | 9 | M20.3 |
| M21.1 | Delete `.data-table` CSS block | 10 | M16 |
| M21.2 | Audit dashboardCSS for dead rules (grep class usage) | 12 | M21.1 |
| M21.3 | Shrink dashboardCSS to token layer + JS-behavior hooks only | 12 | M21.2 |
| M21.4 | Visual parity sweep + commit | 11 | M21.3 |
| M22.1a | Write CSP-header -> DOM nonce test | 8 | M11 |
| M22.1b | Run + fix | 7 | M11 |
| M22.2 | Negative test: inline script without nonce would be blocked (assert none emitted) | 12 | M22.1 |
| M22.3 | Assert toast/copy/global-error scripts carry nonce attr | 12 | M22.1 |
| M22.4a | Run full security suite | 10 | M22.3 |
| M22.4b | Fix findings if any | 6 | M22.3 |
| M22.4c | Commit | 5 | M22.3 |
| M23.1a | Add golden test harness file | 8 | M16 |
| M23.1b | Wire -update flag | 7 | M16 |
| M23.2a | Goldens: StatusBadge + StatCard | 8 | M23.1 |
| M23.2b | Goldens: EmptyState + Button | 7 | M23.1 |
| M23.3 | Document `-update` flow in module README | 10 | M23.2 |
| M23.4a | Ensure test path covers goldens | 10 | M23.3 |
| M23.4b | Commit | 10 | M23.3 |
| M24.1a | Bench skeleton + b.Loop setup | 10 | M16 |
| M24.1b | Hand-rolled vs hybrid sub-benches | 10 | M16 |
| M24.2a | Run -count=5 -benchmem capture | 8 | M24.1 |
| M24.2b | Save machine-pinned baseline artifact | 7 | M24.1 |
| M24.3 | Run + record numbers | 12 | M24.2 |
| M24.4a | Analyze benchstat output | 8 | M24.3 |
| M24.4b | Write per-component perf note | 7 | M24.3 |
| M24.5a | Commit bench artifacts | 6 | M24.4 |
| M24.5b | Verify git clean | 7 | M24.4 |
| M24.6a | Run bench-spike gate | 8 | M24.5 |
| M24.6b | Confirm threshold green | 7 | M24.5 |
| M25.1a | Audit focus/keyboard paths | 10 | M16 |
| M25.1b | Audit roles + aria-sort/aria-live | 10 | M16 |
| M25.2a | Set up axe sweep or checklist doc | 8 | M25.1 |
| M25.2b | Run sweep, collect findings | 7 | M25.1 |
| M25.3a | Add e2e spec: status badges + stat cards | 9 | M25.2 |
| M25.3b | Add e2e spec: toasts + error pages | 8 | M25.2 |
| M25.3c | Add e2e spec: sortable tables | 8 | M25.2 |
| M25.4a | Fix a11y findings | 8 | M25.3 |
| M25.4b | Fix e2e findings | 7 | M25.3 |
| M25.5a | Run full e2e suite | 8 | M25.4 |
| M25.5b | Commit | 7 | M25.4 |
| M26.1a | Extract overview stat-grid renderer | 9 | — |
| M26.1b | Extract projection-health panel renderer | 8 | — |
| M26.1c | Extract remaining overview sections | 8 | — |
| M26.2a | Guard-clause LoadEventByID | 10 | — |
| M26.2b | Extract payload-load helper | 10 | — |
| M26.3a | Split renderEventDetail: header + meta | 10 | — |
| M26.3b | Split renderEventDetail: payload + nav | 10 | — |
| M26.4a | Extract FetchOverview stat computation | 8 | — |
| M26.4b | Extract FetchOverview aggregation | 7 | — |
| M26.5 | Tests + commit | 10 | M26.1 |
| M27.1 | e2e/server: replace ListenAndServe with httputil.NewServer timeouts (G114) | 10 | — |
| M27.2 | e2e/server exhaustruct + param-name fixes | 10 | M27.1 |
| M27.3 | integration_test templ-components indirects: note train policy (do not hand-bump per AGENTS.md) | 10 | — |
| M27.4a | Commit | 7 | M27.2 |
| M27.4b | Verify e2e still builds | 8 | M27.2 |
| M28.1a | Recount adopted capabilities | 8 | M16 |
| M28.1b | Recompute score vs 14/100 | 7 | M16 |
| M28.2 | Update AGENTS.md dashboardui adoption table + score | 12 | M28.1 |
| M28.3 | ANNOTATE (not rewrite) old status report + audit with outcomes | 8 | M28.2 |
| M28.4 | Commit | 10 | M28.3 |

**Micro totals:** 193 tasks · every task ≤ 12 min. Per-medium-task micro sums equal the medium minutes exactly. Tier sums match Section 2.

---

## 4. Execution Graph (mermaid.js)

```mermaid
flowchart TD
    subgraph T0["Tier 0 · Session debt (150 min)"]
        M1["M1 Reconcile reports + fix links"]
        M2["M2 HARVEST docs/TODO/AGENTS"]
        M3["M3 Findings-gate triage"]
    end

    subgraph T1["Tier 1 · 1% → 51% (180 min)"]
        M4["M4 Tailwind v4 gate"]
        M5["M5 StatusBadge spike"]
    end

    subgraph T2["Tier 2 · 4% → 64% (225 min)"]
        M6["M6 errorpage + 404"]
        M7["M7 Badge full swap"]
        M8["M8 StatCard + ValueID"]
    end

    subgraph T3["Tier 3 · 20% → 80% (600 min)"]
        M9["M9 EmptyState + PageHeader"]
        M10["M10 Buttons"]
        M11["M11 Nonce + ToastContainer"]
        M12["M12 GlobalErrorHandling"]
        M13["M13 CopyButton"]
        M14["M14 Events table sortable"]
        M15["M15 Audit tables"]
        M16["M16 Remaining tables"]
    end

    subgraph T4["Tier 4 · → 100% (750 min)"]
        M17["M17 Pagination trio"]
        M18["M18 ThemeToggle"]
        M19["M19 SidebarNav"]
        M20["M20 DefinitionList"]
        M21["M21 CSS cleanup"]
        M22["M22 Security tests"]
        M23["M23 Golden tests"]
        M24["M24 Benchmark"]
        M25["M25 a11y + e2e"]
        M26["M26 Complexity refactors"]
        M27["M27 e2e/server fixes"]
        M28["M28 Re-score + annotate"]
    end

    M3 --> M4
    M4 --> M5
    M5 --> M7
    M4 --> M6
    M6 --> M11
    M4 --> M8
    M4 --> M9
    M4 --> M10
    M7 --> M14
    M14 --> M15
    M14 --> M16
    M11 --> M12
    M11 --> M13
    M16 --> M17
    M16 --> M21
    M16 --> M23
    M16 --> M24
    M16 --> M25
    M6 --> M18
    M4 --> M19
    M13 --> M20
    M11 --> M22
    M12 --> M22
    M26 -. independent .-> T4done
    M27 -. independent .-> T4done
    M15 --> M28
    M22 --> M28
    M25 --> M28
```

**Critical path:** M3 → M4 → M5 → M7 → M14 → M16 → {M17, M21–M25} → M28. Independent tracks (M26, M27) can run anytime.

---

## 5. Execution protocol (per task)

1. Re-check `git status --short` (concurrent sessions).
2. Execute micro tasks in ID order within the medium task.
3. Verify per medium task: `GOEXPERIMENT=jsonv2 go build ./... && go vet ./... && go test ./... -count=1` inside `dashboardui/`.
4. Commit at every medium-task boundary (detailed message). Hook failure on pre-existing findings → documented `--no-verify` + justification (until M3 lands).
5. Visual check after any task that changes rendered HTML.
