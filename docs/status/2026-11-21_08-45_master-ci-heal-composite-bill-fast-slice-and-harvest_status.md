# Status report — master-CI ratchet heal, the composite bill resolved by name, the tidy gate + fast e2e slice landed, the QUERY-ladder red root-caused, and the 03-09 §f harvest (2026-11-21 08:45)

**Scope line — what this session VERIFIED:** the overnight CI reds triaged to root cause and healed on master (both `checks` legs: corpus +133 of unledgered `add7d694` docs growth at `ac34e337`, tail 20-vs-19 from the sibling's 03-15 report at `40a9bd53`; the `lint (.)` red at `40a9bd53` proven to be the sibling's in-flight httperr lane — `golangci-lint run` in `httperr/` at their worktree state prints **0 issues**, so it self-heals untouched); the composite's 3 unknown failures NAMED and each resolved: `corpus-budget-ratchet` (the treadmill — re-trues per docs commit), `cqrs-lint-gate-self-test` (a stale **4.20.1 `~/go/bin/cqrs-lint` shadow binary**, built 03:17 by a sibling — retired; the profile dev build is the only instrument again; gate green **26/26** with zero findings), `branching-flow` (**+11 new findings**, ALL in sibling lanes' committed waves — legalparty #45 ×7, identity-model ×2, usermgmt service-core ×2 — each site read, all standing reject-classes, re-pinned **212 → 220** with a triage-decisions row, gate green **+0**); the isolation `root FAIL (vet)` discrepancy RESOLVED by full re-run (**18/18 modules build+vet standalone green** — the transient class, now gotcha 68); the CHANGELOG `unsafe-hashes` receipt verified in history (`git log -S "unsafe-hashes" -- CHANGELOG.md` → `add7d694`); the hermetic tidy gate landed and **fired live on a real push** (32/32 tidy-clean); the fast e2e slice landed and proven standalone (**20/20 in 45.6s**); the QUERY-ladder triple red root-caused by a throwaway browser probe (both requests fire in order, QUERY first at rest; the real test **green at load 59** — the fill's 350ms debounce twin races the Enter QUERY for the same URL and `waitForResponse` catches whichever lands first under load; test fixed to settle the debounce, probe deleted); erraudit strict receipt intact (TOTAL=364 ×3, no edit needed); libs-report-count green at 22; zero mixed templ-components pins; README quickstart green; the runbook's sibling-lane input paragraph verified surviving. **NOT verified:** the newest CI runs at `cca44a5a`/`d7e9d744` were still `in_progress` at report time (`44881625677` pending; `44881306509` cancelled by the push race — superseded, not red); the full check-modules composite was not re-run end-to-end as ONE green receipt (each of the 3 named legs verified green individually); W3/W4 and the per-row owner packets remain parked per the no-re-ask directive.

## a) FULLY DONE

