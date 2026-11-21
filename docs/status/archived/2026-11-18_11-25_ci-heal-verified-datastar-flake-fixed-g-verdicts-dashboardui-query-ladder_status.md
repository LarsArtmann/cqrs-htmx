# Session report — CI-heal verification, the datastar flake's true root cause, §g verdicts, and the dashboardui QUERY ladder

> **ANNOTATED 2026-11-21** — ~~this report~~ **SUPERSEDED** by [`2026-11-21_08-45_master-ci-heal-composite-bill-fast-slice-and-harvest_status.md`](../2026-11-21_08-45_master-ci-heal-composite-bill-fast-slice-and-harvest_status.md), which the 08-45 report closes the QUERY-ladder red class this heal report verified (debounce-twin race root-caused, test settles the debounce) and receipts the datastar/G-verdict landings via the fleet rows. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-21 (round-48).**

**Date:** 2026-11-18 11:25 CET · **Session:** execution of the owner's blanket GET-SHIT-DONE directive over the 06-53 report's next-steps list · **Tree:** 30-module workspace, master at `7e2991d3`, **16 commits ahead of origin, NOTHING pushed**

**Mission as given:** execute the whole handoff todo list — push + CI watch, gotchas 41/62/63, PR #101, CHANGELOGs, §g verdicts, f.32/f.33, TODO harvest, CURRENT_FOCUS, K2/K4 waves, close-out. The session executed in that order until the tree itself became contested (three live sibling sessions) and one self-inflicted test regression stopped the dashboardui suite.

## Scope line — VERIFIED (all fresh, receipts named)

- The push step was already DONE by the sibling session: `git ls-remote origin master` == local HEAD `440237c5` (my local origin/master ref was stale — "Everything up-to-date" surprise).
- CI run 44249826940 on the pushed HEAD: of the six originally-failing jobs, **FOUR went green on the runner** (lint/systemadapter, security, e2e Warm-cache, module-architecture root-files — the ambiguous-import fix, the go 1.27.4 pin, the SQL DSN fix all PROVEN). TWO NEW failures: `checks`/Root-package-size-ratchet and `test`/datastar-submodule — both triaged and fixed locally (below).
- The datastar failures are **flaky by construction, and PREDATE the dependency sweep**: the producer (`NewSignalsPatch`) marshals map-backed signals via Go's randomized map iteration — no key sort — while `datastartest.RequireSignals` compares bytes. Both orders observed live (full suite run: both fail; filtered rerun same commit: both pass; -count=12 loops: intermittent). The two multi-key pins (`response_test.go:77`, `coverage_test.go:150`) could never be deterministic; the leg never ran since the usermgmt red aborted the job before it (gotcha 40's coverage gap, vindicated). The single-key pin (`{"count":1}`) is fine.
- Fix: `datastar/signals_assert_test.go` `requireSignalsSemantically` (unmarshal-both + require.Equal), the two multi-key call sites swapped. Suite green 4×, `-race` green, vet green.
- Root-size ratchet reproduced locally: +2 LOC = the two `//cqrs-lint:ignore(C009)` directive comments from the BuildFlow-incident triage wave (daemon capture `0113cf7f`) — legitimate, deliberate, gate's own remedy applied: `--update-baseline` same-commit with the reason (`eff367be`).
- §g② answered EMPIRICALLY, premise FALSE: BuildFlow's timing-regression verdict is ADVISORY-ONLY and has never gated an exit code (`buildflow/internal/cli/regression_verdict.go`, advisory since its creation `73e61349e` 2026-09-02, "the run verdict must never depend on" the timing query; installed binary built from `5783c654f` includes it). Live probe under load ~100: dozens of regression lines printed (govalid-generate +403550%), hook **exit 0**. The 06-53 session misread accompanied advisory lines as the blocking cause. **No hook change made; the 06-53 report's §b row and §g② are annotated with the correction.**
- §g③ answered: the "e2e flake" on `9d159fdc` (failed run 43827767098, then "green on same sha" 44010050578) was NOT a flake — the failing run shows the same SIX deterministic causes as the red window (incl. the ambiguous import, since fixed), and the "green" was the train-lag workflow, not ci.yml. **Verdict: fail-loud stays, no annotation** (the REAL flakiness was the datastar signal pins, fixed at source).
- §g① verdict recorded: **accept-bloat + hardening** (`.gitignore` gains `/server` + `/demo`; the `--all --diff-filter=A` sweep over the red window found NO other binaries; history rewrite stays owner-gated and was NOT executed on a blanket directive).
- PR #101 (templ-components popover x-axis): **MERGED upstream**, all checks SUCCESS incl. CSS Freshness. The vendored adminui corrected-CSS layer now retires with the next family train.
- f.33 STRUCK with receipt: the "silently accepts duplicate ops / last-wins" premise is false — `openapi.Spec.PathOp` PANICS on a duplicate same-method registration (`openapi/builder.go:112`). The guard is stronger protection than any comment.
- CHANGELOGs landed: root Added (admin-demo page rule + posture receipts), root Housekeeping (the three-train sweep receipt with the daemon hashes), systemadapter Fixed (the T04 display-form cascade fix).
- Release-train verified 0-unpublished/0-lag in the morning; hook probe receipt (exit 0 under load); gotchas 41 (amended) + 62–65 (new) landed in docs/gotchas.md + the AGENTS index (now "65 numbered traps").

