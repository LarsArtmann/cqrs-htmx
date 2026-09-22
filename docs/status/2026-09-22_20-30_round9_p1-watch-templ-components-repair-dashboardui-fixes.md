# Round 9 — P1 CI watch, templ-components red-master repair (root cause), dashboardui dark-mode + nil-request fixes, a11y batch, WithChildren spike

**Date:** 2026-09-22, report written 20:30 CEST
**Session scope:** cqrs-htmx TODO_LIST full-execution pass (P1 watch + P2/P3 sweep), interrupted for this report.
**Repos touched:** `cqrs-htmx` (5 substantive commits), `templ-components` (2 root-cause fixes, later absorbed into a v1.19.2 release by a concurrent session).

---

## 0. Headline

1. **templ-components red master (4 CI lanes) diagnosed to ONE root cause + one stale golden, both fixed.** A modernize fixer hand-edited generated `website/internal/pages/base_templ.go` from `encoding/json` to `encoding/json/v2`; the flipped JSON-LD key order broke the website goldens, the CSP script-hash guard (Visual Regression), the templ-drift lint check, and the codegen verify step — four red jobs from one line. Separately, `sales.golden` predated the v1.19.1 CopyButton label fix (the pre-existing Website red on 45a2923b). Both fixed and committed (45dd22a8, f9d3788e).
2. **While I worked, a concurrent session ran a full templ-components v1.19.2 release** (NoThemeScript knob): release commit 8a8c9700, replace-restore d6d22b75, 7× v1.19.2 tags PUSHED, master synced. My fixes rode along in that push.
3. **cqrs-htmx master is NOW RED in CI** (9c12ff40, 18:02Z, release-train gate exit 3) — immediately after the templ-components v1.19.2 adoption landed here (00c83a56, 24 files of go.mod/go.sum bumps, CI-green at 17:25Z). The strict train gate now demands a cqrs-htmx family train cut. **This is the top open item, undiagnosed beyond exit-code identity.**
4. **dashboardui had a red-in-waiting on local master:** `renderError(w, nil, …)` segfaulted (nil guard lost when the errorpage shell landed, post-CI daemon commit 00184078). Fixed (5d564839), plus a genuine dark-mode bug: the missing gray-800/900 literal re-pin made BadgeNeutral render a near-white chip with contrast-failing text in dark mode. Fixed + bundle rebuilt (5d564839).
5. **A11y + spike batch landed** (d5a55b6e): pagination label is now a live region; `templ.WithChildren` escape hatch empirically proven in the hybrid path; stale README exclusion claim corrected.

---

## a) FULLY DONE

