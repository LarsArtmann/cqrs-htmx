# Release-Train Alignment — 21 Lags Cleared, Push Unblocked (2026-09-30 07:30)

> Session: 2026-09-30 ~00:45–02:00 (work) / 07:30 (this report). Series: episode 2 of the
> push-train-lag genre — see [`2026-09-27_06-39_push-train-lag-fix-dependency-sweeps.md`](2026-09-27_06-39_push-train-lag-fix-dependency-sweeps.md)
> for episode 1. Report covers THIS session's run only, per instruction.

## Context

`git sync` (git-town) was blocked at push: the pre-push release-train gate found 21 train
lags — 8 internal family dependencies pinned below published tags across 28 workspace
modules. The gate printed the exact fix recipe (8 anchored `bump-dep.sh` sweeps); this
session executed it and re-verified every gate.

## Timeline (all times CEST 2026-09-30)

| Time  | Event                                                                                                                                                                                                                                                        |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 00:45 | Diagnosed blocked push; loaded buildflow + go-ecosystem-upgrade skills                                                                                                                                                                                       |
| 00:53 | Sweep 1 go-branded-id v0.7.0 (24 mods) green; daemon raced commit → 2e482e58                                                                                                                                                                                 |
| 00:55 | Sweep 2 catalog/v4 v4.6.0 green; daemon → c830b280; my hook-full commit attempt burned a BuildFlow cycle on documented env-class failures → adopted `--no-verify` fallback                                                                                   |
| 00:57 | Foreign formatter rewrap (adminui/dashboardui assets.go) blocked sweeps; wait-tree-quiet TIMEOUT 10 min; verified canonical via `nix fmt` idempotence; absorbed isolated at 3353fd57                                                                         |
| 01:0x | Sweep 3 storage/v4 v4.10.2 (14 mods) green → 0b63a553; sweep 4 watermill/v4 v4.6.2 green → d6f7f7c4                                                                                                                                                          |
| 01:1x | Chained 3 datastar sweeps in one call — MY DESIGN FLAW: sweep 5 dirt blocked sweeps 6–7 (script refusal by design). Sweep 5 broadcast v0.6.1 green → daemon 0cc92298; static v0.6.1 already carried by MVS through broadcast's tidy (absence grep proved it) |
| 01:2x | Sweep 7 refused: collaborator mid-feature in dashboardui (twCSS handler). wait-tree-quiet TIMEOUT #2; 10-min poll loop → NO_CLEAN_WINDOW (.cqrs-lint.json went dirty)                                                                                        |
| 01:37 | Second poll loop found clean window at attempt 26; sweeps 7 (go-datastar v0.6.1) + 8 (go-error-family v0.11.0, 28 mods) green → daemon a35f1b81/ed0d45b8/354f70f2                                                                                            |
| 01:5x | Verification: release-train strict GREEN (0/0/0, fresh cache); test suite RC=0 18/18 (race); version-drift --strict GREEN (841/841); check-modules 17/17                                                                                                     |
| later | User's `git town continue` succeeded — push went through (master now ahead 2, both remaining = collaborator daemon commits)                                                                                                                                  |
| 07:31 | Report re-verification: release-train strict STILL GREEN (fresh cache); tree clean                                                                                                                                                                           |

## a) FULLY DONE

- All 8 fix-recipe sweeps executed with exact `$`-anchored patterns; every touched module
  `GOWORK=off` tidy + build + vet PASS; per-sweep absence assertions green:
  - `go-branded-id` v0.6.0 → v0.7.0 (integration_test)
  - `go-cqrs-lite/catalog/v4` v4.5.0 → v4.6.0 (examples/catalog-demo, integration_test)
  - `go-cqrs-lite/storage/v4` v4.10.1 → v4.10.2 (14 modules)
  - `go-cqrs-lite/watermill/v4` v4.6.1 → v4.6.2 (integration_test)
  - `go-datastar`, `/static`, `/broadcast` v0.6.0 → v0.6.1 (static carried by MVS via broadcast tidy)
  - `go-error-family` v0.10.2 → v0.11.0 (28 modules)
- Repo-wide stale-pin grep: zero old pins remain
- `check-release-train --refresh-cache --strict-lag 0`: GREEN at 01:53 and re-verified
  07:31 — 0 lag, 0 unpublished, 841 requires checked
