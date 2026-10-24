# Archive eviction — prepared, not yet executed

**Prepared:** 2026-10-24 · **Status:** ⏸ **BLOCKED on the owner answer to Q2** (where the closed
history goes). Everything except the deletion is done: the dry run is green, the keep-list is
measured, and the removal is one reviewed command.

> **Why this file exists.** 578 of the 591 files under `docs/**/archived/` and
> `docs/**/processed/` are closed history that no living document cites. They sit inside every
> `grep`, every file listing, and every agent's context window — 43% of the repository. Removing
> them is safe _only_ because the citation set was measured, not guessed.

## The measured keep-set (13 files)

Twelve files survive because a living document cites them (or a kept archive file does, one hop
out); every archive `README.md` survives because it is the pass manifest.

| #   | File                                                                                     | Cited by                                                    |
| --- | ---------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| 1   | `docs/planning/archived/2026-06-21_07-31_identity-model-redesign-pareto.md`              | `docs/adr/0015-identity-model-redesign.md`                  |
| 2   | `docs/planning/archived/2026-06-21_08-23_membership-wiring-execution.md`                 | `docs/adr/0015-identity-model-redesign.md`                  |
| 3   | `docs/planning/archived/2026-06-28_09-38_offline-first-command-sync.md`                  | `docs/adr/0023-command-sync.md`                             |
| 4   | `docs/planning/archived/2026-08-30_appkit-foldin-revalidation.md`                        | `docs/planning/2026-08-30_gated-work-index.md`              |
| 5   | `docs/planning/archived/2026-08-30_upstream-issue-drafts.md`                             | `docs/planning/2026-08-30_gated-work-index.md`              |
| 6   | `docs/status/archived/2026-06-21_07-58_identity-redesign-brutal-self-review.md`          | `docs/adr/0015-identity-model-redesign.md`                  |
| 7   | `docs/status/archived/2026-10-05_14-59_branching-flow-full-analysis.md`                  | `docs/analysis/triage-decisions.md`                         |
| 8   | `docs/status/archived/2026-10-06_19-31_SUPERB-hardening-m11-m12-done-m13-m15-pending.md` | the live hardening plan                                     |
| 9   | `docs/status/archived/2026-10-06_23-23_SUPERB-m05-m10-brutal-status-review.md`           | `docs/status/2026-10-07_00-15_…` (live until the tail pass) |
| 10  | `docs/status/archived/2026-10-06_21-35_SUPERB-hardening-m11-m15-complete.md`             | transitive: #8 links to it                                  |
| 11  | `docs/status/archived/2026-10-06_23-20_SUPERB-phase2-complete-m05-m10.md`                | transitive: #9 links to it                                  |
| 12  | `docs/status/archived/2026-10-06_22-34_SUPERB-phase1-complete-phase2-m06-midedit.md`     | transitive: #11 links to it                                 |
| 13  | every `archived/README.md`                                                               | the bulk-pass manifest (gate-exempt, kept by pattern)       |

The machine-readable list is [`ARCHIVE_KEEP.txt`](status/ARCHIVE_KEEP.txt).

## What would be removed — 578 files

| Directory                       |   Total | To remove |                        Kept |
| ------------------------------- | ------: | --------: | --------------------------: |
| `docs/status/archived/`         |     473 |       465 | 8 (7 reports + `README.md`) |
| `docs/planning/archived/`       |     104 |        99 |                           5 |
| `docs/feedback/processed/`      |      10 |        10 |                           0 |
| `docs/reviews/archived/`        |       2 |         2 |                           0 |
| `docs/modularization/archived/` |       1 |         1 |                           0 |
| `docs/plans/archived/`          |       1 |         1 |                           0 |
| **Total**                       | **591** |   **578** |                      **13** |

## How the list was derived (re-run this, never hand-edit)

1. Enumerate every `.md` outside any `archived/`/`processed/` directory — the 293 live docs.
2. Parse every markdown link target in those files; normalise relative to the file's directory.
3. Intersect with the 591 archived files. Result: **10 direct edges** (9 file targets plus the
   archive README).
4. Take the transitive closure one hop into the kept archive files. Result: **3 more targets**
   (`2026-10-06_21-35`, `…_23-20`, `…_22-34`) — each is cited by a file that is itself kept.
5. Everything else is unreferenced closed history.

## The command

```bash
# dry run — prints the per-directory counts above, removes nothing (DEFAULT)
bash scripts/tools/evict-archive.sh

# dry run with every path listed
bash scripts/tools/evict-archive.sh --list

# the eviction itself (refuses without --yes; stages a `git rm`)
bash scripts/tools/evict-archive.sh --apply --yes

# then commit the deletion + this file's "EXECUTED" stamp together
```

The tool refuses to run when the keep-list is empty or contains a path that no longer exists, so a
stale list cannot silently wipe the corpus.

## The restore command

Deletion goes through `git rm`, so the files stay in history. Record the pre-eviction commit:

```bash
BEFORE=$(git rev-parse HEAD)   # capture BEFORE committing the eviction
#   ... commit the eviction ...
git revert --no-edit <eviction-commit>                     # whole-batch restore
# or, selectively:
git restore --source="$BEFORE" -- docs/status/archived/    # one directory
```

`git revert` is the preferred path: it restores the exact tree the eviction commit removed, with a
receipt in history, and leaves the working tree consistent with `ARCHIVE_KEEP.txt`.

## Dry-run verdict (S11, 2026-10-24)

Executed in a throwaway `git worktree add --detach` checkout with the 578 removals applied:

| Gate                          | Result                                                               |
| ----------------------------- | -------------------------------------------------------------------- |
| `check-docs-links.sh`         | **rc=0** — 640 links resolve (after the keep-set grew from 10 to 13) |
| `check-docs-freshness.sh`     | **rc=0**                                                             |
| `check-status-annotations.sh` | **rc=0** — 5 gated reports annotated, 1 legacy-exempt                |
| `check-status-rows.py`        | **rc=0** — 7 files PARTIAL-free, 1 deliberately-mixed table          |
| `check-docs-tail-budget.sh`   | **rc=0** (advisory; the live tail is a separate milestone)           |
| `check-large-files.sh`        | **rc=0**                                                             |

The first dry run found **one** broken link (`…_23-20` → `…_22-34`), which is exactly why the
dry run exists: the plan predicted 9 keep targets; measurement says 13.

## Execution checklist (when Q2 is answered)

1. `df -h` and `git status` — expect a clean tree with no foreign commits.
2. `bash scripts/tools/evict-archive.sh --list | wc -l` — expect 579 (1 header line + 578 paths).
3. `bash scripts/tools/evict-archive.sh --apply --yes`.
4. Run the six gates above; all must stay at the verdicts in the table.
5. Record `git rev-parse HEAD~1` (the pre-eviction commit) in this file, remove the ⏸ banner.
6. Commit: `chore(docs): evict 578 unreferenced archive files`.
7. Update the corpus metrics in the master plan's Appendix D.
