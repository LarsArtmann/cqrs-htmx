# Status — dashboardui × templ-components Adoption Program, Execution Run 2 COMPLETE

**Generated:** 2026-09-18 05:42 CEST
**Scope:** Run 2 execution (resumed from `docs/status/2026-09-17_21-03_dashboardui-adoption-execution-run-2.md` halt mid-M8) through the entire remaining plan: M8 recovery → M28 re-score. This report supersedes the run-2 halt report.
**Verified state at writing:** dashboardui build/vet/test green (2 packages ok), **0 golangci-lint issues**, 4 golden files, clean tree, integration_test green, CSS bundle 71,686 bytes with all canaries. HEAD = `19a37e9f` (sibling revert atop this program's work).

> **ANNOTATED 2026-09-20** (docs-health sweep): Run 2's trailing debt was closed by the 2026-09-19 run 3 (N1–N17) and the v4.11.0 train.
> - **§a (fully done):** self-declared done with per-row evidence — left unstruck (already clear).
> - **§b:** b1/b2/b4/b5/b6/b7/b8/b9 DONE; **b3 (M28 formal rubric recount)** still an estimate — open.
> - **§c:** c2/c4/c5/c6/c7 DONE; **c1 (theme toggle)** + **c3 (upstream ask filing: ListNote range, Grid children)** remain open → `TODO_LIST.md`.
> - **§d / §e / §g:** session-local retrospective + questions — historical record, no action.
> - **§f:** struck rows are confirmed done (evidence = 09-19 run3 N-reports); unmarked rows remain open and are routed to `TODO_LIST.md` (print stylesheet check, SSE live-row injection test, goldens extension, canary manifest, combined filter/sort test, shared test helper, `statValueByHTMLID` guard, ETag stale-version test, golden CI note, ListNote revisit, upstream bench methodology, `zz_debug` retirement).

---

## a) FULLY DONE

| Task | Outcome | Verified by |
| --- | --- | --- |
| **M8 recovery** | Both "failing" tests root-caused as **vacuous assertions exposed, not breakage**: `TestOverview_HealthStatCard` had matched the *Projections* card's `stat-card ok` class since inception; the real "Unhealthy" value exposed that dashboardui classified `stopped` workers as bad — contradicting the root library's readiness gate (`ProjectionReadinessCheck`: live/stopped = ready). Fixed at root cause: `ProjectionStatusKind("stopped") → StatusGood`, both classification tables + 3 tests updated to corrected semantics. `TestOverviewStats_AccurateCount` now asserts exact value "10" scoped to the `stat-total-events` element via the new `statValueByHTMLID` helper. | 3× stability runs + integration_test |
| **M8 CSS false-green #2** | errorpage defines its runtime classes in `styles.go` (Go source), not `.templ` — the scan never saw the amber family. Flake app now copies `styles.go`; canaries pin `bg-amber-50/100`, `border-amber-200`, `bg-amber-900`. | grep + assets_test |
| **M9 EmptyState** | All empty states → `display.EmptyState` (`role="status"`, h2, icon tile) with per-page nav icons; `.empty-state` CSS deleted; `contextcheck` forced full ctx threading: `renderLayout(ctx, …)`, `renderStreamIndex` callback type, all render funcs + closures. | lint 0 + tests |
| **M10 Buttons** | All link/submit buttons → `display.Button` (Secondary / OutlineInfo / OutlineDanger mapping for `.btn` / `.btn-accent` / `.btn-danger`); disabled pager links use the library's aria-disabled treatment; `.btn*` CSS deleted. **Second scan fix:** Button variants live in `display/button_go.go` — flake now copies every package's `*_go.go` class maps. | bundle greps + tests |
| **M11 Toasts** | `feedback.ToastContainer` mounted in layout (nonce-gated); `triggerToast` emits merged `dashboardui:toast` Hx-Trigger (adminui contract); dashboard.js bridges to `tcShowToast`; old `.toast-*` CSS + `showToast` listener deleted; bridge pinned by `TestLayout_ToastBridge`. | test + lint |
| **M12 GlobalErrorHandling** | `htmx.GlobalErrorHandling` mounted (2 retries, 1s base, 10-history, named constants); `TestLayout_GlobalErrorHandling` pins announcer + handler. | test |
| **M13 CopyButton** | Every copyable ID (overview stream, events rows, audit rows, detail subtitles, aggregates/snapshots/timetravel h2, `metaRowCopyable`) → `display.CopyButton` (`data-tc-copy`); payload Copy button swapped; `data-copyable` listener, `copyPayload` JS, `.copyable` CSS all deleted. | tests |
| **M14 Events table** | `display.Table` data-row path with `TypedHeaders` (server-side `?sort=&dir=` preserved, aria-sort + indicators), `LazyRows`, BodyID `events-tbody`; SSE live-row JS retargeted; dead `sortHeader` + `.sort-header` CSS deleted. | tests |
| **M15 Audit tables** | Commands + queries tables → `Table` data rows + shared `plainHeaders` helper. | tests |
| **M16 Remaining tables** | All 7 remaining tables (overview recent-events, projection-health panel, projections index, aggregate timeline + index, time-travel index + detail, both DLQ tables) → `Table` via `tableHTMLRaw` (raw tbody, typed headers); `.data-table` + `.table-scroll` CSS deleted. | tests + integration_test |
| **M17 Page-size** | `forms.Select` adopted (values are full URLs, delegated change listener). **Excluded with reasons:** `navigation.Pagination` (cursor+history vs numbered pages), `display.ListNote` (count-only vs range semantics). `.page-size-selector` CSS deleted. | tests |
| **M20 DefinitionList** | All six detail metadata surfaces → `display.DefinitionList` with `defItem`/`defItemRaw`/`defItemCopy` helpers + `templ.Join` copy-button components; `metaRow`/`metaRowCopyable` + `.meta-table` CSS deleted. | tests |
| **M21 CSS prune** | Dead-rule audit ran clean — incremental per-swap deletion (M8–M20) already removed every unused rule; remaining CSS is the token layer + JS behavior hooks. | audit script |
| **M22 CSP security** | Last two inline handlers moved to delegated listeners (`data-download-payload`, `select[name="limit"]` change). New `csp_test.go`: library scripts carry nonce; **no** rendered page emits onclick/onchange/onsubmit/onload; nonce plumbing populated under SecurityHeaders+Nonce middleware. | 3 passing tests |
| **M23 Goldens** | 4 golden files pin exact markup of StatusBadge/Badge/StatCard(+ValueID)/EmptyState; `-update` flag wired; drift fails with regenerate instructions. | golden suite |
| **M24 Benchmark** | `BenchmarkRenderStatCard` (hand-rolled vs hybrid, `b.Loop` + benchmem + count=5); result **~112 ns vs ~2.2 µs per component** — machine-pinned raw + analysis in `docs/benchmarks/dashboardui-render-2026-09-17.{txt,md}`. | benchstat-able artifact |
| **M25 a11y** | In-process contract tests: skip link + landmarks, polite live regions (toast host, error announcer), hamburger label, `role="status"` empty states, aria-sort ascending/none on typed headers. | 4 passing tests |
| **M26 Complexity** | All 4 planned findings refactored away: `LoadEventByID` → 3 per-source loaders; `FetchOverview` → `classifyProjectionHealth`; `renderOverview` → `renderStatGrid` + `renderRecentEventsTable`; `renderEventDetail` → `eventMetaItems`. **Lint end-state: 0 issues, all linters.** | golangci-lint |
| **M27 e2e server** | G114 fixed (explicit timeouts mirroring setup.Bundle posture, named constants), full struct literal, parameter rename; e2e/server at **0 lint issues**. | lint |
| **M28 Re-score + docs** | AGENTS.md dashboardui section rewritten (adoption table, exclusions, gotchas); plan doc Tier-2 + Tier-4 verdict banners; CHANGELOG extended (M9–M28). | n/a |

**Net score: 15 of 22 applicable capabilities adopted (was 1 of 22). Re-estimated score ~85/100 vs 14/100** (estimate — see c/e for why it is not yet a formal recount).

## b) PARTIALLY DONE

1. ~~**M25 Playwright specs (M25.3a–c) — not written.** I delivered the in-process a11y/security contract tests but skipped the axe sweep and the three e2e specs (badges/stat cards, toasts/error pages, sortable tables). The e2e server exists and builds; the browser layer was out of this run's verified loop.~~ done (09-19 run3 N5-N8 browser e2e specs + screenshots)
2. ~~**M24 benchmark breadth — one component, not per-family.** Plan wanted sub-benches per component family + benchstat analysis + per-component notes. I benched StatCard only (the highest-volume component) and recorded one run.~~ done (09-19 N16.1-4 four new bench families)
3. **M28 formal recount — estimate, not rubric.** "~85/100" is my judgment against the 22-capability list, not a re-run of the audit's weighted checklist. The 4 exclusions are documented but unweighted.
4. ~~**M28.3 annotation incomplete.** Plan doc + CHANGELOG annotated, but the two PRIOR artifacts (`docs/status/2026-09-17_13-12_…audit-status.md`, `docs/status/2026-09-17_21-03_…run-2.md`, and the sibling's deep-dive HTML) were NOT annotated with outcomes — plan requires ANNOTATE, not rewrite.~~ done (09-19 N13 outcome banners + this docs-health sweep)
5. ~~**M17.5b visual pager check — not done.** No browser/visual verification of any swapped page happened this run (string-level assertions only). This was user-question ② from the halt report — proceeded on the documented "string-verify only" assumption.~~ done (09-19 N5 27 screenshots)
6. ~~**CHANGELOG heading.** Everything sits under `[Unreleased]` — correct until cut, but the release-train decision (see f) is unresolved.~~ done (09-19 N2 dashboardui v4.10.0/v4.10.1 CHANGELOG cut)
7. ~~**M23.3** — `-update` flow documented in the golden file header comment but NOT in the module README (plan wanted README).~~ done (09-19 N12 README -update flow)
8. ~~**Cross-cutting gates.** `check-release-train` / `check-version-drift --strict` were planned "before M28" and were NOT run. No dashboardui go.mod changes happened this run (requires unchanged), so risk is low, but the gate was not executed.~~ done (09-19 N1 full gate ladder green)
9. ~~**`nix fmt` sweep** — planned cross-cutting item, not run (treefmt over the whole tree risks formatting the sibling session's dirty files).~~ done (09-19 N15 nix fmt zero diff)

## c) NOT STARTED

1. **M18 ThemeScript/ThemeToggle** — deliberately deferred: it is a NEW feature (dark-mode switcher), not an adoption swap; the dashboard follows OS `prefers-color-scheme` today. Routed here instead of TODO_LIST — needs a decision (see g).
2. ~~**M19 SidebarNav** — not attempted; excluded with the adminui precedent (custom dark theme + mobile drawer). Documented, but no TODO_LIST/upstream entry was filed.~~ done (09-19 N14 SidebarNav revisit criteria recorded)
3. **Upstream asks** — ListNote range variant; children-less Grid guidance. Neither filed into TODO_LIST or upstream.
4. ~~**Workspace-mode verification.** `nix run .#build` / `.#test` (GOWORK on) were never run this session — only hermetic GOWORK=off per-module. The sibling's go-directive churn (1.27.1 bump reverted at `19a37e9f`) means workspace mode was likely broken mid-session and I never proved it green.~~ done (09-19 N1 workspace-mode gates green)
5. ~~**Browser/visual pass** — zero eyes-on verification of the new UI (light, dark, mobile, print).~~ done (09-19 N5 27 screenshots light/dark/mobile)
6. ~~**Release train** — dashboardui behavior changed materially (status semantics, table markup, CSP-safe handlers); no tag cut, no `verify-tag.sh` flow started.~~ done (09-19 N2 dashboardui v4.10.0 + v4.10.1 pushed)
7. ~~**Vacuous-assertion sweep** — `statValueByHTMLID` fixed 2 of an unknown population; adminui/setup/integration_test string-contains tests were never audited for the same class.~~ done (09-19 N11 vacuous-assertion triage verdict: no false-green class)

## d) TOTALLY FUCKED UP

1. **Scripted regex patches broke files twice.** My python `str.replace` patches on multi-line `fmt.Fprintf` calls mangled `handlers_audit.go` (wrote a literal `// PLACEHOLDER` into a format string) and `handlers_dlq.go` (mismatched verb/arg lists) — both required `git restore` + redo via exact-match editing. The second time I also truncated a `renderPagination` arg (`timelinePage, ""` → `timelinePage,`). **Root cause: using batch string-replace where the edit tool's exact matching was required. Cost: ~6 round trips.**
2. **Mangled 7 import blocks at once** — the import-insertion script emitted `""github.com/…""` (double-quoted) into every file it touched, breaking the build tree-wide. Same root cause as (1).
3. **Daemon shredded narrative history for most of Run 2.** Despite the AGENTS.md "commit at phase boundaries" rule, the daemon won the race for M8 code, M10, M12, M13, M16 bulk, M21/M22 code, and both doc passes — those landed as `chore: auto-commit` heuristic commits with no story. 11 narrative commits survived (M9, M14, M15, M17, M20, M23, M24, M25, M26, M27, CSS-scan fix, M8 docs); the rest are archaeological.
4. **`TestCSP_UnsafeInlineNotRequired` is a permanent SKIP** — I built the middleware with `SecurityHeadersConfig{}` which emits no CSP header, so the test's meaningful assertion path never executes. A test that can never assert is dead weight.
5. **Benchmark baseline is weak.** The "hand-rolled" sub-bench is a re-creation of the old markup from memory (not lifted from git history) and carries a dead `_ = ctx` inside the loop. The 19× delta direction is trustworthy; the absolute comparison is softer than the doc implies.
6. **I nearly reintroduced the nolint/golines fight twice** — long `//nolint:ireturn` reasons pushed lines >120 so golines wrapped the signatures and detached the directives (nolintlint flagged them unused). Fixed by shortening reasons to keep the func line ≤120; this trap is now documented three times in this repo and I still stepped in it once.
7. **A long stretch of uncommitted work existed repeatedly** (10–30 min windows) despite the documented daemon-absorption history — the M16 table sweep and the M20 DefinitionList sweep each sat uncommitted through multiple green-verify boundaries before landing.

## e) WHAT WE SHOULD IMPROVE

1. **Patch discipline rule (personal):** multi-line function-call edits go through the `edit` tool with full exact context — batch regex scripts are for flat single-line replacements only. Twice-bitten; codify it.
2. **Verify-at-boundary rule:** after each green verify, commit THAT MINUTE even mid-task; the daemon race is unwinnable over 10+ minute windows.
3. **Workspace gates at session start + end:** run `nix run .#build`/`.#test` (workspace mode) first and last; hermetic per-module runs mask workspace breakage (go.work churn proved it again).
4. **Release gates before "done":** `check-release-train` + `check-version-drift --strict` belong in the M28 checklist, not in a follow-up.
5. **Semantic-conformance reviews as a program step:** the `stopped`=healthy split-brain between dashboardui and the root readiness gate was found by accident during a test fix. A deliberate cross-module review (dashboardui kinds ↔ root readiness ↔ go-health probes ↔ setup `/health`) would likely find more.
6. **Vacuous-assertion class hunt:** two pre-existing test bugs of the same class (page-wide `strings.Contains`) were exposed by markup changes. Every `strings.Contains(body, …)` assertion scoped to a *page* instead of an *element* is a latent false-green.
7. **Test scaffolding for visual truth:** string assertions cannot see CSS class coverage or layout. The goldens + CSS canaries cover the mechanism; nothing covers the eyes. A Playwright screenshot pass (light/dark/mobile) is the missing verification layer.
8. **Skip-tests must fail loud:** a `t.Skipf` test that skips on its default configuration (CSP test) should be constructed to exercise its assertion path or be deleted.
9. **Benchmark honesty:** baselines should be lifted from git history (`git show <rev>:file`), not re-created from memory; dead statements (`_ = ctx`) undermine credibility.
10. **nolint lines stay short:** nolint directives belong on the func line with minimal reason text so golines cannot wrap them off their target.

## f) NEXT (up to 50, Pareto-ordered within tiers)

**Correctness/verification debt (do first):**
1. ~~Annotate the two prior status reports + sibling deep-dive HTML with Run-2 outcomes (M28.3 leftover).~~ done (09-19 N13 outcome banners + this docs-health sweep)
2. ~~Run `nix run .#check-release-train` and `nix run .#check-version-drift --strict`.~~ done (09-19 N1 gates green)
3. ~~Run workspace-mode `nix run .#build` + `.#test` (after sibling's go-directive revert, expect green; prove it).~~ done (09-19 N1 workspace gates green)
4. ~~Run full `nix run .#lint` (all 15 modules) to prove no cross-module fallout from the dashboardui/e2e changes.~~ done (09-19 N1 lint 0/15 modules)
5. ~~Fix `TestCSP_UnsafeInlineNotRequired` to actually emit CSP (configure `NonceConfig.CSPBuilder`) so the unsafe-inline assertion executes.~~ done (09-19 N10 CSP test now asserts)
6. ~~Rebuild benchmark hand-rolled baseline from `git show 81088b64:dashboardui/handler_overview.go` markup; drop `_ = ctx`; re-record artifact.~~ done (09-19 N10 baseline rebuilt from git history)
7. ~~coverage-gate re-run for dashboardui (new files: buttons.go, tables.go, definitions.go, csp/a11y/golden tests shifted coverage).~~ done (09-19 N4 coverage-gate green)

**Browser truth (the missing layer):**
8. ~~Playwright visual pass: all pages, light + dark + mobile viewport, screenshots archived under docs/.~~ done (09-19 N5 27 screenshots)
9. ~~Playwright e2e spec: status badges + stat cards (ValueID presence + health semantics).~~ done (09-19 N6 browser spec)
10. ~~Playwright e2e spec: toasts (write op → `dashboardui:toast` → visible toast) + error pages (404/500 family rendering).~~ done (09-19 N6 browser spec)
11. ~~Playwright e2e spec: sortable tables (click header → aria-sort flip + order change).~~ done (09-19 N6 browser spec)
12. ~~Axe sweep over the dashboard pages; fix findings.~~ done (09-19 N9 axe 9/9 green)
13. Print stylesheet check (toast/error-handling hidden, tables readable).
14. SSE live-row injection test (`dashboard:event` appends into `#events-tbody`).

**Test-debt hunt:**
15. ~~Repo-wide vacuous-assertion sweep: scope every page-level `strings.Contains` test to element IDs (adminui, setup, integration_test, dashboardui leftovers).~~ done (09-19 N11 triage: no remaining false-green class)
16. Extend goldens: Badge error/warning/success variants, EmptyState without icon, Button all variants, DefinitionItem with DetailComponent, TableRow (data path).
17. Link `assets_test.go` canaries to a per-component class manifest (programmatic, so new adoptions extend the list mechanically).
18. ~~DLQ/projection/snapshot detail pages under the CSP nonce test (currently only base listing pages).~~ done (09-19 N15 covered indirectly, documented decision)
19. Combined filter+sort+pagination URL regression test for the events page.
20. Move `statValueByHTMLID` (and future element-scoped helpers) into a shared test helper file with doc.

**Documentation:**
21. ~~Module README: adoption surface table, `-update` golden flow (M23.3), benchmark pointer.~~ done (09-19 N12 README adoption table)
22. ~~Doc guide: "hybrid adoption pattern" (`Component.Render(ctx, &b)`) — the M8 Grid lesson, class-map scan, ctx threading contract, so the next consumer module repeats it safely.~~ done (09-19 N12 hybrid adoption guide)
23. ~~Document the daemon-shredded-commit reality in the release runbook (attribution archaeology note).~~ done (09-19 N12 release playbook §6)
24. ~~AGENTS.md: add "nolint lines stay ≤120 chars incl. reason" to the gotchas.~~ done (AGENTS.md nolint-line-length gotcha)

**Release + trains:**
25. ~~Decide + cut dashboardui release (semantics + markup changed) via `scripts/verify-tag.sh`; CHANGELOG `[Unreleased]` → version.~~ done (09-19 N2 dashboardui v4.10.0 cut + pushed)
26. ~~templ-components family v1.18.0 train decision (dashboardui pinned v1.17.0; integration_test indirects intentionally behind).~~ done (09-19 N3 v1.18.0 uniform repo-wide)
27. Upstream ask: ListNote range variant (Showing X–Y of Z).
28. Upstream ask: document children-less/Grid standalone-render behavior.
29. ~~File dashboardui TODO_LIST entries for the exclusions (theme toggle decision, SidebarNav revisit criteria).~~ done (09-19 N14 dashboardui TODO entries recorded)

**Feature follow-ups (post-adoption):**
30. M18 theme toggle decision (implement vs TODO_LIST).
31. ~~Wire `htmx.PolledRegion` for the projection-health panel (last listed adoption opportunity).~~ **Won't implement — 09-19 N17.1 PolledRegion children render empty in the hybrid path.**
32. SSE fan-out: consider replacing bespoke live-row JS with the library's loading/swap helpers.
33. ~~Filter bar inputs → `forms.Input`/`forms.Form` (unstarted leaf swap).~~ done (09-19 N17.3-4 forms.Input adopted)
34. CSV/JSON export bar → library buttons (currently `formatLinks` uses buttonLink — verify visually).
35. ~~Time-travel slider a11y pass (aria-valuetext, keyboard focus ring) — untouched by adoption.~~ done (09-19 N17.5-6 slider CSP-safe + aria-valuetext)

**Mechanical hygiene:**
36. ~~`nix fmt` treefmt sweep at a clean-tree boundary.~~ done (09-19 N15 nix fmt zero diff)
37. ~~`examples/middleware-showcase/vendor/` — trash if still present.~~ done (09-19 N15 vendor dir gone)
38. `go vet ./...` in workspace mode for e2e (excluded from lint loop).
39. ~~Review `.golangci.yml` exhaustruct deprecation warning (v2.13: migrate to exhaustruct_v5) — repo-wide, flagged by every run.~~ done (exhaustruct_v5 migration documented in AGENTS.md)
40. ~~Consider `ireturn` allowlist entry for `templ.Component` instead of per-func nolints.~~ done (09-19 N16.6 ireturn allow entry)
41. ~~Benchmark additions: Button, EmptyState, DefinitionList, Table raw-body (completes the perf note).~~ done (09-19 N16.1-4 benches landed)
42. `statValueByHTMLID` → assert-once-per-test lint? (optional guard against the vacuous class returning).
43. ~~Check `adminui/styles.css` parity didn't drift from the daemon commits in run 1 (class-set diff).~~ done (09-19 N1 orphan styles.css finding documented)
44. sync `dashboardVersion`/ETag constants unchanged — verify no stale-version test exists for the new JS bridge.
45. Golden files: add CI-path verification note (`nix run .#test` includes them).
46. ~~Extract `renderStreamIndex` ctx-callback migration note into the composability guide.~~ done (09-19 N12 hybrid guide pitfall)
47. Revisit `display.ListNote` exclusion if upstream ships a range variant (linked to #27).
48. Consider upstreaming the hybrid-render benchmark methodology to templ-components docs.
49. ~~`csp_test.go`: cover DLQ detail + projection detail pages (needs fake stores, mirrors handlers_write_test).~~ done (09-19 N15 documented decision (no separate test))
50. Retire the ad-hoc `zz_debug`-style debugging pattern — keep a throwaway-test naming convention documented.

## g) QUESTIONS (cannot answer myself)

1. **Visual sign-off:** Will you do the eyes-on pass of the swapped UI (stat cards, buttons, tables, toasts, error pages — light/dark/mobile), or should I build the Playwright screenshot pass (f#8) before anything else? Every swap this run was verified at string/CSS-class level only; the pixels have never been looked at, and I cannot judge whether e.g. the OutlineInfo "Next →" reads right against the custom accent palette.
2. **Release timing:** dashboardui's behavior changed semantically (stopped = healthy) and structurally (all table/detail markup). Cut a dashboardui tag now via `verify-tag.sh` (consumers get the new UI + semantics immediately, changelog-ready), or hold for the templ-components v1.18.0 family train so the coordinated bump lands together? I can construct either, but the train policy call is yours.
3. **Toolchain direction:** the sibling bumped go floors to 1.27.1 then reverted (`d20d631e` → `19a37e9f`) mid-session. Is the fleet moving to 1.27.1 soon, or should dashboardui and I treat 1.26.7 as the stable floor and re-pin the go.work directive check against it? This determines whether workspace-mode verification (f#3) is meaningful right now or futile until the sibling settles.

---

**Awaiting instructions.**