**Scope line — NOT verified / NOT done:** see b) and c). The short version: the dashboardui suite is red by MY OWN new regression (d1), the 16-commit train is unpushed, and the K2/K4 waves were deliberately not started in a contested tree.

## a) FULLY DONE

| Item | What landed | Receipt |
| --- | --- | --- |
| CI heal verified on the runner | 4 of 6 previously-failing jobs green on run 44249826940 (ambiguous-import fix, go 1.27.4 pin, SQL DSN fix, root-files restore all PROVEN) | `gh run view 44249826940` |
| Datastar flake root cause + fix | `NewSignalsPatch` map-iteration order vs byte-exact `RequireSignals`; semantic-comparison helper; 2 pins swapped; suite 4× + race green; the flake PRE-DATES the sweep (masked-leg proof at parent commit `e533bceb`) | `datastar/signals_assert_test.go`; daemon capture `eb049c68` |
| Root-size ratchet | +2 LOC identified as the C009 directive comments (not code growth); baseline bumped 8,881 → 8,883 with the reason | `eff367be` |
| Gotchas 41 + 62–65 | 41 amended (root cause; rerun-once obsolete); 62 canonical-sed/ambiguous-import (+ repro recipe); 63 daemon binary-guard bypass (+ .gitignore `/server`, `/demo`); 64 fixture-mirrors-bug (T04); 65 pathspec-commit hook trip; AGENTS index 61 → 65 | daemon capture `337c74b6` |
| §g② verdict | Regression verdict advisory-only — verified at source + live probe; report corrections annotated; NO hook edit | 06-53 report §b/§g② annotations |
| §g③ verdict | Fail-loud stays (the "flake" was the six deterministic causes + a workflow misread) | `gh run view 43827767098` |
| §g① verdict | Accept-bloat + `.gitignore` hardening; sweep clean | `.gitignore`; `git log --all --diff-filter=A` |
| PR #101 | MERGED, all checks green incl. CSS Freshness | `gh pr view 101` |
| CHANGELOGs ×3 | admin-demo receipts (root Added); three-train sweep Housekeeping; systemadapter T04 Fixed | daemon capture `fc4ca511` |
| f.32 dashboardui QUERY ladder (CODE) | `QUERY /events/search` + POST twin (one `eventsSearchHandler` over a shared `renderEvents` core), urlencoded body parser (the adminui form-encoding precedent — NOT JSON), `MethodOverrideMiddleware` inside `routes()`, `/-/ext/cqrs-query.js` asset row, form rides `hx-ext="cqrs-query"` + `hx-post` + `hx-query` with gotcha-17 target/select pins, layout script tag, pinned-templ v0.3.1100 regeneration (2 files), ladder-parity + invalid-body-400 + ext-asset tests GREEN | daemon captures `5653dcfb`/`7818f908`/`a0940e91` lineage |
| f.33 merge-loop note | STRUCK — premise false (PathOp panics on true duplicates) | `openapi/builder.go:112` |

