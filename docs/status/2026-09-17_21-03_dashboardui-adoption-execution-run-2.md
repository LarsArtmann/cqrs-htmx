> **OUTCOME (2026-09-19):** Run-2 completed and shipped. All adoption tiers landed (M4-M8, M22-M24, goldens, benchmarks); the verification items this report left open were closed by Run 3 (verify-tag releases v4.10.0/v4.10.1, Playwright harness, axe sweep, coverage gate 85.3%, fmt-marker guard). See docs/planning/2026-09-18_16-09_run3-verify-ship-see-harden.md for the closing plan and docs/status/2026-09-18_05-42_dashboardui-adoption-run-2-complete.md for the Run-2 completion snapshot.

# Status Report — dashboardui × templ-components Adoption Program, Execution Run 2

**Written:** 2026-09-17 21:03 CEST
**Scope:** This session only (resumed after the Run-1 halt + "GET SHIT DONE" instruction). Covers M4 closure through M8 mid-flight, plus the three pending questions from Run 1.
**Plan under execution:** `docs/planning/2026-09-17_13-22_dashboardui-templ-components-pareto-execution-plan.md` (M1–M28)

---

## TL;DR

M4 (Tailwind gate) is now **genuinely** open — the Run-1 false green had a second, deeper root cause which is fixed and canary-guarded. M5 (StatusBadge spike), M6 (errorpage adoption), M7 (full badge sweep) are **done, verified, committed with narrative messages**. M8 (StatCard) is **mid-flight and the daemon committed a test-FAILING state to HEAD** — that is the top item to fix on resume. The compiled Tailwind bundle is missing the amber utility family that the new error pages reference (bundle last built before errorpage adoption). Two sibling-session interferences recurred all session: go-directive churn (gopls broken in workspace mode) and daemon absorption of mid-flight work.

---

## a) FULLY DONE (verified + committed)

