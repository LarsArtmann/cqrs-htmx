# Status: M06 landed green, train assembled and armed — blocked on the concurrent session's in-flight root edit

> **ANNOTATED 2026-11-04** — ~~this report~~ **SUPERSEDED** by [`2026-11-04_00-28_sweep-committed-breach-owned-awaiting-train-authorization_status.md`](../2026-11-04_00-28_sweep-committed-breach-owned-awaiting-train-authorization_status.md), which overtaken minutes later by the corrected final-state report (sweep commit landed, breaches owned). Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-04 (round-23 (2026-11-04)).**

Session start: 2026-11-03 ~20:20 (resumed from the 20-15 WAIT report; owner blanket GO).
Session end: 2026-11-04 00:04 CET (owner: report + WAIT).
Head at end: `34aa58dd` (local master, 13 ahead of origin — push deliberately held, see §b3).
Environment: the box REBOOTED at ~21:45 mid-session; load ranged 84–446 all night
(concurrent session's go/rust builds + fleet services). Two BuildFlow steps were OOM-killed
(`signal: killed`) before `BUILDFLOW_MAX_CONCURRENCY=1` was found as the starvation workaround.

## a) FULLY DONE (verified green)

1. **adminui funlen fix**: `superAdminRoutes` (79>60) split into
   `superAdminUserRoutes` + `superAdminTenantRoutes` + `superAdminTenantMemberRoutes`,
   concatenated with `slices.Concat` — manifest order byte-identical (golden green).
   Pinned golangci-lint 2.15.0: **0 issues** in root, adminui, dashboardui, loginpage, setup.
   All five changed modules `go test ./... -count=1` green.
2. **vcs-cache repaired**: the 2 broken entries were torn clones from my own 17:32/17:36
   hermetic runs (empty HEAD, 12 K skeleton, git refuses to open) — trashed; gate green 118/118.
3. **css-bundle-classes fixed without a rebuild**: the fresh build's phantom `.shadow` class was
   a Tailwind token harvested from MY OWN COMMENT ("would shadow every route",
   adminui/routes.go:320) — reworded to "would take precedence over every route"; both bundles
   class-set exact again. Zero bundle churn.
4. **branching-flow adjudicated + re-pinned 198 → 202** (`d26073a9`): all 5 new findings are one
   family — Route manifest-row shape-twins (adminui.Route ↔ dashboardui.Route ↔ authztest.Route ↔
   cqrshtmx.RoutePosture). Rejected under the standing shape-twin class (UI modules must not
   share a manifest type; authztest cannot reference either UI module; embedding was evaluated
   and would not dissolve the duplicate-type pair). Full verdict in the triage-decisions
   amendment log. Gate: `+0 added` vs the new baseline.
5. **M06 COMMITTED** (`119e0f31`, 28 files, explicit pathspecs — the concurrent session's files
   untouched): route-posture manifests across root/authztest/adminui/dashboardui/loginpage/setup,
   `check-route-postures` gate + self-test wired into check-modules (both stage lists) and CI,
   PERMISSIONS L3→L4, guide, CHANGELOGs. Route-posture gate + self-test PASS in the battery.
6. **Archive completion** (`07899d3e`): the 18-31 report's move/manifest/rebase that 5e5c1dba
   missed (daemon committed mid-ritual).
7. **FEATURES.md formatter commit** (`9f3fc4a9`): treefmt realignment of 45235828's tables,
   zero content delta (whitespace-ignoring diff proved only separator-dash rows) — committed to
   unblock bump-dep's dirty-tree guard. Provenance caveat: §d1.
8. **Deferred six-tag remote audit: 151/151 tags peel to local commits, 0 missing, 0 mismatch.**
9. **Consumer-wave prep (read-only)**: browser-history `reconcileSuperAdmins` confirmed on the
   ephemeral `AddGroupPolicy` path (gotcha 45, their comment admits grants vanish on restart);
   InboxClean's `sseDeadlineFree` confirmed duplicating root's shipped `transport.ClearWriteDeadline`
   (deletable after the dashboardui bump); crm confirmed on a local `replace` onto this tree with
   no seeded admin; SwettySwipperWeb **cloned** (was absent locally) — `bootstrapAdmin` confirmed
   on the ephemeral path, needs durable + idempotent GrantRoles.
10. **check-modules: 81/86 stages green** including the new route-postures, shellcheck, and all
    self-tests. The 5 fails decompose into exactly two structural classes (see §b) — all five
    were diagnosed to root cause with fix paths attached.

## b) PARTIALLY DONE (mid-transition, known state)

1. **module-isolation RED (3 modules)**: loginpage/setup/integration_test GOWORK=off builds fail
   because submodules consume `cqrshtmx.Posture`/`RoutePosture` — root symbols that no published
   root tag carries yet. Structural: resolved ONLY by the family train (root tag + submodule
   require bumps). Not a code defect; the route-posture gate runs workspace mode on purpose for
   the same reason.
2. **release-train RED (lag 27)**: `go-cqrs-lite/id/v4` v4.7.3→v4.7.4 and `catalog/v4` →v4.12.0
   published upstream. Both sweeps (`bump-dep … --commit --lint`) were attempted and BLOCKED:
   first by a (stale) live-sweep PID warning, then by the dirty-tree refusal — correctly, because
   the concurrent session's wave kept the tree dirty continuously. An autonomous driver
   (`/tmp/train-wave-driver.sh`: wait for quiet+clean → sweeps → strict train check) was built,
   tuned (load threshold 40→120 after proven successes at 84–130; bc replaced by bash arithmetic),
   and ran for ~90 min without a clean window. **Stopped before this report.** Current pins
   verified still v4.7.3 (their parked go.mod adds dedup/commandlifecycle, NOT id/catalog).
3. **Push held**: 13 commits local-only. Pushing before the train would push a tree CI must
   redden (module-isolation + strict-lag). The push belongs INSIDE the train wave (wave-tag leg 1).
4. **Family train PLANNED, not started**: order usermgmt v4.19.0 (concurrent session's five
   feature commits deserve a tag) → root v4.16.0 (M06 + their transport change) →
   adminui v4.14.0, dashboardui v4.15.0, loginpage v4.14.0 → setup v4.17.0; wave-tag per carrier
   (5 carriers exceeds the --cascade limit of 3 — run individually); upstream sweeps FIRST
   (verify-tag leg 0 requires resolvable requires); train-preflight before all of it (S52).

## c) NOT STARTED

1. `train-preflight` (mandatory pre-train battery).
2. The train itself (6 tags, 5 carrier re-tags, integration_test/examples alignment).
3. `push-and-watch` of the final wave.
4. All four consumer waves (browser-history, InboxClean, SwettySwipperWeb, crm) — each blocked
   on the published tags, each's exact edit surface pre-verified (§a9).

## d) TOTALLY FUCKED UP (honest)

1. **Commit `9f3fc4a9` provenance unresolved**: it landed at 22:15 with my exact message and
   correct content, but no live shell of mine can be positively identified as its author — my
   best theories (an interrupted-but-surviving retry, the auto-commit daemon adopting a staged
   COMMIT_EDITMSG, a reboot clock jump scrambling timestamps) are all unfalsifiable from inside
   the session. Content is verified correct (single file, 35/35 formatter lines) and signed, but
   "a commit appeared that I cannot attribute" is a real auditability failure. Should have been
   escalated instead of moved-past.
2. **Two poll loops built on `bc`**, which is not on PATH — one silently no-oped for 15 minutes
   (comparisons never fired), the first driver version crashed at launch. Both fixed with bash
   integer arithmetic; the wasted wall-clock is on me.
3. **Three owner interrupts** from long blocking `job_output` waits before I switched to short
   non-blocking polls. Pattern-recognition too slow.
4. **First check-modules run piped through `tail -30`**, hiding 4 of 5 failures — forced a full
   battery re-run (~6 min). Lesson written to §e1.
5. **The driver's initial load threshold (40) was wrong** for this box's baseline load — commits
   demonstrably passed at 84–130 all night; I tuned to 120 only after ~40 min of unnecessary
   waiting.

## e) IMPROVEMENTS (do differently)

1. Gate/battery output ALWAYS to a full log file on first run; read with grep, never tail-pipe.
2. No `bc`/non-core utilities in throwaway scripts — bash arithmetic only.
3. **Record `BUILDFLOW_MAX_CONCURRENCY=1` as the box-load starvation workaround** (test-compile
   went OOM-killed→green at load>100 with it; pre-commit hooks inherit it). Candidate gotcha-47.
4. **Record the phantom-Tailwind-token trap**: word choice in Go comments inside CSS-scanned
   trees changes bundle class sets ("shadow" in a comment ⇒ `.shadow` in the fresh build ⇒
   css-bundle-classes red). Candidate gotcha-48.
5. Short polls only (~4–8 min sleeps, non-blocking reads); long blocking reads invite interrupts.
6. The archive ritual's ordering hazard (daemon committing mid-ritual) bit twice now — the
   ritual should run as ONE committed unit immediately after writing the report, before anything
   else touches the tree.

## f) NEXT 50 (ordered; the first 4 unblock everything else)

1. Wait for the concurrent session's parked root edit (go.mod/go.sum/transport/serve.go, quiet
   since 23:33) to land; re-derive tree state.
