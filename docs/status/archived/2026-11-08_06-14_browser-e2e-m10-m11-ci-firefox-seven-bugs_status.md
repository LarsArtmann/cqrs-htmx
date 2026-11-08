# Browser E2E train: M10–M11 landed, CI job added, firefox promoted then flaked — 7 bugs total this wave

> **ANNOTATED 2026-11-08** — ~~every open item in this report is resolved~~ **ARCHIVED 2026-11-08 (round-23 (2026-11-08)).**
> The next session in the same arc is [`2026-11-08_13-11_firefox-divergence-resolution-f20-promotion_status.md`](../2026-11-08_13-11_firefox-divergence-resolution-f20-promotion_status.md), which resolved.
> Nothing here is a live work item: it is kept as the session's evidence.

**Date:** 2026-11-08 06:14 CET
**Plan of record:** `docs/planning/2026-11-07_08-55_SUPERB-browser-e2e-battery-ci-and-datastar-coverage.md` (§Execution log now carries per-milestone receipts)
**Session scope:** M10 finish → M11 → M2 (CI) → M4 (firefox) → flake-stabilization → M5 (docs) → M6 (commits+push) — interrupted mid-M6 by this report request.

## a) FULLY DONE (verified green this session)

1. **M10 — bridges battery** (`e2e/tests/bridges.spec.ts`, 5 tests green):
   - Health dashboard: page + "All Systems Operational" + in-page proof the SSE patch arrived (init-script marker) + connection-pill attribute state machine.
   - Auditlog viewer: title, `#connection-status` → connected, invoke → live events-table growth (one invoke records TWO phase rows — growth contract, not exact delta), services tab shows `bridgeProbeService`, console guard.
2. **Bug #6 found + FIXED (the battery's ROI continues):** the health dashboard mount was page-only — `Handler()` serves the page but neither the `/sse` route nor the pusher goroutine existed (`RegisterRoutes` + `Start` were never called). Every consumer copying the old mount ships a dashboard whose connection pill is stuck on "Reconnecting" forever. Fixed in `e2e/server/bridges.go` with `healthdashboard.WithRoutes{...}` + `RegisterRoutes` + `Start(ctx)`.
   - Bonus: `WithEmbeddedDatastarSDK()` swapped the jsdelivr CDN script for a same-origin one — the bridge surface is now hermetic (no CDN dependency in CI/offline).
3. **Upstream finding (documented, filing candidate):** go-health-dashboard v0.13.2 renders all three connection pills simultaneously — Tailwind display classes (`inline-flex`) defeat the UA `[hidden]` rule (author display always wins). Verified empirically (computed style `flex` + `hiddenAttr=true`). No upstream fix exists (v0.13.2 is latest); the spec asserts the hidden-ATTRIBUTE state machine with a citation instead of visual visibility.
4. **M11 — loginpage posture spec** (`e2e/tests/loginpage-posture.spec.ts`, 3 tests green): `/auth/me` 401-JSON anonymous → register issues a browser-usable session (request-space shares the page context cookie jar; `/post-login` renders the email) → logout revokes everywhere (401 again, gated page bounces to /login). Each test is self-contained (fresh context per test — the serial-mode context sharing trap hit once and was fixed).
5. **M4 — firefox engine, verified locally green ONCE, then promoted:**
   - `playwright.config.ts`: chromium `E2E_BROWSER_PATH` scoped into the chromium project only; firefox project env-gated (`E2E_FIREFOX_PATH` explicit build / `E2E_FIREFOX=1` registry), `loginpage-webauthn.spec.ts` excluded (CDP-only).
   - `nix run .#e2e-firefox` flake app: builds a nixpkgs lib-forest (`runCommand` symlink farm) for the prebuilt firefox (FHS expectations), locates the cached Playwright build, exports `E2E_FIREFOX_PATH`/`E2E_FIREFOX_LIBS`.
   - The forest is scoped to firefox LAUNCHES ONLY (`launchOptions.env`) — a runner-wide `LD_LIBRARY_PATH` poisons chromium's bundled nspr resolution (hit live: chromium launch died on `libnspr4.so`).
   - `/favicon.ico` now served (redirect to the dashboard SVG) — firefox's internal FaviconLoader logs a CSP console error against the auditlog viewer's strict `default-src 'none'` CSP; the bridges console guard filters that one noise class with a citation.
   - First full both-engine battery: **150 passed / 0 failed / 5.7 min** (13 did-not-run = 11 screenshot under-load skips + 2 webauthn firefox excludes).
6. **Flake-stabilization:**
   - Screenshot harness: single-shot `goto` non-200 → 3× retry before skipping (the "availability-probe skip" class was transient server hiccups under machine load, never real unavailability).
   - Sync-tail specs (`getEvents`, `sync:reset`): poll ceilings 15s → 30s + `test.setTimeout(120_000)` (the 2026-09-13 boot-race class).
