# SUPERB: e2e train push, firefox rate stabilization, and loud skip-gates

**Date:** 2026-11-08 06:26 CET
**Precursor status:** `docs/status/2026-11-08_06-14_browser-e2e-m10-m11-ci-firefox-seven-bugs_status.md`
**Parent plan:** `docs/planning/2026-11-07_08-55_SUPERB-browser-e2e-battery-ci-and-datastar-coverage.md` (M7–M11 + M2/M4/M5 landed; its §Execution log carries the receipts)

## Where we actually are

The battery grew from 80 passed/18 skipped to **106/106 chromium green (0 skipped)** and found+fixed **7 shipped bugs** across two sessions (#1 dead datastar demo, #2 dead forms, #3 ds-demo triple-dead, #4 htmx indicator CSP, #5 slider-under-boost, #6 page-only health mount, #7 roleless aria-label). The CI e2e job exists but has **never run red-or-green in CI** — nothing is pushed. Firefox is locally green once (150/0) and flaky once (11 firefox-only failures, sync interception protocol errors; cached firefox-1543 vs runner-expected 1555 skew suspected). The daemon snapshot-reverted two session edits (firefox CI promotion, sync/screenshot stabilization) — both re-applied 2026-11-08 06:3x and now guarded by this plan's daemon-revert rule.

**The verschlimmbesserung guard is explicit:** nothing in this plan adds a new abstraction, a new module, a config surface, or a second way to do something that already works. Every task either lands what exists, measures what exists, or writes down what was learned.

## Pareto

### The 1 % that delivers 51 %

**Land the push.** The CI e2e job is written, linted, and locally rehearsed — but unpushed, which means the entire two-session train still ships silently. One push converts the client-facing-bug class from ship-silent to PR-red and validates the job end-to-end (sibling clone, go.work rewrite, warm build, bun pin, both engine lanes). Everything else in this plan is worth less than this single action.

### The 4 % that delivers 64 %

1. **CI first-run triage** — the first real run will surface what rehearsal could not (runner cold caches, go.work rewrite reality, browser install size). Budget one fix-forward cycle.
2. **Firefox rate, not vibes** — three consecutive both-engine runs produce a real flake rate; provision firefox-1555 locally to kill the revision-skew hypothesis; the CI advisory lane (registry build, revision-correct) measures the true engine rate in parallel. The promote-to-blocking decision then has data behind it.

### The 20 % that delivers 80 %

3. **Loud-skips gate** — silent "did not run" masked bug #7 for weeks. A ran==listed check makes the class structurally impossible to miss again.
4. **Firefox follow-ups** — sync-spec `testIgnore` on firefox if the rate data says interception-heavy flake; revision-mismatch warning in the `e2e-firefox` app; promotion to blocking on 3 clean runs.
5. **Memory** — gotcha #56 (element-bound JS dies under hx-boost swaps; bugs #5's generalization) and the DecodeForm empty-body sharp edge, into `docs/gotchas.md` + the AGENTS index.
6. **Upstream filing** — the go-health-dashboard pill bug (evidence already captured in the spec comment; verify-before-filing pass, then file).
7. **Boost-guard spec** — the parent plan's parked dashboard spec is UNBLOCKED: `/api/debug/emit` now exists as the server-side trigger.
8. **Stale-port guard** — the kill ritual is prose; mechanize it so nobody tests yesterday's binary again.

### The other 20 % (to 100 %)

9. **Pixel-regression baselines** — precondition-gated (pinned runner image); parked with a trigger, not forced.
10. **Parallel worker split** — the suite is 1.5 min; serialization is not the bottleneck. Parked.
11. **CSP guide section** — docs-only rider.
12. **Deeper example flows** — beyond smoke for basic/setup-demo.
13. **The standing TODO_LIST backlog** (release-train lag, A25 upstream watch, erraudit drain, audit_max re-pin) — owned there, not replanned here.

## Medium tasks (30–100 min, sorted by importance / impact / effort / customer value)

| # | Task | Est | Impact | Value rationale |
| --- | --- | --- | --- | --- |
| M1 | Commit the train + push + `.#push-and-watch` + triage the first CI e2e run (fix-forward) | 45 min | ★★★ | The 1 %→51 %: converts the class to PR-red; validates everything |
| M2 | Firefox rate: provision firefox-1555, 3× both-engine runs, rate table, promote/demote decision by data | 90 min | ★★★ | The 4 %→64 %: turns a guess into a measurement |
| M3 | Loud-skips gate: post-run ran==listed assertion (local script first, wire into the runner path) | 45 min | ★★ | Mechanizes the bug-#7 lesson; structural, not vigilance |
| M4 | Firefox stabilization follow-ups: sync testIgnore (if data says so), app revision warning, promotion flip after 3 clean | 60 min | ★★ | Closes the engine lane honestly |
| M5 | Memory: gotcha #56 + DecodeForm sharp edge + AGENTS index rows | 30 min | ★★ | Next session starts warm; these traps recur |
| M6 | Upstream filing: go-health-dashboard pill bug (verify-before-filing, then gh issue) | 45 min | ★ | Upstream fix beats our spec-side scoping |
| M7 | Boost-guard spec: hx-boost wrapper + `/api/debug/emit` → every self-fetching element survives | 75 min | ★★ | The bug-#5 class gets a standing tripwire |
| M8 | Stale-port guard mechanization + e2e/README serial-context note | 30 min | ★ | Kills the yesterday's-binary trap |

## Micro tasks (≤ 12 min each, execution order)

| # | Task | Parent | Est |
| --- | --- | --- | --- |
| F01 | git status; confirm no sibling in-flight files in scope | M1 | 2 |
| F02 | Commit: ci.yml firefox advisory split + restored sync/screenshot edits (detailed message) | M1 | 8 |
| F03 | Commit: this plan + status report + baselines (docs message) | M1 | 5 |
| F04 | Push (gotcha-50 protocol if pre-push reds on the documented API-vs-tag class: hand-verified battery + `--no-verify` + named) | M1 | 10 |
| F05 | `nix run .#push-and-watch` (workflow-filtered); read the e2e job's step-by-step output | M1 | 10 |
| F06 | Triage: expected classes = cold-cache webServer timeout, go.work rewrite miss, browser install size | M1 | 12 |
| F07 | Fix-forward commit if red; re-watch | M1 | 12 |
| F08 | `nix run .#verify-remote-tags` after green | M1 | 3 |
| F09 | Provision firefox-1555: `./node_modules/.bin/playwright install firefox` (PLAYWRIGHT_BROWSERS_PATH=/mnt/buildcache/playwright) | M2 | 8 |
| F10 | Verify the flake app picks 1555 (its find is newest-first) + revision match vs browsers.json | M2 | 4 |
| F11 | Both-engine run 1 (record: passed/failed/did-not-run per engine) | M2 | 10 |
| F12 | Both-engine run 2 (record) | M2 | 10 |
| F13 | Both-engine run 3 (record) | M2 | 10 |
| F14 | Write the rate table into this plan's §Execution log; decide promote/demote per the 3-clean rule | M2 | 8 |
| F15 | Loud-skips: `scripts/checks/`-style ran==listed script (parse `--list` vs run summary) | M3 | 12 |
| F16 | Self-test fixture for F15 (one planted skip case) | M3 | 12 |
| F17 | Wire F15 into the e2e runner path (flake app post-run hook or check-modules leg) | M3 | 10 |
| F18 | If rate data says sync-flake: firefox-project `testIgnore` for sync.spec.ts with citation | M4 | 6 |
| F19 | Revision-mismatch warning in the `e2e-firefox` app (cached rev vs browsers.json) | M4 | 10 |
| F20 | If 3 clean runs: merge CI firefox into the blocking step (drop continue-on-error) | M4 | 5 |
| F21 | Gotcha #56 full text in docs/gotchas.md + AGENTS index row | M5 | 10 |
| F22 | DecodeForm sharp-edge gotcha + index row | M5 | 8 |
| F23 | verify-before-filing pass on the pill bug (source-level: confirm v0.13.2 latest still affected) | M6 | 10 |
| F24 | File the upstream issue (evidence: computed-style + hiddenAttr, repro snippet) | M6 | 10 |
| F25 | Boost-guard spec skeleton: emit → boosted page → assert no element lost its own attrs | M7 | 12 |
| F26 | Boost-guard: cover PolledRegion + events filter (the two known victims) | M7 | 12 |
| F27 | Boost-guard: run green ×2, receipt in plan | M7 | 8 |
| F28 | Stale-port guard: script (or preflight check) killing :18923–:18933 strays before a run | M8 | 12 |
| F29 | e2e/README serial-context note (fresh context per test even in serial describes) | M8 | 4 |
| F30 | Corpus/tail re-true for any docs growth in M5–M8 | M5–M8 | 6 |

```mermaid
graph TD
    A[train landed locally: 106/106 chromium] --> B[F01–F03: commit restored edits + plan]
    B --> C[F04: push — gotcha-50 protocol]
    C --> D[F05–F07: push-and-watch + first-run triage]
    D --> E{CI e2e green?}
    E -- no --> FF[fix-forward] --> D
    E -- yes --> F[F08: verify-remote-tags]
    F --> G[F09–F10: firefox-1555 provision]
    G --> H[F11–F13: 3× both-engine runs]
    H --> I{rate table: stable?}
    I -- "3 clean" --> J[F20: promote CI firefox to blocking]
    I -- flaky --> K[F18: scope sync.spec off firefox + F19 revision warning]
    J --> L[F15–F17: loud-skips gate]
    K --> L
    L --> M[F21–F22: gotchas #56 + DecodeForm]
    M --> N[F23–F24: upstream pill filing]
    N --> O[F25–F27: boost-guard spec]
    O --> P[F28–F29: stale-port guard + README note]
    P --> Q[(train complete: PR-red battery, measured engine lane, loud skips)]
    R[Parked 20%: pixel baselines / worker split / CSP guide] -.precondition-gated.-> Q
```

## Guards

- **No verschlimmbesserung:** no new abstractions, modules, or config surfaces. Land, measure, or write down — nothing speculative.
- **Daemon-revert guard (bite twice, shame):** after EVERY daemon auto-commit while this train is in flight, `rg`-scan the sentinel lines of every session file (`E2E_FIREFOX` in ci.yml, `30_000` in sync.spec, `attempt` in screenshots.spec, `role="img"` in layout.templ) — the daemon already snapshot-reverted two of these once.
- **Promotion rule:** firefox joins the blocking CI step only after 3 consecutive clean both-engine runs (one green run is not a rate — the lesson of the premature promotion).
- **Contract waits only** in every new spec; no sleeps outside retry backoff.
- **Push discipline:** pre-push hook may red on the documented API-vs-tag class — gotcha-50 protocol (hand-verified battery green, `--no-verify`, named in the message), never a silent skip.
- **History:** no rebase/squash of daemon commits while sibling sessions are active (gotcha 47).

## Execution log

- [x] F01–F03 (2026-11-08 06:3x): daemon-revert damage found and repaired (firefox CI promotion + sync ceilings + screenshot retry were snapshot-reverted; re-applied and sentinel-scanned), firefox CI landed in the ADVISORY shape (chromium blocking + firefox `continue-on-error`), plan + status report committed.
- [x] F04 (2026-11-08 07:0x): PUSHED (`e85b1613..0c371e35`, pre-push hook fully green — no gotcha-50 needed).
- [x] F05–F07 partial (2026-11-08 07:2x): first CI run 42454366652 — the e2e job's infrastructure WORKED first try (sibling clones, go.work rewrite, warm build, bun pin, browser installs, suite ran). One real failure: **bug #8** — the events page's "Auth stream" chip aria-label hid its visible text (`label-content-name-mismatch`, deterministic in the registry Chromium; the local Nix build never surfaced it — the engine lane paying for itself on day one). FIXED (`dashboardui/events.templ` label now contains the visible text) + pushed. The run's other red jobs (lint ×4, module-architecture, checks) are SIBLING in-flight work (auth-stream preset feature, huma drift test), not this train's.
- [x] F09–F10 (2026-11-08 07:1x): firefox-1555 provisioned into `/mnt/buildcache/playwright`; the flake app picks it (newest-first); the forest expression's `dbus.lib` output-quirk fixed (the last missing lib); launch verified (`TITLE: ok1555`).
- [x] F11–F14 (2026-11-08 07:3x): **the rate table —**

  | Run | Build | Result | Verdict |
  | --- | --- | --- | --- |
  | pre-plan both-engine | 1543 | 150 passed / 0 failed / 13 DNR | green-once |
  | 06:0x (post axe-fix) | 1543 | 186 passed / **11 failed** / 13 DNR | deterministic divergence |
  | rate run 1 | 1555 | 186 passed / **11 failed** / 13 DNR | identical set → revision-skew hypothesis DEAD |
  | rate run 2 (sync scoped off) | 1555 | 186 passed / **4 failed** / 13 DNR | stable: the four single-test divergences |
  | rate run 3 | 1555 | (pending — expected = run 2's set) | |

  Decision per the data: NOT flake — **deterministic cross-engine divergences in shipped artifacts**. Applied: `sync.spec.ts` scoped off the firefox project with a citation (F18 executed early); the four singles stay RUNNING and visible in the advisory lane; all five documented as TODO rows with per-item diagnosis leads (DataStar double-patch 2→4, OOB no-op, tab-2 SSE never-live, partial-vs-full Accept negotiation, SharedWorker cache-reset no-wipe).
- [x] F15–F16 (2026-11-08 07:4x): loud-skips gate `scripts/checks/check-e2e-ran-count.sh` + 8-case self-test (a spec with NO results vs an intended skip's skipped RESULT; `--expect-unrun N` ceiling).
- [x] F17 (2026-11-08 07:4x): wired into CI — chromium gate BLOCKING, firefox gate advisory, reports via `PLAYWRIGHT_JSON_OUTPUT_NAME`; actionlint green.
- [x] F21–F22 (2026-11-08 07:0x): gotchas #57 (element-bound JS dies under hx-boost; document-delegated listeners + `bubbles:true`) and #58 (DecodeForm zero-struct silence; browser-encoding rule) + AGENTS index rows. (#56 was taken by a sibling's zero-ULID gotcha — numbered past it.)
- [x] Rate run 3 (2026-11-08 07:4x): **4F/13DNR/186P — identical to run 2**; the divergence set is fully deterministic; the table above is final for this train.
- [x] F19 (2026-11-08 07:5x): revision-drift warning in the `e2e-firefox` app (cached build's revision vs `playwright-core/browsers.json`, with the re-provision command).
- [x] F23–F24 (2026-11-08 07:4x): upstream pill bug VERIFIED at source (master still affected, no `[hidden]` override CSS anywhere) and FILED: [LarsArtmann/go-health-dashboard#31](https://github.com/LarsArtmann/go-health-dashboard/issues/31); the bridges spec cites the issue number.
- [x] F25–F27 (2026-11-08 07:5x): `boost-guard.spec.ts` (2 tests) — the PolledRegion and the events filter form pin their own hx-target/hx-select, and the pins SURVIVE a live `/api/debug/emit`-driven swap (the bug-#5 class tripwire). Full chromium battery with it: **108/108, 1.4 min**.
- [x] F28 (2026-11-08 07:4x): `scripts/tools/e2e-kill-strays.sh` (the README kill ritual as one command, ports 18923–18941, fails loud when a port stays occupied; self-verified on an occupied port) + wired into the `e2e-firefox` app.
- [x] F29 (2026-11-08 07:4x): e2e/README serial-context note (fresh context per test even in serial describes).
- [x] CI follow-up fix (2026-11-08 07:5x, from run 42455839640's triage): the ran-count steps failed because `--reporter=line` REPLACES the config's reporters so the json report was never written → both steps now `--reporter=line,json` (verified end-to-end locally: file written, gate green); the unused conditional shell-lib source in check-e2e-ran-count.sh removed (the checks job's shellcheck leg flagged SC1091) — local `check-shellcheck` 144 scripts clean.
- [x] **CI GREEN (2026-11-08 08:0x, run 42456536460 @ c691104e): the `e2e` job passed ALL SIX steps on its second wired run** — sibling clones + go.work rewrite + warm build + bun install + both engine lanes + BOTH ran-count loud-skips gates. Job wall time 6 m 05 s (chromium blocking ~2.5 min, firefox advisory clean). **The firefox lane's CI result is clean run #1** (registry build, sync scoped off — the promotion rule needs 3). The run's remaining red jobs (lint ×4, module-architecture, checks) are SIBLING in-flight work (the auth-stream preset feature + huma drift test), verified not-train files. F08 is a no-op for this train: no tags were cut, nothing to verify-remote.
- [ ] F18 remainder: keep-or-verify the sync scoping after each divergence fix.
- [ ] F20: promotion flip after 3 consecutive clean CI firefox runs.

### Divergence resolution (2026-11-08 08:3x–12:0x, the follow-up session)

The four "deterministic firefox divergences" were investigated with page-level probes (fetch interceptor + `body.tee()` chunk logger; background-vs-fronted tab probes; sequential-tab probes) and **three of four were TEST-DESIGN artifacts, not engine bugs** — full root causes in gotcha 59 and the TODO_LIST row:

- **(a) setup-bundle "signal 4"**: the server-side `broadcastCount` is GLOBAL and one webServer backs BOTH playwright projects — chromium's posts landed 1/2 first, firefox's at 3/4; the chunk probe showed the client rendering exactly the server's patch values (`{5}`→"5", `{6}`→"6"). Client faithful; assertion over-absolute. FIXED: delta-based assertion + stream-liveness probe loop (`setup-bundle.spec.ts`).
- **(b) root OOB "3 items"**: chromium's serial run left 3 items in the shared store; firefox failed the fresh-`0 items` assumption BEFORE the OOB swap ever executed — the swap machinery was never implicated. FIXED: baseline-relative badge/list asserts + per-run unique item name (`root-htmx.spec.ts`).
- **(c) datastar multi-tab**: genuine headless-firefox properties — background-tab stream suspension AND no reliable concurrent SSE streams (with tab 1's stream open, tab 2's `@get` dies `NS_BINDING_ABORTED` and its replacement hangs; sequential tabs — close the first, open the second — connect on the FIRST probe). FIXED: sequential proof choreography, identical on both engines (`datastar.spec.ts`).
- **(d) admin pagination "50 of 65"**: chromium's modal test DELETED the seeded team60 filler (firefox saw "Showing 50 of 64"); content negotiation was never involved. FIXED: totals parsed from the page's own "Showing X of Y" + the modal test seeds its own victim via `POST /users` with the panel's `<meta name="csrf-token">` token (probe: cookie-as-header token = 403, meta token = 303 — nosurf masked-vs-plain) (`admin-behavior.spec.ts`).
- **(e) sync SharedWorker**: probed once (raw IndexedDB dump path prepared); still the open investigation — `testIgnore` stands with its citation.
- **Tooling fix found on the way**: the `e2e-firefox` app died under `set -e` when invoked from repo root (the revision-drift grep ran before `cd e2e`, missed `node_modules/`, errexit killed the app before playwright) — reordered, REPO_ROOT resolved once, drift check advisory. Receipt: `nix run .#e2e-firefox -- --list` works from repo root.
- **FINAL VERIFICATION RECEIPT (2026-11-08 12:0x): both-engine run over the four specs — 54/54 PASSED, 0 failed, 0 skipped, 54.7 s** (`nix run .#e2e-firefox -- --reporter=line setup-bundle.spec.ts root-htmx.spec.ts admin-behavior.spec.ts datastar.spec.ts`).
