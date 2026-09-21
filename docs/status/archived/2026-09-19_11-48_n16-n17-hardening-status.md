# Run-3 Status — N16+N17 Hardening Pass (verify · ship · see · harden)

**Recorded:** 2026-09-19 11:48 · **Session scope:** N16 (benches + ireturn) and N17 (feature follow-ups) from `docs/planning/2026-09-18_16-09_run3-verify-ship-see-harden.md`, plus one live toolchain incident.
**Tree at recording:** clean, master `ab5a2b6d`, go directive 1.26.7 (restored), dashboardui build/vet/test/lint/fmt all green, root lint+test green.
**Scope note:** per instruction this report covers only this session's run and what it surfaced — not a whole-repo re-audit. Format is `.md` per explicit user instruction (status-report skill default is styled HTML; divergence flagged, not propagated back into the skill).

> **ANNOTATED 2026-09-20** (docs-health sweep): the N16/N17 hardening shipped in the v4.11.0 train.
> - **§a (fully done):** ID-less evidence table — self-declared done with per-row evidence; left unstruck (already clear).
> - **§b (bullets):** mostly open → `TODO_LIST.md` (screenshot regen vs HEAD, axe re-run, e2e re-run, 243 auto-fix audit, `.gitattributes` review, SSE e2e deferral, systemadapter unblock, toolchain policy — the last is now resolved by the 1.27.1 bump).
> - **§c (bullets):** HARVEST is this sweep; v1.19 prep + theme toggle + toolchain are routed (`TODO_LIST.md`); the "~40 Run-3 §f items" were routed by the Run-3 report's own annotation.
> - **§d / §e:** retrospective incidents + improvement list — historical record; the toolchain-class fix landed (1.27.1 bump).
> - **§f:** struck rows confirmed done; open rows (f1/f3–f10/f16–f20/f22/f24–f36/f38–f50) are routed to `TODO_LIST.md`/`ROADMAP.md` (auto-fix audit, axe/e2e/screenshot re-runs, vacuous sweep, favicon, aria-live, slider keyboard, mobile wrap, upstream filings, README, filterInput doc, hook-incident note, CI wiring, render-bench app, fuzz, SSE spec, JS extraction, toolchain policy, hook hygiene, templ-conversion, theme toggle, benchstat gate).
> - **§g:** q1 resolved (1.27.1 coordinated bump); q2/q3 resolved by the train (sibling reconciled; v4.11.0 tagged).

---

## a) FULLY DONE

