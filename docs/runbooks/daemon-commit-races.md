# Runbook: Auto-Commit Daemon Races (HEAD lock, shredded commits)

Recovery procedure for the shared-tree failure class where the auto-commit
daemon (or a concurrent session) interleaves with your git operations.
Prevention lives in `scripts/preflight-tree-check.sh` and
`scripts/wait-tree-quiet.sh`; this page is what to do AFTER an interleave.

Historical narrative and quantification: `docs/agents-notes.md` ("The
five-wave evening" and the gotcha-4 history). Standing rules: AGENTS.md
gotcha 4.

## Symptom 1: `fatal: cannot lock ref 'HEAD'` (or `.git/index.lock`)

Your `git commit` lost the lock race against the daemon's own commit.

1. **Do not retry blindly.** First check what the daemon did:
   `git log -3 --oneline && git show --stat HEAD`.
2. **If the daemon committed your staged set** (a `chore: auto-commit …
   (heuristic)` message containing exactly your files): you are done —
   content is what matters. Optionally amend the message (see rules below).
3. **If your changes are still staged/dirty**: re-verify with
   `git status --porcelain`, then retry the commit ONCE. If the daemon
   wins again, accept daemon pickup — staging and waiting beats racing.
4. **If your changes vanished from status entirely**: the daemon committed
   them; find them with `git log -- <paths>` before assuming loss. NEVER
   `git reset --hard` / `git checkout` / `git clean` to "recover".

## Symptom 2: your logical change is shredded across heuristic commits

The daemon polls faster than a long verification tail; batched work lands
as several `chore: auto-commit` commits with correct content but useless
messages. This is expected behavior, not corruption.

- **Verify content, not messages**: `git show --stat <hash>` per commit.
- **Amend rules** (all three must hold):
  1. the commit is the current tip,
  2. it contains ONLY your files (no foreign in-flight work mixed in),
  3. it is local-only (not on origin).
  Otherwise leave the heuristic message standing — rewriting interleaved
  history is sabotage of the other session's work.
- **The amend dance is theater at scale**: the daemon re-races within
  seconds (six shredded commits in one evening despite phase-boundary
  discipline; two more the next round). Amend opportunistically; do not
  block on it.

## Symptom 3: a mid-batch `git add` split across commits

If the daemon commits while you are still staging a multi-file batch, the
result is two commits (daemon-partial + your remainder). Verify both
together contain the full set, then continue — the split is cosmetic.

## Prevention (the mechanical protocol)

| When | Tool |
| ---- | --- |
| Before ANY batch-mutating loop | `nix run .#preflight-tree-check` (abort on surprise: dirty tree, fresh foreign commit, velocity burst) |
| Before push retries / acting on possibly-orphaned state | `nix run .#wait-tree-quiet` (clean tree + stable HEAD for a full window) |
| Always | commit at phase boundaries; verify → commit → next phase |
| Never | `reset --hard`, `checkout`, `clean`, force push, reverting diffs you did not author |

## Related

- `docs/runbooks/dependency-train-bump.md` — sweeps refuse dirty trees
  precisely because of this class.
- The pre-push CI-parity hook (`.githooks/pre-push`) re-runs blocking
  gates at push time; a daemon commit that lands mid-hook does not poison
  the push (gates re-evaluate the tree they see).