2. Re-arm the driver (or run by hand): sweep `go-cqrs-lite/catalog/v4$` → v4.12.0 `--commit --lint`.
3. Sweep `go-cqrs-lite/id/v4$` → v4.7.4 `--commit --lint`.
4. `check-release-train --refresh-cache --strict-lag 0` → green (27 lag → 0).
5. Re-run `nix run .#check-modules` → expect ONLY module-isolation red (train-resolved).
6. Read docs/runbooks/release playbook §1; run `nix run .#train-preflight` (record any skips).
7. `wave-tag usermgmt v4.19.0 --execute` (their CBOR/DLQ/catalog/lifecycle/ProjectionHostOptions
   features; verify-tag guards first).
8. `wave-tag . v4.16.0 --execute` (root: M06 + transport/serve.go dedup change) — sweep aligns
   all root consumers; carriers: adminui, dashboardui, loginpage, setup + test go.mods.
9. `wave-tag adminui v4.14.0 --execute`.
10. `wave-tag dashboardui v4.15.0 --execute`.
11. `wave-tag loginpage v4.14.0 --execute`.
12. `wave-tag setup v4.17.0 --execute` (last carrier; integration_test/examples align here).
13. `nix run .#check-modules` → 86/86 green.
14. `nix run .#push-and-watch` → CI green end-to-end (intermediate reds during the wave are
    superseded; the final run is the evidence).