| Item | Evidence |
| --- | --- |
| **N16.6** — `templ\.Component` added to root `.golangci.yml` ireturn allow; both `//nolint:ireturn` in `dashboardui/buttons.go` deleted | `5c119e12` (daemon); root + dashboardui lint 0 via CLI, not LSP |
| **N16.1–4** — 4 new bench families in `render_bench_test.go`: Button (link+submit), EmptyState, DefinitionList, Table raw-body vs data-row; hand-rolled baselines extracted **verbatim** from git history (`6294d73e~1`, `21c13e22`) with the escaping the real sites performed | `0c5fa7f5` (daemon, content verified) |
| **N16.5** — 5× `-benchmem` sweep run; per-bench medians computed and written into `docs/benchmarks/dashboardui-render-2026-09-19.md` with honest-baseline notes; raw output pinned as `dashboardui-render-2026-09-19-sweep.txt` | Key findings: DL structure swap ≈1.3x (CopyButton dominates both sides); Table raw-body ≈4.3x cheaper than typed data-rows at 10 rows; StatCard ratio reproduced (~8x) across load regimes |
| **N16.7 / N17.7** — full hermetic verify cycles (build/vet/test/golangci/gofmt) for dashboardui + root; final state re-verified after every incident | multiple `/tmp` logs; final sweep clean |
| **N17.1–2** — `htmx.PolledRegion` evaluated and **rejected with reasons**: children-based (`templ.GetChildren`) → empty in the hybrid standalone render, same class as `display.Grid`. Decision documented in `docs/guides/hybrid-templ-components-adoption.md`, `dashboardui/README.md`, `AGENTS.md`, incl. the upstream-ask (props-based body slot) | guide section "Pitfalls #1" extended |
| **N17.3–4** — event filter bar swapped to `forms.Input` via new `filterInput` helper; explicit DOM ids (`filter-type`, `filter-stream-type`, `filter-stream-id`) preserved; hx-get wiring untouched; new `TestA11y_FilterInputsKeepLabelPairing` pins label[for]↔input[id] + hx attributes; Tailwind bundle rebuilt (`nix run .#build-dashboardui-css`, canaries OK) and new utility classes confirmed present in the compiled artifact | `f5c39764`, `ae70ef61`, `b58bffd2` |
| **N17.5–6** — **real CSP bug found and fixed**: time-travel slider's inline `onchange`/`oninput` (blocked under nonce CSP → slider dead for CSP-enforcing consumers, and invisible to `TestCSP_NoInlineEventHandlers`, which swept only listing routes = vacuous pass). Replaced with `data-nav-base`/`data-slider-display` + CSP-safe external listeners; static + dynamic `aria-valuetext`; `:focus-visible` ring. CSP test hardened: seeds a 3-event stream, sweeps the slider detail page, asserts the slider rendered, `oninput=` added to the blacklist — **mutation-verified** (fails with the attribute re-added, passes without) | `1c25be3e`, `ae70ef61` |
| **Browser verification** — throwaway module in `/tmp` with a local replace to the working dashboardui; Playwright computed-style + behavior check in light AND dark: filter input bg/color/border theme-consistent (no white-box), label contrast correct, form hx wiring intact, slider has zero inline handlers, and a synthetic `input` event updates the display + aria-valuetext live | session log; scaffold deleted after use |
| **CHANGELOG** — `dashboardui/CHANGELOG.md` gained an `Unreleased` section: the CSP fix as *Fixed*, forms.Input adoption + benches + ireturn change as *Added* | `5b72cb73` (daemon) |
| **Toolchain incident resolved (twice)** — root `go.mod` re-raised to 1.27.1 by the sibling toolchain, absorbed by the daemon; then the pre-commit hook's own gomod-check repair **re-applied it mid-run and the hook failed on the mismatch it created**. Restored to 1.26.7 both times; final commit `ab5a2b6d` uses the documented `--no-verify` fallback with full justification, and HEAD was re-verified after (go.work ↔ go.mod agree at 1.26.7, dashboardui tests + root vet green) | `ab5a2b6d`, `ccf5d0aa` |
| **Train-lag alignment landed en passant** — examples/*, integration_test, setup dashboardui requires bumped v4.9.0 → published v4.10.1 (hook repair + daemon); each affected module verified **hermetically** (GOWORK=off): root, setup, setup-demo, dashboard-demo, integration_test, e2e/server | `5b72cb73`, `ab5a2b6d` |
| Perfsprint/golines/gci findings in my own new code fixed same-session (`strconv.Itoa`, line rewraps); gofmt clean tree-wide at the end | lint logs |

## b) PARTIALLY DONE

- **Visual regression coverage for the new markup.** Computed-style + behavior checks done in a live browser, but the 27-PNG golden screenshot set (N5) was **not regenerated** — it still documents the published v4.10.1 rendering, not the filter-bar/slider changes. The e2e harness consumes the published module, so verifying local changes requires a temporary replace (I used one ad hoc; it is not wired into the harness).
- **Axe sweep not re-run.** N9's 9-page axe gate exists, but the DOM changes (FormFieldWrapper wrappers in the filter bar, slider attributes) were validated only by unit-level a11y contract tests, not by the actual axe run.
- **e2e dashboard specs not re-run** against the new markup (`dashboard.spec.ts` targets the published server; a local-replace run was not set up).
- **Unreviewed auto-fix surface.** The failed hook run reported "cqrs-lint: 243 fixed" and "gomod-check: 5 fixed" repo-wide before failing. The staged result was committed by the daemon across its commits and `git status` is clean, but **nobody has diffed what those 243 fixes touched** — they may include files the sibling session owns (adminui work was mid-flight). Risk assessed as low (cqrs-lint fixes are suppression-comment normalizations) but unverified.
- **`.gitattributes` gained `*.png binary`** (landed inside a daemon commit, unattributed, probably sibling or BuildFlow). Harmless and kept, but unreviewed-by-owner.
- **N8 SSE e2e deferral unchanged** — event.Bus remains too rich to stub; documented in the spec header, no movement this session.
- **Systemadapter gate still red** — externally blocked on the sibling's unpublished `go-cqrs-lite/claiming/v4.0.0`; unfixable in this repo.
- **Toolchain policy** — state restored, but the *decision* (GOTOOLCHAIN=go1.26.7 pin fleet-wide vs coordinated 27-module bump to 1.27.1) remains open; until then every session risks re-fighting this.

## c) NOT STARTED (this session; known backlog items I deliberately did not touch)

- The remaining ~40 items of the Run-3 status report §f list (mobile title alignment, detail-page captures, go-etag v0.4.0 train, catalog v4.4.0 train, upstream filings, CI wiring for the new gates, …).
- HARVEST of this report's §f into `TODO_LIST.md` (belongs to a docs-health run).
- v1.19 templ-components prep (sharp-cards visual change + CSS rebuild + screenshot pass).
- Theme-toggle decision (prefers-color-scheme vs manual toggle).
- The 3 open user questions from the morning status report (toolchain policy, release authority, screenshot sign-off).

## d) TOTALLY FUCKED UP

Nothing in shipped code — dashboardui is green at every gate and the CSP fix makes the module *more* correct for consumers. But three process failures this session, honestly ranked:

1. **Lost the commit race again, at scale.** The "commit at phase boundaries" rule exists *because* of seven-plus prior losses, and I still batched. Result: every one of this session's code changes (ireturn, benches, filter swap, slider fix, test hardening) landed inside `chore: auto-commit` heuristic commits with no narrative. The content was verified intact each time, but the history is mush again — `git log` tells none of the story. Two narrative commits (`ab5a2b6d`, the CHANGELOG-less state) were all I salvaged.
2. **The hook self-sabotage loop** — I ran commits under the ambient (sibling-polluted) toolchain, so BuildFlow's gomod-check repair re-bumped `go.mod` to 1.27.1 *during* the hook, and golangci-lint then failed in 17 modules on the mismatch the repair itself created. I burned two ~90s hook runs discovering this. Mitigation existed and I didn't use it from the start: prefix commits with `GOTOOLCHAIN=go1.26.7` (or check `head -4 go.mod` immediately before and after).
3. **Test written against assumptions instead of observed behavior.** `TestA11y_FilterInputsKeepLabelPairing` failed twice before passing: first on a hardcoded `hx-get="/dashboard/events"` (BasePath is config-dependent), then because an empty store renders the events page *without* the filter bar at all. A 30-second probe of the rendered markup would have written the test right the first time — the same lesson as the vacuous-sweep findings, re-learned at micro scale.

## e) WHAT WE SHOULD IMPROVE

1. **Pin the toolchain for all local gates** until the policy decision lands: `GOTOOLCHAIN=go1.26.7` in `scripts/lib/go-cache-env.sh` would neutralize the whole tug-of-war class for every script that sources it. This is the highest-leverage one-line fix available.
2. **Commit after every verified micro-task**, not after phases — the daemon polls faster than any verification tail, and this repo's own AGENTS.md says exactly this.
3. **Make the e2e/screenshot harness able to run against a local module replace** (env-var-gated replace or a `make`-style target), so visual regression doesn't lag code changes by a release.
4. **Probe rendered markup before writing markup assertions** (render → print → assert), especially in this builder-style codebase where composition decides what's on the page.
5. **Never trust a green check that can pass vacuously** — the CSP sweep now asserts the target page actually rendered (the slider must be present). Audit the remaining route-list sweeps for the same hole.
6. **Audit or fence BuildFlow auto-fix blasts** — 243 mechanical fixes committed without review is the kind of change a human should at least skim; a post-hook `git diff --stat` gate before daemon pickup would help.
7. **Bench recordings need a quiet-machine gate** (or at minimum a recorded load counter) — two runs from the same day differ 3x; the md now says "ratios only", but recording absolute numbers under unknown load invites misreading.
8. **Daemon attribution** — release-playbook §6 documents the pattern; the follow-through (annotating which heuristic commit carried which logical change) still has to be done by hand each session. A tiny `docs/status/daemon-map` convention would make future archaeology cheaper.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Brainstorm, ranked roughly by impact; the top ~12 are real TODO_LIST candidates, the tail is ROADMAP fuel.

**Correctness / security**
1. Audit the 243 cqrs-lint auto-fixes that landed unreviewed (diff the daemon commits; confirm no behavior change, confirm no sibling-owned files mangled).
2. ~~Fleet-wide `GOTOOLCHAIN=go1.26.7` in `go-cache-env.sh` (kills the tug-of-war for every sourced gate) — pending the user's policy call.~~ done (1.27.1 coordinated bump landed 2026-09-19)
3. Run the axe sweep (9 pages × light/dark) against the new filter-bar/slider DOM; fix any contrast/focus findings.
4. Re-run `dashboard.spec.ts` (7 specs) against a local-replace server; confirm sort flip / NotFound / badge specs still pass.
5. Regenerate the 27 golden PNGs against HEAD (filter bar + slider changed; also captures the new focus ring).
6. Sweep remaining route-list tests for vacuous-pass potential (assert each target page renders a signature element, like the CSP test now does).
7. File/fix the missing favicon route (every page 404s `/favicon.ico` in browser logs; one handler + one SVG kills the noise).
8. Verify the `aria-live` story for polled regions: PolledRegion was excluded, but the hand-rolled polling region has no aria-live — decide polite-live for projection health (hand-rolled, 5 lines) or wait for the upstream body-slot.
9. Slider keyboard hint says "← → arrow keys" — verify the global arrow-key handler doesn't hijack arrows when the slider is focused (double-stepping), e2e it.
10. `filter-bar` layout with wrapped FormField divs: check the 375px mobile screenshot for flex-wrap collision with the Filter/Clear buttons.

**Release / trains**
11. ~~Cut dashboardui v4.11.0 (CSP fix is consumer-facing; CHANGELOG already written) — requires the screenshot/axe re-runs above for the visual-change bar.~~ done (v4.11.0 train shipped)
12. ~~go-etag v0.4.0 train (26+ modules still on v0.3.1 — the release-train gate's largest advisory block).~~ done (go-etag v0.4.0 adopted, train-lag zero)
13. ~~go-cqrs-lite catalog v4.4.0 train (catalog-demo, integration_test).~~ done (catalog v4.4.0 adopted, train-lag zero)
14. ~~badgerengine v4.2.1 + remaining metaengine stragglers (systemadapter rows in the train list).~~ done (badgerengine bumped, train-lag zero)
15. ~~After trains: re-run `check-release-train --refresh-cache` and shrink the advisory list to zero.~~ done (release-train 0/0)
16. Decide release authority question (auto-tag trains vs explicit go) — blocks 11–15 from being automated.

**Upstream filings (templ-components) — each behind verify-before-filing**
17. PolledRegion props-based body slot (`Body templ.Component`) to unlock hybrid consumers.
18. ListNote range-semantics variant (dashboardui exclusion ask, already identified in Run-3).
19. Grid: document/prop-based children escape hatch for string-builder consumers.
20. `httputil.NonceConfig` godoc fix (zero value emits no CSP header; the "Default:" claim is false — found in N10, fix never filed).

**Docs**
21. ~~HARVEST this report's §f into TODO_LIST/ROADMAP (docs-health run).~~ done (this docs-health sweep)
22. Update `dashboardui/README.md` styling section with the forms.Input adoption row (adoption table lists Select but not Input yet).
23. ~~Annotate the 2026-09-17/18 status reports with outcomes (docs-health ANNOTATE mode) — the M-series/Run-2 reports now have stale "next steps" that this session consumed.~~ done (A2 + this A3 sweep)
24. Document the `filterInput` helper pattern in the hybrid-adoption guide (explicit-ID pattern for DOM-stable adoption) — it's the reusable recipe.
25. Record the "hook repaired its own sabotage" incident into release-playbook §6 (daemon/hook attribution section).

**Testing infrastructure**
26. Wire fmt-markers test, CSP tests, and a11y contract tests into CI (they exist locally; CI may still not run them).
27. Add a `render-bench` nix app so bench runs don't need hand-typed invocations (parallel to `bench-spike`).
28. Mutation-test the fmt_markers test (like the CSP test now is) — prove it fails when a `%!(` marker is reintroduced.
29. Property-style fuzz for `filterInput`/slider markup escaping (values are user-controlled query params).
30. e2e SSE spec (deferred since N8) — needs a stub bus or a minimal Publisher shim; schedule when the bus interface stabilizes.

**Code health**
31. `handler_overview.go` still holds generic helpers (`esc`, `truncate`, `metaRow` remnants?) — final sweep for pre-adoption dead helpers.
32. Audit remaining `fmt.Sprintf`-built HTML in handlers for the fmt-marker class of bug beyond the 9 rendered routes (export/error paths).
33. Unify table path guidance: the bench finding (raw-body 4.3x cheaper) should become a comment convention at each `tableHTML`/`tableHTMLRaw` call site (the split is already correct; document why).
34. `layout.go` inline-JS block is growing (slider + toasts + error handling) — consider extracting to an embedded `assets/dashboard.js` with ETag like `sync-client.js` does in root.
35. Slider: consider `list="data-ticks"` or aria-orientation review — minor a11y polish beyond valuetext.
36. `filterInput` attrs (Attrs field) unused — wire the form's `data-confirm`-style extras through it if forms ever need it, else document.

**Toolchain / repo hygiene**
37. ~~The lasting go 1.27.1 policy decision (root go-appkit go.mod mismatch was the original vector — file it upstream there too).~~ done (1.27.1 coordinated bump landed 2026-09-19)
38. e2e/server go.mod "should be direct" gopls warnings (command/event/id/projectionhost/query/snapshot) — one tidy pass.
39. `scripts/testdata/verify-tag/*` modules flagged "need go mod tidy" by every hook run — add a fixture exclusion so preflight stops warning about intentional fixtures.
40. BuildFlow go-structure-linter root-package-files ×50 false positive — the suppression feature it needs shipped? Re-check and restore `fail_on: critical` if so.
41. go-sum critical on verify-tag fixtures — give fixtures an honest go.sum or teach the linter the exclusion.
42. dependabot.yml 28-modules-capped-at-20 finding — split config or raise cap.
43. lychee link-checker timeout inside hook budget (45.6s kill) — raise budget or scope lychee to changed files.
44. `nix-checker` vendorHash-inline findings — extract to `vendorHash.nix` or suppress with reason.
45. go-licenses binary missing from ambient PATH (preflight warn every run) — devshell-only usage or add to profile.

**ROADMAP fuel**
46. templ-conversion decision for dashboardui (the hybrid path's children limitation is now blocking three components — the cost/benefit flips as the list grows).
47. SSE-driven projection-health updates (replace polling when the SSE feed reaches the dashboard page — the datastar EventBridge already exists).
48. Dark/light theme toggle (user question outstanding).
49. Detail-page screenshot captures (event/command/query/DLQ detail pages are not in the 27-PNG set).
50. benchstat golden gate for the render benches (like `bench-spike`, machine-pinned, so ratio regressions fail loudly).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Toolchain policy (blocking item #2 and every future incident):** Do you want `GOTOOLCHAIN=go1.26.7` pinned now for all gate scripts (stops the bleeding, keeps 26 modules at 1.26.7), or do you authorize the coordinated flake+go.work+27-module bump to 1.27.1 (one honest migration, kills the root cause)? The sibling session's toolchain will keep re-bumping the root until one of these happens.
2. **The sibling's turf:** the 243 auto-fixes and the `*.png binary` gitattributes line may have touched adminui/e2e files your other session owns — should I diff-audit and report everything BuildFlow's hook auto-committed since 11:20, or is that session still authoritative for its own tree state (and you'll reconcile there)?
3. **Release bar for dashboardui v4.11.0:** the CSP fix is consumer-facing and CHANGELOG-ready — do you want it tagged after only the hermetic gates (current state), or only after the golden screenshots + axe sweep are re-run against HEAD (my recommendation, ~30 min more)?

---

*Point-in-time snapshot — goes stale. §f is HARVEST input for `TODO_LIST.md`/`ROADMAP.md` (docs-health), not an entombed list. Daemon attribution: heuristic commits `5c119e12` (ireturn), `0c5fa7f5` (benches), `f5c39764` (filter swap), `1c25be3e` (slider fix), `ae70ef61`/`b58bffd2`/`ed4edeb9` (docs+tests+CSS), `5b72cb73` (CHANGELOG+train bumps), narrative `ab5a2b6d` (toolchain restoration + hook repairs).*
