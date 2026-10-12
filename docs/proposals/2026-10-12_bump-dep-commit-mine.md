# Proposal: `bump-dep --commit-mine` — ownership-scoped sweep commits

**Status:** draft, 2026-10-12 (18-20 plan A7/G21). Internal tooling proposal —
bump-dep lives in this repo (`scripts/tools/bump-dep.sh`), so there is no
external upstream to file against; if the pattern is ever promoted into
BuildFlow's go-mod tooling, this doc is the filing draft.

**Problem — the half that is still true (2026-10-12):** `--commit` stages with
`git add -u` (bump-dep.sh:224) — ALL tracked modified files repo-wide, including
any file a concurrent session authored mid-sweep. Concurrent sessions are the
norm on this tree (gotcha 4), so a sweep commit can silently carry foreign work
under a deps-only subject.

**What already exists (do not re-propose):** the commit rc IS checked
(bump-dep.sh:232-251, landed 2026-10-07 plan R18) — a failed commit prints
"did NOT land (rc=…); changes remain STAGED" plus the daemon-absorption caution.
The 2026-10-12 status report's §d1 leg-1 phrasing ("prints success WITHOUT
checking") predates R18 and is corrected in that report's annotation.

**Proposed flag:** `--commit-mine`

- Stage ONLY the paths the sweep itself edited — the tool knows its edit set
  (it produced every `go.mod`/`go.sum` mutation); collect them during the sweep
  and `git add <that-list>`.
- Refuse when a tracked-modified file exists OUTSIDE the edit set: print the
  foreign paths and abort the commit (the sweep edits stay in the working tree;
  the operator decides). This is the expected-dirty-set precondition from the
  G20 wrapper sketch, embedded where the sweep actually commits.
- Keep the existing rc-check and postcondition behavior; print the new HEAD.

**Acceptance (extends `scripts/selftests/test-bump-dep.sh`):**

1. With a foreign dirty file present, `--commit-mine` refuses and the commit
   contains only the sweep's paths (fixture case).
2. Deliberately failing pre-commit ⇒ nonzero exit, no success claim (regression
   pin for the R18 behavior).
3. Plain `--commit` unchanged this train; deprecate toward `--commit-mine`
   semantics after one quiet train.

**Cost:** S — the edit set already exists in-process; ~30 lines + two fixture
cases. Companion: the G20 wrapper sketch in `docs/agents-notes.md` (same date)
generalizes the precondition/postcondition to every sweep-type tool.
