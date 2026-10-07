# Status: CI-Repair Round (mod-tidy residue + StreamID pin ratchet → master green)

> **Point-in-time report (2026-10-07 07:21 CEST).** Session window ≈ 06:16–07:21 CEST
> (round 2 of the 06-02 push-unblock session — resumption of the same working tree).
> Co-existed with the **still-active parallel session** (the art-dupl/pre-commit-zero
> train — three live buildflow processes at 06:03/06:11/06:13; it landed gotchas 28–33
> mid-round). All work below is MY round's; attribution noted where it matters.
> Base at resume: `ea8fe94f` (ahead 1, clean); tip at write time: `98a6ff59`
> (local == origin, clean).

## Headline

**The red master CI that the 06:02 push created is repaired and master is fully green.**
CI run 37569591700 (on `eda06ff1`) failed exactly 2 of 18 jobs; both classes were
root-caused, fixed, verified locally (check-modules + hermetic battery + both fixture
self-tests), and pushed as `eda06ff1..b660d534` through strict pre-push gates →
**run 37574144312 FULLY GREEN**. The session's own collateral damage (two flattened
verify-tag fixtures, captured by the auto-commit daemon into a pushed commit) was
detected, restored, and mechanized as **gotcha 34**. Final tip `98a6ff59` (report
annotation) is green at **18/18 jobs** (run 37575129022). The prior session's three
open questions are now answered with evidence and annotated into its status report.

---

## a) FULLY DONE