| # | Item | Evidence |
| - | ---- | -------- |
| 1 | templ-components root-cause diagnosis: `encoding/json/v2` in generated file → 4 red lanes (Lint drift check, Build&Test codegen, Website JSON-LD goldens, Visual CSP guard) | CI logs 35726718266/35726718722; local repro of all four |
| 2 | Fix: regenerated `base_templ.go` from source (root-level regen = canonical form for that repo; module-dir regen emits bare `FileName:` and was rejected) | commit 45dd22a8; post-regen diff = exactly 1 line |
| 3 | Fix: re-recorded stale `sales.golden` (predates v1.19.1 CopyButton `text-gray-700 dark:text-gray-200` span) — the pre-existing Website red on 45a2923b | commit f9d3788e; website golden package green |
| 4 | P1 verified: pkg.go.dev renders v1.19.1 (README, directories, published date) | fetched `@v1.19.1` page |
| 5 | P1 verified: cqrs-htmx CI green through f51f354b / 00c83a56 (the pushed range of the round-8 TODO item) | `gh run list` (CI success 17:25Z) |
| 6 | bench-spike refusal #10 recorded mechanically by the gate (load 31.29 ≥ BENCH_MAX_LOAD 8) | `nix run .#bench-spike` output (TODO_LIST update still pending, see c) |
| 7 | Gate-integrity spot checks (a)–(e): ALL PASS — (a) verify-tag fixtures exempt via `*/testdata/*` in check-version-drift.sh:50; (b) e2e webServers GOWORK pin intact (playwright.config.ts:10,65,81); (c) pre-push hook is repo-root-scoped, both-repos-pending is safe; (d) check-vcs-cache + self-test correctly placed in ci.yml module-gates job with honest trivially-green comment; (e) go.work.sum absorbed v1.19.1 (31 hashes), drift gate green on 805 requires | greps + `check-version-drift.sh --strict` run |
| 8 | FEATURES.md: new "Repository Quality Gates" section with the 3 unrecorded round-8 gates (VCS-cache health, dep-budgets self-test, pre-push CI-parity hook) | commit bcba05ec (daemon-raced), content verified in HEAD |
| 9 | Dark-mode surface pin re-check: adminui pin PRESENT and correct post-v1.19.1 (`html.dark` literal re-pin, tailwind.css:91-92) | compiled bundle contains `--color-gray-800:#1f2937` in dark `:root` |
| 10 | dashboardui missing pin FOUND + FIXED: `@media (prefers-color-scheme: dark)` literal re-pin added to `tailwind.css`, bundle rebuilt via `nix run .#build-dashboardui-css` (canaries OK, pin verified at offset 41 inside the dark block) | commit 5d564839; BadgeNeutral `dark:bg-gray-800` now renders literal #1f2937 |
| 11 | dashboardui nil-request segfault FIXED: `renderError` HTMX check derefs nil request (guard lost in errorpage adoption); doc comment already promised nil-safety; test contract restored | commit 5d564839; `TestRenderError_NilRequestStillWrites` green; full module green |
| 12 | DLQ count-note a11y verdict: CORRECT — `role="status"` (implicit polite live region) + `aria-label="Dead letter count"`; HTMX-inserted live-region content is announced by conforming SRs; count-as-blast-radius intent holds | library list_note.templ:62-68 + goldens |
| 13 | Count-notice consistency sweep: paginated lists rendered a plain SR-silent span vs DLQ's live region → added `role="status"` to `renderPaginationInfo` (both branches); no test/e2e pins on the markup, CSS-only selector unchanged | commit d5a55b6e; module tests green |
| 14 | CopyButton contrast fleet check: CONSISTENT — library v1.19.1 span carries its own colors (no inheritance from re-colored tables); dashboardui adds deliberate `var(--text)` override (WCAG comment, layout.go:288); adminui uses fixed library default; loginpage doesn't use CopyButton | greps |
| 15 | Post-adoption exclusion-claims sweep: ListNote-count claim already lifted (README correct); Pagination exclusion still valid (ListNote has no X–Y range variant); SidebarNav criteria (2)/(3) still gating; **Grid/PolledRegion claim was STALE** (ignored v1.19.1's `templ.WithChildren` escape hatch) → README updated with dated UPDATE block | commit d5a55b6e |
| 16 | `templ.WithChildren` SPIKE: PASSING test proves children-slot population works in dashboardui's strings.Builder path (PolledRegion renders child markup + polling attrs) — adoption of PolledRegion/Grid on the projection-health polling region is unblocked | `dashboardui/render_polled_region_spike_test.go` (d5a55b6e) |
| 17 | templ-components v1.19.1 consumer-eye half-check: proxy propagation + pkg.go.dev render confirmed (throwaway-module compile remains open) | see c) |

## b) PARTIALLY DONE