15. `nix run .#verify-remote-tags` for all six fresh tags.
16. browser-history: replace `reconcileSuperAdmins`' `AddGroupPolicy` loop with durable
    `GrantRoles` (rewrite the bootstrap comment — the config-revocation semantics CHANGE:
    removal from SUPER_ADMIN_EMAILS no longer revokes; revocation is RevokeRoles/admin UI).
17. browser-history: delete the drain-hook reconcile call; seed at boot only, idempotent.
18. browser-history: adopt `authztest.Suite` for its route postures.
19. browser-history: bump cqrs-htmx pins to the new train; its own CI green; commit+push.
20. InboxClean: bump dashboardui/root; delete `sseDeadlineFree` (wrapper + comment); verify SSE
     survives a 30 s WriteTimeout via its own tests.
21. InboxClean: note the branch situation (`crmpeople-run-hardening`) — bump on that branch.
22. SwettySwipperWeb: rewrite `bootstrapAdmin` onto durable `GrantRoles`, idempotent re-run
    (re-register/re-grant must converge, not error).
23. SwettySwipperWeb: strip the local replaces for adminui/loginpage after the train publishes;
    tidy; its CI green.
24. crm: seed an admin durably (GrantRoles at boot from env) OR set `DisableAdmin` honestly;
    resolve which with a one-line decision in its AGENTS.md.
25. crm: drop the local `replace …/cqrs-htmx/setup` once setup v4.17.0 is published.
26. crm: bump usermgmt v4.18.0 → v4.19.0 (+ the rest of its pins) in the same change.
27. Consumer CI policy: each repo green before the next wave starts (pending §g3).
28. Add gotcha 47 (BUILDFLOW_MAX_CONCURRENCY=1 under load) to docs/gotchas.md + AGENTS index.
29. Add gotcha 48 (phantom Tailwind tokens from comment word choice) to docs/gotchas.md.
30. Update AGENTS.md Quick Reference: route-postures gate row exists but the train now includes
    usermgmt-by-default when its master carries unreleased features (train-order note).
