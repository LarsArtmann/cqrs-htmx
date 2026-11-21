# Status report — the family-train session, brutal self-review: twelve tags landed, five lies my own push output told me, and what I forgot (2026-11-21 19:21 CET)

**Scope line — this session only (19:19 CET boundary: the pareto close-out P01–P18 window's report already exists at 13-15; the 14-12 sibling's toolchain round has its own report).** The owner directive was blanket execution of the TODO list. What follows is what I did, what I fumbled, and what I forgot — with the fixes I applied before writing this (three stale trueings + one fan-out re-verify, the fix-on-sight class).

## a) FULLY DONE

1. **The 2026-11-21 family train, end to end.** Twelve tags cut and pushed in dependency order — identity-model/v4.18.0, root **v4.19.0**, datastar/v4.13.0, auditlog/v4.13.0, health/v4.15.0, huma/v4.1.0, usermgmt/v4.21.0, adminui/v4.18.0, dashboardui/v4.18.0, loginpage/v4.19.0, setup/v4.18.0, systemadapter/v4.14.0 — each through verify-tag's guards, all remote-verified (peeled audit 240/240), **root v4.19.0 confirmed live on proxy.golang.org**. Version folds in root + 7 module CHANGELOGs before tagging; the two `### Housekeeping` blocks merged (close-out item 30) so the P02 receipts survived the fold (item 37 ✓).
2. **All 85 internal requires swept to the fresh tags; final strict gate 0 unpublished / 0 replace-exempted / exactly the 4 receipted appkit holds.** Eleven bump-dep family sweeps (daemon shredded several commit messages — the CHANGELOG train entry is the durable receipt); both sibling-local replaces stripped after symbol-verification at the tags (gfl-usermgmt → systemadapter v4.14.0's deriver; integration_test → dashboardui v4.18.0's FromSystem/systembridge); hermetic builds green after each strip.
3. **The CSP knob shipped at tag level** — `cqrshtmx.CSPBuilder` + `RecommendedSecurityStackWithCSP`/`RecommendedSecurityMiddlewareWithCSP` (root) + `Config.CSPBuilder` (adminui), admin-demo adopted on both mounts with its rebuilt-stack twin deleted; root/adminui suites + API-surface compat pin it; **20/20 e2e fast-slice green** (console guards prove zero CSP errors on admin surfaces — the train's own verify row).
4. **The pre-tag lockstep window opened and closed inside one session:** the 5 CI red jobs on `7eab8c16` (all the unpublished-API class) healed to green locally post-train — route-postures ✓, adminui lint 0 ✓, admin-demo lint 0 ✓, integration_test hermetic tests ✓, module-isolation `--examples` ✓.
5. **The appkit hold defended with a verified fact.** The 17:05 sweep re-moved appkit to v0.13.0 (+ four directives to 1.27.4) against the flake's 1.27.3 toolchain — I re-healed (one reviewed bump-dep commit, all legs green) because **nixpkgs-unstable HEAD `680275f0` (probed ~17:20) still ships go 1.27.3**: the floor train's first step has no target. go-legalparty v0.1.0 declares `go 1.27` — floor-clean, not a forcing function. Receipts: fleet-floor row addendum 2 + the [v4.19.0] Housekeeping entry.
6. **go-branded-id v0.8.0 swept** across all 29 lagging consumers mid-session (floor-clean `go 1.27`; the only failure leg was adminui's pre-tag window, not the bump).
7. **The nested-retag-reminder gate exists** (P01's rule mechanized): `scripts/checks/check-nested-retag-reminder.sh <dep> <fix-date>`, live-proven against the real oauth2 state (green — v4.13.1 postdates the fix), self-test 5/5, documented in the release runbook.
8. **The release-train gate's demonstrated reds** (§f45): two new self-test cases prove both UNPUBLISHED verdicts (zero-tags; above-published-max; rc=1 with the require named) — self-test 7/7.
9. **The `.#render-receipt` flake app**, live-tested end to end: `e2e/render-receipt.mjs` + the hoisted shared `playwrightBrowserLibs` forest (single definition now, e2e-firefox deduplicated onto it), binary-guard-aware (exit 3 + the PROVEN ImageMagick quantize recipe — 1.72 MB → 750 KB measured), runbook at `docs/runbooks/render-receipt.md` (§f48 + §f17).
10. **GH release created for `usermgmt/oauth2/v4.13.1`** (close-out item 29) — and the discovery that this repo has NO Release workflow and no releases since 2026-09-30 (the release-dance-helper gap, now with 30+ unreleased-tag evidence).
11. **Q3 decided: verify blocks freeze as audit-time records; the living replay receipt carries corrections.** Precedent written into `FORMAT.md` (replay rule + §f49's non-adopters pointer). The audiblesync false-negative upgraded to FACT in the same receipt (§f19): the constants ship only in the `htmx` subpackage — the root façade never aliased them (zero at v4.17.0/v4.18.1/master); a root-surface lookup saw a true absence.
12. **The docs batch:** FEATURES "Worth Considering" (the 8 correctly-not-adopted verdicts + reopen triggers) and the JSON Directory API row; README fleet-audit link + the fit-test quote pin (7, re-derived with the exact grep); setup README footprint pointer (§f46); skill file-wide editing contract (§f16) + Path D dashboardui-only (§f42); libs-index presentation overlay row (§f27) with its strict-lag pin trued to the receipted hold; self-audit Prior-reports annotation (§f24); check-docs-links HTML boundary documented (§f41); CHANGELOG receipt rule satisfied at tag level (FlashErrorHandler/CSPBuilder/Retryable/NewFromJSON/artifact/deriver all resolve AT the tags).
13. **Batteries:** docs-gates 8/8 PASS (twice); erraudit strict re-true **TOTAL=354, DOWN from the 364 ceiling** (hold satisfied, AGENTS receipted); treadmill worst-lag 1 (appkit only — metaengine aligned, B18 has nothing to wait on); tail ratchet green after archive round-23 (one fully-harvested report, tail 14); corpus baseline bumped twice for my own docs growth (65284 → 65383 → 65477).
14. **CI shepherded to the terminal posture:** 18 green / 1 red (`module-architecture`, the receipted hold) on `258d9b3b` — including healing two of my own CI finds (shellcheck SC2164/SC2319 in my new self-test; the root-size ratchet +41 LOC for the knob, baseline bumped same-change).
15. **Three stale trueings + one re-verify applied on sight before this report** (the 2026-09-06 permission): TODO row 45 (the CSP carve-out row was still saying "train-gated" and describing the deleted `adminSecurity()` — now FULLY DONE with receipts), the fleet-presentation row's FlashErrorHandler clause (now "PUBLISHED at tag level"), and the skill fan-out re-verified AFTER my later SKILL.md edits (Path D + editing contract — 2 hits through the symlink).

## b) PARTIALLY DONE

1. **The train-preflight receipt is incomplete.** The first run died at quiescence (my `--keep-going` arg was silently ignored — the flag is `TRAIN_PREFLIGHT_KEEP_GOING=1` env). The second (env-var) run's captured tail shows tests/train/e2e but I never printed the quiescence/surprise/lint stage verdicts into a durable receipt — the lint stage almost certainly red on the then-open pre-tag window (expected), but "almost certainly" is not a receipt.
2. **The wave-tag deviation is undocumented in the runbooks.** I cut the train via manual verify-tag + pushes because wave-tag's final `--strict-lag 0` leg cannot pass under the receipted hold — the correct call, but the runbook still presents wave-tag as THE rhythm with no hold-shaped exception noted. The CHANGELOG entry carries it; the runbook does not.
3. **One-green check-modules composite** — not re-run post-train (now possible except the strict train step; queued).
4. **coverage-gate** — not run after the knob added root/adminui code (the thresholds are single-sourced; a move would surface at the next receipt).
5. **`check-reexport-policy --strict` + `branching-flow` re-runs** — not executed locally after my root API addition; CI's green tip covers both, but my verification was CI's, not mine.

## c) NOT STARTED

1. **The fleet-floor train** — [owner]; blocked upstream (no nixpkgs rev ships go ≥ 1.27.4, verified today ~17:20).
2. **The go-taskqueue erratum `0d0a647d` push** (pareto close-out Q1) — untouched per its gate: another repo, 92 commits of sibling work, active session.
3. **§f34** — the oauth2 consumer-sweep recipe verified in a throwaway consumer module (dnsblockd's move is evidence, not proof).
4. **The GH release backlog** — oauth2 v4.13.1 got its release; the Oct/Nov trains' ~30 tags did not.
5. **The dated-hold mechanism for the strict-lag gate** (14-12 §f48 [owner]) — would retire the whole `--no-verify` protocol class AND un-block push-and-watch, whose pre-push leg refuses on the hold today.
6. **§f43** cv vs CV dedupe — next fleet refresh; **§f14/§f26** — v5 planning triggers.
7. **The release-dance helper** (14-12 §f43) — now with stronger evidence.

## d) TOTALLY FUCKED UP

1. **Five of my push receipts were LIES.** The tag batch printed "Everything up-to-date"/"PUSHED" while the pre-push hook had blocked every tag — I trusted `tail -1` of piped push output and nearly closed the session believing five tags published that were not on origin. Caught only because I re-checked `git ls-remote` per tag. The rule I now hold: **a push receipt is the remote's answer (ls-remote), never the local command's output or exit code.** The retry loop verifies remote presence per tag before declaring victory.
2. **`set -e` was defeated by my own pipes.** In the tag batch, `verify-tag … | tail -2` and `git push … | tail -1` made every failure invisible to `set -e` — which is why the batch sailed past root's verify-tag failure. `set -o pipefail` or capturing rc explicitly is mandatory in compound scripts.
3. **I lost a full ~20-minute preflight run to an ignored flag.** `--keep-going` is not a flag; the env var is `TRAIN_PREFLIGHT_KEEP_GOING=1`. I grep'd the script for stages but never read its argument handling before launching the long run.
4. **I created the root v4.19.0 tag without `--dry-run`** (verify-tag's rehearsal mode) — it created a real tag I then had to delete. Rehearse-then-cut was the documented pattern; I skipped the rehearsal on the most important tag of the day.
5. **The corpus baseline raced its own commit twice** (daemon swept the bump mid-hook; ref-lock failures). Recoverable, but the ordering is now obvious: land the one-line baseline bump FIRST, alone, before any hook-heavy work.
6. **~10 minutes lost to a ghost.** bump-dep's "another sweep appears LIVE (pid N)" warnings during my sequential loop were my own loop's stamps — I attributed them to a sibling and waited two idle cycles. One `ps -p N` would have settled it immediately.
7. **The render-receipt recipe shipped half-proven:** the quantize recipe pointed at PIL, which the ambient python3 does not have (the 14-12 sibling's environment had it) — caught in my own self-test, fixed with the verified ImageMagick one-liner. And the first live render failed on relative paths (the wrapper `cd`s into e2e/ before args resolve). Both fixed and re-proven, but "prove the recipe, then codify it" is the order I skipped.
8. **Shellcheck missed on my own new script.** I ran the self-test but not `check-shellcheck.sh` on `test-check-nested-retag-reminder.sh` before pushing — CI's checks job caught SC2164/SC2319. The gate exists LOCALLY; the CI-parity legs exist precisely so CI is never the first to see my findings. Same class: the root-size ratchet (+41 LOC from the knob) surfaced in CI, not locally — I added root API and never ran the one gate whose number my change moves.

## e) WHAT WE SHOULD IMPROVE

1. **Push receipts must be remote-answers.** Every push helper (tag_one-style loops, wave-tag, push-and-watch) should end with an `ls-remote` verification of the exact ref — local exit codes and stdout are evidence of intent, not outcome.
2. **`pipefail` everywhere.** Any `cmd | tail` under `set -e` silently de-fangs failure handling; the repo's gate scripts mostly do this right — my ad-hoc session scripts did not.
3. **Read a long-running tool's arg/exit contract before launching it**, not after its first failure (train-preflight's env-var keep-going; wave-tag's strict-leg refusal under holds).
4. **Run the CI-parity gates on every NEW file before push** — shellcheck for scripts, root-size for root-package edits, branching-flow/reexport for Go API changes. CI should confirm, never discover.
5. **A dated-hold mechanism for the strict-lag gate** (holds file: module, held version, reason, expiry) would make the appkit posture first-class, retire the `--no-verify` protocol, un-block push-and-watch AND wave-tag — five tools, one mechanism (14-12 §f48, still owner-gated).
6. **verify-tag's subject naming** — `. v4.19.0` is ugly-but-consistent; a dir→display-name map would make `git tag -n` match the CHANGELOGs.
7. **The preflight's quiescence stage under a live daemon + siblings** always times out — a documented expected-timeout shape (or a `--quiescence-soft` mode) would stop burning 10 minutes of every preflight in this era of concurrent sessions.
8. ** bump-dep's live-sweep stamp should carry the invoking tty/pid visibly** — my ghost-stamp detour is the second attribution confusion this week (the 08-45 shadow-binary class was the first).

## f) Up to 50 things to get done next (ranked; [owner] = owner-gated, [watch] = trigger-gated)

1. **[owner] The fleet-floor train** — the moment a fresh `nix eval nixpkgs#go.version` says ≥ 1.27.4: flake rev + directives + appkit v0.13.0 + CI pins, ONE train (row 21's recipe). Retires the hold, the 4 lag entries, the `--no-verify` protocol, and master's one named CI red.
2. **[owner] The dated-hold mechanism** (`release-train-holds.tsv`) — first-class deliberate holds; un-blocks push-and-watch + wave-tag under any future hold.
3. **[owner] go-taskqueue erratum `0d0a647d` push timing** (carried since the pareto close-out).
4. **[owner] Release cadence** — restore per-train GH releases (backfill the Oct/Nov trains, then one release per train going forward), or ratify tags-only/pkg.go.dev as the surface and retire the release step from the helper design.
5. Confirm the tip CI run (`258d9b3b`+) holds the 18-green/1-named-red posture after the baseline bump.
6. **The one-green check-modules composite receipt** — re-run post-train; every leg should pass except the strict train step (the hold).
7. **coverage-gate receipt on the post-train tree** (the knob's root/adminui code is unmeasured).
8. **§f34**: verify the oauth2 consumer-sweep recipe in a throwaway consumer module.
9. **Wave-tag under holds**: document the manual verify-tag deviation in the release runbook (§b2 of this report) until the dated-hold mechanism lands.
10. **Preflight receipt completion**: capture the quiescence/surprise/lint stage verdicts of a keep-going run into the next train's receipt.
11. **[watch] templ-components PR#101 release** → sweep + retire adminui's vendored popover CSS same-change.
12. **[watch] go-cqrs-lite#113 kv constructor** → bump + delete usermgmt's view-cache adapter.
13. **[watch] storage v4.12.0 StateShape tag** → flip the sentinel test.
14. **[watch] metaengine B18** — aligned today; the ROADMAP bridge row + the §f12 spike wait only on demand.
15. **[owner] M2M D1–D4** per-row surfacing (packets pre-staged).
16. **[owner] adminui D1–D3** per-row surfacing.
17. **The legalparty lane M03 / family train** (lane-owned; go-legalparty extraction landed this window).
18. **C28 dispatch-gate consumer waves** (inventories pre-staged: browser-history, InboxClean have real command surfaces).
19. **adminui `Config.CSPBuilder` consumer docs** — the production-readiness guide §10 still teaches the hand-rebuilt pattern as the primary path; rewrite it to teach the knob (the carve-out recipe moves to "legacy fallback").
20. **SKILL.md**: add the CSP-knob recipe to recipe territory (the carve-out pattern is now a one-liner — consumers should find THAT first).
21. **Examples sweep**: setup-demo still hand-rebuilds its security chain (`demoSecurity`) — adopt the knob like admin-demo did, retire the twin.
22. **`check-nested-retag-reminder` wiring**: call it from bump-dep post-sweep (the dep+fix-date inputs are known there) instead of manual invocation.
23. **bump-dep stamp visibility**: print the stamp's owning command/pid in the "appears LIVE" warning.
24. **train-preflight flag alias**: accept `--keep-going` as the env var's alias.
25. **verify-tag subject map**: dir → display name for tag subjects.
26. **archive-status successor allowance**: accept `../TODO_LIST.md`-class repo-root successors for honestly-harvested reports.
27. **[watch] go-datastar#31 deterministic signals ordering** (filed; maintainer cadence).
28. **[watch] httputil#48 + go-cqrs-lite#117** maintainer cadence.
29. **[owner] The five usermgmt API asks filing decision** (upstream-asks row (h), batched per adminui D2 recommendation).
30. **[owner] go-cqrs-lite `event.New` pooled-encode ask** (A20 (g)).
31. **e2e sync offline-reads posture** — raise the 30s catch-up ceiling or document the retry (row 48's open question).
32. **Spec-race audit**: sweep the battery for `waitForResponse`-by-URL patterns sharing a URL with a debounced twin (the QUERY fix is the template).
33. **Build-then-run pre-flight inside the fast-slice script** (08-45 §f7).
34. **Hook byte-match stage** in check-modules via `install-git-hooks.sh --verify` (08-45 §f8).
35. **Widen `e2e-kill-strays.sh` to the full demo-port range** (08-45 §f9).
36. **`BUILDFLOW_MAX_CONCURRENCY=1` default for bump-dep commit legs** (08-45 §f10).
37. **Fleet-cache hygiene probe** of `cache.home.lan/*` (gotcha-10's substituter class; owner-gated host side).
38. **bench-spike re-pin** on a verified-quiet window (twice-quiet rule).
39. **README↔FEATURES quarterly claim-sweep** (due ~2027-01-17).
40. **Shared-tree branch guard** (preflight-tree-check branch assertion + self-test).
41. **loginpage test-depth debt** (WebAuthn ceremony e2e + full-page render golden).
42. **`ReadLastN` SQL event-store upstream ask** (S55 packet U2 — owner tick still pending).
43. **GCL hardening follow-through** (b)-remainder + (e) — owner-gated via the S55 packet.
44. **BuildFlow diverged-checkouts reconciliation** (lowercase vs capitalized trees; owner-gated cross-repo).
45. **Fleet release-dance helper design** (tag-release + versions-manifest + gh release create; now with the dark-since-09-30 evidence).
46. **dashboardui custom-sidebar revisit criteria re-check** (v1.24.0-era verdict stands; re-check at the next family bump).
47. **ExtraNav owner decisions (a)/(b)** (Href scheme validation; aria-current).
48. **Quarterly `/mnt/buildcache` watch** (`df -h` before any gate battery).
49. **Harden archive-status against silently-truncated specs** (14-12 §f47 — the rc-must-reflect-dropped-lines ask).
50. **TODO hygiene self-test after the next new-TODO batch** (`.#test-check-todo-hygiene` — standing practice; this report adds none).

## g) Three questions I cannot answer myself

1. **The fleet floor — leap or hold, as POLICY?** Today the hold is mechanically forced (nixpkgs ships 1.27.3). But the 17:05 sweep shows the pressure resolves itself into half-moves whenever someone follows the gate's fix recipe. Do you want cqrs-htmx to bump its flake rev UNILATERALLY the day any rev ships 1.27.4 (owning the fleet drift until siblings catch up), or wait for a fleet-coordinated ADR-0003 rev move even if that means weeks of `--no-verify` pushes? A standing answer would let me execute the floor train without asking again.
2. **GH releases: cadence or retirement?** Releases have been dark since 2026-09-30 while ~30 tags shipped; I created oauth2 v4.13.1's release by hand. Do you want releases as a standing per-train step (backfill included — ~2 hours of `gh release create` work), or are tags + pkg.go.dev the intended surface and the release-dance helper should DROP the release step?
3. **go-taskqueue erratum `0d0a647d`** (carried, third ask): the branch sits 92 commits ahead of origin with sibling work. Push my erratum commit now (publishing the 92 with it), let the sibling's own push carry it, or wait for a quiescent window and push everything then?

*Report ends. Waiting for instructions.*