| Task | What landed | Verification | Commit |
|---|---|---|---|
| **Q1 (go.mod 1.26.7 vs 1.27.1)** | Sibling session self-repaired root go.mod to 1.26.7 early in the session. No repair needed by me. Churn continued elsewhere (see d-3). | `grep "^go" go.mod` → 1.26.7 | — |
| **Q2 (middleware-showcase/vendor/)** | Already covered by existing `.gitignore` line 26 (`vendor/`). Directory left in place (generated artifact, regenerable via `go mod vendor`). | `git check-ignore` → IGNORED | — |
| **Q3 (M5.1 pin)** | **Decision: v1.17.0** — family uniformity (adminui/setup/etc. all at v1.17.0; integration_test deliberately behind per AGENTS.md). `go mod tidy` initially drifted the family to v1.18.0; re-pinned all four family requires to v1.17.0. | go.mod pins | inside M5 commits |
| **M4.1–M4.8 Tailwind gate (CLOSED)** | Root cause of Run-1 false green found: the module cache lays out `templ-components/icons@vX` as a versioned dir — **the icons dir's parent is NOT the root module** (the Run-1 flake fallback copied zero `.templ` files, Tailwind emitted zero library utilities, exit 0). Fixed flake app: derives root version from the icons module, downloads the root module if not extracted, fails loudly on unresolvable TC_DIR / zero scanned `.templ` files, and **canary-asserts the output** (`bg-green-100`, `bg-blue-100`, `bg-green-900`, `dark\:` — note minified CSS escapes the colon). | `nix run .#build-dashboardui-css` → 58,126 bytes, canaries OK (was 10,634 false bytes); `nix flake check --no-build` green | flake.nix in `98ebf293` (daemon-absorbed) |
| **M4.7 serve path** | `dashboardui/assets.go`: `//go:embed assets/dashboard-tw.css` + adminui-style assetHandler (Content-Type, Cache-Control 1d, nosniff, ETag). Route `GET /-/dashboard-tw.css` (guarded). Second `<link>` in `renderLayout`. **Bonus fix:** ETag must be quoted per RFC 7232 — net/http's `etagStrongMatch` requires a leading quote, so the copied adminui pattern never actually 304'd. Fixed in BOTH dashboardui and adminui (fix-on-sight). | New `assets_test.go`: route headers, 304-on-If-None-Match, layout-link, canary utilities — all green | `87ccb802` (daemon) + ETag fix commit |
| **M5 StatusBadge spike** | `statusKindToStatus` (single mapping point: good→healthy, warn→degraded, bad→error, unknown→neutral `display.Badge` with raw text) + `statusBadge(ctx, …)` rendering via `.Render(ctx, &b)` into the existing `strings.Builder`. Old badge switch in `renderProjectionRow` deleted. Badges wrapped in `<td>` (library emits bare `<span>`). | build/vet/test green; tests updated to new classes; badge classes verified present in compiled CSS | `a33c62cd` |
| **M5.8 spike verdict** | Written into the plan doc (§ Tier 1 banner): hybrid path PROVEN + 5 findings for remaining rungs (mapped-word coloring, `<td>` wrapping, tidy-resolves-@latest trap, ctx plumbing expectation, module-cache layout). | — | in `a33c62cd` |
| **M6 errorpage adoption** | `renderError` → method `d.renderError` (25+ call sites swept via sed), renders `errorpage.ErrorPage` with status→Family mapping (`statusToFamily`: 409→Conflict, 4xx→Rejection, 503/502/504→Transient, else Infrastructure). HTMX requests get the bare styled card; full loads get `renderErrorShell` (minimal doc loading both dashboard stylesheets, `.error-shell` centering rule). 404 catch-all (`GET /` → `notFoundHandler`) renders `errorpage.NotFound404` with Overview link (upgraded the sibling's just-landed hand-rolled version in place — kept their route registration). **M6.5 done:** `pageData.Nonce` via `httputil.NonceFromRequest`, plumbed in `page()`. | New `errors_test.go`: family-mapping table, styled-page shape, HTMX-gets-bare-card, nil-request safety, 404 shape with links; 3 stale tests updated | `56dcbfdd` |
| **M7 full badge sweep** | New `badges.go`: `badgeHTML(ctx, text, type)` single mapping point + `statusKindToBadgeType`, `encodingBadgeType`, `countBadgeType`. All remaining hand-rolled badge spans swapped: DLQ counts, projection rows + detail header, event-detail schema/encoding badges, DLQ error-family badges (×2 literal `badge badge-err` found by grep). Deleted: `badgeNeutral/badgeWarn/badgeOK/badgeErr` consts + the whole `.badge*` CSS block. **Went beyond plan:** threaded `ctx` through `renderDLQ`, `renderDLQEntryDetail`, `renderEventDetail`, `renderProjections`, `renderProjectionDetail` (contextcheck demanded it; templ-style ctx-first signatures). exhaustruct satisfied by fully-specified `BadgeProps` (no nolint churn — nolintlint kept deleting the earlier nolint after golines reformats). | build/vet/test green; lint clean except the 4 pre-planned M26 complexity findings | `81088b64` |

**Adoption score effect (rough, formal re-score is M28):** capabilities went from 1/22 (score 14/100) to ~5/22 (Tailwind pipeline, StatusBadge/Badge, errorpage+404, error-family mapping, nonce plumbing). Estimate ≈ 35–40/100. Formal recount pending.

---

## b) PARTIALLY DONE

1. **M8 StatCard + Grid (≈70%, IN PROGRESS, tests failing — see also d-1).** Done: `stats.go` with `statCardHTML(ctx, valueID, value, label, tone)` + `healthKindToTone`; overview's 5 stat cards swapped to `display.StatCard` with stable ValueIDs (`stat-total-events`, `stat-total-aggregates`, `stat-projections-active`, `stat-system-health`, `stat-dlq-count`); projection-detail's 4 cards swapped with `stat-<name>-<metric>` scheme; `statCard()` helper + `healthKindToVariant()` deleted; `.stat-card*` CSS (6 rules) deleted. **Not done:** 2 tests still failing on the new markup (details in d-1); Grid container consciously NOT adopted (documented decision: `display.Grid` renders templ `{children...}` which is empty in a standalone hybrid `.Render` — keeping the 1-line `.stat-grid` container rule; revisit only if M21 wants a pure-Tailwind replacement); M8.7 SSE-JS note (verified safe: `dashboard.js` targets `data-*` attributes and `#projection-health`, NOT `.stat-card-value` — note not yet written into the plan doc); lint pass not run; commit not made by me.
2. **Compiled CSS freshness (forgotten gap).** `assets/dashboard-tw.css` was last built BEFORE the errorpage adoption. Verified just now: `bg-amber-50/100`, `border-amber-200` (errorpage's Rejection family styling) are **absent from the bundle** — styled error pages will render with unstyled amber accents until the bundle is rebuilt. The flake app now resolves the errorpage module (dashboardui requires it), so one `nix run .#build-dashboardui-css` fixes it — plus amber canaries should be added to the app's assertion list.
3. **CHANGELOG / TODO_LIST / AGENTS.md harvesting for M4–M8** — not started (was planned for the M28 re-scoring pass, but CHANGELOG entries should accumulate per task per repo convention).
4. **Plan-doc annotation for M8** (decision + verdict banner like M5's) — not written yet.

---

## c) NOT STARTED

- **M9** EmptyState + PageHeader swap
- **M10** Button swap + `.btn*` CSS deletion
- **M11** ToastContainer + old toast JS/CSS deletion (nonce plumbing prerequisite landed with M6.5)
- **M12** `htmx.GlobalErrorHandling`
- **M13** CopyButton + `data-copyable` JS deletion
- **M14** Events table conversion + sortable headers
- **M15** Audit commands/queries tables
- **M16** Projections/DLQ/snapshots/time-travel tables
- **M17** Pagination trio
- **M18** ThemeScript/ThemeToggle
- **M19** SidebarNav
- **M20** DefinitionList for metaRow
- **M21** `.data-table` CSS deletion + dead-rule audit
- **M22** Security tests (nonce→CSP end-to-end)
- **M23** Golden tests
- **M24** Benchmark hand-rolled vs hybrid
- **M25** a11y pass + e2e extension
- **M26** Complexity refactors (the 4 lint findings: renderOverview 28, LoadEventByID 27, renderEventDetail 26, FetchOverview 22 — all pre-existing, pre-planned)
- **M27** e2e/server G114 + exhaustruct + train note
- **M28** Re-score + AGENTS.md adoption table + report annotations
- Upstream asks (go-structure-linter tag + BuildFlow pin bump to restore `fail_on: critical`; gomod vendor-consistency double-count; missing-submodule-replace false positive) — not yet written into TODO_LIST

---

## d) TOTALLY FUCKED UP (honest list)

1. **The daemon committed a test-FAILING M8 WIP state to HEAD (`edfdcb2c`, 11 files).** `go test ./dashboardui/...` is red right now on master:
   - `TestOverviewStats_AccurateCount` — asserts `>1<` / `>10<` bare text; library StatCard renders values differently (needs a ValueID-scoped assertion).
   - `TestOverview_HealthStatCard` — asserts `"stat-card ok"` class; new markup has library classes + `stat-system-health` ValueID.
   Both are mechanical test updates (~10 min), but the repo's most basic gate (dashboardui tests) is red in HEAD until then. **First action on resume: fix these two tests, lint, rebuild CSS, commit M8 with a narrative message.** Root cause: the daemon polls faster than my task boundary; I did M8 as one big edit batch instead of micro-committing test fixes alongside.
2. **Stale compiled CSS shipped in the embed (amber gap).** The 404/error pages reference `bg-amber-*` utilities that the embedded bundle does not contain (verified: zero matches for `bg-amber-50/100`, `border-amber-200`). My own M6 test asserted `bg-amber-` in the *handler output* but nothing asserts it in the *compiled CSS* — the M5.5 canary check was not re-run after M6/M8 added new class families. Fix: rebuild + extend canary list with `bg-amber-100` + `dark\:bg-amber-900`.
3. **Sibling-session go-directive churn persisted all session.** gopls is broken in workspace mode ("module … requires go >= 1.27.1, but go.work lists go 1.26.7") — every LSP diagnostic I received all session carried this error. I trusted only CLI verification (per AGENTS.md rule), which held, but the LSP was dead weight and the pre-commit hook's workspace golangci failed on it repeatedly (all my commits used the documented `--no-verify` + justification fallback).
4. **Narrative-commit losses continue.** M4's bulk was daemon-absorbed into `98ebf293` (mixed with the sibling's `integration_test/go.mod` churn — attribution muddied), and M8's WIP went into `edfdcb2c`. My explicit `git commit` attempts raced the daemon twice (one landed as a daemon commit behind my back between `git add` and `git commit`; one hook failure let the daemon win). The per-task commits that DID land (M5 `a33c62cd`, M6 `56dcbfdd`, M7 `81088b64`) prove the protocol works when I commit immediately after each verify.
5. **Minor self-inflicted: removed the in-use `errorfamily` import from dashboard.go during an M6 import edit.** Caught immediately by build; restored. Should have scoped that edit to the `pageData` construction only.

---

## e) WHAT WE SHOULD IMPROVE (process, applies going forward)

1. **Micro-commit within medium tasks.** The plan's 12-min micro tasks are the right commit granularity for this tree. Commit after every green `go test` run, not after every medium task — the daemon leaves no alternative.
2. **Re-run the CSS build + canary grep as part of EVERY adoption task.** New components bring new utility families (amber was missed). Candidate: extend the flake canary list now (amber + `dark\:` forms) and make `build-dashboardui-css` part of my per-task verify loop until M23's golden tests cover it.
3. **Add a compiled-CSS canary assertion to `assets_test.go`** (mirror of the flake canary) so a stale bundle fails tests, not just the build app.
4. **Check `git status` immediately before staging** (I did, mostly) — but ALSO re-check after hook failures; that's the window where the daemon wins.
5. **Library-container components (Grid/Table shells with templ children) don't fit the standalone-hybrid path** — document the pattern "keep a minimal container div, adopt the leaf components" in the plan doc so M14–M16 table conversions don't rediscover it. For tables, `display.Table`'s `Body` slot may have the same limitation — investigate BEFORE starting M14.
6. **`go mod tidy` resolves new imports at @latest** — always `go get <family>@v1.17.0` before tidy in this program; this bit once and cost a re-pin.
7. **The `nolint` + formatter interaction is hostile** (nolintlint deletes directives after golines reformats the line). Prefer fully-specified props structs over nolint for exhaustruct; prefer code shape over directives wherever possible.
8. **Run `nix run .#check-release-train` / `check-version-drift --strict` before the M28 finish** — dashboardui now requires the templ-components ROOT module + errorpage + httputil directly; confirm no new drift advisories beyond the known v1.18.0 train-lag.
9. **Verification commands that matter (memory-refresh):** hermetic per-task: `cd dashboardui && GOEXPERIMENT=jsonv2 GOWORK=off go build ./... && go vet ./... && go test ./... -count=1` + `golangci-lint run ./...`; CSS: `nix run .#build-dashboardui-css` (now canary-guarded); flake: `nix flake check --no-build`.
10. **Don't edit import blocks outside the exact scope of the task** (the errorfamily incident) — add/remove only the lines the change itself requires.

---

## f) UP TO 50 THINGS TO DO NEXT (Pareto-ordered within the program)

**Immediate recovery (before anything else):**
1. Fix `TestOverviewStats_AccurateCount` — assert event count via the `stat-total-events` ValueID region instead of bare `>10<`.
2. Fix `TestOverview_HealthStatCard` — assert `stat-system-health` + StatCard tone classes.
3. Rebuild CSS (`nix run .#build-dashboardui-css`); extend flake canaries with `bg-amber-100` + `dark\:bg-amber-900`; grep-verify errorpage amber family + StatCard classes (`bg-blue-50`, `text-green-600`) in the bundle.
4. Add compiled-CSS canary assertions to `assets_test.go` (task 3 as a permanent test).
5. Lint dashboardui (expect only the 4 M26 findings); write M8.7 SSE-JS decision note + M8 verdict banner into the plan doc.
6. Commit M8 with a narrative message immediately after green (beat the daemon).
7. CHANGELOG entries for M4–M8 (one Added block per adopted capability).

**Tier 3 sweep (the 20% → 80%):**
8. M9.1–M9.3: EmptyStateProps helper + swap `emptyState` call sites + delete `.empty-state` CSS.
9. M9.4–M9.7: PageHeader swaps (6 pages) + delete `.page-header` CSS + tests + commit.
10. M10.1: button variant mapping helper (default/danger/accent → display.Button variants).
11. M10.2–M10.5: swap `.btn`/`.btn-accent`/`.btn-danger` call sites; delete `.btn*` CSS; tests; commit.
12. M11.1: port adminui `toastHost(nonce)` wrapper; align `adminui:toast` → `tcShowToast` event bridge.
13. M11.2–M11.6: mount ToastContainer in renderLayout (uses `pageData.Nonce`); rewrite `triggerToast` Hx-Trigger body; delete old toast JS + `.toast-*` CSS; update tests; visual toast check; commit.
14. M12.1–M12.4: mount `htmx.GlobalErrorHandling` in layout head (MaxRetries/RetryDelayMS config); kill-server-mid-swap check; tests; commit.
15. M13.1–M13.5: CopyButton props helper; replace `window.copyPayload` + button; CopyButton for command/query/stream IDs; delete `data-copyable` listener + `.copyable` CSS; tests; commit.
16. M14 pre-check: read `display.Table`'s `Body`/children mechanics in the hybrid path BEFORE starting (the Grid lesson).
17. M14.1–M14.6: events listing → Table (Headers/Rows); sort query param + typed headers with aria-sort; LazyRows decision; live-row strategy decision; tests incl. sort param; commit.
18. M15.1–M15.6: commands + queries listings → Table; extract shared sortable-header helper; tests; commit.
19. M16.1–M16.6: projections/DLQ/snapshots+aggregates/time-travel → Table; update all table tests; commit.

**Tier 4 long tail:**
20. M17.1–M17.5: pagination URL builder + `navigation.Pagination` swap + `display.ListNote` + `forms.Select` page-size + delete `.pagination` CSS.
21. M18.1–M18.4: `ThemeScript(nonce)` + `ThemeToggle` + `@custom-variant dark` class strategy in tailwind.css + tests.
22. M19.1–M19.5: `SidebarNav` structural swap with `BaseProps.Class` dark-sidebar override; document mobile-hamburger decision.
23. M20.1–M20.4: `DefinitionList` replacing `metaRow`/`metaRowCopyable`; CopyButton in definition rows; delete `.meta-table` CSS.
24. M21.1–M21.4: delete `.data-table` CSS; grep-audit `dashboardCSS` for dead rules; shrink to tokens + JS hooks; visual parity sweep.
25. M22.1–M22.4: CSP-header→DOM-nonce test; negative test (no nonce-less inline scripts); assert toast/copy/global-error scripts carry nonce; full security suite; commit.
26. M23.1–M23.4: golden-test harness (`-update` flag); goldens for StatusBadge, StatCard, EmptyState, Button; document `-update` flow in module README; wire into `go test`.
27. M24.1–M24.6: bench skeleton (b.Loop + ReportAllocs, adminui bench pattern); hand-rolled vs hybrid sub-benches; `-count=5 -benchmem`; machine-pinned raw baseline artifact; benchstat analysis; per-component perf note; commit.
28. M25.1–M25.5: keyboard/focus audit; roles + aria-sort/aria-live audit; axe sweep or checklist doc; e2e specs (badges+statcards, toasts+error pages, sortable tables); fix findings; full e2e run; commit.
29. M26.1–M26.5: the 4 complexity refactors (extract overview stat-grid/panel renderers; guard-clause LoadEventByID + payload helper; split renderEventDetail header/meta + payload/nav; extract FetchOverview stat computation + aggregation) — zeroes the last 4 lint findings.
30. M27.1–M27.4: e2e/server `httputil.NewServer` timeouts (G114); exhaustruct + param-name fixes; integration_test templ-components train-policy note (do NOT hand-bump indirects); verify e2e builds; commit.
31. M28.1–M28.4: recount adopted capabilities; recompute score vs 14/100; update AGENTS.md dashboardui adoption table + score; ANNOTATE (not rewrite) the two audit reports + old status reports + this program's plan doc; commit.

**Cross-cutting / housekeeping:**
32. Run `nix run .#check-release-train` and `check-version-drift --strict` — confirm dashboardui's new direct requires (templ-components root, errorpage, httputil v1.2.0) introduce no new UNPUBLISHED/train-lag findings beyond the known v1.18.0 advisory.
33. Update `AGENTS.md` dashboardui adoption section incrementally at M11/M16/M28 (not only at the end) so a fresh session sees current state.
34. Write the upstream asks into TODO_LIST: go-structure-linter suppressions tag + BuildFlow pin bump → restore `fail_on: critical`; gomod vendor-consistency double-count report; missing-submodule-replace false positive.
35. Decide + execute the `examples/middleware-showcase/vendor/` fate (currently gitignored + present; trash the directory to zero the gomod-check vendor-consistency findings — it regenerates in seconds; NOT deleted this session because it appeared mid-session and ownership was unclear).
36. `nix fmt` (treefmt) over the touched files before the M28 finish; verify no formatter fights with the golines-formatted flake app.
37. Coverage gate re-run for dashboardui after M16 (`nix run .#coverage-gate`) — the adopted-module paths need to keep 83.7%/60.
38. `nix run .#check-cqrs-lint` after M16 (new render paths shouldn't trip anything, but the gate is cheap).
39. Wire `build-dashboardui-css` (or its canary) into CI's checks job so a stale committed bundle fails CI, not just local builds.
40. Consider a `check-dashboard-css-freshness` gate: recompile in temp + diff against committed `assets/dashboard-tw.css` (Tier 4 candidate from the plan).
41. Document the module-cache root-module extraction layout + the icons-parent trap in AGENTS.md (this cost Run-1 its false green and Run-2 its root-cause hunt).
42. Document the RFC 7232 quoted-ETag requirement in AGENTS.md (adminui shipped an unquoted ETag for releases without anyone noticing — the 304 path was dead).
43. After the sibling session lands: re-run workspace-mode `nix run .#build` + `.#test` (this session verified hermetically only) and confirm the hook's workspace golangci is green again so future commits don't need `--no-verify`.
44. Re-run `nix run .#check-modules` (isolation stage does per-module go vet GOWORK=off) once M16 lands.
45. Consider adopting `display.EmptyState`'s icon field with the existing `navIconSVG` icons for visual continuity (M9 detail).
46. Verify the HTMX error-card path renders acceptably when swapped into `#main-content` (the card assumes a centered page context; HTMX swaps it into the existing layout — check padding/duplication).
47. Plan-doc: add the "library containers need templ children" pattern + "ValueID scheme is the live-update contract" to §0.1 guards.
48. Sweep for remaining `context.Background()` in render paths once M26 refactor starts (renderOverview's closure currently uses it; thread `r.Context()` when the signature refactor lands).
49. Kill/deprecate `encodingBadge` vs `badgeHTML(ctx, …, encodingBadgeType(…))` duplication if M14's table conversion gives encodings a column anyway (avoid two call shapes for the same badge).
50. At M28: write the program retrospective into the plan doc (actual time per tier vs estimate; the daemon-race and formatter findings are reusable for the next adoption program — setup's UI modules are the likely next consumer).

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (max 3)

1. **Go toolchain direction:** the sibling session keeps bumping module go-directives toward 1.27.x while go.work/flake/AGENTS.md pin 1.26.7. Is a coordinated fleet-wide 1.27 bump PLANNED (then I'll stop treating it as breakage, stay hermetic, and update dashboardui's docs when it lands), or are those bumps accidental (then I should keep repairing to 1.26.7 and flag each occurrence)? This determines whether `--no-verify` stays necessary for dashboardui commits.
2. **Visual sign-off:** I verified everything with string-contains tests and canary greps — zero browser-level rendering checks this session (M4.8's visual-parity step was downgraded to CSS-canary verification). Will you do a quick eyeball pass of `/dashboard` + force a 404 + trigger an error page and tell me whether the token mapping (@theme: gray→surface/bg, blue-600→accent, amber error styling) reads right in both light and dark? If you'd rather I do it, I'll prioritize a Playwright screenshot pass (adds ~45 min before M9).
3. **templ-components v1.18.0:** finish the M9–M28 sweep on v1.17.0 and leave the family bump to a dedicated coordinated train run (my current assumption, keeps this program single-axis), or fold the v1.18.0 family bump into this program now (re-verifies every adopted component once, but couples two change axes)?

---

**Resumption instruction for the next session:** start at f-1 (the two failing tests), then f-2 through f-7 in order — that returns HEAD to green and closes M8 cleanly before the M9 sweep resumes. Recreate the todo list from §a/b/c above (M4–M7 completed; M8 in_progress; M9–M28 pending).
