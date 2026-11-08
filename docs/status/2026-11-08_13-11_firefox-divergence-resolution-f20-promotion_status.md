# Firefox-divergence resolution, F20 promotion, and the probe-driven root-cause wave (status report)

**Date:** 2026-11-08 13:11 CET · **Session:** full-execution of the remaining e2e-train items (the four "firefox divergences", F20 promotion, the (e) sync investigation, the `e2e-firefox` tooling fix) · **Predecessor:** the 06-14 browser-e2e report and the 06-26 SUPERB plan (§Execution log carries this session's receipts) · **Tree at end:** master = origin at `aecb54c2`, clean, both-engine battery 207/207, corpus 58,090 + this report's true-up, tail 12.

## a) FULLY DONE (verified, receipts in commits)

1. **All four "deterministic firefox divergences" root-caused and FIXED — three were TEST-DESIGN artifacts, not engine bugs** (probe receipts in `docs/gotchas.md` #59 + the TODO_LIST row + the plan log): (a) setup-bundle "signal 4 not 2" — the server-side `broadcastCount` is GLOBAL and one webServer backs BOTH playwright projects; chromium's posts landed 1/2 first, firefox's at 3/4; a fetch-interceptor + `body.tee()` chunk-logger probe proved the DataStar client renders EXACTLY the server's patch values (`{5}`→"5", `{6}`→"6"). Fixed delta-based with a stream-liveness probe loop (`setup-bundle.spec.ts`). (b) root OOB "3 items" — chromium's serial run left 3 items in the shared store; firefox failed the fresh-`0 items` assumption BEFORE the OOB swap ever executed (the swap machinery was never implicated). Fixed baseline-relative with per-run unique item names (`root-htmx.spec.ts`). (d) admin pagination "50 of 65" — chromium's modal test had DELETED the seeded team60 filler (firefox saw "Showing 50 of 64"; content negotiation was never involved). Fixed: totals parsed from the page's own "Showing X of Y" + the modal test seeds its own victim through `POST /users` (`admin-behavior.spec.ts`).
2. **Divergence (c) multi-tab — the one genuine engine property, mapped by three probes:** firefox SUSPENDS a background tab's stream processing (background second tab: zero deliveries across 3 probes; same tab fronted: delivery) AND headless firefox cannot hold two CONCURRENT SSE streams (with tab 1's stream open, tab 2's `@get` dies `NS_BINDING_ABORTED` and its replacement hangs; sequential tabs — close the first, open the second — connect on the FIRST probe). Fixed: sequential proof choreography, identical on both engines (`datastar.spec.ts`, probe receipts in the spec comment).
3. **F20 EXECUTED — firefox PROMOTED TO BLOCKING CI** (commit `1b75074e`): three clean CI firefox observations (42456536460, 42457053291, 42477578702; the intermediate b86fd0cb run SKIPPED the lane — chromium red on a mid-flight spec state, no firefox observation either way, named honestly). Deliberate deviation from the plan's literal "merge into one step": the lanes stay SEPARATE invocations because one webServer set backs every project in an invocation and cross-engine shared server state is exactly the race class gotcha 59 documents — each engine keeps a fresh server set and its own ran-count gate. actionlint green.
4. **`e2e-firefox` flake app silent-death bug found and FIXED** (flake.nix): invoked from the repo root, the revision-drift grep ran BEFORE `cd e2e`, missed `node_modules/`, and `set -e` killed the app before playwright ever started (the "no output, exit 1" runs that opened this session). Fixed: repo root resolved once up front, kill-strays via the resolved root, the drift check runs after `cd e2e` and is advisory (`|| true` guards). Receipt: `nix run .#e2e-firefox -- --list` works from any CWD.
5. **Divergence (e) sync — the investigation EXECUTED with a split verdict:** an IndexedDB dump probe DISPROVED the headline claim (the backendId cache reset works in isolation under firefox: events 2→0, backendId A→B, meta persisted — exactly the contract); an evidence run with the `testIgnore` temporarily lifted proved the REMAINING firefox sync failures are real and specific — online flush delivers (121), offline reads (352), reset-in-spec-context (392) — 3F/11P. The ignore was RESTORED with the sharper citation (`aecb54c2`); the TODO row now carries the split verdict.
6. **FINAL VERIFICATION: full both-engine battery 207/207 PASSED, 3.2 min, zero failed, zero skipped** (`nix run .#e2e-firefox`, no filters — every spec on both engines). The battery's screenshot refresh is committed (`1316b8a6`) after pixel spot-checks (dark + light overview renders: layout intact, live-data drift only).
7. **Memory landed:** gotcha 59 (the full class: shared webServer, delta assertions, self-seeded victims, the csrf-meta token rule — nosurf rejects the masked cookie value 403 while the meta token passes 303 — and the firefox stream properties) + the AGENTS index row + the e2e/README "Engine scope and shared-state discipline" section + the TODO_LIST divergence row rewritten with per-item verdicts + the plan's divergence-resolution log section with the 54/54 and 207/207 receipts.
8. **CI receipts this session:** 42477578702 (@ `1316b8a6`, the final divergence-fix tree): **e2e job SUCCESS** — both lanes + both ran-count gates. 42478653025 (the promotion commit) was CANCELLED (superseded by the daemon's 9f6d1e70 push); the promoted shape's verification rides run 42479310064 (@ `aecb54c2`, queued at report time). The sibling reds (lint ×4, module-architecture, checks) remain the sibling lane's (auth-stream preset + huma drift test) — not touched.
9. **Corpus/tail trued twice** (58,072 → 58,088 → 58,090, the second +2 for sibling drift above the freeze) with the ledger sentence appended per the round-46 §d2 ritual; conflict-markers green (1,823 files), shellcheck clean on the e2e scripts, both ratchets green.

## b) PARTIALLY DONE

- **The promoted CI shape's own verification** (run 42479310064 @ `aecb54c2`) was QUEUED at report time — the firefox lane now reds the job if it fails, which is the promoted behavior; watch with `gh run watch 42479310064` or `nix run .#push-and-watch`.
- **Divergence (e)**: root-caused enough to keep the ignore honest (the reset mechanism works; the spec-context failures are real) but the actual firefox SharedWorker/route-interception failure mechanism inside the spec context is UNDIAGNOSED — the three failing tests have not been probed individually.
- **The 9f6d1e70 daemon push** carried a stale `e2e/probe-sync.mjs` to the remote before I deleted it locally (`aecb54c2` removes it) — the remote is clean at aecb54c2; noting the two-commit churn.

## c) NOT STARTED

- Upstream filing of the headless-firefox stream findings (background-tab suspension + concurrent-stream `NS_BINDING_ABORTED`) to mozilla/Playwright — evidence captured, no report filed (owner-gated: verify-before-filing pass first).
- The go-health-dashboard#31 upstream response watch (pre-existing).
- The M14-F63 sibling item and the sibling CI heal (lint ×4 + module-architecture + checks) — theirs, untouched per the handoff.

## d) TOTALLY FUCKED UP

1. **An INTERMEDIATE, UNVERIFIED spec state reached the remote and redd CI** (run 42475825677 @ b86fd0cb): I iterated on the specs in the working tree while the auto-commit daemon swept and something pushed mid-flight — CI ran my cookie-CSRF variant (403, probed wrong) and the mid-iteration multi-tab assert. The exact class the plans warn about ("never test yesterday's binary"), inverted: never SHIP yesterday's spec. Mitigation now in place: every fix wave was verified 207/207 before its push, and the final tree's runs are green — but the red run sits in the CI history caused by my own churn.
2. **The TODO_LIST divergence-row edit was daemon-snapshot-reverted THREE TIMES** before I changed tactics: edit + `git add` + `git commit` in ONE shell invocation leaves no window. The plan's execution-log section was lost twice the same way. Both finally landed (`8eb3cff7`, `b8618fc9`) — but three losses is three too many; the atomic pattern should have been the default from the first loss (the context literally documented it).
3. **The corpus baseline needed two re-trues** (sibling drift raced both pins) — expected under concurrent sessions, but my first pin didn't account for the daemon's in-flight doc writes landing between measure and commit.
4. **My first client probe was self-sabotaging**: the fetch wrapper called `getReader()` twice (a lock error) which CAUSED a bogus second `/ds/events` fetch, nearly steering the diagnosis toward a phantom "SDK retry". The clean non-invasive probe (log-only interceptor + `tee()`) was the second attempt.
5. **The csrf cookie-value guess cost a full both-engine run**: I shipped the cookie-as-header variant (403 on both engines) before probing; the meta-token probe (403-vs-303) settled it in one round. Probe first, ship second — the session's own lesson, violated once.
6. **probe-sync targeted the WRONG PORT** (18940 from a stale context line; the sync tests run against `e2e/server` on :18923) — one wasted probe round.

## e) WHAT WE SHOULD IMPROVE

1. **Edit+commit atomically under this daemon** — one shell call, no lingering modified files. Every loss this session (three TODO losses, one plan loss) came from edits that rested in the tree between two daemon cycles.
2. **Never let spec iterations rest in the tree uncommitted-and-unverified**: a red CI run from shipped intermediates is worse than a delayed push. Verify the wave (full battery), commit, push, THEN iterate the next wave.
3. **Probe before shipping a mechanism guess** (the CSRF token shape) — the probe is one round; a wrong guess is a full battery round plus a red run.
4. **Run-count sentinels after every daemon commit** worked exactly as designed this session (caught the swept-probe and the formatter tug-of-war within one command) — keep it mandatory; the structural `git show HEAD:<path> | rg <sentinel>` form, never counts or label-paths.
5. **Formatter ping-pong between the pre-commit hook and stale staged copies** (the LD_LIBRARY_PATH array collapsed/re-expanded) — resolve by pinning to HEAD's form and re-committing; a `git diff --cached` glance after the hook runs would catch it earlier.
6. **The both-engine app runs both projects against ONE server set** — that is now a DOCUMENTED property (gotcha 59 + README) with CI deliberately running per-project invocations; keep that asymmetry (CI = fresh servers per engine, local = shared) in mind when a test passes locally but reds CI or vice versa.
7. **Load-flake discipline confirmed again**: the sync chromium failure under parallel load (CI-watch + builds running) vanished on the quiet rerun — rerun-once-before-triage (gotcha 41's pattern) extends to Playwright polls.

## f) Up to 50 next things (ranked; 1–3 are this session's tails)

1. **Watch run 42479310064** (@ `aecb54c2`) — the promoted firefox-blocking shape's first verification; triage if the firefox lane reds (it now fails the job by design).
2. **Probe the three firefox sync failures individually** (online flush 121, offline reads 352, reset-in-context 392) — decide fix-vs-permanent-divergence with an owner verdict; the ignore's citation is already evidence-backed.
3. **File the headless-firefox stream findings** (background-tab suspension; concurrent-stream abort) upstream after a verify-before-filing pass — the probe scripts are deleted, so re-derive from the gotcha text or resurrect from git history (`8eb3cff7`'s plan section carries the numbers).
4. Commit the remaining corpus true-up for THIS report (+its ledger sentence, same change — done as part of this report's commit).
5. The sibling CI heal (lint golines ×3, errname, huma tidy, root-size ratchet, checks) — the sibling's §f1, still red on every run.
6. M20 docs-health follow-through: archive the settled 2026-11-06 set; the tail sits at 12 vs budget 3 (advisory) — the next archive pass is due.
7. The `audit_max` re-pin decision (docs-gates audit-all red at 393 vs pinned 322) — owner question, parked in TODO.
8. F18 remainder bookkeeping: the sync scoping is now evidence-backed — fold the TODO row's (e) wording into the final verdict once item 2 lands.
9. Boost-guard spec extension: the filter-form and PolledRegion pins survive `/api/debug/emit` swaps; consider an axe-style sweep over EVERY self-fetching element in both UI modules (a generated tripwire, not per-element specs).
10. The `e2e/README` coverage matrix lists 13 specs — the tree has 16 files; true the matrix.
11. Screenshot review cadence: the battery refreshes 27 PNGs per full run — decide whether the commit-ritual (spot-check + commit) stays manual or gets a drift-budget gate.
12. The `e2e-firefox` app's revision-drift check is now advisory-only — consider `--strict` mode for CI parity once the registry build is guaranteed.
13. CI's chromium lane runs 108 tests vs local 106 — reconcile the count difference (retry artifacts? project composition) so the ran-count gates compare honestly.
14. The sync spec's 30s boot-catch-up ceiling raced load once this session — consider raising to 45s or adding a load-guard skip.
15. admin-behavior's `waitForTimeout(400/500)` blocks — contract-wait conversion (the plan's own guard says no sleeps outside retry backoff; these predate it).
16. loginpage-webauthn remains firefox-ignored (virtual authenticator support) — M18's Playwright WebAuthn E2E covers chromium only; revisit when Playwright ships firefox virtual authenticators.
17. Deeper example flows beyond smoke (the parked 20%): setup-demo admin write actions, dashboardui time-travel round-trip under boost.
18. Pixel-regression baselines (parked, precondition-gated on a pinned runner image).
19. CSP guide section (parked docs rider).
20. The worker-split question (suite is 3.2 min both-engine — serialization is not the bottleneck; park permanently unless the suite triples).
21. Route-posture manifest: add the e2e servers' debug endpoints (`/api/debug/*`) to a posture table so the ungated debug surface stays explicit.
22. `check-e2e-ran-count` self-test: add a case for the `--expect-unrun` ceiling under firefox's DNR set.
23. The plan's §"Divergence resolution" entry says "final green state recorded in the TODO_LIST row and the next execution-log entry" — the 207/207 receipt is in; cross-link it from the parent plan's execution log.
24. Gotcha 59's csrf-meta rule is adminui-specific — check whether dashboardui/setup mounts need the same token pattern documented for consumer POSTs.
25. Consider a `docs/guides/browser-e2e.md` distilling the battery's contracts (serial modes, shared-state discipline, engine scope) from the README+gotchas pair.
26. The datastar-demo's `/api/events` live-only feed + the aborted-stream behavior: worth a server-side SseKeepalive ping probe (would distinguish dead vs stalled client streams in future diagnostics).
27. Add a battery smoke test for the `e2e-firefox` app itself (repo-root invocation, `--list`) so the CWD-death class stays fixed structurally.
28. Prune `docs/status/` tail via the archive pass (item 6) before the next report adds a line — the ratchet is advisory but the cadence rule says archive-before-add at >15.
29. The 06-26 plan's F18 remainder closes with (e)'s verdict — mark it done in the plan log when item 2 lands.
30. Upstream watch: Playwright issues on firefox background-tab streaming — subscribe/search so the environment limitation gets an upstream link in gotcha 59.
31. Check whether `bridge`/`dashboard-live` specs assume fresh server state anywhere (the gotcha-59 sweep covered four files; a grep for absolute assertions across the remaining 12 specs is cheap insurance).
32. `sync/jstests` — the SharedWorker's node-side test suite: add the reset-in-context case (the spec-context failure mode, node-side, cheap).
33. The admin self-seed uses `Date.now()` emails — collision-safe today; a suffix counter would be belt-and-braces if two runs start within the same millisecond (paranoid).
34. `setup-bundle`'s console-guard regex `/0|1|2/` — with the counter now unbounded, re-derive whether that assertion still means what it says (it passes because the fresh page renders 0; document or tighten).
35. The three sibling-clone steps in ci.yml e2e could share a cache (M14-F63, the sibling's item) — blocked on their lane.
36. Tempting-but-parked: parallelizing the two CI engine lanes into one job with two steps is DONE (separate steps); a matrix strategy would give per-engine status checks — weigh against the webServer duplication cost.
37. The `#users-table` fixture: seeding 65 users per server start costs the admin suite ~2s — a seed-count env knob would let tests shrink it if the suite grows.
38. Screenshots.spec's 3× goto retry — proven necessary once this session? It never retried (green first pass); keep.
39. Document the probe techniques (fetch interceptor, `tee()` chunk logger, IndexedDB dump, bringToFront sequencing) as a reusable snippet file under `e2e/probes/README.md` — the scripts themselves stay out of the tree.
40. The `checks` job's corpus-ratchet red (sibling) will vanish when their docs settle — verify it's not my report's +N lines by re-truing in this report's commit (done in the same change as this file).
41. AGENTS.md Quick Reference's battery row could name `nix run .#e2e-firefox` as the both-engine entry (it names the battery; the app is discoverable but one clause would help).
42. The firefox lane's CI runtime (~2.5 min) vs chromium (~2.5 min) — if the promoted lane makes the e2e job the longest pole, consider splitting the job to keep branch feedback fast.
43. Re-run `docs-gates` battery after this report (freshness + annotate verdicts on the new row wording).
44. The TODO_LIST "Post-change verification battery" row's bench-spike remainder — still waiting on the twice-quiet window (load was 20+ all session).
45. When go-datastar ships the next client, re-run the multi-tab probe — the concurrent-stream property may be SDK-influenced (fetch vs EventSource transport), not purely browser.
46. `adminui` user-create toast + redirect flow: the e2e now covers create→detail via API; a UI-path create test (fill the form, submit) would cover the form itself.
47. The browser-e2e skill (`cqrs-htmx/.agents/skills/`) — fold gotcha 59's rules into its next revision.
48. Check `git log --all` for any OTHER mid-flight daemon commits pushed this session (b86fd0cb and 9f6d1e70 were both daemon sweeps that shipped) — confirm nothing else unverified shipped.
49. The `e2e/README` CI section now says both lanes blocking — verify the runs-on image's firefox install stays pinned (browsers.json drift warning exists locally; CI installs fresh each run — fine, but note it).
50. Celebrate the number: **seven shipped client-facing bugs found by this battery across its first two days, four false "engine bugs" debunked, and the engine lane is now a blocking gate** — the battery has paid for itself several times over.

## g) Owner questions (max 3)

1. **Mid-flight pushes:** b86fd0cb (my intermediate spec state) and 9f6d1e70 (a stale probe script) both reached the remote WITHOUT my pushing them — a sibling session or an auto-push mechanism is pushing daemon commits. Should I treat the remote as shared-mutable (gate my pushes on a fresh `git fetch` + rebase-read each time), or can the pusher be identified and told to hold during active trains? I cannot see who pushed from this session's vantage.
2. **Divergence (e) funding:** the sync-on-firefox failures (3 tests) are real but scoped-off with an evidence-backed citation. Invest roughly a half-session in probing firefox's SharedWorker/route-interception behavior for a fix (re-including the suite on the now-BLOCKING lane), or accept the permanent documented divergence and spend the time elsewhere? My lean: accept-and-document — the sync surfaces are chromium-proven and the firefox failures are environment-shaped, but the blocking lane makes this a real risk-transfer decision.
3. **Upstream appetite for the firefox stream findings:** file to mozilla/Playwright (background-tab stream suspension + concurrent-stream abort, with repros) — worth the filing effort now, or fold into the next natural firefox work? It is evidence-rich (probe receipts) but environment-specific and non-blocking for the library.

## h) Verification receipts

- Full both-engine battery: **207/207 passed, 3.2 min, EXIT=0** (`nix run .#e2e-firefox`).
- Four-spec verification ladder: 49/54 → 51/54 → 53/54 → **54/54** (the chase fixes: csrf meta token, multi-tab reload, sequential choreography).
- Sync evidence run: firefox 3F/11P with the ignore lifted; quiet chromium rerun 7/7 (the load-flake rerun-once rule).
- CI: run 42477578702 e2e SUCCESS on the final tree; promotion commit pushed `1316b8a6..1b75074e`; cleanup pushed `9f6d1e70..aecb54c2`; shape-verification run 42479310064 queued.
- actionlint green on ci.yml; conflict-markers green; shellcheck clean; corpus 58,090 + tail ratchets green (pre-report); pre-push hook fully green on both pushes.
- The firefox lane's three clean CI observations: 42456536460, 42457053291, 42477578702.
