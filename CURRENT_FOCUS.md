# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-02 15:25 · **Owner:** Lars · **Module count:** 29 (`go.work`)

## The active thread

**Round 46 — the permission system.** Authorization is now a single contract, one
durable write path, a proof kit, and a ratchet instead of four decision shapes and
three "is this an admin?" predicates. The library half is written, committed, and
pushed; what is open is **publishing it**: master's CI is red with the
root-tag-gap class ([gotcha 30](docs/gotchas.md#gotcha-30)) because six modules
use symbols that only exist on HEAD.

- Contract: `cqrshtmx.Request` / `Decision` / `Decide` + `IsForbidden` /
  `IsUnauthorized` ([ADR-0057](docs/adr/0057-authorization-contract.md)).
- Durable grants: `Service.GrantRoles` / `RevokeRoles` / `HasAdminGrant` over the
  membership journal ([gotcha 45](docs/gotchas.md#gotcha-45)).
- Proof kit: [`authztest`](authztest/README.md) — posture matrix plus grant
  semantics (tenant isolation, revoke, durability).
- Gates: `check-permission-surface` (four anti-patterns) and
  `check-permission-maturity` (`--ratchet` over the declared levels).
- Guide: [docs/guides/authorization-system.md](docs/guides/authorization-system.md);
  ladder + inventory: [docs/PERMISSIONS.md](docs/PERMISSIONS.md); plan:
  [docs/planning/2026-11-02_10-26_SUPERB-permission-system-hardcore-levelup-pareto-plan.md](docs/planning/2026-11-02_10-26_SUPERB-permission-system-hardcore-levelup-pareto-plan.md).

**Landed on the same day, on its own branch (`modularize/root-package-split`):** the
root package is decomposed — `htmx/`, `httperr/`, `eventcatalog/`, and the shared
`internal/hx/` plumbing now sit beside it, with three `*_reexport.go` facades keeping
the published API byte-identical (`api_surface_compat_test.go` compile-pins it) and
`nix run .#check-module-layers` enforcing the one-way edges. Root production files
57 → 51, root-package LOC 8,121 → 7,325, zero consumer changes. It is deliberately NOT
part of the active thread above: it touches no tag and no published symbol, so it can
land before or after the round-46 publish. Receipt:
[docs/status/2026-11-02_14-13_root-package-decomposition-go-modularize-status.md](docs/status/archived/2026-11-02_14-13_root-package-decomposition-go-modularize-status.md).

**Done when** master's CI is green with `identity-model`, root, `usermgmt`,
`dashboardui`, `adminui`, and `setup` tagged on the symbols they now consume, the
grant suite is exercised against the real service in `integration_test`, and each
of our own admin/dashboard routes has a declared posture checked by
`authztest.Undeclared`.

## The next thread (pick ONE)

1. **Consumer waves** — one repo at a time, each with its own tests:
   `browser-history` (`reconcileSuperAdmins` → `GrantRoles`), `InboxClean`
   (delete `sseDeadlineFree`), `SwettySwipperWeb` (drop the `Count() > 0` guard),
   `crm` (seed an admin or `DisableAdmin`). The exact call to replace is the
   "Next step" column of [docs/PERMISSIONS.md](docs/PERMISSIONS.md).
2. **Promote the maturity gate** from "blocking but unexercised" to "blocking and
   proven" after two green waves.
3. **Advisory month for erraudit strict** (passive): green month at TOTAL ≤ 303,
   then flip `--strict --gate`.

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
