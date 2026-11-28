# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-28 14:00 CET (every number below was re-measured by command that day) · **Owner:** Lars · **Module count:** 31 (`go.work`)

## The active thread

**Everything queues behind the next root tag — `v4.20.0` — and the tag is one owner action.** The last train is from 2026-11-24 (root v4.19.1, identity-model v4.18.1, usermgmt v4.21.1, adminui / dashboardui / setup v4.18.1, loginpage v4.19.1, systemadapter v4.14.1, health v4.15.1, datastar v4.13.1, huma v4.1.1; legalparty-model has no tag yet). Since then `master` carries unpublished work in nine modules' `[Unreleased]` sections: root 11 entries (the props seam `QuerySet`/`Views`/`OnLoad`, client tracking's `TrackHandler`), adminui 31, dashboardui 4 (one BREAKING: the `/-/` route prefix is gone), identity-model 2, usermgmt 3 (the `BotService`/`TenantService` composition), systemadapter 1, datastar 2, oauth2 2, totp 1.

A sub-module test or example may only assert PUBLISHED root behavior (gotcha 30), so three pieces are red by construction until the tag exists, in this order:

1. **Client tracking, setup stage** — finished on `feat/client-tracking-setup`, lands after root v4.20.0 (TODO row 167 carries the exact sequence).
2. **`examples/basic` on the props seam and its three browser specs** (TODO row 166).
3. **adminui on the seam** (TODO row 165; deferred after the spike, unblocked by the same tag).

Then the train itself: `nix run .#train-preflight` (mandatory), identity-model → root → usermgmt → adminui → dashboardui → setup → systemadapter, then `.#push-and-watch` and `.#verify-remote-tags`.

Until the tag nothing on `master` needs it: the ADR-0038 increments, the skill and FEATURES truth passes, the docs harvest and the archive round all proceed without it.

## Waiting on the owner

- **Root `v4.20.0`** — the train above. Tags and pushes are owner actions in this repo.
- **Commit policy for sessions on the daemon-shared tree (2026-11-28 14:00 report §g.1)** — the daemon commits heuristic snapshots; a session that lands named commits works on its own branch in a private worktree and fast-forwards `master` (the `feat/client-tracking-*` shape). Say whether that is the norm for every session.
- **audit-all: drain or re-pin (2026-11-28 14:00 report §g.2)** — 668 unharvested historical items against a pin of 263: 205 in the 32 live reports, 463 in archived ones. Raising the pin needs an owner tick.
- **ADR-0038 continuation (2026-11-28 09:00 report §g.3)** — four increments remain (the three OAuth2 delegates, `SessionService`, `UserService`, `MembershipService`). Running default: no dedicated wave; each lands when a change touches its aggregate.
- **`Service.ChangeEmail`** ships as changed (refuses invalid and taken addresses, stores lower-cased) — running default, unanswered since the 2026-11-27 23:08 report.
- **§g.1 (20-47 report)** — the NixOS system templ (0.3.1123, unpublished master build) is the hook-poison's last standing cause: repin it to a published release (one-line system-flake change, owner-side), or ask for a runbook patch note first.
- **§g.3 (20-47 report)** — K2 multi-instance follow-up: a reference SQL/Redis `idempotency.Store` guide example, or the seam + docs suffice (§g.2 is answered — the anomaly was settled same-evening).
- **Per-row M2M D1–D4 / adminui D1–D3** — the packets stay parked; each decision surfaces individually through the native decision tool per the owner's 2026-11-19 verdict.
- **Fleet-cache hygiene (from the gotcha-10 poison find)** — a substituter served stale fixed-output content under new revs (`cache.home.lan/*` is the suspect class, unattributed): worth a fleet-side prune/provenance check.
- **§g.2 (03-09 report)** — gate robustness vs fail-loud for shared-worktree transients: should one-retry tolerance become the norm for the local-only gates, or does strict fail-loud + the documented re-run-before-triage rule (now gotcha 68) stand?
- **§g.3 (03-09 report) — migration-plan D1/D2, the W3/W4 gate.** The 11-08 owner decision sheet says "Defaults already ADOPTED — no tick needed" while the plan's §6 says W3/W4 "hinge only on the D1/D2 owner ticks" — those two statements contradict each other; pick one (adopt-the-defaults unblocks the read-tier cutover waves, or keep W3/W4 parked until an explicit tick). Recorded per the no-re-ask directive — not surfaced via the question tool.

## Known standing reds (measured 2026-11-28)

- **docs-gates 8 of 10:** `cadence` is ARCHIVE-OVERDUE (32 live reports against 15) and `audit-all` reads 668 against 263. The archive round is the fix for both and needs a verdict on every row of 17 reports.
- **`GOTOOLCHAIN=local` flake apps on `setup` and `integration_test`:** `go.work` and both modules declare go 1.27.4 while the flake's `pkgs.go_1_27` ships 1.27.3, so `.#coverage-gate`'s setup leg is a load error until the fleet rev moves (TODO row 1, addendum 3). CI runs 1.27.4 and is green.
- **Green:** CI at `64ea7092` (all jobs; `test` was red for six hours on two dashboardui contract tests and is fixed in `320c01ef`), the strict release-train gate (go-flightrecorder v0.2.2 swept across 19 modules, `64ea7092`), version drift.
- **Not re-measured since 2026-11-21:** erraudit TOTAL=354.
