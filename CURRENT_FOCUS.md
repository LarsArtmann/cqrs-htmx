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

**Progress (2026-10-24):** counts in `docs/status/README.md` are derived, not hand-maintained;
`check-todo-hygiene` + the tail **ratchet** + the eviction tool are wired into `check-modules`
and CI; the eviction is measured (578 files, 13 kept) and one command away
([`docs/ARCHIVE_EVICTED.md`](docs/ARCHIVE_EVICTED.md), waiting on Q2); the tail went 58 → 44 via
14 superseded reports. Remaining: the rest of the tail, the eviction itself, the AGENTS module
split (deferred: the always-loaded file is already 147 → 127 lines and the remaining blocks are
the high-frequency surface).

## The next thread (start immediately after)

**Close the open-issue sweep.** Five issues are shipped but not closed (#23/#28/#29/#18/#19);
`origin/wip/issue-23-auth-audit` still exists. Plan:
[`docs/planning/2026-10-19_03-00_SUPERB-open-issue-sweep-pareto-plan.md`](docs/planning/2026-10-19_03-00_SUPERB-open-issue-sweep-pareto-plan.md).

**Done when:** `gh issue list --state open` shows only issues opened after 2026-10-19, and the
`wip/` branch is gone from origin.

## Resolved since the last update

**The systemadapter declarative red is gone.** It was the module-CI red for a week: ten
`Declarative__`/`Harness_` tests failed with `system: not found` because published
`metaengine/projectionadapter` keyed views with the display form of a stream ID. The upstream fix
was published and pinned here on 2026-10-19 (`99e0df63`, v4.6.1); the hermetic suite is green
(`GOWORK=off go test ./... -race`). Nothing left to watch.

## Ground rules that keep this honest

1. **One thread at a time.** Anything not listed above is in [TODO_LIST.md](TODO_LIST.md) and waits.
2. **Never add a row to `TODO_LIST.md` without a verify command** or a named evidence path.
3. **Never leave a finished row in `TODO_LIST.md`** — it moves to [CHANGELOG.md](CHANGELOG.md).
4. **Closed history is not documentation.** `*/archived/` is out of the reading path by design;
   open it only for provenance questions.
