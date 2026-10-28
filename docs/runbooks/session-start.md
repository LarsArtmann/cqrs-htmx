# Session start — four commands

> The smallest reliable orientation for a fresh session. Everything else is optional reading.

```bash
git fetch origin --tags && git branch --show-current && git status --short && git log --oneline -5   # 1. WHERE AM I — fetch FIRST (a stale local origin/master silently omits just-landed merges), then branch: the auto-commit daemon + owner branch switches relocate work silently (round-42 split-brain)
cat CURRENT_FOCUS.md                            # 2. the ONE active thread + its "done when"
nix run .#check-modules                         # 3. is the tree green before I touch it?
nix run .#check-release-train -- --refresh-cache --strict-lag 0   # 4. has upstream tagged anything since? (a docs-only commit can redden master)
```

> **One writer per worktree.** The daemon commits whatever branch is checked out in the
> shared tree, so a second session working there collects work it never wrote. Start
> queue/release work in a dedicated worktree (`git worktree add /tmp/cqrs-work -b <branch>
> origin/master`) — after the fetch above.

## What is loaded automatically (and what is not)

- `AGENTS.md` — auto-loaded by Crush for this project. It is deliberately small; the long form
  lives in `docs/gotchas.md`, `docs/agents-notes.md`, and the per-module READMEs.
- `docs/agents-notes.md` — **not** auto-loaded. Open it only for a provenance question ("when/why
  was this decided?"), never as a backlog source.
- `docs/status/*.md` — not auto-loaded, and the tail is ratcheted to shrink. Use
  `CURRENT_FOCUS.md` and `TODO_LIST.md` first.

## Before you finish

1. Run the milestone's named gate and capture the rc before any other assignment.
2. Commit at the milestone boundary — never batch two milestones.
3. If the live status tail grew, archive a replaced report and lower
   `docs/status/TAIL_BASELINE` in the same commit (the ratchet gate fails otherwise).
4. Same rule for the whole corpus: markdown under `docs/` past `docs/CORPUS_BASELINE`
   fails `check-docs-corpus-budget --ratchet`, so either shrink something or bump the
   baseline in the commit that justifies the growth.
5. If the session touched error construction or `//nolint:erraudit` directives: run the
   canonical strict audit per module — `GOEXPERIMENT=jsonv2 erraudit lint ./... --max
   --disable generic_return` — plus `nix run .#erraudit-inventory`. The disable flag is
   the recorded posture, not an escape hatch (gotcha 44: `--max` bypasses ALL nolints,
   and `generic_return` fires on the decoder's callback-bound identity, a documented FP
   class — an annotation asserts an invariant a test backs, never the reverse).