31. The 2026-11-03_19-39 report (concurrent session's) is still live — archive it when their
    thread closes (their lane, not mine).
32. Consider a `check-corpus-baseline-dirty` guard: refuse the corpus ratchet when
    CORPUS_BASELINE itself carries foreign staged edits (tonight's entanglement).
33. Sweep examples/* go.mods onto the new train tags (wave-tag carriers list will name them).
34. Post-train: re-run `nix run .#bench-spike`? NO — box loaded; baseline discipline forbids.
35. Post-train: `nix run .#check-dependabot-alerts` (new upstream pins → verify 0 open alerts).
36. Post-train: treadmill (`check-upstream-treadmill`) — go-cqrs-lite tags moved again tonight.
37. browser-history: its `vendor/` (if any) needs the pin bump reflected — check.
38. Each consumer repo: CHANGELOG entry for the durable-grant migration (breaking-semantics note).
39. authztest README: document that consumer suites should declare postures matching their
    WIRING, not their intent (the crm/browser-history lesson generalized).
40. docs/guides/route-posture.md: add the consumer migration recipe (how a consumer reads
    Bundle.RoutePostures() into its own authztest declarations).
41. Consider `check-route-postures --workspace-mode-only` comment expansion (why GOWORK=off is
    wrong) into the guide — one paragraph, links the gotcha.
42. The 27-lag class: add `check-release-train` advisory output to the session-start runbook so
    trains are never surprised by upstream publishes again.
43. Verify the driver script pattern (wait-for-window + fire) is worth a tools/ script
    (`scripts/tools/wait-and-sweep.sh`) — tonight proved the need twice.
44. InboxClean/SwettySwipperWeb/browser-history: after bumps, run their authztest suites in CI
    if a CI workflow exists; add one if not.
45. crm: its OAuth2 pin (v4.13.0) is two trains behind — align in the same bump.
46. Re-verify `nix run .#check-readme-badges` post-train (new tags change shields.io verdicts).
47. Post-train status report for THIS wave (the train + consumer closures), then archive this
    report in the same commit.
48. Memory: project AGENTS.md gains the "concurrent session etiquette" actually used tonight
    (pathspec commits, foreign-dirt patience, never gc/lock fights).
49. If the box stays >120 load: propose moving trains to CI-side tagging (owner decision).
50. Keep this thread's todo list in the todos tool (drifted tonight during the interrupt storm).

## g) QUESTIONS (cannot figure out alone)

1. **Concurrent-session handoff**: their root edit (go.mod/go.sum/transport/serve.go) has been
   parked since 23:33 with their session apparently paused mid-wave. May my thread cut the
   family train over their commits the moment they land (including a usermgmt v4.19.0 tag for
   their five features), or should the train wait for their explicit thread-close report?
2. **Provenance of `9f3fc4a9`**: does the auto-commit daemon replay a staged COMMIT_EDITMSG from
   a killed `git commit`? If not, the remaining explanation is a survived/interrupted shell —
   either way I cannot verify from inside; the owner can (daemon logs).
3. **Consumer-repo CI policy**: must each consumer repo's CI be green before the next wave
   starts, or is local-gates-green sufficient (each repo's CI watching later)? The blanket GO
   does not answer cross-repo sequencing.

## Environment appendix (for the record)

Reboot ~21:45 (uptime 27 min at 22:12). Load 1-min trace: 84→292→446→225→121→104→216→138→127.
BuildFlow failures under load: `test-compile` OOM-killed twice (pre-commit 9 steps failed once);
`BUILDFLOW_MAX_CONCURRENCY=1` made the identical tree pass (30/30). Pre-commit pass times
tonight: 332–383 s (vs 54 s cached). Two interrupted `job_output` reads + one owner GO re-issue
mid-wait; all background jobs verified dead or harvested before this report. Driver stopped at
00:04 (pid confirmed gone) — nothing of mine will mutate the tree during WAIT.