7. **Bug #7 found + FIXED (masked until this session):** with skips gone, the axe sweeps finally RAN and found a real, long-standing a11y violation — `aria-label` on a roleless `<span>` (`dashboardui/layout.templ:135`, the live-indicator, ships in every dashboard page). Fixed with `role="img"`; `_templ.go` regenerated with the canonical pinned generator (clean 1-line diff); dashboardui Go tests green; axe 19/19 green. **The prior "18 skips under load" had been silently hiding this for every earlier run.**
8. **Chromium full battery final: 106 passed / 0 failed / 0 skipped / 1.5 min** (from 80 passed + 18 skipped at session start).
9. **M2 — CI e2e job** (`.github/workflows/ci.yml`, actionlint green, verification = the pending push): sibling clones (go-cqrs-lite, go-idempotency, httputil) + `go.work` absolute-path rewrite, warm Go build of all five servers (keeps the 120s webServer timeouts off the cold toolchain), `setup-go` with `go-version-file: go.work`, `oven-sh/setup-bun@0c5077e` `# v2.2.0` (tag→SHA resolved live via `gh api`, peeled-ref verified), `bun install --frozen-lockfile`, `bunx playwright install --with-deps chromium firefox`, suite with `E2E_FIREFOX=1`.
10. **M5 — docs:** `e2e/README.md` rewritten (13-row coverage matrix spec × surface × port, stale-server kill ritual, sanctioned-run warning, firefox entry, CI section — the old one claimed "73 specs" and "CI does not run browsers"); plan §Execution log filled with per-milestone receipts + flake receipts; corpus baseline re-trued (57,190); freshness + links gates green.

## b) PARTIALLY DONE

1. **M6 (commits + push):** NOT executed. The daemon absorbed all file changes into heuristic auto-commits (tree was clean before my final README fmt pass); the logical-commit plan is moot without history rewriting (which I refuse to do under active sibling sessions — gotcha 47). The remaining work is: one small commit (formatted README), the push itself, the gotcha-50 pre-push protocol, and `nix run .#push-and-watch`.
2. **M4 CI promotion is now QUESTIONABLE** (see d): firefox was promoted to a BLOCKING CI step on the strength of one green run; the final verification run flaked 11 firefox tests. The promotion needs demotion to advisory or scoping before the push, or CI will redden randomly.

## c) NOT STARTED

1. Gotcha #56 entry (element-bound JS dies under hx-boost swaps — the bug #5 generalization) in `docs/gotchas.md` + AGENTS index.
2. DecodeForm-silently-empty sharp-edge gotcha candidate.
3. Upstream filing for the go-health-dashboard pill bug (evidence captured in the spec comment; filing gated on the verify-before-filing skill).
4. e2e/server stale-port guard mechanization (the kill ritual is documented prose, not a gate).
5. The parked 20% from the plan (pixel baselines, parallel workers, CSP guide section).

## d) TOTALLY FUCKED UP / needs a decision

1. **The final both-engine run: 186 passed / 11 FAILED / 13 did-not-run (4.2 min).** All 11 failures are FIREFOX-only and were GREEN in the first both-engine run — i.e. flaky, not deterministic:
   - 6× sync suite: `page.route: Protocol error (Network.setRequestInterception)` — request interception against the cached firefox-1543 build while @playwright/test 1.64.1 expects firefox-1555 (revision skew, or a load-timing race in interception setup).
   - root-htmx OOB `toHaveText`, admin pagination `toContainText`, setup-bundle dual-transport, datastar multi-tab — timing/patch-shape differences under firefox.
   - The 13 did-not-run = serial cascade after the sync failures + the 2 webauthn excludes.
