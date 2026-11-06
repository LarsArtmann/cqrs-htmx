# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-05 (post-train) · **Owner:** Lars · **Module count:** 29 (`go.work`)

## The active thread

**Round 46/47 tail — the train shipped; take the verified-green.** The v4.16.0 family
train was cut and pushed 2026-11-05 (root v4.16.0 carries the root package split +
`Posture*` in ONE tag; usermgmt v4.19.0 carries `ViewCacheConfig`/shape stamps;
adminui/dashboardui/loginpage v4.16.0; setup/v4.16.0 stays the 2026-11-02 tag —
nothing post-v4.16.0 is consumed). The `--no-verify` window is closed; the release-train
gate reads 0 unpublished / 0 lag.

What stands between this tree and fully green CI is exactly ONE upstream blocker:
**the go-cqrs-lite `catalog/v4` release** (v4.12.0's docserver ships templ-poisoned
generated code; their master already carries the regenerated templ but is not
releasable from here — in-flight work behind a local-path replace). Once it ships:
bump `integration_test` + `examples/catalog-demo`, run `push-and-watch`, close the
TODO P1 CI row with evidence, and archive per docs-health.

Then the post-train battery (plan M02) on the merged tree: `nix run .#check-modules`
aggregate, `.#coverage-gate` (root LOC moved 8,121 → 7,325 in the split),
`.#check-readme-quickstart` against TAGGED v4.16.0, `.#check-cqrs-lint` /
erraudit-strict / branching-flow re-runs — none have run on the final post-train tree.

- Plan of record: [`docs/planning/2026-11-05_03-11_SUPERB-round46-tail-train-green-and-backlog-pareto-plan.md`](docs/planning/2026-11-05_03-11_SUPERB-round46-tail-train-green-and-backlog-pareto-plan.md) (M01 done; M02–M05 next, M06+ after).
- Authorization thread receipts: [ADR-0057](docs/adr/0057-authorization-contract.md),
  [gotcha 45](docs/gotchas.md#gotcha-45), [`authztest`](authztest/README.md),
  [docs/guides/authorization-system.md](docs/guides/authorization-system.md),
  [docs/guides/route-posture.md](docs/guides/route-posture.md).

**Done when** master's CI is green end-to-end (catalog released + consumers bumped),
the post-train battery is green on the merged tree, the TODO P1 CI row is closed with
CI evidence, and the keep-green traps are mechanized (API-vs-tag drift gate, templ
codegen hook audit).

## The next thread (pick ONE)

1. **Direct imports only — the v5 alias cut** (W0 landed 2026-11-06). The
   preference is now a policy with a gate: [ADR-0062](docs/adr/0062-direct-imports-only.md),
   `nix run .#check-reexport-policy -- --strict`, frozen
   [`scripts/checks/reexport-baseline.tsv`](scripts/checks/reexport-baseline.tsv)
   (47 alias files / 413 declarations), and codemod rules R11/R12/R13. The three
   root façades are deprecated and root no longer uses them itself — production
   code AND its 25 own test files were repointed by the codemod, with
   `api_surface_compat_test.go` deliberately left pinning the aliases until v5 —
   so the v5 cut is a file delete plus a consumer repoint. Zero living doc still
   teaches a removed spelling unlabelled (the 10 survivors are labelled contrasts
   or frozen `CHANGELOG` history). Remaining waves (all inside the ONE
   v5 window, never a second break): W1 delete the 9 root alias files + 176
   `usermgmt` aliases and run the codemod on `examples/`, `e2e/`,
   `integration_test/`, `setup/`, `dashboardui/`; W2 alias-free extractions
   `authz/` → `readiness/` → `sync/` → `realtime/`; W3 `internal/handler`.
   Plan: [`docs/planning/2026-11-06_07-16_SUPERB-direct-imports-re-export-eviction-and-root-burndown-pareto-plan.md`](docs/planning/2026-11-06_07-16_SUPERB-direct-imports-re-export-eviction-and-root-burndown-pareto-plan.md).
2. **Consumer waves** (plan M06–M09) — one repo at a time, each with its own tests:
   `browser-history` (`reconcileSuperAdmins` → `GrantRoles`), `InboxClean`
   (delete `sseDeadlineFree`), `SwettySwipperWeb` (drop the `Count() > 0` guard),
   `crm` (seed an admin or `DisableAdmin`). The exact call to replace is the
   "Next step" column of [docs/PERMISSIONS.md](docs/PERMISSIONS.md).
3. **Advisory month for erraudit strict** (passive): green month at TOTAL ≤ 303,
   then flip `--strict --gate`.

(The former item 2 — cqrs-lint 4.19.1 triage, plan M10 — is DONE 2026-11-05:
binary rebuilt at the published v4.20.0 tag, gate green 14/14, `3c544dc9`.)

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
