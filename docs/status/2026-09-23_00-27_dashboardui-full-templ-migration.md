# Status: dashboardui full-templ migration (2026-09-23 00:27)

> Point-in-time report for the session that answered **"dashboardui/ STILL doesn't use templ!?!?!"** by executing the migration end-to-end. All gates green at time of writing (see Verification Matrix). Written per docs-health status-report hygiene: append-only, annotate inline when resolved.

## Executive Summary

dashboardui rendered every page via `strings.Builder` + `fmt.Fprintf` HTML — 49 render functions, ~4,840 lines across 20 files, 155 manual `esc()` calls, zero `.templ` files. The deferral decision doc (`dashboardui/docs/planning/templ-migration-evaluation.md`) pinned a revisit threshold of ~5,000 lines; that threshold was effectively reached, the user had now asked twice (2026-07-30 and 2026-09-22), so the session executed the full templ migration instead of re-deferring.

**Result:** dashboardui now renders entirely through `a-h/templ` — 9 new `.templ` files, generated `_templ.go` committed, every strings.Builder HTML renderer deleted. Repo-wide gates: build ✅, test ✅, race ✅, lint 0 issues/15 modules ✅, coverage gate 15/15 ✅, codegen ✅, CSS bundle byte-identical ✅, flake check ✅.

## a) FULLY DONE

