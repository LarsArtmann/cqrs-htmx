# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-10-23 · **Owner:** Lars · **Module count:** 29 (`go.work`)

## The active thread

**Shrink the documentation corpus so a fresh session can find what matters.**
43% of this repository was closed historical documentation (695 files / 189k lines) and the
first file every session opened had an 86%-dead "High impact" section. Plan:
[`docs/planning/2026-10-24_05-36_SUPERB-master-plan-docs-eviction-delivery-and-backlog-burn-down.md`](docs/planning/2026-10-24_05-36_SUPERB-master-plan-docs-eviction-delivery-and-backlog-burn-down.md)
(supersedes the 2026-10-23 plan for execution order; that file keeps the measured evidence).

**Done when:** `docs/status/` + `docs/planning/` + `docs/reviews/` + `docs/brainstorming/`
hold ≤25 live files, the evicted history has a manifest with a restore command, `TODO_LIST.md`
carries zero dead rows, and the four new hygiene gates run in `check-modules` + CI.

## The next thread (start immediately after)

**Close the open-issue sweep.** Five issues are shipped but not closed (#23/#28/#29/#18/#19);
`origin/wip/issue-23-auth-audit` still exists. Plan:
[`docs/planning/2026-10-19_03-00_SUPERB-open-issue-sweep-pareto-plan.md`](docs/planning/2026-10-19_03-00_SUPERB-open-issue-sweep-pareto-plan.md).

**Done when:** `gh issue list --state open` shows only issues opened after 2026-10-19, and the
`wip/` branch is gone from origin.

## The blocked thread (watch, do not start)

**The systemadapter declarative red** — the single module-CI red. Root cause is fixed but
untagged upstream (`metaengine/projectionadapter/v4`, commit `b30156686` on go-cqrs-lite master).

**Done when:** upstream publishes the tag, this repo re-pins it, and
`nix run .#test-systemadapter` is green.

## Ground rules that keep this honest

1. **One thread at a time.** Anything not listed above is in [TODO_LIST.md](TODO_LIST.md) and waits.
2. **Never add a row to `TODO_LIST.md` without a verify command** or a named evidence path.
3. **Never leave a finished row in `TODO_LIST.md`** — it moves to [CHANGELOG.md](CHANGELOG.md).
4. **Closed history is not documentation.** `*/archived/` is out of the reading path by design;
   open it only for provenance questions.