1. **Handoff state adopted; the daemon reality honored.** The report set (03-09 report + both baselines + ledger row) had been swept by the daemon (`f5c8e782`); the CHANGELOG `unsafe-hashes` Housekeeping receipt verified in `add7d694`; the httperr/flash uncommitted files identified as a sibling lane's in-flight work and left alone — their worktree state lints clean, so the CI `lint (.)` red self-heals on their next capture (it did: the tree later reads clean).
2. **Overnight CI reds healed and pushed.** The corpus ratchet (`64364 > 64231`, then `64617 > 64480`) and the tail ratchet (`20 > 19`) both got the documented remedy: ledger sentence naming the sibling share (07-01 precedent — "named not numbered, their lane"), baselines re-trued same-change, TAIL 19 → 20. Landed via daemon capture `e176eb1e`, pushed through `af139cba` with green pre-push gates (1061 requires, 0 unpublished).
3. **The isolation-leg discrepancy is closed with evidence.** `bash scripts/checks/check-module-isolation.sh` full re-run: all 18 modules build and vet standalone, `✓ All modules build and vet standalone (GOWORK=off)`. The previous session's `root FAIL (vet)` was the shared-worktree transient (a sibling's mid-flight httperr edit inside the vet window). Mechanism + verdict rule landed as **gotcha 68**: a local-gate red whose exact command then passes green on the same tree is a transient — re-run once before triaging; a red surviving one re-run is real; CI has no such transients.
4. **The composite bill is fully paid, by name.** Run 2's three failures: (i) corpus ratchet — treadmill semantics, healed by re-true; (ii) cqrs-lint gate — a sibling's 03:17 build installed a **4.20.1 dev-build at `~/go/bin/cqrs-lint`** that shadowed the profile binary and tripped the gate's v4.22.2 floor guard (the 2026-11-17 shadow-binary class resurrected); retired (trash, recoverable), `nix run .#check-cqrs-lint` re-run → **all 26 candidates pass strict, zero findings**; (iii) branching-flow — `+11 added, −2 removed, ~174 modified`: every site spot-checked (`gleif.go` wire twins + functional-options plumbing, `profile.go` flat wire vocabulary, `identifiers.go`/`address.go` constructor params, `commands.go:349` the sanctioned B59 pointer-embed, `events.go:144` the golden-pinned payload family, `service_core.go` composition-root structs) — all standing reject-classes, zero logic defects; re-pinned **212 → 220** with the dated triage-decisions row (commit `43018794`), gate green `+0 added`.
5. **The hermetic go-mod-tidy gate is landed, tested, and live.** `scripts/checks/check-go-mod-tidy.sh`: `go mod tidy -diff` per go.work member under `GOWORK=off` (the workspace-mode mask is exactly why the seven-job go.sum red read green locally), offline-safe, drift named per module; **4-case fixture self-test green** (drift named rc=1, clean rc=0, missing go.work rc=2, go.mod-less member named); shellcheck clean; real workspace run **32/32 tidy-clean**; wired into the pre-push hook after release-train + version-drift.
6. **The hook layering trap found and fixed.** The first push after wiring ran the OLD hook signature — `.git/hooks/pre-push` was a stale Nov 15 copy (`core.hooksPath` unset → `.git/hooks` fallback), and the template of truth lives at `scripts/hooks/pre-push.template` (the `.githooks/` copy I first edited is a deployment, not the source). Synced all three layers via `install-git-hooks.sh --force` (the installer's refuse-and-diff then force protocol), committed the template (`cca44a5a`), and the next push **fired the new leg live**: `✓ tidy-gate: 32 module(s) tidy-clean` + the new green signature printed.
7. **The fast e2e slice is landed, wired, and proven.** `scripts/tools/e2e-fast-slice.sh` + flake app `.#e2e-fast-slice` (kill-strays → chromium → loginpage-golden + admin-behavior + setup-bundle); the train-bump runbook now names it REQUIRED for render-touching family bumps (the §g.1 process gap). Standalone proof at the stable tree: **20/20 passed (45.6s)**.
8. **admin-demo carries the auto-fit-grid CSP carve-out on BOTH mounts.** `adminSecurity()` rebuilds the recommended stack (recovery, security headers, nonce) with a runtime-computed `'unsafe-hashes' 'sha256-…'` grant from the `tcAutoFitGridStyle` constant — the setup-demo pattern, **published API only** (hermetic build + vet green), and the tenant-admin mount gains the security layer it previously lacked entirely. Verified live in the slice's console guards (green).
9. **The library-level CSP knob was built, tested, and deliberately reverted.** `Config.CSPBuilder` on adminui + `RecommendedSecurityStackWithCSP` in root + both module tests green — then reverted with reason: adminui pins published root and admin-demo pins published adminui, so unpublished-API usage reddens CI's hermetic legs until a root+adminui train (the gotcha-30 root-tag-gap class), and a train is another active lane's territory today. The demo-side fix ships the user-facing value now; the knob is a train-gated TODO row.
10. **The QUERY-ladder triple red is root-caused and fixed, not band-aided.** Three slice failures (confirm-modal ERR_ABORTED, QUERY-method assertion, pagination) had one real core: the QUERY test's `waitForResponse` matches `/admin/users/search` by URL only, and the input's 350ms debounce schedules the POST twin against the SAME URL — under load the twin's response wins the race. Probe evidence (throwaway spec, deleted after use): at rest BOTH requests fire in order **QUERY first**, and the real test **passes green at load 59**. Fix: settle the debounce (600ms) before pressing Enter, keeping the strict method assertion. Two diagnostic misses owned in §d2.
11. **The 03-09 report's §f is harvested.** Four NEW TODO_LIST P2 rows (adminui carve-out split verdict, the tidy leg, the fast slice, the sync load-flake posture); the rest resolved: §f1–5 executed this session (CI/ratchet/composite/isolation/CURRENT_FOCUS), §f6–8/11 landed (agents-notes receipts, gotcha 68, screenshot note in both spec headers + AGENTS, production-readiness § 10), §f24/25/26/29/23/27/28/32/39/42/44–46 deduped onto existing rows with citations (PR#101 filed+merged, upstream-asks row (g)/(h), D18 verdict, blanket-row coverage), §f35/37/38/44/50 verified green in passing (uniform pins, cqrs-lint 26/26, CSS class-sets exact ×2, libs-count 22, runbook paragraph survived).
12. **CURRENT_FOCUS trued; owner questions recorded, not re-asked.** The stale carryover line rewritten (erraudit done 364 ×3, composite resolved, archive still owed at tail 20); the 03-09 report's §g.2 (gate retry-tolerance) and §g.3 (the migration D1/D2 contradiction) added to Waiting-on-owner per the no-re-ask directive; §g.1 answered by action (the agents-notes receipts section).
13. **The 20-47 report's §g.2 and §f5 annotated done** (`cc7f3bfd` — the true-1118 landing; §a3 already carried the full inline resolution; both open forms struck with the gotcha-10/67(e) pointers).
14. **The go-appkit treadmill wave swept and pushed.** Strict train gate red on push (4 requires at v0.12.0, v0.13.0 published — gotcha 39's tail-grows-while-you-sweep): `bump-dep 'larsartmann/go-appkit$' v0.13.0 --commit --lint` — all four consumers' tidy+verify+build+vet+lint legs green; the sweep commit died in the hook (load starvation) and rode daemon captures (`769ad2cb` + `d5adf8ef` — the second also carrying a sibling's go.work 1.27.4 floor bump, theirs not mine); `--refresh-cache --strict-lag 0` → **0/0/1063**; pushed green.
15. **The docs-gates battery's annotate leg driven to CLEAN.** Two rule-F findings (the glued-suffix form of `git restore` in my 03-09 report, the sibling 01-49 report's quoted plural form) reworded; `docs-annotate: 222 file(s), 1514 struck item(s), 0 finding(s)`; re-true + push `d7e9d744`.

## b) PARTIALLY DONE

1. **The check-modules composite has no single end-to-end green receipt at the final tree.** Each of the three named failures is individually verified green (corpus within 64663; cqrs-lint 26/26; branching-flow +0), but the 106-check composite takes long enough under load 60–110 that re-running it whole risks new worktree-transients; the honest state is "every named leg green, composite receipt pending a quieter window."
2. **The docs-gates battery reads 2 of 5 green.** routing PASS, annotate CLEAN (my repairs); freshness red is the SIBLING's newest report (03-15, their active pareto lane — 5 unharvested §f items matching only generic tokens); audit-all and cadence reds are the standing documented debts (the TODO P2 top row; tail 20 vs absolute budget 3). No new debt added by this session.
3. **The library CSP knob exists as history, not code.** Implemented + tested green, reverted in the same session; the TODO row carries the split verdict and the train-gated landing recipe.
4. **The archive round remains owed at tail 20** (the ledger's loudest standing item); this session drained §f routes but did not archive reports.
5. **CI verdicts for the final pushes unconfirmed** (`44881625677` in progress at report time; the run covering `cca44a5a`/`d7e9d744` is the one to watch; `44881306509` was cancelled by the push race — superseded, not a red).

## c) NOT STARTED

1. **The three §g questions below** (owner-gated; recorded in CURRENT_FOCUS, not re-asked via tool).
2. **W3/W4** (metaengine read-tier cutover + v5 cut) — owner-gated on the D1/D2 contradiction (§g.3), untouched.
3. **adminui `Config.CSPBuilder` + root `RecommendedSecurityStackWithCSP`** — rides the next root+adminui train (TODO row has the recipe).
4. **The docs-gates audit-all drain** (standing TODO P2 top row) and **the tail archive round** (tail 20 → the absolute budget of 3; the settled 2026-11-18/19 set is the named candidate).
5. **The sibling 03-15 report's §f harvest** — their active pareto lane's duty (the freshness gate names their 5 items; `<!-- fresh:skip -->` or harvest is their call).

## d) TOTALLY FUCKED UP

1. **I re-committed the exact sin I was executing the receipts for.** The 03-09 report's §d3 documented "battery started before the rebuild was live" — and this session, hours later, edited `examples/admin-demo/main.go` WHILE slice run 1 was executing; the mid-edit window broke the webServer compile on restart and produced the ERR_ABORTED confirm-modal red. Rule now written into the slice's own docs path: finish edits → verify the build → THEN run. Cost: one wasted 2.5-minute run and a false failure name.
2. **Two self-inflicted diagnostic misses burned ~30 minutes of wrong theorizing on the QUERY triage.** (a) My first manual server ran with `GOWORK=off`, so it rendered the STALE published adminui (GET filter form, no cqrs-query) — I nearly concluded the ladder had been deleted from the panel, while it was the gotcha-62 stale-proxy class I had LEDGERED in triage-decisions.md earlier that same day; (b) the "workspace-mode" retry never bound (`:18999` already in use) and I read the stale server a SECOND time before the bind-failure log corrected me; killing my own stray then needed `kill -9` escalation. The probe spec finally settled it with in-browser evidence — the right tool, used two wrong-server detours late.
3. **One `git checkout --` where policy mandates `git restore`** (reverting my own root/adminui knob changes — zero damage, my authorship, but the letter of the safety rule; noted, not repeated).
4. **A TODO-hygiene RULE4 violation shipped in my own harvest**: the `**Updated:**` header paragraph exceeded the 300-char ceiling — the gate caught it (working as designed) and the fix was a reword, but writing an over-long header while harvesting is the exact rot the gate exists to stop.

## e) WHAT WE SHOULD IMPROVE

1. **Make "build-then-run" mechanical for e2e work:** the fast-slice script should compile the served modules (`go build ./...` in the webServer dirs) BEFORE killing strays and starting playwright — turns the §d1 class into a fast, loud pre-flight instead of a mid-run ERR_ABORTED.
2. **bump-dep `--commit` under load needs `BUILDFLOW_MAX_CONCURRENCY=1` by default** — the sweep commit has now died to hook starvation twice in two days (gotcha 51); the verification legs all pass, only the ref-update loses the race.
3. **The hook layering (template → `.githooks/` → `.git/hooks/`) needs a byte-match stage** in check-modules (`install-git-hooks.sh --verify` exists) — this session proved a template edit silently misses the installed copy until manually synced.
4. **Audit the battery for the waitForResponse-by-URL race class:** any spec matching a URL that a debounced twin also hits is one load spike from this session's triple red; the QUERY fix (settle-then-assert) is the pattern.
5. **`e2e-kill-strays.sh` should sweep the full demo-port range** (my :18999 stray survived it; widen or document the port list).
6. **The probe pattern deserves promotion:** a throwaway `*-probe.spec.ts` that captures form attrs + registered extensions + request methods + console errors settled in one run what three theory passes could not — a reusable `e2e/tests/_probe.template.ts` (gitignored or deleted-after) would make the next triage start at evidence.

## f) Up to 50 things we should get done next

*(ordered roughly by impact; [owner] = owner-gated, [lane] = another lane's active surface, [watch] = upstream/trigger-gated, rest = session-actionable; dispositions from this session's harvest marked)*

1. Confirm CI green at `d7e9d744` (run `44881625677` was in_progress at report time; `44881306509` cancelled by push race).
2. One full check-modules composite green receipt at a quiet window (all three prior failures verified green individually).
3. The tail archive round (tail 20; the settled 2026-11-18/19 reports are the named candidates — `check-rows --strict` + `bare-rows` BEFORE any `git mv`).
4. [lane] Harvest the sibling 03-15 report's 5 unharvested §f items (the freshness gate's named red; their pareto lane owns it or marks `<!-- fresh:skip -->`).
5. adminui `Config.CSPBuilder` + root `RecommendedSecurityStackWithCSP` — land with the next root+adminui train (code existed this session; recipe in the TODO row).
6. Spec-race audit: sweep the battery for waitForResponse-by-URL patterns where a debounced twin shares the URL (the QUERY fix is the template).
7. Build-then-run pre-flight inside the fast-slice script (§e1).
8. Hook byte-match stage in check-modules via `install-git-hooks.sh --verify` (§e3).
9. Widen `e2e-kill-strays.sh` to the full demo-port range (§e5).
10. `BUILDFLOW_MAX_CONCURRENCY=1` default for bump-dep commit legs (§e2).
11. erraudit blocking-promotion watch — the restarted window closes **2026-12-21** (TOTAL=364 receipt; existing P2 row).
12. [owner] §g.2 (below) — gate retry-tolerance vs fail-loud+gotcha-68 posture.
13. [owner] §g.3 (below) — the migration D1/D2 contradiction unblocking W3/W4.
14. [owner] §g.1 (below) — daemon-commit message receipts: amend vs docs-only.
15. e2e sync offline-reads posture (TODO row: raise the 30s ceiling or document the retry posture — CI already retries=1).
16. coverage-gate receipt on the current tree (existing blanket row; last full receipt pre-dates today's trains).
17. docs-gates `audit-all` drain (standing TODO P2 top row; archived reports replay, so the drain annotates individual items).
18. v4.19.0 wave planning (QUERY W4/W5 receipts ride it; D18 verdict is the recorded recommendation).
19. bench-spike re-pin on a verified-quiet window (twice-quiet rule; advisory-only).
20. [watch] templ-components PR#101 release → sweep + retire adminui's vendored popover CSS same-change (existing P2 row).
21. [watch] go-cqrs-lite#113 kv constructor → bump + delete usermgmt's view-cache adapter (existing upstream-asks row (d)).
22. [watch] storage v4.12.0 tag → flip the `StateShape` sentinel test (existing row; fix on upstream master, untagged).
23. [watch] CI security job `go-version: 1.27.4` → `go-version-file` when fleet nixpkgs ships ≥1.27.4 (existing P1 row; NOTE: a sibling bumped `go.work` to 1.27.4 mid-session — their lane, likely this row moving).
24. [owner] M2M D1–D4 per-row surfacing (existing row + packet).
25. [owner] adminui D1–D3 per-row surfacing (existing row + packet).
26. [owner] §g.1 20-47: repin the NixOS system templ (existing CURRENT_FOCUS row).
27. [owner] §g.3 20-47: K2 multi-instance follow-up (existing CURRENT_FOCUS row).
28. [owner] Fleet-cache hygiene probe of `cache.home.lan/*` (existing CURRENT_FOCUS row).
29. [lane] legalparty-model M03 / family train (the billing lane's next milestone; existing P3 row).
30. [lane] webphone C27 family-pin sweep (existing blanket row; their tree, their timing).
31. C28 dispatch-gate consumer waves (existing L5 row (b); inventories pre-staged).
32. [owner] The five usermgmt API asks filing decision (existing upstream-asks row (h)).
33. [owner] go-cqrs-lite `event.New` pooled-encode ask (existing upstream-asks row (g)).
34. [watch] go-datastar#31 deterministic signals ordering (`NewSignalsPatch` marshal instability; existing P2 row).
35. [watch] httputil#48 + go-cqrs-lite#117 maintainer cadence (existing rows).
36. [owner] K3 templ-components popover evidence pack — DONE-by-action: PR#101 filed and merged (verify state, then strike the stale K3 mention wherever it lingers).
37. Fleet libs report-count gate re-run after the next `docs/review/libs` refresh (baseline 22, green today).
38. README quickstart gate sanity after any README touch (green today; recurring).
39. setup-demo `app.css` byte-delta eyeball — SATISFIED-BY-GATE: `check-css-bundle-classes` proves class-set equality (1078 tokens exact ×2 bundles); a one-time human diff glance remains optional.
40. Annotate remaining superseded reports' settled §g items (the 20-47 §g.2/§f5 done today; sweep the siblings next annotate pass).
41. README↔FEATURES quarterly claim-sweep (due ~2027-01-17; existing P3 row).
42. Shared-tree branch guard (existing row: preflight-tree-check branch assertion + self-test).
43. loginpage test-depth debt (existing row: WebAuthn ceremony e2e + full-page render golden).
44. `ReadLastN` SQL event-store upstream ask (existing row; rides go-cqrs-lite).
45. GCL hardening follow-through (b)-remainder + (e) (existing row; owner-gated via S55).
46. BuildFlow diverged-checkouts reconciliation (existing P2 row; owner-gated cross-repo).
47. Fleet release-dance helper design (existing P2 row; design note first).
48. dashboardui custom-sidebar revisit criteria re-check (existing row; criterion 2 still unmet at last verify).
49. ExtraNav owner decisions (a)/(b) (existing row; aria-current gap real, goldens churn).
50. Quarterly `/mnt/buildcache` watch (existing P3 row; `df -h` before any gate battery).

## g) Three questions I can NOT figure out myself

1. **Gate robustness vs fail-loud for shared-worktree transients (the 03-09 report's §g.2, still open).** Gotcha 68 now documents the re-run-before-triage rule, but the underlying choice is yours: should the LOCAL-only gates (route-postures real-tree case, module isolation, the composite) get a one-retry tolerance inside the gate + self-test, or does strict fail-loud with the documented re-run rule stand as the permanent posture?
2. **The migration-plan D1/D2 contradiction (the W3/W4 gate; the 03-09 report's §g.3, still open).** The 11-08 owner decision sheet says "Defaults already ADOPTED — no tick needed" while the plan's §6 says W3/W4 "hinge only on the D1/D2 owner ticks." Adopt-the-defaults unblocks the read-tier cutover waves; keep-parked waits for an explicit tick. One word from you settles it.
3. **Daemon-commit message receipts — how far should repair go?** The go-appkit sweep's message again landed as daemon heuristic commits (`769ad2cb`/`d5adf8ef` — one mixing MY sweep with a SIBLING's go.work bump). I wrote the durable receipts into agents-notes/CHANGELOG instead of rewriting history. Do you want a sanctioned amend/rewrite flow for daemon-captured substance commits (owner-gated per gotcha 63), or is the docs-receipt pattern the intended terminal state?

*Report ends. Waiting for instructions.*
