# Plan: Purge the setup-demo binary blob from pushed master history

**Status:** PREPARED, awaiting user approval (force-push class).
**Prepared:** 2026-08-30 · Source: TODO_LIST P-gated item.

## Current state

- `examples/setup-demo/setup-demo` (27 MB compiled binary) was removed from
  HEAD and from the 17 then-unpushed commits on 2026-08-14 (filter-branch;
  the 6 family tags riding those commits were re-pointed and re-signed;
  backup ref `backup/pre-blob-purge` + `refs/original/*` kept locally).
- The blob STILL EXISTS in already-pushed history (range `5604e810..73ff1556`),
  so every fresh clone downloads it.

## Plan

1. **Preconditions:** all family tags pushed and verified (`git ls-remote --tags`
   vs local); no open PRs based on pre-2026-08-14 history; a fresh clone as
   the crash target.
2. **Rewrite (in the fresh clone, never the working repo):**

```sh
git clone --no-checkout git@github.com:larsartmann/cqrs-htmx.git rewrite
cd rewrite
git filter-repo --invert-paths --path examples/setup-demo/setup-demo --force
# filter-repo drops remotes by design; re-add and force-push with lease:
git remote add origin git@github.com:larsartmann/cqrs-htmx.git
git push origin --force-with-lease master
# re-push ALL family tags touched by the rewrite (they were re-signed
# locally on 2026-08-14 — confirm tag→commit mapping before pushing):
git push origin --tags --force-with-lease
```

3. **Aftermath:** announce the rewrite (all clones must re-clone, not pull);
   GitHub support request to run an immediate server-side GC (otherwise the
   blob persists via cached refs for ~90 days); locally `git gc` + drop
   `refs/original/*` and the backup branch once confident.

## Risk

- force-push master + tags — the exact class the tag protocol forbids for
  NEW mistakes; this one repairs an OLD mistake and is only safe because
  the module proxy serves tags by name+content that consumers already
  resolved (verify no consumer pinned pre-rewrite tag hashes).
- The module proxy caches every tag FOREVER by (name, commit): re-pushing
  identical tag names pointing at REWRITTEN commits will be REJECTED by
  sum.golang.org for anyone who already resolved them. **This makes the
  rewrite consumer-hostile unless every rewritten tag was never resolved
  by anyone.** Given the family tags are publicly live, the honest
  assessment: the blob costs ~27 MB per clone; the rewrite costs every
  existing consumer a broken `go get` unless coordinated. **Recommendation:
  DO NOT rewrite. Close this item as "accepted cost"** — or ask GitHub
  support to purge the reachable-blob caches only if clones matter.