2. **My CI promotion decision was premature** — one green run is not a flake rate. I promoted firefox to blocking in the same session where the promotion criterion ("locally green") was met by exactly one observation. The honest fix is one of: demote the CI firefox step to `continue-on-error: true` until it stabilizes; exclude the interception-dependent sync specs on firefox (`testIgnore` addition); or provision firefox-1555 (matching the runner's playwright) instead of the cached 1543.
3. Session hygiene note: two `multiedit` batches against `bridges.go` mangled the file mid-edit (duplicated `RegisterRoutes`, displaced config block) — caught and repaired by re-reading, but it cost a cycle and was avoidable by reading the freshly-gofmt'ed file first.

## e) WHAT WE SHOULD IMPROVE

1. **Promotion decisions need a flake RATE, not a single green run** — the plan's kill-point said "promote if green"; one observation met the letter and violated the spirit. Any future CI-blocking promotion should require ≥2 consecutive green runs.
2. **The skip class was masking a real bug for weeks** — axe's 18 tests silently not running hid bug #7. Lesson mechanized by the screenshot retry, but the deeper fix is making "did not run" loud (a post-run assertion that ran-count == listed-count) — candidate gate.
3. **Playwright revision pinning on NixOS:** the cached firefox-1543 vs registry-1555 skew is a standing liability; the flake app should warn when the cached build's revision ≠ `node_modules/playwright-core/browsers.json`.
4. **Serial-mode context sharing:** playwright gives every test a fresh context even inside `test.describe.configure({mode:"serial"})` — bit me twice (loginpage-posture, sync cascades). Worth a one-line note in e2e/README.

## f) Next 50 (roughly priority-ordered; P0/P1 first)

**P0 — the push (M6 finish):**
1. Decide firefox-in-CI shape (question 1 below) and apply it.
2. Commit the formatted README (+ any daemon leftovers).
3. Push with the gotcha-50 protocol if the pre-push hook reds on the documented API-vs-tag class.
4. `nix run .#push-and-watch`; triage the CI e2e job's first real run (expect first-run cache-cold timing; warm-build step exists for exactly that).
5. Fix-forward any CI red; verify `.#verify-remote-tags` after.

**P1 — firefox stabilization:**
6. Provision firefox-1555 into `/mnt/buildcache/playwright` and point the flake app at it (removes the revision-skew hypothesis).
7. If sync-on-firefox still flakes: `testIgnore` the sync spec on the firefox project (interception-heavy, SharedWorker semantics differ) with a citation.
8. Re-run the both-engine battery 3× to get a real rate; only then revisit blocking vs advisory.
9. Add the cached-revision-≠-expected warning to the `e2e-firefox` app.

**P1 — gates & hygiene:**
10. Ran-count == listed-count post-run check (make silent skips loud) — new gate or reporter hook.
11. Gotcha #56 (hx-boost kills element-bound JS) into `docs/gotchas.md` + AGENTS index.
12. DecodeForm empty-body sharp-edge gotcha.
13. File the go-health-dashboard pill bug upstream (verify-before-filing pass first).
14. e2e/server stale-port guard mechanization (script or config-level `reuseExistingServer:false` under a flag).
15. Serial-context note in e2e/README.

**P2 — plan remainder / parked 20%:**
16. Pixel-regression baselines (needs pinned runner image).
17. Parallel worker split (suite serialization isn't the bottleneck yet).
18. CSP guide section (docs-only, TODO_LIST rider).
19. Dashboard boost-guard spec (server-side event trigger now EXISTS via `/api/debug/emit` — unblocked).
20. `examples/basic` + `setup-demo` deeper flows beyond smoke.

**P2 — observed-but-not-mine:**
21. docs-gates `audit-all` red at 393 vs pinned 322 (pre-existing TODO P2 row; report volume, not rot).
22. Sibling sessions actively landing docs; expect baseline contention again on the next corpus re-true.

23–50. The standing TODO_LIST backlog (release-train tag lag, A25 catalog-demo upstream watch, erraudit drain, audit_max re-pin decision, route-posture tag-lag reds healing at the v4.17.0 re-tag, …) — unchanged by this session; see `TODO_LIST.md`.

## g) Questions I cannot answer myself

1. **Firefox in CI — demote or scope?** The final run flaked 11/104 firefox tests (all green in the prior run). Options: (a) `continue-on-error: true` on a separate firefox step until it shows ≥3 clean runs, (b) exclude the sync spec on firefox and keep blocking, (c) revert to chromium-only CI and keep firefox local-only. My recommendation: (a) — honest, keeps signal, doesn't redden PRs on engine noise.
2. **CI e2e budget:** the job adds ~8–12 min per PR (both engines). Keep both engines in the per-PR lane, or split firefox into the release train lane only? (Owner question #2 from the 00-19 report, still unanswered; now concrete.)
3. **Push now or after firefox-shape decision?** The standing SUPERB authorization covers commit+push; I can push everything as-is (firefox blocking) and fix-forward, or apply question 1's answer first and push once. My recommendation: apply (a) first — one push, one CI truth.

## Receipts

- Chromium battery: **106/106, 0 skipped, 1.5 min** (was 80 passed/18 skipped at session start).
- Both-engine battery run 1: 150 passed / 0 failed / 5.7 min. Run 2 (post axe-fix): 186 / 11 failed / 13 did-not-run — all failures firefox-only.
- axe spec: 19/19 after the `role="img"` fix. bridges 5/5. loginpage-posture 3/3. dashboardui Go tests green.
- Docs gates: corpus ratchet re-trued (57,190 + this report), freshness green, links green (1,538 links), tail ratchet green (bumped for this report).
- actionlint green on ci.yml; bun pin peeled-ref verified; flake builds (`.#e2e-firefox`, lib forest) green.
