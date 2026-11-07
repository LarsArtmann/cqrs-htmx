# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-07 (evening, blanket-directive execution) · **Owner:** Lars · **Module count:** 30 (`go.work`)

## The active thread

**Blanket-directive execution — the 2026-11-07 wave landed; drain the receipts and the two owner-held packets.**
Three tags shipped this evening on a strict 0/0 train: **root v4.18.1** (pristine htmx 2.0.11 +
idiomorph 0.8.0 + SSE 2.2.4 assets with sha pins), **loginpage v4.17.0** (train-alignment maintenance
release), **huma v4.0.0** (first tag of the Huma v2 adapter, cqrs-htmx#41). All 21 consumer requires
swept, e2e/server's stale third-party family pins swept (that sweep also repaired a hermetic typecheck:
go-health-dashboard v0.12.1 was templ-poisoned), docs-gates **5/5 PASS** (audit-all pinned ≤268 and held),
tail back at 13 after archiving the 18-49 report.

Two decision packets now sit one tick from GO, both with recommendations pre-staged:

- **M2M D1–D4** ([`docs/proposals/2026-11-07_m2m-owner-decisions-d1-d4.md`](docs/proposals/2026-11-07_m2m-owner-decisions-d1-d4.md)) —
  no plan-recorded defaults exist (the plan §9 says each answer is un-inferable), so W1/W2 code waits for
  the tick; the packet's recommendations (supported / event-derived scopes / single-delegator /
  per-authentication-outcome events) make the tick one word per row.
- **usermgmt → system/metaengine D1–D5** ([`docs/proposals/2026-11-07_usermgmt-system-metaengine-migration-decisions-d1-d5.md`](docs/proposals/2026-11-07_usermgmt-system-metaengine-migration-decisions-d1-d5.md)) —
  defaults adopted under the directive; W1–W2 probes are GO without further owner input.

Consumer waves C27 landed where the tree allowed: browser-history (durable `GrantRoles` bootstrap),
InboxClean (`sseDeadlineFree` deleted — the pinned v4.16.0 stream handler owns its deadline),
github-local-sync (route-posture manifest + truth-sync tests + their stale 1070-era `dashboard_templ.go`
regenerated). **Swetty/SwettySwipperWeb is unlocatable** (not local, not in `gh repo list`, not archived) —
the row records it. webphone's leg is deferred (active session in that tree at execution time).

**Done when** the post-tag battery is green on master's CI (check-modules, coverage-gate, erraudit
receipt, badges ✓, dependabot 0 ✓), the C27 leftovers are dispositioned (webphone when quiet, Swetty
row trued), and the two packets sit in the owner's queue with nothing else blocking.

## The next thread (pick ONE)

1. **C28 — dispatch-gate adoption in the consumer repos that dispatch own commands**
   (browser-history, InboxClean; github-local-sync is vacuous — zero commands by design;
   webphone deferred). Each repo: `cqrshtmx.NewDispatchGate` on its dispatcher, per-command
   posture declarations with reasons, `authztest.CommandSuite` in its own suite
   ([docs/guides/authorization-system.md](docs/guides/authorization-system.md) § dispatch gate).
2. **v4.19.0 wave planning** — the HTTP QUERY remainder rides it: W3 DataStar lane
   (fork `go-datastar/static` bundle bump, gated on NV1/D4), W5 upstream legs (httputil CSRF
   safe-list, verified-only proxy table). ROADMAP q35's 4.x-default clock started at v4.18.0.
3. **NV4 probe** (ExtSSE/ExtIdiomorph compat under htmx 4.x) — the sibling e2e lane covers the
   runtime battery; the library-level verdict + the `examples/basic` htmx-4 dogfood are ours.

## Ground rules that keep this honest

1. **One thread at a time.** Anything not listed above is in [TODO_LIST.md](TODO_LIST.md) and waits.
2. **Never add a row to `TODO_LIST.md` without a verify command** or a named evidence path.
3. **Never leave a finished row in `TODO_LIST.md`** — it moves to [CHANGELOG.md](CHANGELOG.md).
4. **Closed history is not documentation.** `*/archived/` is out of the reading path by design;
   open it only for provenance questions.
5. **Master stays green.** Never push a stack whose battery is red on content (the 15-57 hold rule).
6. **One session per working tree.** A second agent in the same checkout will stash your
   uncommitted work (a Git Town run took 59 files and left a conflicted merge on
   2026-11-02). Use `git worktree add ../<repo>-<thread> -b <branch>` — [gotcha 47](docs/gotchas.md#gotcha-47).
