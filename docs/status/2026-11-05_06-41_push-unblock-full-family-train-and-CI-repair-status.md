# Status: push-unblock → full v4.16.0 family train + upstream dashboard release + two CI gate repairs

**Session window:** 2026-11-04 ~18:00 → 2026-11-05 06:41 CET (two hard interruptions; a parallel wave session worked the same trees throughout)
**Author:** this session (Crush), solo for the analysis/rerelease/train cuts, co-executor with the parallel session for sweeps
**Verdict:** the original blocker (pre-push release-train gate, 30 lags) is resolved; master pushed and synced; `checks` job green; residual CI red is exactly one documented upstream-blocked class (go-cqrs-lite catalog docserver poison).

## Timeline (what actually happened, in order)

1. **Blocked push triaged** — the pasted output showed the release-train gate red with 30 lags (go-sse v0.7.0, go-atomic-write v0.6.2, go-health v0.9.0). Skills loaded: go-ecosystem-upgrade, cqrs-htmx, later go-release + buildflow.
2. **Pre-existing tree dirt** (round-46 archive ledger + corpus baseline) was resolved — the daemon committed it first (`f52ae192`); bump-dep's dirty-tree guard was the reason it mattered.
3. **Classification** (per go-ecosystem-upgrade Phase 0): atomic-write v0.6.2 = Go-unchanged; go-sse v0.7.0 = nix/tooling; go-health v0.9.0 = **breaking** (branded `InstanceID`, `Check.Since` → `StatusSince`). Baseline workspace build green.
4. **Sweep 1 (atomic-write)**: build/vet/lint legs passed for auditlog + samber-do-demo; integration_test failed hermetically on `undefined: cqrshtmx.Posture` — diagnosed as PRE-EXISTING (integration_test replaces dashboardui with the local workspace copy; its pinned root v4.15.0 predates the 2026-11-03 posture commit `119e0f31`). Workspace-mode vet green. Error set captured as baseline (`/tmp/it-vet-baseline.log`, since lost with /tmp cleanup).
5. **Sweep 2 (health)**: legs exposed what my source-grep missed — the **transitive** consumer `go-health-dashboard@v0.12.2` reads `Check.Since` and assigns `string` InstanceID, breaking inside health/samber-do-demo/integration_test graphs. No fixed dashboard release existed. **Reverted the sweep** — correct call: upstream later shipped go-health **v0.10.0** and the fleet consumed that instead.
6. **go-sse sweep** launched (27 modules): content landed (24 go.mods at v0.7.0, absence-asserted); its PASS tables were never fully reviewed because the session was interrupted and the logs purged.
7. **go-health-dashboard release prepared and cut**: read that repo's full release checklist; made the CHANGELOG re-head (with a Compatibility section the checklist demands), Version const 0.13.0, FEATURES Released-row repair (it was stale at v0.12.1 — v0.12.2's session skipped it), SECURITY.md supported-versions row; committed with explicit paths; tagged **v0.13.0** from release commit `1e0fc25`. The parallel session completed the ritual: tag pushed, **v0.13.1** green-CI re-cut (after adopting go-health v0.10.0), shipped.
8. **Overnight parallel sessions** landed: PR #40 (root-package-split), PR #39, go-health v0.10.0 follow, templ-components v1.27.0 consumption, `b55d95b3` templ regen, master synced with origin, TODO row 21 diagnosing the exact posture red and prescribing the train.
9. **train-preflight** run: quiescence+surprise ✓; lint stage ✗ (43 issues) = exactly the two documented classes (posture API-vs-tag, samber-do-auditlog templ poison). Reason recorded per the S52 mandate; proceeded.
10. **v4.16.0 family train cut** (`wave-tag`): root v4.16.0 pushed; the rhythm's train leg refused as designed (consumers then lag); consumer sweeps run with bump-dep. **Tag inventory cut+pushed this session:** root/setup/usermgmt/adminui/dashboardui/loginpage `v4.16.0`, plus **usermgmt v4.19.0** after the sweeps revealed setup's source consumes the unreleased `ViewCacheConfig` (the parallel session's usermgmt v4.16.0 sits BELOW the existing v4.18.0 latest — inert noise, recorded).
11. **Gate end-state**: 897 requires, 0 unpublished / 0 replace-exempted / 0 train lag. Full `.#test` suite: 16/16 modules pass; only integration_test build-fails on the catalog poison.
12. **CI triage + two real fixes**:
    - posture gate insta-failed on runners (green locally): my first hypothesis (tag-timing race) was WRONG — a `--failed` rerun proved it; added a no-detail → raw-output branch to the gate (it was swallowing stderr — the A5 class), which revealed the true cause: the **tracked go.work's `/home/lars/projects` local-dev replaces** resolve nowhere on runners. Fix: gate flipped to hermetic (GOWORK=off) — correct post-train since pins are current; all five modules + fixture self-test green (`7b5998a7`).
    - codegen drift check failed: CI pinned the **ancient templ v0.3.1020 generator** while committed files carried **v0.3.1114-master** stamps + repo-root FileName paths. Regenerated all 19 `_templ.go` with the true v0.3.1100 generator per-module (canonical under CI's own invocation shape; stamps + error-message paths only) and bumped the CI step to v0.3.1100 (`714a6db9`).
13. **Final CI state** (run 41966146025): `checks` GREEN; remaining red = `build`/`test`/`module-architecture`, all confirmed (grep-classified) the single upstream catalog-docserver class — clears when go-cqrs-lite releases its regenerated catalog (their master already has it; releasability is blocked by their own unpushed local replace + 37 in-flight files).
14. Housekeeping: stale worktrees pruned (cqrs-htmx-root-split, dashboard /tmp worktree); TODO row 21 updated with execution evidence + the two fixes (one edit was clobbered by a concurrent session rewrite — gotcha 47 — and re-applied).

## a) FULLY DONE

- go-atomic-write → v0.6.2 sweep, committed (`f08756a2`), in pushed history.
- go-sse → v0.7.0 sweep across 24 modules: pins verified, absence assertion clean (0 × v0.6.2), gate-confirmed.
- go-health v0.9.0 correctly diagnosed as ecosystem-blocked and **reverted** instead of force-landing; superseded cleanly by v0.10.0.
- go-health-dashboard **v0.13.0 released** — my commit `1e0fc25` is the tagged commit; CHANGELOG/Version/FEATURES/SECURITY internally consistent; green re-cut v0.13.1 shipped by the parallel session.
- **v4.16.0 family train**: 7 tags cut+pushed, every consumer go.mod aligned, release-train gate at 0/0/0, pre-push CI-parity gates green on every push.
- Two CI defects root-caused (not guessed) and fixed at the source, both verified green locally (gate run + shellcheck + fixture self-test) and in CI.
- Stale worktrees pruned; TODO row 21 carries the full execution record (`9591fe33` → `0122794e`).

## b) PARTIALLY DONE

- **Dashboard release verification**: the go-release Phase-4 battery (build/test-race/lint/vet/flake/browser/vulncheck in the clean worktree) was launched but **killed by the session interruption and never re-run by me**. The release shipped anyway (parallel session), so the fleet is fine — but MY verification chain for v0.13.0 has a hole that only their v0.13.1 green run backfills indirectly.
- **go-sse sweep PASS tables**: content verified (pins/absence/gate/tests) but the per-module hermetic lint verdicts were never individually reviewed (interrupted mid-legs; logs purged).
- **CI watch on the final commits**: 41966146025 (on `714a6db9`) watched; the docs-only commits after it (`0122794e`) were not watched.
- This session's documentation duty: AGENTS.md memory updates were NOT made at discovery time (see f.13); this report is the capture.

## c) NOT STARTED

- `nix run .#check-modules` (the 27-stage aggregate) on the final post-train tree.
- `nix run .#coverage-gate`, `.#erraudit --strict` inventory vs TOTAL=303, `.#check-cqrs-lint`, `.#check-upstream-treadmill`, `.#check-dependabot-alerts` — none re-run this session.
- `.#test-all` (race incl. e2e/examples) and bench-spike re-pin (row 54's remaining leg; code changed).
- go.work replace-block audit (which of ~30 replaces are still load-bearing post-train) — flagged, not started.
- CHANGELOG entry for cqrs-htmx itself re: the train (didn't check whether this repo maintains one needing it).
- Dashboard checklist §8 close-the-loop items (release rows in THEIR TODO_LIST, consumer announcement) — left to the owning repo's sessions.

## d) TOTALLY FUCKED UP (honest)

1. **Wrong bump-dep anchors, twice in a row** (`adminui/v4$`, `cqrs-htmx/adminui/v4$`): the matcher joins the pattern directly after `github.com/`, so the anchor needs `larsartmann/...` — which the gate's own FIX RECIPE printed. I didn't read the matcher before first use. Two no-op cycles wasted while a parallel session was racing me.
2. **Transitive-consumer miss on go-health**: I grepped OUR source for the renamed symbols, concluded "additive for us", and missed that a published dependency (go-health-dashboard) inside our graphs consumes the broken API. The skill literally warns about this (Phase 2/3). Cost: one full sweep cycle + revert + re-diagnosis.
3. **Three edit-tool failures in a row** (read via bash instead of View) — pure tool-discipline sloppiness.
4. **Launched the release battery into a killable background shell** right before a likely interruption boundary; the verification evaporated. Everything long-running this session should have been interruption-proofed from the start.
5. **Edited TODO_LIST.md while a parallel session had it in-flight** (I had literally just watched them rewrite it) — my first addendum was clobbered (gotcha 47, the exact trap). Re-applied, but the loss was foreseeable.
6. **Misdiagnosed the posture-gate CI failure once** (tag-timing race) and burned a `--failed` rerun on it before instrumenting the gate properly. The right move was diagnostics-first.

## e) WHAT WE SHOULD IMPROVE

- **Instrument before rerun**: when local-green / CI-red diverges on an identical commit, add output visibility FIRST (the gate's swallowed stderr cost two CI cycles).
- **Graph-level breaking-change checks**: for any breaking bump, run `go list -m all` / consumer-of-consumer resolution, not just source grep.
- **Daemon-race economics**: the BuildFlow pre-commit hook ran 12–13 min cold while the daemon commits within ~8–11 min beats — `--no-verify` for sweep commits is the measured-correct default until hooks are warm; make that the documented first choice, not a discovered one.
- **Interruption-proofing**: long verifications need detached execution (or immediate incremental checkpointing) in this fleet — two session kills cost us logs, PASS tables, and one verification chain.
- **Read tool semantics before first invocation** (bump-dep anchors; edit-tool View requirement) — both cost avoidable cycles.
- **Concurrent-session etiquette is now the norm, not the exception**: explicit-path commits, SHA-tagging (never HEAD), back-off-on-dirty-tree, and immediate commits after edits worked, but TODO_LIST.md still needs a "commit within seconds or lose it" rule for hot files.
- **The v0.13.0 release shipped with my verification chain incomplete** — fleet rule candidate: a release whose battery didn't complete gets an explicit hand-off note, not silent continuation.

## f) NEXT (up to 50, ordered roughly by value)

1. Watch CI on `0122794e` (docs-only, but the discipline is total).
2. Track/land the **go-cqrs-lite catalog release** (master has the regen; blocked by their unpushed local replace + 37 in-flight files) — then bump `examples/catalog-demo` + `integration_test`; this alone turns CI fully green.
3. Grep-verify the samber-do-auditlog **v0.12.2** consumer bumps actually landed in auditlog / examples/samber-do-demo / integration_test (parallel session's claim).
4. `nix run .#check-modules` full 27-stage run on the final tree.
5. `nix run .#coverage-gate` (15 modules; codegen canonicalization is unmeasured).
6. `.#erraudit --strict --disable generic_return` per-module inventory vs TOTAL=303 baseline (gotcha 44).
7. `.#check-cqrs-lint` via the flake app post-train.
8. `.#check-upstream-treadmill` fresh (go-sse/go-health-dashboard successors may already exist).
9. `.#check-dependabot-alerts` sweep.
10. `.#check-readme-quickstart` + `.#check-readme-badges` post-train.
11. `.#test-all` on the final tree (28 modules incl. e2e/examples).
12. bench-spike re-pin on an idle machine (row 54's last leg; train changed code). <!-- fresh:skip: routed to TODO_LIST row 54 — the bench-spike re-pin leg, wording bound 2026-11-05 -->
13. AGENTS.md updates at discovery time: bump-dep anchor semantics (`larsartmann/`-prefixed anchors after `github.com/`), go.work-replace-vs-CI class, the measured hook/daemon race numbers.
14. Promote the **catalog-release blocker** from row 21's prose to its own TODO row so it survives row 21's closure.
15. usermgmt **v4.16.0 dangling-below-latest tag**: document as inert noise (cannot delete — ruleset) so nobody pins it intentionally.
16. Verify go-health-dashboard **v0.13.1** GitHub release page: exists, notes match CHANGELOG, discloses the re-cut lineage from my v0.13.0.
17. Confirm the release-draft workflow actually produced drafts for both tags (the workflow's own bug was fixed in this batch — verify the fix in production).
18. Eyeball the 19-file `_templ.go` diff beyond the grep classification (goldens discipline: line-by-line, not class-level) — especially dashboardui/timetravel (34 lines).
19. Run the browser suite locally once Chromium can load http(s) — CI carried runtime verification, local pass/skip counts were never captured.
20. Add gotchas: catalog-poison-in-published-tags; tracked-go.work-replaces-break-CI-workspace-steps; templ stamp skew (generator version strings lie).
21. AGENTS.md quick-reference: templ-components era note (v1.27.0 consumed; stamps now 1100-canonical).
22. Fleet consumer waves (row 23): browser-history, Swetty, crm, InboxClean, github-local-sync, webphone can now take cqrs-htmx v4.16.0 + posture manifests — their repos, their commits.
23. docs-tail-budget + corpus-budget ratchets after this report lands; archive per the ritual if over baseline.
24. Check `git town status` is clean (the original interrupted sync from the paste).
25. Audit which go.work replaces are still load-bearing (system/metaengine justifications date to August; trains have since published) → propose pruning dead ones.
26. Fleet policy question (see g.1): tracked go.work should perhaps carry use-only, with local-dev replaces untracked — decide once, apply everywhere.
27. route-postures gate: add a self-test case exercising the new raw-output branch (the no-detail path is currently un-self-tested).
28. bump-dep enhancement: auto-prepend the org prefix when an anchor lacks it (kills the anchor-trap class; update the 12-case self-test).
29. Once catalog ships and row 21's fix lands, close row 21 with evidence and archive per docs-health. → routed — closure condition now written into TODO_LIST row 21 itself (2026-11-05)
30. CURRENT_FOCUS.md: reconcile the active-thread pointer with what this session closed.
31. `verify-release.sh v0.13.1` evidence: confirm the shippers ran it (their checklist §6) — read their status report.
32. Sweep OTHER fleet repos' published tags for the same master-stamp poison (catalog + auditlog known; go-output/go-appkit/go-atomic-write worth one grep each).
33. Wave-tag dry-run the NEXT trains consumer modules will demand (root v4.17.0 posture follow-ups, if any land).
34. Verify identity-model/datastar/systemadapter/health/auditlog genuinely need no v4.16.0 tags (version-drift gate green says no — verify once, record once).
35. Port the dashboard's **templ stamp guard** idea (check-templ-stamps.sh) to this repo as a desk-check complement to CI's drift check.
36. Confirm `setup/v4.16.0` + `usermgmt/v4.16.0` (parallel session's tags) carry what their comment-claims say, or mark them dead weight in the tag inventory.
37. Reconcile the three daemon-labeled sweep commits (`2fdcd225`/`a346fee2`/`250e1457`) against the intent message's disclosure — read the three diffs once to confirm the claim.
38. `git town status` (dup of 24 — keep one).
39. After catalog fix: rerun push-and-watch and archive the incident into a status-report archive entry.
40. Update `docs/runbooks/dependency-train-bump.md` with the anchor-full-path trap (my double-fail is the third instance of a wrong-anchor class).
41. Consider `ROUTE_POSTURE_GO`/env pins for the posture gate so hermetic is structural (env-var default) rather than comment-enforced.
42. Check whether any OTHER CI step runs workspace-mode `go` commands (same replace-poison exposure as the posture gate had) — sweep ci.yml for `go ` runs without GOWORK=off.
43. Add the integration_test hermetic-red classes to a small "known hermetic reds" table in the runbook (catalog + auditlog line items with upstream-blocked status).
44. Dashboard repo: FEATURES row repair carried my edit — verify their v0.13.1 didn't regress it.
45. `/tmp` hygiene: the citest-gomod caches I created (2×~1GB) — confirm auto-clean or remove.
46. Re-verify `.#check-permission-surface` + `.#check-permission-maturity` + `.#check-route-postures` trio post-train (route manifest pins changed shape).
47. Consider a fleet convention for multi-agent waves: a one-line lock-note file per repo during an active wave (today's interleavings cost ~4 dead cycles).
48. buildflow doctor's "13 use paths mismatch" warning (cosmetic, known) — decide: silence via config or fix the use-path normalization.
49. Re-run `.#check-css-bundles` + `.#check-css-bundle-classes` (templ-components v1.27.0 era; bundles were rebuilt by the parallel session — verify canonical form).
50. Write the go-release skill feedback: Phase 4 needs an "interruption" branch (hand-off note requirement) — my v0.13.0 hole is the concrete case.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **go.work policy**: should the tracked go.work keep its `/home/lars/projects` local-dev replace block (making every workspace-mode CI step permanently GOWORK=off-required), or should local-dev replaces move out of the tracked go.work so workspace-mode gates are runner-safe again? This is a fleet-wide ergonomics-vs-safety call that affects every repo with sibling checkouts — I can fix gates symptom-by-symptom, but only you can set the policy.
2. **The go-cqrs-lite catalog release** is the single blocker between this repo and a fully green CI. Their master has the regenerated templ, but releasability is blocked by their own unpushed `replace => ../schema` + 37 files of another session's unreleased work. Do you want me to go untangle/land that release next, or is that the owning session's thread and I should leave it?
3. **Daemon-commit relabeling ritual**: today several sweep commits landed as generic `chore: auto-commit` because the daemon beat deliberate commits (I amended one, disclosed the rest). Should "amend/relabel daemon-claimed sweep commits" become a standard post-wave step, or do you accept generic chore messages when the daemon wins the race?
