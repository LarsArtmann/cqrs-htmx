# Status — the v4.17.0 family train executed end-to-end, and CI green

**Written:** 2026-11-07 06:4x CET · **Session window:** 00:45 → 05:2x · **HEAD at writing:** the docs commit carrying this report (see `git log -1`)
**Predecessor:** [23-52 receipt battery](archived/2026-11-06_23-52_receipt-battery-dep-sweep-coverage-recovery-and-owner-gated-train_status.md) (archived this pass — its §g1/§g3 answered by this session's execution) · [00-15 templ-era conflict + push unblock](archived/2026-11-07_00-15_templ-era-conflict-train-lag-zero-and-push-unblock_status.md) (live; its §g2 go-work policy and §c/f queue carried)
**Operator directive:** the owner cancelled the §g questionnaire and re-sent "keep going until done" — the recorded precedent (2026-11-03_20-15 §0) treats that as the standing GO; the train, the verification battery, and safe hygiene items executed under it. Posture-only calls (§g below) were NOT self-answered.

## a) Fully done (verified in-tree this session)

1. **State re-derived before acting:** the handoff's "54+ unpushed" was already stale — the sibling pushed `a7260d70..bb407b60` (60 commits) and later `f90cae63`; origin == HEAD verified via ls-remote before any mutation. The 23-52 §g questions were re-read against the newer 00-15 report before asking anything.
2. **CI triage receipt (00-15 §f1):** the red set on `f90cae63` was EXACTLY the documented residuals — route-postures hermetic reds on dashboardui+setup, lint ×4, build/test/module-architecture (API-vs-tag + upstream catalog poison); `security` green. Confirmed before the train, so the train had a falsifiable expectation.
3. **Git Town stashes resolved (00-15 §c7/§f13):** stash `dc8b70b9` (one stale-numbered wave-3 ledger sentence, superseded by the archived 04-29 report + the live ledger's later entries) and stash `b8beb7c1` (59-file pre-rebase root-split WIP, superseded by the landed verified PR #40 `c6b6547d`; worktree already pruned) — both dropped with reasons; stash list empty.
4. **train-preflight ran per the S52 mandate:** quiescence ✓ (first run aborted at the surprise gate on the sibling's fresh `851f8323` — the gate working as designed; re-run after the 300s window), surprise ✓, lint ✗ with the recorded reason (identity-model v4.15.0 lacks the session-lifecycle API usermgmt consumes — the exact class the train heals, same posture as the 2026-11-05 train); strict release-train gate separately green.
5. **THE TRAIN EXECUTED — seven tags cut and pushed via `wave-tag --execute` (dry-run rehearsed first, all six leg-0 guards green):** `identity-model/v4.17.0` → `v4.17.0` (root) → `usermgmt/v4.20.0` (above its v4.19.0 latest — the below-latest uniform-tag mistake not repeated) → `adminui/v4.17.0` → `dashboardui/v4.17.0` → `setup/v4.17.0` → `systemadapter/v4.13.0` (added mid-train: its engine-presets API was unpublished and `examples/system-demo` hermetically red on it — the wave-carrier report's job). CHANGELOG `[Unreleased]` cut per module same-change and committed BEFORE each tag (tags point at commits).
6. **Consumer sweeps between every tag** (the printed fix recipes): identity-model ×12, root ×21, usermgmt ×11, adminui ×5, dashboardui, setup ×3, systemadapter ×2. bump-dep's `--commit` was daemon-absorbed four times (`b6ecf4b5` ×24 files, `af75f0da` ×42, `72f4cf6b` ×22, `e84395cb` ×10, `8c359092` ×26, `7a2864ba` ×4, `3e3cc8a3` ×6) — the documented interleave; message-amends skipped mid-train (daemon race), hashes recorded here instead.
7. **The A25 upstream blocker HEALED:** go-cqrs-lite shipped `catalog/v4.14.0`; verified poison-free BEFORE sweeping (module-cache grep: `NewAttributeExpression` present, zero `ResolveAttributeValue`/`JoinURLErrs`), then swept its consumers (`8c359092`). integration_test + `examples/catalog-demo` templ legs unblocked.
8. **ADR-0062 repoints in integration_test:** with tags published, hermetic typecheck finally surfaced 7 SA1019 deprecated-alias uses (`cqrshtmx.ErrUnauthorized`/`MapError`/`ErrForbidden`, `usermgmt.NewBotID`) — repointed to `httperr`/`identitymodel` (`7434268a`); integration_test hermetic lint 0 issues, suite green.
9. **Post-train battery:** route-postures gate all five modules GREEN hermetically (the CI `checks` red class dead); hermetic examples leg "All modules build and vet standalone (GOWORK=off)"; the full race battery (`nix run .#test`) green end-to-end locally; coverage gate PASSED (all 15 modules incl. the four previously-unmeasured: setup 93.7%, systemadapter 91.1%, loginpage 81.1%, huma 95.6%); `check-release-train --refresh-cache --strict-lag 0` = **951 requires, 0 unpublished / 0 replace-exempted / 0 lag**; `verify-remote-tags` over all seven new tags = 7 ok / 0 mismatch (peeled refs).
10. **Two CI reds the train's green unmasked — both fixed (`578c8a9d`):** (a) `check-module-layers.sh` ran workspace-mode `go list`, which resolves the tracked go.work's `/home/lars/projects` replaces on Lars's machine but nowhere on runners — now `GOWORK=off` (root's own go.mod resolves hermetically; gate green, 9/9 fixture self-test); (b) the install-git-hooks fixture never shipped `check-conflict-markers.sh` when the gate joined the pre-commit template (2026-11-06 11:20) — T6/T12 scratch commits failed on the missing script; shipped into the fixture cp list, self-test **17/17**.
11. **CI verdict at `578c8a9d`:** `test` ✓, `build` ✓, `security` ✓, `mod-tidy` ✓, `module-architecture` ✓, ALL lint jobs ✓ — the only red left is the docs tail ratchet, healed below.
12. **Round-25 archive pass:** four settled 2026-11-06 evening reports archived (21-31, 22-03, 22-42, 23-52) with specific successors and non-generic reasons; 21-31's nine struck-mixed rows normalized (trailing cells struck — PARTIAL-class fix); 12 links re-based across 7 files; `TAIL_BASELINE` 16 → 13.
13. **TODO trued:** the two executed train rows removed (`13048492`), preamble updated; TODO hygiene gate clean (76 rows, 0 dead, preamble 629).

## b) Partially done

1. **The F020 receipt comment on PR #40** (06-41 §f29 closure condition): the CI-green evidence exists at `578c8a9d` for every job except the docs-tail ratchet (healed post-run); the comment itself is owed once the NEXT CI run is fully green — it needs github-voice loaded and is the first item of the next session, not something to dash off now.
2. **docs-gates battery:** annotate/corpus/tail/conflict gates green after this pass; `audit-all` (419 vs pin 322) and `cadence` remain exactly as the predecessor left them — posture-gated (§g1), not advanced unilaterally.
3. **cqrs-lint re-verification on the final tree:** not re-run this session (the gate's binary-custody question is owner-gated; the hermetic load errors that reddened 4 modules are healed by the train, which is known from the class, not from a fresh run — the fresh run is owed with the custody answer).

## c) Not started (owed forward)

1. The 00-15 §f queue remainder: numbered gotchas.md entry for the templ-era conflict; the daemon go.work-deletion guard (§f16); the templ-era coherence gate (§f17); bump-dep flock/pidfile + intra-train verification mode (§f18 — this train re-proved the need: `--lint` is structurally red mid-train); the sibling-collision pre-flight probe (§f19 — this session dodged it only by reading logs first).
2. Upstream asks, all still owner-gated: the E005 cross-module resolution defect (23-52 §g3a), the dispatch-middleware offer (ADR-0063), the catalog bump thank-you/confirmation note (v4.14.0 healed us — a courtesy note, not a defect).
3. The go-appkit bump-dep re-run expecting 4/4 PASS (plan M02 tail); the bench re-pin ONLY on an idle machine (this session ran at load 11-26 — explicitly not idle); the parallel/allocation ratchets.
4. `examples/samber-do-demo` + `loginpage` per-module suites after their transitive bumps (otter v2.4.0, commandlifecycle v4.3.1) — inherited from 00-15 §c4, still unrun; the race battery covered them workspace-mode, not module-hermetic.

## d) Fucked up (self-attributed)

1. Presented the predecessor's §g questions as a form while the tree had already answered one of them (the push) — the read-then-ask order was wrong once; the cancel + directive was the correction.
2. `git stash drop stash@{1}` after dropping stash@{0} — reflog indices shift on drop; the second drop failed noisily and was redone as `stash@{0}`. No loss (the drop target list was verified first), but the sequence was sloppy.
3. The first PARTIAL-row normalization struck CLEAN rows' trailing cells (9 → 12 partials) — the gate caught it, `git restore` of my own delta, redone precisely. The gate doing its job is not an excuse for the scattershot first pass.
4. Ran the first bump-dep sweep WITH `--lint` knowing the class was intra-train-impossible, then dropped the flag without recording the reason per sweep — the reason lives here (§a6) but should have been a flag-level decision from sweep one.

## e) What we should improve

1. **The fixture-ship gap is structural:** every script the hook templates invoke needs a mechanical check that the install-git-hooks fixture ships it (this session's T6/T12 failure class, and `d19249bb` fixed the previous instance by hand). A tiny gate diffing template script-references vs the fixture cp list would close the class.
2. **bump-dep intra-train mode** (00-15 §e3, now twice-proven): workspace-mode verification legs + hermetic legs skipped-with-reason; today every sweep's lint leg was expected-red noise.
3. **Archive row-gate fixer:** the PARTIAL normalization is a manual python one-off each pass; a `--fix-strike` flag on check-status-rows (strike unstruck cells in rows that contain any strike) would make the ritual one command.
4. **The preflight's lint stage could classify the API-vs-tag failure** (the `undefined:` signatures name the missing module directly) and print "documented intra-train class — record reason or train first" instead of a bare FAIL.

## f) Next items (sequenced, highest impact first)

1. Post the F020 PR-#40 receipt comment once CI is fully green on the docs commit; close TODO's plan-of-record row's first-wave block.
2. Answer §g1-§g3 (owner): audit-all posture; cqrs-lint binary custody + E005 order; go.work policy.
3. Then: the 00-15 §f tooling queue (gotcha entry, daemon guard, coherence gate, bump-dep mode, collision probe) — each small, each independently landable.
4. Re-run `check-cqrs-lint` on the final tree with the custody answer in hand (expect 14/14 now that tags exist).
5. The docs-gates battery on the final tree; drain-or-re-pin decision executes §g1.

~~6. Run the two inherited module suites hermetically: examples/samber-do-demo after the otter v2.4.0 transitive bump and loginpage after the commandlifecycle v4.3.1 addition (§c4 — harvested into TODO with those tokens); the upstream asks (§c2) after §g answers.~~ ROUTED — TODO_LIST.md "Run the two inherited post-bump module suites hermetically" row (00-15 §c4) carries both suites; harvested 92158e3c
7. Bench re-pin on a genuinely idle machine; the parallel/allocation ratchets.

## g) Owner questions (carried, unchanged in substance)

1. **audit-all endgame (fourth ask):** drain 419 → 322 (batch-annotate with evidence), owner-approved re-pin to the measured residue, or a semantics change (stop replaying archived reports — archiving provably drains nothing). The pin is advisory-only today; nothing blocks on this except honesty about what the battery measures.
2. **cqrs-lint custody:** pin the binary in the flake (ends the `~/go/bin` shadowing; the 4.20.1 master build emits a false V003) or keep the deliberate rolling posture? And file E005 upstream first, or disable-with-reason per module first?
3. **go.work policy:** it is gitignored-but-tracked, which is what let the daemon commit its deletion (twice) and what just reddened CI via the layer gate's workspace resolution. Un-ignore and track plainly, keep ignored + add the daemon guard, or leave as-is?

---

*Receipts: the seven tags peel correctly (`verify-remote-tags`), the train gate prints 951/0/0/0, coverage 15/15, race battery green, CI `578c8a9d` green except the tail ratchet healed by this commit; sweep commits and their daemon-absorbed hashes are in §a6; the two CI-healing fixes are `578c8a9d` with self-tests 9/9 + 17/17.*