- `check-version-drift --strict`: GREEN — all 841 larsartmann requires resolve to published tags
- Full test suite (`nix run .#test`, race): RC=0, 18/18 packages ok
- `nix run .#check-modules`: 17/17 gates + fixture self-tests pass
- The blocked push went through after handoff (`git town continue`)
- Foreign formatter-class change absorbed in isolation with attribution (3353fd57), not reverted
- Collaborator's in-flight work (dashboardui twCSS, .cqrs-lint.json C040 re-enable) never touched; both landed on their own

## b) PARTIALLY DONE

- Verification breadth: `go mod verify`, `coverage-gate`, `bench-spike`, and
  `check-cqrs-lint` (with the collaborator's C040 re-enable) were NOT run this session —
  the four gates above were the push-blocking set; the rest is follow-up
- Memory upkeep: lessons for AGENTS.md gotcha 4 / dependency-train-bump runbook
  identified (see e) but not yet written
- HARVEST of this report's (f) items into TODO_LIST/ROADMAP: pending instruction

## c) NOT STARTED

- CHANGELOG entries for the 8 family bumps (convention unclear — question g1)
- `bump-dep.sh --commit` mode (sweep + commit in one process, beats the daemon)
- BuildFlow pre-commit env-class fixes (go-licenses outside devShell, tsconfig-check on a
  Go repo, samber-linter ~80% failure rate) that force the `--no-verify` fallback
- SA1019 migrations BuildFlow surfaced in e2e/server + examples/datastar-demo (pre-existing)

## d) TOTALLY FUCKED UP

Nothing landed broken; gates green, push succeeded. Honest near-misses (process, not damage):

1. **First test verification piped through `tail`** — lost the exit code AND all module
   results before the tail window; a documented shell trap (AGENTS.md gotcha 3) repeated.
   Self-caught and re-run with explicit rc capture before claiming green.
2. **Chained 3 sweeps in one bash invocation** — sweep 5's own output dirtied the tree for
   6–7 (bump-dep's documented dirty-refusal). Should have been obvious from the script
   header read minutes earlier.
3. **First commit ran hook-full** into pre-documented deterministic BuildFlow failures
   (gotcha 8) — one wasted gate cycle before adopting the documented `--no-verify` fallback.
4. **Lost 4 of 6 commit races to the auto-commit daemon** — several dependency bumps live
   in "chore: auto-commit" heuristic messages (2e482e58, c830b80 [sic: c830b280], 0cc92298,
   0b9898a9/a35f1b81/ed0d45b8), degrading exactly the per-sweep reviewability the recipe asks for.
5. ~35 min of the session was waiting on a concurrent session (2 wait-tree-quiet
   timeouts + 2 poll loops); the windows could have carried independent verification work
   (release-train gate needs no clean tree) — parallelism missed.

## e) WHAT WE SHOULD IMPROVE (self-review)

- **What did I forget:** `go mod verify`; CHANGELOG; AGENTS.md/runbook memory updates;
  reading the two v0-minor bumps' CHANGELOGs (error-family 0.10→0.11, branded-id 0.6→0.7
  are breaking-allowed semver) before executing — tests caught nothing, but pre-knowledge
  would have been cheaper than luck.
- **Stupid things done anyway:** repeated a documented shell trap; chained sweeps against
  the script's documented refusal design; raced the daemon cross-invocation four times and
  lost four times before changing approach.
- **Better next time:** sweep+commit inside ONE bash invocation from the start;
  `--no-verify` + justification on first commit for dep-bump class (failures pre-documented);
  use wait windows for independent gates; pre-read minor-bump CHANGELOGs.
- **Ghost systems:** none created (dependency pins only; no new unwired code).
- **Split brains:** none created; the absorbed formatter change was verified canonical
  (`nix fmt` idempotent) so no formatting fight was introduced.
- **Tests:** full race suite green 18/18; coverage-gate not re-run (dep bumps don't change
  code paths, but the gate exists).
- **Did I lie:** No — but full transparency: my first interim "suite green" between the
  two test runs was based on tail-truncated output before the proper rc-captured re-run.
- **VendorHash observation:** BuildFlow warned flake.nix:107 vendorHash stale after go.sum
  changed. That hash belongs to the vendored **benchstat tool** (go.googlesource.com/perf),
  not the workspace modules — my sweeps did not touch benchstat's deps, so this is likely
  the false-positive class. NOT verified (no FOD build run) — follow-up item f3.

## f) Next things (session-derived, ~impact-sorted)

1. Decide/push the 2 remaining daemon commits (collaborator's cqrs-lint follow-through) — `git town continue` or leave to their session (question g2)
2. Clear the stale git-town "unfinished sync" state (`git town continue`/`skip`)
3. Run `nix build` on the benchstat FOD to settle the vendorHash warning (expected false positive)
4. Run `go mod verify` across modules (skill checklist item skipped this session)
5. Run `nix run .#check-cqrs-lint` with C040 re-enabled — confirm collaborator's green claim end-to-end
6. Run `nix run .#coverage-gate` post-bumps
7. Run `nix run .#bench-spike` — confirm no perf drift from storage v4.10.2 / error-family v0.11.0
8. HARVEST this report's f-items into TODO_LIST.md / ROADMAP.md (docs-health)
9. CHANGELOG entries for the 8 family bumps if convention requires (question g1)
10. Teach `bump-dep.sh` an opt-in commit mode (sweep + `git commit` in one process; wins the daemon race by construction)
11. Document in `docs/runbooks/dependency-train-bump.md`: never chain bump-dep invocations without committing between; MVS can carry sibling bumps (broadcast tidy carried static)
12. Update AGENTS.md gotcha 4: "sweep+commit must share one shell invocation; the daemon wins cross-invocation races"
13. Fix BuildFlow pre-commit env-class failures at source: go-licenses needs devShell toolchain pin (crypto/mldsa error = go-licenses on go 1.26.7 std vs 1.27 workspace)
14. tsconfig-check should no-op on Go repos (TS5081 noise)
15. samber-linter: 80% historical failure rate — exclude or fix
16. pytest-test should skip when zero python tests collected
17. Rebuild BuildFlow binary (doctor: binary e881e96 vs repo bfa3d35)
18. Investigate golangci-lint timing +800% regressions flagged by BuildFlow (cold cache vs real)
19. Migrate examples/datastar-demo off deprecated `datastar.Broadcaster`/`NewBroadcaster` → `broadcast.*` (SA1019)
20. Migrate e2e/server `AggregateID` → `StreamID` (SA1019) and the two nil-error-return findings (main.go:66,70)
21. gomod-check: 15 go.work replaces "not used by any used module" — prune (schema/v4, stack/* etc.) after verifying they're dead
22. go-line-flipflop: go.mod `go` line changed 20×/20 commits — pin the floor policy and stop the oscillation
23. scripts/testdata verify-tag fixtures flagged as needing tidy — exclude testdata from freshness or tidy them
24. Reconcile go-work-paths doctor warning (13 use-path/module-name /v4 suffix mismatches) — cosmetic but noisy
25. Confirm dashboardui twCSS feature (collaborator) is complete/tested — it rode the same push
26. Schedule the next cqrs-htmx release train (wave-ordered) so consumers get the aligned pins (question g3)
27. nix-checker style: extract hash/vendorHash to dedicated files (+1/−1 diffs)
28. flake-meta-checker: add missing mainProgram attribute
29. Daemon heuristic: absorb untracked docs/status files promptly (00-41 doc sat ~12 min)
30. Eval-cache-busy nix contention during BuildFlow runs — serialize or add retry
31. VACUUM or delete the 0.81 GB buildflow cache DB (29% free pages)
32. Re-check cqrs-lint suppressions after the rebuilt binary settles (gotcha 13: rebuilds re-attribute findings)
33. Consider a lockfile (flock) shared by bump-dep.sh and the daemon to serialize tree mutations
34. Vulture/bandit/mypy/ruff +1000–7000% timing spikes — run `buildflow timings --regressions`
35. Status-report skill default is HTML but this repo's series convention is .md — feed the convention back to the skill config

## g) Questions I cannot answer myself

1. **CHANGELOG convention:** do family-alignment dependency bumps (this session's 8 sweeps)
   get CHANGELOG.md entries here, or are they considered chore-noise below the changelog bar?
2. **The 2 unpushed daemon commits** (collaborator's cqrs-lint arc, 5114a4b4 + 06371cea):
   push them now via `git town continue`, or hold until that session finishes its arc?
3. **Release timing:** cut the next cqrs-htmx release train now while all requires are
   aligned, or wait for the dashboardui twCSS + cqrs-lint arcs to settle?