1. **Recon & falsifiable design** — read all 20 render files + 6,245 test lines before writing code; verified `templ.EscapeString` in a-h/templ v0.3.1020 is literally `html.EscapeString` (runtime.go:83-84), making faithful transcription byte-safe for all `strings.Contains` test assertions.
2. **Old-vs-new layout differential test** — throwaway test diffed the strings.Builder layout against the templ layout with identical adversarial inputs before committing to the approach; confirmed only HTML5-equivalent deltas.
3. **9 `.templ` files authored** — `layout.templ` (shell/sidebar/header/HTMX-partial/errorShell), `components.templ` (16 shared leaf wrappers: statCard, emptyStatePanel, statusBadge, badge, encodingBadge, buttonLink, buttonSubmit, copyButton, dataTable, rawDataTable, definitionList, listNoteCount, filterInput, paginationNav, paginationInfo, pageSizeSelector, formatLinksNav), plus one page file per panel: `overview`, `events`, `aggregates`, `projections`, `audit` (commands+queries), `dlq`, `timetravel` (shared stream-listing page), `snapshots`.
4. **All handlers rewired** — `renderPage`/`writeHTML` now take `templ.Component` and render straight to the `http.ResponseWriter`; `renderStreamIndex` takes a component factory; `renderError`/`notFoundHandler` render the library errorpage inside a new `errorShell` templ component (the last strings.Builder HTML path, killed).
5. **Dead code deleted** — `buttons.go`, `format.go`, and the string bridges in `tables.go`/`stats.go`/`badges.go`/`definitions.go`/`render.go`/`layout.go`/`export.go` plus every per-page `render*` method. Kept: pure mapping funcs, `plainHeaders`, definition-item data builders, cursor helpers. `esc()` survives only where Go-side helpers compose display markup for `defItemCopy`.
6. **Go helper layer** — `detail_items.go` (definition-item builders, payloadBytes, timelinePageState, combinedFilterSortParams), rewritten `pagination.go` (href/info/options builders).
7. **Tests updated** — 8 DOCTYPE assertions lowercased, render-func call sites migrated to `goldenRender(t, component(...))` (pagination ×9, projection rows/panel ×3, stat grid ×3, definition list, empty states ×5, layout ×2), bench file rewritten to measure the templ path against the same hand-rolled baselines.
8. **Security upgrade landed free** — templ's `href` handling sanitizes URL schemes (`templ.URL()` allowlist); `javascript:`-style values in data-derived hrefs now render inert `about:blank#blocked` instead of raw anchors. No test regressed.
9. **False-green killed: `check-codegen`** — it only iterated `adminui loginpage`; dashboardui's generated code was unverified (the session's first "PASSED" was vacuous). Fixed in flake.nix: `check-codegen` + `gen` + `build-dashboardui-css` (now scans dashboardui's own `.templ`) all cover the module.
10. **All module gates green** — build, vet, test (0 failures), test -race, golangci-lint 0 issues (after fixing gci ×3, varnamelen ×3, unused ×3, modernize ×3, perfsprint ×1, embedlit nolint), coverage 72.9% (threshold 60), core 88.8% (80).
11. **All repo gates green** — `.#build`, `.#test`, `.#lint` (0/15 modules), `.#coverage-gate` (15/15), `.#check-codegen`, `nix flake check --no-build`.
12. **Docs updated** — root AGENTS.md (dashboardui bullet + hybrid-rendering gotcha closed), dashboardui README (full rewrite of the styling/rendering/architecture sections), ROADMAP (migration marked DONE with evidence note), IMPROVEMENT_IDEAS (T23 resolved inline), CHANGELOG (full Changed entry), root TODO_LIST (WithChildren spike item superseded by PolledRegion adoption; SidebarNav criteria updated), `docs/agents-notes.md` (dated war-story entry).
13. **`go-health-dashboard` v0.9.0→v0.10.0 train-lag sweep** — 3 consumers (health, integration_test, examples/samber-do-demo) via the gate's own `bump-dep.sh` recipe; release-train now 0 unpublished/0 lag (strict-lag 0).

## b) PARTIALLY DONE

1. **`check-modules` meta-gate** — was red on the go-health-dashboard train lag (now fixed and release-train verified green); final full re-run was in flight at report time. All sub-gates it composes are individually green.
2. **History hygiene** — the auto-commit daemon shredded the migration into heuristic `chore: auto-commit` commits; the narrative is preserved in `docs/agents-notes.md` + CHANGELOG, but the commit history for the conversion itself is not reviewable as one logical change (daemon behavior is documented in AGENTS gotcha 4; concurrent sessions shared the tree — one of them landed the module-local `.golangci.yml` and the `_templ.go` gitignore fix).
3. **Golden coverage of page markup** — goldens pin the five library component outputs; page-level HTML remains `strings.Contains`-asserted (pre-existing state, unchanged by this migration). The migration relied on the 155 escaping-equivalence + differential-test argument instead of new page goldens.

## c) NOT STARTED (known follow-ups, deliberately out of scope)

1. **`htmx.PolledRegion` adoption** for the projection-health polling region (TODO_LIST has the item) — now unblocked; kept hand-rolled during migration for markup faithfulness.
2. **`display.Grid` adoption** — same unblocking story.
3. **The 5th templ-components family member** — `datastar` submodule still not required by dashboardui (unchanged, by design).
4. **bench-spike re-pin** — not needed (no bench-path change to setup), recorded here to preempt the question.

## d) TOTALLY FUCKED UP (own mistakes, all fixed same-session)

1. **First "check-codegen PASSED" was a false green** — the gate never looked at dashboardui. Discovered by reading the flake app (which also had the stale "dashboardui has NO .templ files" comment in the CSS builder). Fixed both apps; lesson: a gate that doesn't know a module exists silently passes it.
2. **Perl mass-edit broke closing parens** in 9 test call sites (`goldenRender(t, paginationNav(...` missing one `)`) — `go vet` caught them one file at a time; fixed precisely, but the right move was a single python AST-aware rewrite.
3. **Wrote `templ helper foo(...)` syntax that doesn't exist** — data builders belong in `.go` files; compile error caught it immediately.
4. **Reach-into-struct from templ components** — first draft called `d.config.PayloadRenderer` inside a component (components only see params); fixed by passing payload bytes as parameters.
5. **Python import-inserter damaged a11y_test.go** (regex targeted the last `)` in the file, splicing an import into an Errorf string) — repaired precisely; lesson: never auto-insert imports by positional hack, use `goimports`.
6. **Fought the write-guard on `pagination.go`** — two wasted write attempts ("modified since last read") because the auto-daemon touched mtimes between reads; after the second refusal, read-then-write worked. Annoying, not harmful.
7. **`golangci-lint fmt` "did nothing" scare** — it actually fixed all 3 gci findings silently (and the subsequent run also showed contextcheck gone because a concurrent session had landed the module-local config between runs). git diff showed nothing because the daemon had already committed the fmt changes. Chaos, but convergent.

## e) WHAT WE SHOULD IMPROVE (process/infra lessons)

1. **Gate discovery must be data-driven** — `check-codegen`/`gen` hardcoded module lists; new templ modules will be silently skipped again. A discovery loop over `*/` with `.templ` files would have made dashboardui's coverage automatic. (Partially mitigated: list is now 3 modules.)
2. **`templ.WithChildren` hybrid escape hatch made the hybrid path a trap** — it kept the strings.Builder layer looking viable for another sprint while the codebase grew past the migration threshold. Decision docs with numeric revisit thresholds need a gate that CHECKS the threshold.
3. **Differential-test-first worked** — the old-vs-new layout byte-diff test should be the standard opening move for any renderer migration (write it before the migration, delete after).
4. **Concurrent-session lint fixes landed without coordination** — `dashboardui/.golangci.yml` appeared mid-session (same root-cause fix the session had queued). Harmless here, but two sessions editing the same gate config is a collision risk; the CRUSH/crush-config layer has no cross-session claim mechanism.
5. **The pre-commit BuildFlow failures remain ambient noise** (license-check under ambient go 1.26.7, examples golangci) — documented gotcha 8 fallback (`--no-verify` + justification) worked, but the steps keep costing every session a failed-commit round trip.
6. **varnamelen vs shadow-prevention**: renaming `fmt :=` to `f :=` traded a lint pass for a different lint; `format` was the right name all along — skip the intermediate.

## f) NEXT 50 (prioritized, actionable)

**Immediate (this train)**
1. Re-run `nix run .#check-modules` to full green (release-train already verified; finish the meta-gate).
2. Watch CI on the pushed range (TODO_LIST P1 carries this for both repos).
3. Adopt `htmx.PolledRegion` in `overview.templ` for projection-health polling (TODO_LIST, now unblocked).
4. Adopt `display.Grid` for the stat grid (same unblocking).
5. Consider `display.Sparkline` for projection trends (ROADMAP Tier-4 candidate, now trivially adoptable).
6. Add page-level golden files for the templ pages (overview + events detail first) — the differential test was deleted with the migration; goldens would pin the new baseline permanently.
7. Extend `check-modules` docs-freshness to flag "has NO .templ files"-class claims (the stale CSS-builder comment class) — a grep gate for known-stale phrasings.
8. Make `check-codegen`/`gen` module-list-free (discover `*/` with `.templ`).
9. Re-record `docs/benchmarks/dashboardui-render-*` with the templ arms (old artifact documents the hybrid bridge).
10. Delete the now-moot `templ.WithChildren` recipe reference from dashboardui README if the guide gets archived (guide lives upstream; local reference already rewritten).

**templ follow-through**
11. Survey `loginpage` for library-component adoption (TODO_LIST item stands; the only hand-rolling UI left).
12. Unify adminui + dashboardui error-shell patterns (dashboardui now has `errorShell.templ`; adminui uses its own shell) — candidate for a shared recipe.
13. Move `streamListPageConfig`-style page configs into the templ layer as components with props structs (cosmetic).
14. Extract the shared `layout` into a recipe candidate for templ-components (the dark-sidebar shell that SidebarNav criteria gate).
15. Add `templ.URL()` wrap audit: confirm every `hx-get`-class attribute is intentionally NOT scheme-sanitized (templ only guards href/src).
16. Benchmark templ render path vs response streaming for the largest page (DLQ detail) — templ writes straight to `w`; measure whether buffering strings is ever still wins.
17. Add `-update` golden workflow doc to dashboardui CONTRIBUTING (goldens mentioned; workflow exists only in README).
18. Consider `templ.IsFirstRendering`-free static hash for SSE-partial caching (projection-health partial is polled every 10s; ETag it).

**Hygiene**
19. Squash-or-annotate the heuristic auto-commits for this migration if history rewriting is ever done (they're local; pushed range already contains similar).
20. Update `docs/guides/hybrid-templ-components-adoption.md` with a "HISTORICAL — dashboardui migrated 2026-09-22; only applies to future hybrid consumers" banner.
21. Sweep other docs for "dashboardui strings.Builder" claims (grep gate candidate): e2e guides, examples READMEs, architecture-understanding docs.
22. `docs/status/README.md` index entry for this report.
23. Annotate this report's items as they complete (inline strike + evidence).
24. Re-check `/mnt/buildcache` fill state next session (fill-drain cycle per gotcha 12).
25. Rebuild the BuildFlow binary (prefetch warned binary-vs-HEAD drift).

**Family/release**
26. Decide whether the templ migration warrants a dashboardui MINOR bump now or rides the next family train (behavioral surface unchanged; CHANGELOG carries it under Unreleased).
27. Consumer-eye verification of the migration from a throwaway module (TODO_LIST already carries the v1.19.1 equivalent; add dashboardui v-next when tagged).
28. pkg.go.dev render check for dashboardui on next tag.
29. Align go-health-dashboard v0.10.0 adoption into the next family-train notes (sweep done; train notes pending).
30. Check whether go-health-dashboard v0.10.0 adds health-check APIs dashboardui's `health/` bridge should surface.

**Dashboard product surface**
31. DLQ bulk-replay progress feedback (replay is synchronous; long replays look hung).
32. Event-detail "next/prev" keyboard navigation (matches the time-travel arrow-key pattern).
33. Snapshot compare view (two versions side-by-side) — roadmap candidate.
34. Projection restart-count alerting threshold (badge goes red at N restarts).
35. Export: CSV/JSON for projections + DLQ pages (events/commands/queries have it).
36. Per-page `<title>` uniqueness check in e2e (HTMX boost relies on it).
37. Dark-mode screenshot regression pass (the migration touched every page's DOM; whitespace deltas are inert but a visual check is cheap insurance).
38. a11y: axe-core sweep of all templ pages (a11y_test covers landmarks/labels; a full axe run is the next rung).
39. i18n readiness audit (all strings are hardcoded English; fine for observability tooling, document the stance).
40. Rate-limit recommendation for the write endpoints in README security section.

**Platform**
41. Investigate BuildFlow `type-check`/`tsc` ambient failure once more — it burns 60s of every pre-commit budget.
42. Excluding `examples/` golangci from BuildFlow pre-commit (matches the flake lint gate's exclusion).
43. nix eval-cache SQLite busy errors (seen in pre-commit log) — consider `nix.config` sqlite tuning or cache relocation.
44. BuildFlow state DB VACUUM (0.16 GB state / 1.34 GB cache flagged by its own doctor).
45. Crystallize the "differential-test-first migration" pattern into `references/lessons.md` (crush-config repo, by commit).
46. Add `templ generate --watch` guidance to the devShell docs (templ LSP post-generation events were noisy in this session's diagnostics).
47. gopls/golangci_ls "requires go >= 1.27.1" LSP warnings in templ files — document the expected-ambient-toolchain stance (gotcha 14 corollary for templ LSP).
48. Consider `exhaustruct_v5` ignore-pattern for `display.TableProps` (every bridge/restatement builds it partially).
49. Audit remaining `//nolint:modernize` nested-BaseProps comments — the templ migration deleted several carriers; keep the ones that remain accurate.
50. Next family train: fold the go-health-dashboard bump + this migration's CHANGELOG entries into one release-notes pass.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Release cadence:** should the templ migration ship as a dashboardui-only `v4.13.0` tag now (library-internal change, zero public API delta, CHANGELOG ready), or ride the next coordinated family train? (The module's public surface — `New`/`FromBundle`/`Mount`/`Handler`/`Middleware`/`Config` — is byte-identical; only rendered HTML whitespace differs.)
2. **HTML delta sign-off:** the three cosmetic deltas (lowercase doctype, void-element slash removal, inter-element whitespace) are HTML5-equivalent and browser-invisible — do you want a rendered-DOM screenshot diff (Playwright, both builds) as formal sign-off evidence, or is the test-suite + differential-test argument sufficient to close it?
3. **`contextcheck` stance:** the new `dashboardui/.golangci.yml` (landed by the concurrent session) disables contextcheck module-wide with a documented rationale. Do you want the same treatment formalized for adminui/loginpage (their configs live at the root exclusions), or is root-level exclusion with a templ-aware comment the preferred long-term shape?

---

*Session evidence: module `go test -race` ok; `nix run .#lint` 0 issues/15 modules; `.#coverage-gate` PASSED (dashboardui 72.9%/60); `.#check-codegen` PASSED (covering dashboardui); `.#build`+`.#test` OK; `check-release-train --strict-lag 0` 0/0/0; CSS bundle byte-identical (76,984 bytes, canaries OK).*