| # | Item | State |
| - | ---- | ----- |
| 1 | **P1 "watch CI, fix forward same-day"** — templ-components leg DONE (root cause fixed; v1.19.2 released+pushed by concurrent session; their tip d13c1445's CI/Website verdict NOT yet observed at report time). cqrs-htmx leg OPEN: new red at 9c12ff40 (release-train exit 3) after the v1.19.2 adoption — family train likely needs cutting, NOT yet diagnosed or fixed. |
| 2 | **P1 bench-spike idle re-run** — 10th documented refusal (load 31.29 ≥ 8); the TODO_LIST annotation of refusal #10 is not yet written. Structural question (automate-or-retire, OQ16) untouched — owner decision. |
| 3 | **P3 consumer-eye verification v1.19.1** — pkg.go.dev render verified; the throwaway-module `go get` + minimal-consumer compile (the actual "consumer eye") NOT done. Superseded in part by the concurrent session's v1.19.2 release: the verification target is now v1.19.2, not v1.19.1. |
| 4 | **templ.WithChildren spike → adoption** — mechanism proven by test; production adoption on the projection-health region NOT done (needs `Trigger: "every 10s, refresh"` to preserve the SSE-driven refresh trigger + golden/e2e churn). Same escape hatch un-surveyed for adminui/loginpage. |
| 5 | **Session hygiene** — my two templ-components fixes were committed by the auto-daemon with heuristic messages (45dd22a8, f9d3788e); I amended only my cqrs-htmx dashboardui commit (4171446d → 5d564839) to a proper message. The templ-components history carries the fix content but not the explanation. |

## c) NOT STARTED

| # | Item (TODO_LIST source) |
| - | ----------------------- |
| 1 | **Diagnose + fix the NEW cqrs-htmx red** (9c12ff40, release-train exit 3 post-v1.19.2-adoption — almost certainly requires a wave-ordered cqrs-htmx family train: verify-tag.sh per changed module, --refresh-cache for the fresh templ-components tags) |
| 2 | e2e DLQ count-notice assertion in `e2e/tests/dashboard.spec.ts` + full Playwright run (57/57 precedent) + human-grade contrast/dark-mode look at the note |
| 3 | P2 CSS-bundle sanity guard (committed bundles = canonical MINIFIED builder output: 1 line / byte range / canaries) + staged-state producer hunt — atomic per gate checklist |
| 4 | P3 exact-class-set CSS bundle drift gate (sorted-selector-set diff, exit 1 on delta) — atomic per gate checklist |
| 5 | P3 ListNote render bench in `dashboardui/render_bench_test.go` (N16 pattern) |
| 6 | P3 loginpage adoption pass (`recipes.AuthLayout`, `forms.Input`/`Form`, `feedback.Alert`) |
| 7 | P3 release-train gate UX: print copy-pasteable per-module `go get` fix recipe on train-lag failure (this would have helped LITERALLY TODAY on 9c12ff40) |
| 8 | P3 small hardening chores: (a) wire check-vcs-cache.sh into prewarm-gocache.sh; (b) pin scratch caches in test-verify-tag + test-check-docs-* self-tests; (c) README contributor note about the pre-push gate; (d) archive-or-finish `docs/status/2026-09-21_18-20_round6-*.md.new` |
| 9 | P3 runbook: recovering from `fatal: cannot lock ref 'HEAD'` daemon races |
| 10 | P3 concurrency-safety helper scripts (`scripts/lib/preflight-tree-check.sh`, `scripts/lib/wait-tree-quiet.sh`) |
| 11 | P3 agents-notes narrative: 2026-09-22 concurrent-session incidents (near-double-bump, false "gutted CSS" alarm, git town continue race — and today's v1.19.2 abort/re-cut + my tidy interference, see d2) |
| 12 | TODO_LIST + CHANGELOG updates for this session (bench refusal #10; a11y verdicts; sweep results; dark-mode pin fix; spike; FEATURES rows) |
| 13 | Wire remaining check-* apps into CI (only check-cqrs-lint outstanding; blocked on Go-installable distribution) — unchanged |
| 14 | SidebarNav revisit criteria (criteria-based, not yet triggered; criterion (1) now satisfiable per the spike — inputs updated in README) |
| 15 | DataStar Tier 4 (demand-gated), BuildFlow go-version-auto-configure re-enable (upstream-blocked), usermgmt V007 migration (metaengine-gated), ProjectionLayer v5 removal (v5-window), examples/datastar-demo owner decision, cqrs-lint CI gate (Nix-only block) — all unchanged from TODO_LIST |

## d) TOTALLY FUCKED UP

| # | Incident | Damage | Lesson |
| - | -------- | ------ | ------ |
| 1 | **Ran tree-mutating verification (`go mod tidy` inside ci-repro) inside templ-components while a concurrent session had an in-flight v1.19.2 release train.** My tidy reverted their version pins in the worktree mid-verify. | None visible: they aborted the attempt ("release attempt aborted mid-verify", 376f1174), re-cut cleanly (8a8c9700), and pushed. But I cannot prove my tidy didn't CONTRIBUTE to their abort, and I spent several poll cycles guessing at their intent before standing down. | Before ANY repo-mutating step in a shared tree, check `git log` recency (their commits were minutes old) and treat an active release train as a no-go zone. The preflight-tree-check helper (c10) would have caught this mechanically. |
| 2 | **Lost the commit-message race twice.** My templ-components commit was blocked by the (pre-existing, foreign) scaffolder drift in the pre-commit hook, and the daemon committed my staged fixes as `chore: auto-commit` (45dd22a8, f9d3788e). My cqrs-htmx commit attempt hit a buildflow step-failure and the daemon again won (4171446d); I amended that one to a real message. | History carries two heuristic messages for substantive fixes in templ-components; content intact. | Commit at the moment the work is verified, not after the next verification pass; on hook-block, re-evaluate and re-commit quickly rather than investigating the foreign drift inline. |
| 3 | **Committed the dashboardui CSS batch before running the module's full test suite** — the nil-request segfault (pre-existing, but in the module I was shipping) surfaced only on the broader run afterward. | No damage (fixed in the same session), but the commit boundary was wrong: CSS fix + test fix should have been one verified commit, not two. | Run the touched module's full suite BEFORE `git commit`, not after. |
| 4 | **Declared "cqrs-htmx CI green through HEAD" at ~16:20 and moved on** — by 18:02Z the train went red on 9c12ff40 (a dependabot actions-bump landing the v1.19.2-adoption lag into the strict gate). The P1 item is a watch, and I stopped watching. | Master is red RIGHT NOW; undiagnosed beyond exit code. | A "watch CI" task needs a terminal condition (green + no pending range), not a point-in-time check. |
| 5 | Minor: two edit-tool rejections (FEATURES.md, pagination.go) for editing before viewing — cost two round trips, zero damage. | — | Keep the read-before-edit discipline even when the content was already grepped. |

## e) WHAT WE SHOULD IMPROVE

1. **Concurrent-session protocol is still tribal knowledge.** Today it bit twice (tidy interference d1; daemon message races d2). The planned `scripts/lib/preflight-tree-check.sh` + `wait-tree-quiet.sh` (c10) would mechanize the checks I ran ad hoc — promote them from P3 to first thing next session.
2. **The release-train gate should print its fix recipe** (c7). Today's exit-3 tells a human (and every agent) THAT it failed, not that ~13 modules need `go get github.com/larsartmann/templ-components@v1.19.2`-style alignment + a wave-ordered train. This is now the highest-leverage 30-minute fix in the repo.
3. **"Green locally / green earlier" is not a verdict.** CI-parity must be witnessed at push time (M03 ritual exists in templ-components; the pre-push hook now enforces the gates in cqrs-htmx — but nothing re-runs when a concurrent session pushes a dependabot commit behind you).
4. **The CSS-bundle oscillation class keeps firing across repos** (cqrs-htmx ~13:05 incident; templ-components eb939a19 minify-vs-pretty flip today). The P2 guard (c3) is designed for exactly this and remains unbuilt; today added a third data point.
5. **Generated-file fidelity needs a fleet-wide norm:** a fixer editing `*_templ.go` (or `*_go.go`) out-of-band produced today's four-lane red. A repo-local guard that diffs generated files against fresh generation in pre-commit (cqrs-htmx has `check-codegen`; templ-components has the drift lint — both green today only because I reverted the file) should be verified to actually CATCH the fixer class, not just the manual-edit class.
6. **Status-report hygiene:** this session produced several one-line verdicts (a11y, sweeps) that belong in TODO_LIST annotations NOW (c12) — unwritten verdicts rot into re-research.
7. **pkg.go.dev + proxy propagation checks** are still manual fetches; a tiny script (`scripts/check-pkg-go-dev.sh <module> <version>`) would make c3/P3 consumer-eye checks mechanical.

## f) NEXT (ordered, ≤50)

1. **Diagnose the 9c12ff40 red** (run `scripts/check-release-train.sh --strict-lag 0 --refresh-cache` locally; confirm exit-3 = lag on the 13 bumped modules).
2. **Cut the cqrs-htmx family train** wave-ordered (verify-tag.sh per changed module; leaf-most first), or widen `--strict-lag` deliberately if the train is intentionally deferred — never leave it loose (gate comment).
3. Re-run `#check-modules` + pre-push gate; push; **watch CI to green** (terminal condition).
4. Annotate TODO_LIST: refusal #10, a11y verdicts (DLQ ✓, pagination fixed), sweep results, pin fix, spike, exclusion-claim update.
5. Append CHANGELOG entries for: dashboardui dark-mode pin fix, renderError nil-guard fix, pagination live region, WithChildren spike, FEATURES gate rows.
6. Write the preflight-tree-check + wait-tree-quiet helper scripts (with self-tests per gate checklist) — then USE them.
7. Write the HEAD-lock daemon-race runbook (docs/runbooks/).
8. Extend the agents-notes narrative with today's incidents (v1.19.2 abort/re-cut; tidy interference; four-lane root cause; 9c12ff40).
9. e2e DLQ count-notice assertion + full Playwright run + contrast/dark look.
10. Build the P2 CSS-bundle sanity guard (minified-canonical: line count, byte range, canaries) — checker + fixture self-test + flake app + check-modules stage + CI + README, atomic.
11. Build the exact-class-set drift gate on top of 10's plumbing (sorted selector diff).
12. Hunt the staged-state producer of the ~13:05 near-empty-bundle incident (Part 1 of the P2 item) once the guard exists to catch it.
13. ListNote render bench (N16 pattern) in dashboardui.
14. Consumer-eye verification — retarget to **v1.19.2** (throwaway module, go get, minimal consumer, pkg.go.dev render).
15. templ.WithChildren follow-up survey: adminui + loginpage adoption candidates for the same escape hatch.
16. Production adoption decision: PolledRegion on the projection-health region (needs `Trigger: "every 10s, refresh"`, golden + e2e updates) — go/no-go after 9.
17. loginpage adoption pass (AuthLayout, forms.Input/Form, feedback.Alert).
18. Release-train gate UX: print the per-module fix recipe (see e2).
19. Small hardening chores (a)–(d) from c8.
20. check-pkg-go-dev.sh helper script (e7).
21. Re-check templ-components d13c1445 CI/Website verdict (their tip may still be red for unrelated reasons).
22. Verify the codegen guards actually catch the fixer class (e5): simulate an out-of-band generated-file edit locally, confirm pre-commit/CI rejects.
23. Gate-integrity follow-up: confirm the drift-gate fixture exemption covers the NEW verify-tag fixtures if any were added by the v1.19.2 train.
24. Re-run `nix run .#coverage-gate` after the family train (15-module gate).
25. Re-run `nix run .#check-cqrs-lint` (flake-app-only rule) after the train.
26. Count-notice consistency: decide whether audit/commands/queries pages should ALSO carry total-count notices (sweep found the X–Y label; totals exist for some lists via TotalCount) — small design call.
27. Screen-reader check part 2 for the DLQ note: contrast + dark-mode visual pass rides with the e2e run (9).
28. CopyButton: the dashboardui override (`var(--text)`) could move upstream as a prop/variant ask — record in templ-components TODO (upstream ask pattern).
29. FEATURES.md: verify the 3 new gate rows render correctly in the docs-links/tables gates (`check-docs-*`).
30. AGENTS.md: add the "no tree-mutating verification during foreign release trains" lesson to the concurrent-session gotcha (gotcha 4 area).
31. AGENTS.md templ-components section: add "generated-file fidelity" gotcha (fixers must never edit `*_templ.go`/`*_go.go`; regenerate instead).
32. Consider a `BENCH_MAX_LOAD`-aware CI-free bench scheduler note (OQ16 input): 10 refusals in 12 days — propose retire-or-relocate in ROADMAP.
33. Verify v1.19.2 propagated to pkg.go.dev (fleet eye on the NEW train).
34. Check whether the daemon-committed 24-file bump (00c83a56) and dependabot 9c12ff40 were owner-intended — the train cut in 2 should reference them.
35. Update TODO_LIST P1 items to reflect: templ-components leg done, cqrs-htmx leg = train cut.
36. Roadmap: record the ListNote X–Y range variant upstream ask (the pagination-info exclusion survives only because the library lacks it).
37. Survey `display.Grid` adoption candidates in adminui (escape hatch applies there too).
38. Run `nix fmt` / treefmt at phase end (session touched CSS, Go, MD).
39. Run golangci-lint on dashboardui (module-level) after the train cut to keep 0-issues claim honest.
40. Re-verify `go.work.sum` absorbed the v1.19.2 hashes after the adoption commit (repeat of spot check e for the new pins).
41. Confirm the v1.19.2 NoThemeScript knob needs no dashboardui/adminui adoption note (dark-mode FOUC strategy unchanged here).
42. Check the orphan `dashboardui/styles.css` buildflow artifact churn rule still holds (it was modified mid-session; verify daemon committed the minified-consistent form, not a hook artifact).
43. Wire the spike test's verdict into the hybrid-adoption guide (docs/guides/hybrid-templ-components-adoption.md) — one paragraph + test pointer.
44. Add the "watch CI to a terminal condition" rule to the P1 item text in TODO_LIST (process fix).
45. Confirm no other module's CI lanes silently skipped in the 17:25Z green (e.g. did the 24-file bump trigger e2e?).
46. Consider dependabot grouping config so actions-bumps stop landing solo reds mid-train (9c12ff40 was a single-action bump).
47. Archive-or-finish the stray round6 `.new` status file (duplicate of c8d, listed separately because it's 2 minutes).
48. Spot-check the amended 5d564839 message rendered correctly in git log (amend raced the daemon once already).
49. Re-run the docs-links gate after this report lands (status reports are linked from docs/status/README.md conventions).
50. After the train: schedule a quiet-window bench-spike attempt (the only remaining v4.9.0-appkit verification).

## g) QUESTIONS (cannot self-answer)

1. **The 9c12ff40 train-lag red: cut the cqrs-htmx family train NOW (wave-ordered verify-tag per changed module, ~30-45 min, several tags) — or is the v1.19.2 adoption meant to ride the NEXT scheduled train, in which case should I temporarily widen `--strict-lag` with justification (documented fallback) until that train?** The gate comment says raise the budget "only deliberately (mid-sweep), never leave it loose" — I can't tell which intent applies.
2. **templ-components v1.19.2 is released+pushed by the concurrent session, but I never witnessed their CI/Website verdict on d13c1445, and their master also carries my two root-cause fixes. Do you want me to independently verify their tip's CI (and pkg.go.dev propagation for v1.19.2), or is the other session owning watch-to-green for that repo?** (Avoids double-pushing/repair races like today's.)
3. **bench-spike has now 10 mechanical refusals in 12 days (load 31-825 vs limit 8). OQ16 ("automate or retire") is yours: should I (a) keep documenting refusals, (b) retire the gate into an on-demand app with the baseline re-pinned from a scheduled quiet window, or (c) move the baseline pinning into a CI job on a self-hosted idle runner?** I can implement any of the three; the choice is a policy call on what the baseline is FOR.

---

*Point-in-time snapshot per docs/status convention; append-only. Evidence: commits 45dd22a8/f9d3788e/5d564839/d5a55b6e (templ-components fixes via daemon-heuristic messages), CI runs 35726718266, 35726718722, 35760605422 (green), 35764541983 (red, release-train exit 3).*