## b) PARTIALLY DONE

| Item | Works now | Still open | Blocker | Effort |
| --- | --- | --- | --- | --- |
| f.32 dashboardui QUERY ladder | All code + tests in HEAD; new tests green; build/vet green | **The module suite REDS via my own regression** (d1): the rewritten `TestRoutes_MatchRegisteredMuxPatterns` drives a GET into `/-/events/stream` — the SSE endpoint never closes the response, the recorder blocks, the package hits the 600s wall-clock timeout (gotcha 42's hang class, self-inflicted). Fix is small: skip stream routes (the boost guard now skips Assets the same way) or bound each probe with a context timeout; then full suite ×3 | none — report-first order stopped the fix | S |
| The push | 16 commits ready; the sibling's idempotency v0.6.0 sweep landed properly (`c100e9c5`) + a 53-file sweep capture (`7e2991d3`); my morning commit attempt showed 48 lag entries which their sweep addressed | `git push` + `push-and-watch` — NOT executed: their sweep was LIVE when I attempted mine (bump-dep's live-sweep + dirty-tree guards refused me, correctly); after their sweep landed the report-first order arrived. Re-verify `check-release-train --refresh-cache --strict-lag 0` first | order to push | S |
| K2 idempotency default | The 23-module sweep is DONE (by the sibling, now v0.6.0); `rg "Forever"` still zero | The "default-on" decision has NO pre-staged recommendation — nothing to tick under D18; needs research before it can even be recommended | owner call on whether to research it | M |
| TODO_LIST harvest + CURRENT_FOCUS re-point | not done | Harvest my threads (CI-heal closed, datastar-flake fixed, hook-verdict misattribution resolved, f.32 code landed) + re-point | the sibling was mid-ritual writing its own close-out when I checked (foreign `CORPUS_BASELINE`/status-README edits); contested-file discipline deferred mine | S |
| Baselines | CORPUS re-measured 62,235 (+111 for this report); TAIL will be 10 with this report | Both bumped in THIS report's commit (the ritual) | none | S |

## c) NOT STARTED