| # | Work | Evidence |
|---|------|----------|
| 1 | **Resume preflight** — prior report committed (`ea8fe94f`), tree clean, ahead 1; parallel-session activity detected (3 live buildflow PIDs) and guarded around for the entire round | `preflight-tree-check` + `wait-tree-quiet` before every tree-mutating step; zero foreign-diff touching |
| 2 | **CI failure diagnosed completely** — exactly 2 red jobs identified with root causes, full 18-job inventory enumerated BEFORE repairing (gotcha 27c discipline) | `gh run view 37569591700`: mod-tidy X (23s), module-architecture X (StreamID identity-form check); all lint/test/build/security jobs ✓ |
| 3 | **mod-tidy residue root-caused** — EXTRA `fxamacker/cbor v2.9.4` hashes in committed go.sums: union-graph residue a workspace-mode tidy left after the family bumps removed the old version's last consumer; the INVERSE direction of gotcha 27(a). Local tidy diff matched CI's expected diff byte-for-byte | CI log diff vs my `GOWORK=off go mod tidy` diff (identical 2-line removal); gotcha **27(d)** written up |
| 4 | **Root go.sum fixed** | `544f14f8`, hermetic root tests rc=0 |
| 5 | **StreamID pin ratcheted 2→1** — verified the actual cause first: commit `a3e1cf50` deduplicated overview's two identical recentEvents loops into `recentEventsFrom` (display-form legit, count 1). Pin follows content; guard + fixture self-test green | `83699956`; `nix run .#check-streamid-identity` OK (12 sites / 8 files); `test-check-streamid-identity.sh` 5/5; gotcha 25 extended with the ratchet-down rule |
| 6 | **Full 36-module tidy sweep** — caught 3 MORE residue modules CI's single-failure surface hadn't shown yet (datastar, identity-model, examples/datastar-demo; daemon `b312fff2`) — the next whack-a-mole round pre-empted | sweep loop `git ls-files '*/go.mod'`, drift count 11 → 4 real modules + fixtures |
| 7 | **Self-inflicted fixture damage repaired** — the sweep's `go mod tidy` flattened 6 deliberately-untidy `scripts/testdata/verify-tag/` fixtures; the daemon pushed 2 of them inside `b312fff2`; all restored from the pre-damage commit and the restoration pushed | `git restore --source 83699956 --`; `808def16` (amended from daemon `836e63dd`); `test-verify-tag.sh` **15/15**; mechanized as gotcha **34** |
| 8 | **usermgmt/webauthn go.sum residue fixed** (same cbor class) | inside `808def16`; hermetic webauthn tests rc=0 |
| 9 | **Verification battery green before push** — check-modules (ALL stages incl. release-train + StreamID + every self-test), full hermetic `nix run .#test` battery, targeted root+webauthn hermetic tests | `07F`/`080` background runs; "✓ All module architecture checks passed" |
| 10 | **Docs round** — CHANGELOG [Unreleased]→Changed receipt; TODO_LIST header + flightrecorder lag-train row closed (CH side, evidence-annotated) + new P1 patch-train row (identity-model v4.12.2 / dashboardui v4.13.2 / setup v4.14.2, full flow documented); AGENTS gotchas 25/27d/**34**; prior status report annotated with the 3 resolved questions | `7840206a` + `b660d534` + `98a6ff59` |
| 11 | **Repair push through strict gates** — release-train strict 0/0/0 (852 requires published) + version-drift --strict | push `eda06ff1..b660d534`; CI run **37574144312 FULLY GREEN** |
| 12 | **Final tip green** — annotation commit `98a6ff59` (pushed by the tree's auto-sync before my manual push; verified local == origin) | CI run **37575129022: 18/18 ✓** |

## b) PARTIALLY DONE

| # | Work | State |
|---|------|-------|
| 1 | **Tidy-sweep mechanization** — the sweep that found the 3 extra residue modules is ad-hoc shell; gotcha 34's exclusion rules exist as PROSE only. No `tidy-sweep` script exists yet (should: exclude `scripts/testdata/`, print candidate count, fail loudly on zero, emit a drift report) | rules recorded, tool not built |
| 2 | **Daemon commit-message hygiene** — 2 of 3 heuristic commits captured this round were amended to named messages (`808def16`, `b660d534`); `b312fff2` (3 legit fixes + 2 damaged fixtures in ONE commit) keeps its heuristic message — it's mid-history and pushed; rewriting would need a rebase I chose not to do | 2/3 amended |
| 3 | **The 3-module patch train** — decision made (defer; parallel session live + strict 0/0/0 means nothing requires the debt) and fully scheduled as a P1 TODO_LIST row with per-tag flow; the train itself not executed | scheduled, not run |
| 4 | **GCL sibling sync** — identified `go-cqrs-lite` holds 1 unpushed daemon commit (`9b2ef9b24`); reported in the closed flightrecorder row; not pushed (different repo, owner's call) | identified only |

## c) NOT STARTED

| # | Item | Note |
|---|------|------|
| 1 | identity-model v4.12.2 + dashboardui v4.13.2 + setup v4.14.2 patch train | the scheduled P1 row; untagged debt is committed but unreleasable to consumers until cut |
| 2 | `tidy-sweep.sh` tooling (see b1) | gotcha 34 as code |
| 3 | GCL `9b2ef9b24` push | owner authorization needed |
| 4 | Fixture-poison canary in check-modules | a fast grep-gate that reds the moment a sweep touches verify-tag fixtures (today the self-test catches it only when run) |
| 5 | Coverage re-run for the 2026-10-04→07 wave paths | pre-existing TODO_LIST header debt, untouched |
| 6 | Per-module `GOWORK=off go build ./...` as a pre-push mechanical catch (gotcha 30's root-tag-gap class) | pre-existing tracked TODO, untouched |
| 7 | Dependabot reds noticed in `gh run list` | the actions-group PR CI failed (run 37567615835, 03:37Z) and `npm_and_yarn in /e2e` update failed (03:33Z) — neither triaged this round |

## d) TOTALLY FUCKED UP

| # | Mistake | Damage | Root cause + lesson |
|---|---------|--------|---------------------|
| 1 | **Ran `go mod tidy` over test fixtures** — the 36-module sweep included `scripts/testdata/verify-tag/*`, whose go.mods are deliberately poisoned (unpublished requires, dev-replaces). Worse: the **auto-commit daemon pushed 2 flattened fixtures to master** (inside `b312fff2`) before my restore landed | pushed-to-master damage, reverted in `808def16`; 3 daemon commits exist partly because of it; ~15 min of recovery | loop didn't exclude testdata — I built the file list from `git ls-files` without auditing what it would return. Lesson mechanized as gotcha 34: exclude `scripts/testdata/` from EVERY sweep; run `test-verify-tag.sh` immediately after any tree-mutating sweep |
| 2 | **Wrong rebase form dropped commit CONTENT** — to fix a daemon message I ran `git rebase --onto 8ec572fd~1 8ec572fd`, intending a reword; it DROPPED the daemon commit carrying the CHANGELOG receipt, silently deleting the doc edit | receipt vanished from the tree until caught | caught within one command by the verify habit (`rg` for the entry → 0 matches); re-applied and re-landed. Lesson: `--onto` replays AFTER the excluded base — for message fixes use `commit --amend` (HEAD) or `rebase -i` (reword), never a content-excluding onto |
| 3 | **Wasteful duplicate battery launch** — launched a second `nix run .#test` (counting pipeline) while the first background run's verdict was unreadable from its tail; killed it after noticing | one wasted expensive launch | should have re-run with output redirection or read the completed job's full output first |
| 4 | **Lost two daemon races + one redundant push** — staged docs were daemon-committed twice before my commits ran (forcing amends), and my final push was remote-rejected because the tree's auto-sync had already pushed `98a6ff59` | noise, no content damage; each recovered via amend/verify | commit in the SAME tool call as the edit when the daemon is live; check `git fetch` before pushing when an auto-sync may exist |

## e) WHAT WE SHOULD IMPROVE

1. **Mechanize gotcha 34**: `scripts/tools/tidy-sweep.sh` — testdata exclusion, candidate-count print, fail-on-zero, per-module drift report (the sweep is repeated ad-hoc every incident; make it a named tool with a self-test).
2. **Fast fixture canary**: a check-modules stage that greps the verify-tag fixtures for their poison lines, so a flattening sweep reds in seconds instead of at the next self-test run.
3. **Daemon-race-proof commits**: adopt the `bump-dep --commit` pattern (single tool call = mutate + commit) for doc rounds too, or add a tiny `commit-or-amend` helper that folds content into a matching heuristic HEAD commit instead of stacking.
4. **Inventory-before-repair as a habit**: enumerate ALL failing jobs of the target CI run BEFORE fixing anything (gotcha 27c) — this round it worked because I did it mid-repair; make it step zero.
5. **preflight-tree-check own-commit false positive**: the guard aborted on MY OWN 60-second-old named commit (non-daemon = suspicious). An escape hatch (e.g. `--allow-recent-own`) or committer-identity awareness would save the blind 91–300s waits.
6. **Residue-class detection at commit time**: the mod-tidy residue (extra hashes) is invisible to every local gate (only CI's tidy-diff sees it) — wiring the per-module tidy-diff into pre-push would have caught this class before the red CI.
7. **`wait-tree-quiet` DX**: the mandatory quiet window makes every small push cost 1.5–5 minutes; a shorter window for docs-only diffs would compound.

## f) TOP 40 THINGS WE SHOULD GET DONE NEXT

_HARVEST fuel — routing per docs-health: P1/P2 → TODO_LIST; idea-class → ROADMAP. Sorted by impact._

**Release & push (highest impact)**
1. Cut the **identity-model v4.12.2** tag (fold.go/doc.go/model_test.go + id/v4 v4.7.1) — verify-tag dry-run → tag+push → refresh tag cache.
2. Cut **dashboardui v4.13.2** (asset-serve dedup onto root `ServeAsset`/`AssetFromFS`, overview recentEvents dedup, pre-commit lint fixes).
3. Cut **setup v4.14.2** (bus recovery + 90-line test).
4. Sweep the three new tags into consumers with `bump-dep.sh` (wave-ordered; refresh cache per cut — gotcha 27).
5. Push GCL's straggler `9b2ef9b24` (owner call — different repo).
6. Confirm the parallel session's wind-down before the train (no tree-mutating train inside a foreign in-flight session — gotcha 4).
7. After the train: full local replication in ONE pass before push (gotcha 27c battery: selftests with `env -u CI` where needed + tidy loop + lint + test + check-modules).
8. Ensure the CHANGELOG [Unreleased] receipts (go.sum shrinks, presets, ServeAsset) ride the correct module tags.

**Tooling & gates (mechanize this round's lessons)**
9. Build `tidy-sweep.sh` (gotcha 34 as code; testdata exclusion + candidate count + drift report).
10. Add the verify-tag fixture-poison canary to check-modules.
11. Wire per-module hermetic tidy-diff into pre-push (catches the residue class locally; closes the only red CI could see that local gates couldn't).
12. `preflight-tree-check` own-commit escape hatch (kill the false positive on named fresh commits).
13. Optional `wait-tree-quiet --window` flag for docs-only rounds.
14. Commit-or-amend helper to stop losing daemon races on doc rounds.
15. Extend bump-dep with the union-graph post-sweep check (workspace tidy + per-module tidy diff) so family bumps can't strand residue again.
16. Add the 3 extra residue modules (datastar, identity-model, examples/datastar-demo) to any residue-regression fixture the tidy gate gains.

**Dependabot / CI hygiene (noticed, untriaged)**
17. Triage the failed dependabot PR CI (actions group bump, run 37567615835) — merge or configure.
18. Triage the failed `npm_and_yarn in /e2e` update (03:33Z).
19. Verify CI survives the ubuntu-latest → Ubuntu 26 label migration (announced for 2026-10-19 — 12 days out).
20. Sweep remaining `chore: auto-commit` heuristic messages for content worth re-describing in the next train's notes (policy: amend-before-push going forward).

**Docs debt (this session's + adjacent)**
21. HARVEST this report's §f into TODO_LIST/ROADMAP per routing rigor.
22. Cross-link the 04-03/04-08 art-dupl reports to the overview-pin ratchet (they own the dedup narrative; the ratchet is my round's outcome of their refactor).
23. Long-form narrative of the CI-repair round into `docs/agents-notes.md` (dated-history convention; gotchas 27d/34 are the distilled forms).
24. Annotate the 06-02 report's §f list against tonight's outcomes (several of its 50 items are now done/superseded).
25. Update the TODO_LIST "Coverage:" and "CI:" header segments after the next full re-run (both are stale-claim candidates).

**Pre-existing open work noticed this round (not researched further)**
26. Coverage re-run for the 2026-10-04→07 wave paths (15 gates green as of 2026-10-01 only).
27. Per-module `GOWORK=off go build ./...` pre-push step (gotcha 30 root-tag-gap mechanical catch — tracked TODO).
28. dashboardui test-fixture follow-ups from the art-dupl round (~13 near-vanilla `NewMemoryStore` sites could fold into a config-variant helper).
29. GCL-side F22 (adttest cross-engine pagination conformance) + gate compliance row.
30. GCL M06 compound-cursor ISSUANCE (NextCursor protocol) — queued in GCL's TODO_LIST.
31. httputil doc-fix release (`a891f0c` untagged: per-TCP-connection limiter caveat).
32. Hardening-wave remainder M16–M27 (unscheduled; Pareto plan appendix governs).
33. Re-verify `.#test-all` (e2e/examples race set) on the post-repair tip — covered tonight only via CI's test job; the local race battery ran pre-annotation.
34. Check `/mnt/buildcache` disk headroom (gotcha 12 fill-drain cycle; cheap `df -h`).
35. Repo-root hygiene scan for stray `buildflow-fsprobe-*` blobs after any failed pre-commit run (gotcha 8).
36. Re-check `docs/status/` tail budget (advisory gate) — two new reports landed today; archive-annotate the 06-02 one per convention.
37. Confirm no OTHER modules carry deliberately-untidy fixtures (audit `scripts/testdata/` + any `testdata/` dirs for the same trap class gotcha 34 found once).
38. Golden/etag pin re-verify after the patch train (dashboardui goldens carry display-form; the train changes module content, not the pin set — confirm).
39. Consider CI runbook entry: "red master triage order" (mod-tidy → module-architecture self-tests → test) codifying tonight's sequence.
40. Schedule the next alignment check of go.work floors vs published tags (post-train, release-train --strict-lag 0 hygiene).

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Patch-train timing**: the identity-model/dashboardui/setup cuts are scheduled but deferred because the parallel session was still live at 06:13. Is that session NOW finished (or tell me how to reach its operator), and do you want the 3-tag train executed immediately on your go — or batched into the next natural train?
2. **The auto-sync**: something pushed my annotation commit `98a6ff59` seconds after I committed it (my manual push then bounced off the lock). Is there an intentional push-automation on this tree? If yes, I'll stop manual-pushing docs commits and let it own the boundary.
3. **GCL straggler authorization**: `go-cqrs-lite` holds 1 unpushed commit (`9b2ef9b24`, its daemon's). It's a different repo outside this session's charter — push it as part of finishing tonight's cross-repo state, or leave it strictly to that repo's owner session?

---

*Point-in-time snapshot. Open items → TODO_LIST via docs-health HARVEST; dated history → agents-notes if the round grows a narrative.*
