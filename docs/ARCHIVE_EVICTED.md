# Archive eviction — EXECUTED

**Prepared:** 2026-10-24 · **Executed:** 2026-10-24 (same day) · **Status:** ✅ **DONE**

> Q2 ("where does closed history go") was answered **delete + manifest** by the owner's
> 2026-10-24 directive to execute the plan of record end to end. Everything is recoverable
> from git; nothing outside version control was touched.

## What was removed — 617 files

| Directory                       | Total at execution | Removed | Kept             |
| ------------------------------- | -----------------: | ------- | ---------------- |
| `docs/status/archived/`         |                532 | 491     | 41 + `README.md` |
| `docs/planning/archived/`       |                120 | 112     | 8                |
| `docs/feedback/processed/`      |                 10 | 10      | 0                |
| `docs/reviews/archived/`        |                  2 | 2       | 0                |
| `docs/modularization/archived/` |                  1 | 1       | 0                |
| `docs/plans/archived/`          |                  1 | 1       | 0                |
| **Total**                       |                666 | **617** | **49**           |

The prepared numbers (578 remove / 13 keep) measured the tree BEFORE the round-23 status
archive (37 reports), the S16 planning archive (16 plans), and their citation effects. The
keep-set grew because (a) the live hardening plan and master plan cite their own execution
receipts, and (b) kept files link each other — the keep-set is the **transitive closure**
over links from living documents, which is exactly what `check-docs-links.sh` requires to
stay green.

## The keep-set (49 files)

Machine-readable: [`status/ARCHIVE_KEEP.txt`](status/ARCHIVE_KEEP.txt). Composition:

- 6 files cited by ADRs / `docs/analysis/triage-decisions.md` (the original measured core),
- the live hardening plan's 6 execution receipts + the master plan's 2 predecessor reports,
- the transitive closure over links from those files (successor chains and cross-receipts),
- every archive `README.md` (the pass manifests).

## How the list was derived (re-run this, never hand-edit)

1. Enumerate every `.md` outside any `archived/`/`processed/` directory — the live docs.
2. Parse every markdown link target in those files; normalise relative to the file's directory.
3. Intersect with the archived files — the direct citation set.
4. Take the transitive closure over links FROM the kept archive files (kept files must not
   link evicted files: the link gate checks archives too).
5. Add every archive `README.md`.
6. Everything else is unreferenced closed history.

## The restore command

Deletion went through `git rm`, so the files stay in history:

```bash
BEFORE=e8ce77d4f96ae2df14f5c86f087389f9982c4a99   # pre-eviction commit
git revert --no-edit <eviction-commit>            # whole-batch restore
# or, selectively:
git restore --source="$BEFORE" -- docs/status/archived/    # one directory
```

`git revert` is the preferred path: it restores the exact tree the eviction commit removed,
with a receipt in history, and leaves the working tree consistent with `ARCHIVE_KEEP.txt`.

## Dry-run verdicts (both scratch passes green)

- S11 (2026-10-24, prepared set): 6/6 gates green in a detached worktree; found one broken
  link and grew the keep-set from 10 to 13.
- Round-23 execution (2026-10-24, final set): 6/6 gates green in a fresh detached worktree
  BEFORE the real apply, and again on the real tree after.

## Addendum — generated-artifact pass (same day, S21+S29)

A second measured pass removed **77 more files** under the same living-citation rule:
14 `docs/status/*.html` generated reports (round-14 KEEP-IN-PLACE verdict superseded by the
master plan), 19 uncited `docs/research/` runs (12 cited reports kept), and 44 uncited
`docs/architecture-understanding/` snapshots (2 composability reviews kept). Restore source
is the same pre-eviction commit family; `git restore --source=<pre-addendum-commit> -- docs/research docs/architecture-understanding docs/status` recovers them.

## Pending dispositions (owner-gated — recorded so audits start from state, not archaeology)

**The 7 uncited non-review `2026-0*` generated artifacts (round-37 §g1, opened 2026-10-27;
re-verified 0 living citations each on 2026-11-04 by docs-health round 46).** Evictable under
the standing 2026-10-24 cite-or-evict rule, but the owner has not answered evict-vs-keep, and
`docs/reviews/` has an explicit written "restraint is success" convention these directories
lack — so the hold stands until the owner ticks one:

| File                                                        | Class        | Hold reason                                    |
| ----------------------------------------------------------- | ------------ | ---------------------------------------------- |
| `brainstorming/2026-07-03_verifiable-credentials-vs-crypto-stack.html` | generated | no dir convention; owner verdict pending |
| `brainstorming/2026-07-23_cqrs-dashboard-design.html`       | generated    | no dir convention; owner verdict pending       |
| `brainstorming/2026-07-23_cqrs-dashboard-v2-self-critique.html` | generated | no dir convention; owner verdict pending  |
| `modularization/2026-06-21_EXECUTION_PLAN.html`             | generated    | no dir convention; owner verdict pending       |
| `modularization/2026-06-21_PROPOSAL.html`                   | generated    | no dir convention; owner verdict pending       |
| `modularization/2026-08-05_PROPOSAL.html`                   | generated    | no dir convention; owner verdict pending       |
| `proposals/2026-07-19_10-02_nix-flake-migration-status.html` | generated   | no dir convention; owner verdict pending       |

Default if the owner stays silent: prepare the measured eviction dry-run and hold before apply
(the round-37 posture). The 24 uncited `docs/reviews/*.html` files are NOT pending — they are
kept by the reviews dir's own written convention (`docs/reviews/README.md`).

**Round-46 `2026-0*` corpus manifest (2026-11-04, single-archive pass — commit message is the
manifest per the single-file rule, this table is the durable classification):**

| File                                              | Decision | Reason                                                                                      |
| ------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------- |
| `research/2026-09-20_theme-toggle-strategy-spike.md` | ARCHIVE | every option dispositioned inline (A shipped, B/defer not taken); moved to `research/archived/` after check-rows + bare-rows gates green |
| `research/2026-08-02_iroh-p2p-networking-fit-analysis.md` | SKIP | complete verdict (not adopted, decision triggers recorded), cited by ROADMAP + 2 research files |
| `proposals/2026-07-25_data-mesh-interchange.md`   | SKIP     | open proposal, verdict + gaps routed in ROADMAP § Data Mesh Interchange                      |
| 24 uncited `reviews/*.html`                       | KEEP     | reviews dir "restraint is success" convention                                                |
| 15 cited `2026-0*` files (research/planning/brainstorming/modularization/architecture-understanding) | KEEP | living-doc citations = load-bearing rationale (citation map re-derived 2026-11-04) |
