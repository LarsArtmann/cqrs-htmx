# Session start — three commands

> The smallest reliable orientation for a fresh session. Everything else is optional reading.

```bash
git status --short && git log --oneline -5      # 1. where the tree actually is
cat CURRENT_FOCUS.md                            # 2. the ONE active thread + its "done when"
nix run .#check-modules                         # 3. is the tree green before I touch it?
```

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