1. **K4 waves** (M2M D1–D4, migration D1–D5, adminui D1–D3): not read, not ticked, not executed. With THREE live sibling sessions in this tree (verified via `/proc/<pid>/cwd`), starting multi-session code waves under gotcha 47 would have shredded everyone's work — a deliberate stop, not an omission. They need a quiet tree.
2. §f.44 TODO_LIST harvest, CURRENT_FOCUS re-point (see b).
3. Post-push items: full chromium e2e battery (the QUERY form change is client-facing), `check-modules` composite, erraudit strict ×2 re-true (new dashboardui code moves the TOTAL), cqrs-lint gate on the post-wave tree, `check-upstream-treadmill --refresh-cache` receipt.
4. Upstream asks: go-datastar deterministic signals ordering (premise verified here); templ-components vendored-CSS retirement scheduling (PR #101 merged).

## d) TOTALLY FUCKED UP

1. **The SSE-hang regression (self-inflicted, the session's real bug).** Rewriting `TestRoutes_MatchRegisteredMuxPatterns` from `mux.Handler()` pattern-matching to real `ServeHTTP` requests made the test walk EVERY route — including `/-/events/stream`, an SSE endpoint whose response never completes. The recorder blocked until the package's 600s wall-clock timeout killed the suite. I violated gotcha 42's own rule in the worst way: a wall-clock timeout IS a hang until proven otherwise — and here I BUILT the hang. The stack trace (`transport/serve.go:495`) sat in the output and I nearly wrote the suite off as load noise; the diagnosis only landed when I stopped and re-derived which routes the loop visits. A route-table test must never drive never-ending routes without a skip list or a per-request deadline.
2. **One garbage multiedit hunk**: `mux.ServeHTTP := handler` (invalid Go) landed in `telemetry_test.go` between two correct hunks — caught and fixed in the immediately following edit, before any test run. The multiedit batch was too big for one pass; the second hunk deserved its own call.
3. **Sweep collision attempted**: I launched a bump-dep sweep of go-idempotency v0.6.0 while a sibling's sweep was LIVE — bump-dep refused (live-sweep pid check + dirty-tree guard, both worked as designed). The sibling's own §e.1 rule ("live-sweep check whenever another session is active") existed and I reached for the tool before running the check.
4. **The push surprise**: `git push` answered "Everything up-to-date" because the sibling had already pushed and my local `origin/master` ref was stale. A `git fetch` before comparing would have saved the round and reframed the whole "verify CI on the pushed train" step an hour earlier.
5. **`templ generate -path <file>` silently updated zero files** (the path is treated as a directory pattern); only the module-dir invocation regenerates. One wasted round, caught by `updates=0`.

## e) WHAT WE SHOULD IMPROVE

1. **Route-walking tests need a never-ending-route policy** — any test that iterates the route table must skip or deadline-bound stream/long-poll routes. The boost guard's new Assets-skip is the same shape; generalize it into a shared helper the routes test reuses.
2. **Multi-session window discipline is now codified by the tools** (bump-dep's guards) — the human/agent rule should match: fetch + `ps aux | grep bump-dep` + `uptime` BEFORE any sweep, push, or tree-wide operation. The sibling codified stop-rules; I should have applied them proactively, not after a refusal.
3. **Verify "already done" claims with the remote, not the local ref** — `git fetch` first is the whole fix.
4. **Tool-output contracts deserve gotcha-60 discipline**: the §g② misattribution came from reading an advisory line as a gating verdict. The source took one grep; the probe took one command. Verify tool behavior at the source before building a policy on it.
5. **The `test` job's sequential masking** (gotcha 40) struck again: the datastar flake shipped in a module whose leg had zero evidentiary coverage since the usermgmt red. The per-module CI matrix idea (gotcha 40c) keeps being retroactively vindicated — it remains the durable fix.
6. **CHANGELOG entries should land INSIDE the work window** (the sibling's §e.4 again) — the daemon captured my code before my commit could group it, so the QUERY-ladder CHANGELOG entry is still owed (f15 below).

## f) Up to 50 things to get done next (priority order)

| # | Task | Impact | Effort |
| --- | --- | --- | --- |
| 1 | Fix the SSE-hang regression: skip stream routes (or ctx-deadline) in `TestRoutes_MatchRegisteredMuxPatterns`; rerun the full dashboardui suite ×3 + `-race` | Critical | S |
| 2 | Hermetic `golangci-lint run` for dashboardui (the new QUERY-ladder code has not run the lint leg; the fleet lint legs in my sweep era were `--no-lint` under load) | High | S |
| 3 | `check-release-train --refresh-cache --strict-lag 0`, then push the 16-commit train + `push-and-watch` | Critical | S |
| 4 | Verify on the runner: the six healed jobs STAY green + the two new fixes (datastar, root-size) go green | Critical | S |
| 5 | CHANGELOG entry (dashboardui module): the QUERY ladder adoption (f.32) | High | S |
| 6 | Full chromium e2e battery locally once CI is green (client-facing form change) | High | M |
| 7 | `check-modules` composite on the post-wave tree (never ran this session) | High | M |
| 8 | erraudit strict ×2 re-true (new dashboardui code moves TOTAL) | Medium | S |
| 9 | cqrs-lint gate (14 modules, v4.22.2 floor) on the post-wave tree | Medium | M |
| 10 | §f.44 TODO_LIST harvest: close the CI-heal thread, open the watch thread, mark the datastar flake fixed, mark the hook-verdict question resolved (misattributed), mark K2's sweep done | High | S |
| 11 | CURRENT_FOCUS re-point (watch thread → QUERY-ladder close-out → K4) | Medium | S |
| 12 | K4 under the D18 precedent: read the three packets (M2M D1–D4, migration D1–D5, adminui D1–D3), record the ticks with this directive as the attribution, execute in a QUIET tree | High | L |
| 13 | K2 default-on: research + write the recommendation packet (currently nothing pre-staged to tick) | Medium | M |
| 14 | Upstream ask (repo: go-datastar): deterministic key ordering in `NewSignalsPatch` (premise verified: randomized map iteration, both orders observed; every consumer byte-pinning signals JSON is flaky) | Medium | S |
| 15 | templ-components: schedule the adminui vendored popover-CSS retirement for the next family train (PR #101 merged) | Medium | S |
| 16 | Sweep remaining docs for the obsolete rerun-once wording of gotcha 41 (harness cheatsheet, runbooks) | Low | S |
| 17 | The boost-guard Assets-skip: extract a shared "walk rendered pages" helper for routes/boost/a11y tests so the stream-route skip lives in one place | Medium | M |
| 18 | Verify `check-permission-surface`/`check-route-postures` on the new QUERY/POST rows (the posture derivation answered `ungated` — confirm the dashboardui manifest pins accept it) | High | S |
| 19 | Re-verify the e2e method-assertion specs still pass with the form's new hx-post transport (admin-behavior QUERY specs target adminui; dashboardui has no QUERY spec yet — consider one) | Medium | M |
| 20 | `check-upstream-treadmill --refresh-cache` receipt post-alignment (worst lag should be 0 after the sibling's sweeps) | Medium | S |
| 21 | Confirm `go-etag/entitytag v0.6.2` rode the sibling's sweep (my delta inspection: main source byte-identical) | Low | S |
| 22 | Hook probe receipt into the harness cheatsheet: the pre-commit lane under load exits 0 with regression advisories (the §g② misread prevention) | Low | S |
| 23 | Annotate the 07-20 report's §g③ with the fail-loud verdict (its question is now answered) | Low | S |
| 24 | The 06-53 report's §f.28 annotation (07-20 "master CI GREEN" correction) — still owed | Low | S |
| 25 | Decide the receipt-commit cadence for daemon-shared days (one cumulative receipt commit per wave — sibling §f.49) | Low | S |
| 26 | After the train greens: re-run the fixture self-tests for the touched gates (root-size self-test, corpus/tail self-tests) | Low | S |
| 27 | Watch for go-idempotency v0.7.0 (the supply side shipped three tags in ~24h; the mid-sweep-tag stop rule applies to the NEXT consumer too) | Low | S |
| 28 | The dashboardui events filter's sort state drops on a body-search submit (pre-existing GET-form behavior, now inherited) — decide: hidden sortBy inputs in the filter form, or document | Low | S |
| 29 | Fleet nixpkgs rev watch: bump ADR-0003 rev when nixpkgs ships go ≥ 1.27.4, revert the security-job pin (TODO P1 row) | Medium | S |
| 30 | Post-green: bench re-pin in a QUIET window only (standing rule; load was 58–228 all session) | Low | M |

## g) Questions for the owner (max 3)

1. **Push ordering**: the 16-commit train contains my dashboardui QUERY-ladder work whose suite currently reds via my own SSE-hang test regression (d1, fix = S). Push AFTER the fix lands in the tree (recommended: one clean green train), or push now and let the fix ride the next commit?
2. **K2's second half**: the go-idempotency sweep is done, but the "default-on" decision (enabling idempotency middleware by default on dispatch surfaces) has NO pre-staged recommendation — nothing exists to tick under D18. Should I research it and write the recommendation packet this cycle, or does it stay parked?
3. **Upstream asks**: file the go-datastar deterministic-signals-ordering issue now (premise verified in-session, evidence in hand), and open the templ-components vendored-CSS-retirement follow-up on merged PR #101 — or hold both for the next family train?

---

*Corpus ledger: 62,124 → measured after this row (baselines bumped same-commit: CORPUS 62,235, TAIL 10).*
