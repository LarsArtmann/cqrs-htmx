# Plan: Purge `origin/v4` branch history (3 binary blobs, ~27.7 MB)

**Status:** PREPARED, awaiting user approval (force-push class).
**Prepared:** 2026-08-30 · Source: TODO_LIST P-gated item.

## Current state

- `origin/v4` carries 3 remaining binary blobs (~27.7 MB total) that master
  already shed (examples binaries are gitignored; HEAD is clean).
- The branch is stale: master is the integration branch; v4 predates the
  multi-module layout.

## Options

1. **Delete `origin/v4` (RECOMMENDED).** Zero history risk: master is
   self-contained, tags are the release record, and GitHub keeps the ref
   recoverable for ~90 days. `git push origin --delete v4`.
2. Rewrite the branch without the blobs (`git filter-repo --path ... --invert-paths`
   on a v4 worktree, then `--force-with-lease` push). Only worth it if the
   branch's history has standalone value — it does not.

## Execution (option 1)

```sh
git fetch origin
git log --oneline origin/master..origin/v4 | wc -l   # confirm nothing master needs
git push origin --delete v4
git push origin --prune                              # drop the local remote ref
```

## Verification

- `git ls-remote origin refs/heads/v4` → empty.
- Fresh clone size drops (~28 MB less); `git count-objects -vH` locally after
  `git gc --aggressive --prune=now` (local only — the server reclaims lazily).
