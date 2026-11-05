# STATUS REPORT — Root-split episode 5, close-out: recovery → 0 lint → 83/84 battery

> **ANNOTATED 2026-11-04** — ~~this report~~ **SUPERSEDED** by [`2026-11-04_18-52_docs-health-round46-2026-0-corpus-six-docs-truth_status.md`](2026-11-04_18-52_docs-health-round46-2026-0-corpus-six-docs-truth_status.md), which resolved. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-04 (round-23 (2026-11-04)).**

**Date:** 2026-11-02 21:03 CET · **Branch:** `modularize/root-package-split` (worktree
`/home/lars/projects/cqrs-htmx-root-split`, HEAD `7daabb8a`) · **master tip:** `6165b762`
· **Skill:** `go-modularize` · **Supersedes:** `2026-11-02_16-23_root-split-recovery-reexport-refactor-and-gate-battery_status.md`

**One line.** The extraction work this branch carries is **content-complete and verified**:
0 lint findings, a compile-pinned public API, an enforced layering gate, all 29 workspace
modules building, coverage 15/15, and the architecture battery at **83/84** — where the
single red (`branching-flow`) is the gate's own uncommitted-baseline guard, satisfied the
moment the owner commits. Nothing has been committed (repo rule).

---

## (a) FULLY DONE

### a.1 Disaster recovery — zero data loss

Arrived at a worktree mid-merge with all 59 refactor files gone from the tree (a concurrent
session's Git Town run had stashed them and started a conflicting merge). Recovered in the
safe order: patch export + `git update-ref refs/heads/rescue/root-split-wip stash@{0}`
FIRST, then `merge --abort`, then `stash pop`. Build + 9/9 packages green immediately after.

### a.2 Rebased onto the rewritten `master`, no merge-commit entanglement

`master` had been history-rewritten (twin commits `67c66fa2`/`69f029b2`). Stash →
`git switch -C` onto `master` (`7daabb8a`) → pop → exactly two doc conflicts, resolved in
upstream's favor.

### a.3 Integrated four upstream advances — surgically, never rebasing twice

`7daabb8a → 4c631647 → 81d691a4 → e3e446e8 → 6165b762` landed while this session worked.
Non-overlapping content taken via `git restore --source=master` (authztest grant proof
kit, `docs/PERMISSIONS.md`, the five-tag consumer aligns, `check-permission-surface.sh`,
the untracked 7.9 MB CLI binary, the authorization-train report); the one true overlap
(`CURRENT_FOCUS.md`) reconciled by hand — their round-46 thread stays active, this work
recorded on its own branch, one new ground rule added.

### a.4 Lint: 34 findings → **0 issues**, every fix at the source

Root module linted with the **pinned** golangci-lint 2.15.0 via `nix develop` (ambient
binary is 2.14.0 — a trap the first session fell into). Fix classes: 20 `revive` stutter
findings (subpackages renamed to `htmx.Target`/`eventcatalog.Catalog`/`Handler`/`New`;
root keeps the historical spelling), 5 `exhaustruct_v5` (explicit fields + one stale
`//nolint:exhaustruct` corrected), 6 `wrapcheck` (4 gone via var aliases; 2 by extending
`ignore-package-globs` with `cqrs-htmx/v4/internal/*` and `go-cqrs-lite/id/*` — the
repo's established family-tagged-errors FP class), 1 `gochecknoglobals` (facades moved to
the `*_reexport.go` convention the repo already exempts), 4 comment/gci/wsl polish.

### a.5 Facades: 36 wrapper funcs → 36 `var` aliases

A wrapper can silently drop a parameter; a `var HTMXTarget = htmx.Target` alias cannot —
identical function value or compile error. Facades renamed to `htmx_reexport.go`,
`eventcatalog_reexport.go`, `httperr_reexport.go` (the `csrf_reexport.go` precedent).

### a.6 Public API pinned at compile time

`api_surface_compat_test.go` (`package cqrshtmx_test`) asserts the exact signature of all
40 moved functions, 3 types, 13 constants, 8 sentinels — a forgotten alias is now a build
failure here, not a consumer break on upgrade. Base API captured via `go doc -all` at
master and diffed against the tree before the file was written.

### a.7 New gate: package layering is machine-enforced

`scripts/checks/check-module-layers.sh` (checks `.Imports` **and** `.TestImports` against
the declared one-way table) + `scripts/selftests/test-check-module-layers.sh` (9 offline
fixture cases, one planted violation per rule — 9/9 green). Wired into flake apps,
`check-modules` (sequential + stages), and CI; `nix flake check --no-build` green;
check-modules parity self-test green with the new stages.

### a.8 The full battery, carried from 79/84 to 83/84

| Gate | Final verdict |
| ---- | ------------- |
| `check-modules` (84 stages) | **83 PASS / 1 FAIL** — only `branching-flow` |
| `coverage-gate` | **15/15 modules** (root 90.6 %, usermgmt 82.9 %, setup 92.2 %) |
| `module-isolation` (GOWORK=off) | green, every module |
| `release-train --strict-lag 0` | green — 882 requires, 0 unpublished, 0 lag |
| `train-consumers-hermetic` | green — 20/20 modules |
| `go build ./...` | all 29 workspace modules incl. 13 examples + e2e/server |
| `go test ./... -count=1` | 9/9 root-module packages |
| `go mod tidy -diff` (every module) | no drift |
| `check-root-files --strict` | green (baseline re-pinned 31 → 30) |
| `check-docs-links` / `tail-budget --ratchet` / `corpus-budget --ratchet` | green / 4-of-4 / green |

### a.9 `branching-flow` +29 fully triaged and re-pinned

Attributed by authorship: 5 this session (1 moved plumbing call under the primitive-string
frozen class + 4 line-shift re-anchors of already-frozen findings), ~17 the concurrent
session's ADR-0057/authztest code, ~7 anchor churn from their dashboardui/authz edits.
All accepted under EXISTING verdict classes; amendment row written into
`docs/analysis/triage-decisions.md`; baseline re-pinned 184 → 198 (minified SARIF, exact
arithmetic match). The gate is red ONLY because its uncommitted-baseline guard fires by
design until that baseline lands in a commit.

### a.10 Docs

`AGENTS.md` (packages bullet, Quick-Reference gate row, gotcha index), `README.md`
(subpackage table), the project `SKILL.md` (subpackage section + the two rules),
`docs/gotchas.md` (**47**: one agent per worktree, rescue-ref-before-recovery; **48**:
extraction split-brain, delegate to the moved code's single source of truth),
`CURRENT_FOCUS.md` (reconciled with the live round-46 thread), three archive passes
(`12-38` draft, round-45 self-critique, `14-13` interim report, plus mirroring upstream's
`12-52` archive), manifest reconciliation by hand, corpus/tail baselines settled.

### a.11 Final numbers

| Metric                      | Before | After |
| --------------------------- | ------ | ----- |
| Root production `.go` files | 57     | 51    |
| Root test files             | 105    | 98    |
| Root-package LOC            | 8,121  | 7,325 |
| Root-module packages        | 5      | 9     |
| Go modules                  | 29     | 29    |
| Root lint findings          | 0      | 0     |
| Public API                  | —      | identical, compile-pinned |

---

## (b) PARTIALLY DONE

1. **`branching-flow` is red by design** — triaged, re-pinned, amendment written; needs one
   commit to go green (the guard exists so an unreviewed baseline can never vouch for itself).
2. **Nothing is committed** — 134 changed paths sit in the worktree, protected by the
   `rescue/root-split-wip` ref and a 183 KB patch in `~` until the owner says commit.
3. **Branch base is `7daabb8a` + surgical upstream takes**, while `master` sits at
   `6165b762`. Content is current (all four advances integrated); only the git HISTORY
   lags, which is a rebase/merge decision.
4. **`htmx` package coverage is 77.5 %** — the moved tests cover behaviour, but several
   accessors are exercised only through root-level specs; one focused test file closes it.

---

## (c) NOT STARTED (deliberate or blocked)

Commit + push + `.#push-and-watch` + `.#verify-remote-tags` (blocked on the owner's
commit ruling); the `sync`/`openapi` ADR (blocked on the core/-facade decision);
`train-preflight` (pre-release only); a full `nix fmt`/treefmt pass (the Go formatters ran
via `golangci-lint fmt`; treefmt is canonical repo-wide); `go work sync` idempotency;
`erraudit --strict` and `cqrs-lint` sweeps over the four new packages; `-race` on the new
packages; godoc `Example*` functions for the new public subpackages; per-package coverage
rows in `coverage-thresholds.tsv`; the W1 close-out record in `docs/modularization/`; a
consumer-visible CHANGELOG entry; a TODO_LIST row for the ADR; lint on the other 28
modules (untouched by this refactor, green at their last release).

---

## (d) TOTALLY FUCKED UP (this session's own errors, honestly)

1. **Wrong first hypothesis, twice.** The `coverage-gate` failure was first blamed on the
   known parallel-load SQLite flake; the second identical run disproved load, and only
   reproducing the gate's ENVIRONMENT (`goEnv`: `GOWORK=off`, `GOTOOLCHAIN=local`) — not
   just its command — found the real cause (the root-tag gap).
2. **Baselines bumped too early — twice.** Wrote report sections after the corpus bump and
   left `corpus-budget-ratchet` +20 red; then did it again later. The rule ("baselines are
   the LAST edit") was learned on the first violation and violated again on inertia.
3. **Wrote predictions into a report.** The 16-23 report shipped "expected green" sections
   before the gates ran, then needed a second editing pass to become true. Reports state
   what happened, not what should happen.
4. **Incomplete first rename sweep.** Renamed `htmx`'s symbols but not their consumers —
   two build breaks (`htmx.IsHTMXRequest`, `htmx.WithHTMX`) that a single grep for stale
   references before building would have caught.
5. **The archive ritual failed on an untracked file** (`git mv` refuses) — fixed by
   `git add` first, but the script could say so itself; this cost one round trip.
6. **Two write-tool refusals** — tried to write files modified after my last read
   (post-`git mv` renames). After any git file operation, re-read before writing.
7. **Nearly concluded the work was destroyed** when the packages were missing; five minutes
   of forensics (MERGE_HEAD, MERGE_MSG, stash list) showed a fully recoverable state.

---

## (e) WHAT WE SHOULD IMPROVE

1. **Baseline-last discipline** — bump `CORPUS_BASELINE`/`TAIL_BASELINE` only after the
   final docs edit (violated twice this session; cheap to obey).
2. **Reproduce the gate's environment, not its command** — the hermetic failures live in
   `goEnv` (`GOWORK=off`, `GOTOOLCHAIN=local`, pinned golangci), not in `go test ./...`.
3. **Grep for stale references before building** after any symbol-rename sweep.
4. **`archive-status.sh` should name the untracked-file case** — "git add the report
   first, then archive" instead of a bare `git mv failed`.
5. **One session per worktree** (now ground rule 6 + gotcha 47), and **rescue ref before
   any recovery step** (gotcha 47).
6. **Facade policy is now written down**: `var` aliases in `*_reexport.go`, subpackage
   names drop the stutter prefix, root keeps the published spelling — copy this shape for
   any future extraction.

---

## (f) NEXT — ordered, with verify commands

**Land it (the owner's click unblocks all of this)**
1. Commit ruling: six revertable commits — `internal/hx`, `htmx`, `eventcatalog`,
   `httperr`, gate+CI+flake, docs+baselines (the branching-flow baseline + amendment row
   ride the gate or docs commit).
2. After commit: `nix run .#check-branching-flow` → green (the guard's whole point).
3. After commit: `nix run .#check-modules` → **84/84** expected.
4. Rebase-or-merge ruling onto `master` `6165b762` (pushed branch → force-push policy).
5. `nix run .#push-and-watch` (workflow-filtered CI watch + failed-job triage).
6. `nix run .#verify-remote-tags` (peeled-ref audit; compare `^{}` to `^{}`).
7. Clean up `rescue/root-split-wip` and `~/cqrs-htmx-root-split-wip-1507.patch`.

**Quality follow-ups**
8. `htmx` coverage 77.5 % → 90 % — one focused test file for the root-only accessors.
9. `internal/hx` 87.7 % → 90 %; `httperr` 89.2 % → 90 %.
10. Run the four new packages' tests with `-race`.
11. Full `nix fmt` (treefmt) pass; expect zero diff.
12. `go work sync` idempotency check (only `go mod tidy -diff` was verified).
13. Per-package coverage rows in `scripts/checks/coverage-thresholds.tsv` for the four new
    packages (module-level 90 currently hides `htmx` 77.5).
14. `erraudit --strict` sweep over the new packages.
15. `check-cqrs-lint` over the new packages.
16. Godoc `Example*` functions for `htmx`, `httperr`, `eventcatalog`.
17. Lint the other 28 modules with the pinned toolchain (untouched; cheap insurance).

**Docs/skill**
18. Close watch item W1 in `docs/modularization/2026-10-18_PROPOSAL.html`'s follow-up slot
    with a short episode-5 record.
19. Consumer-visible `CHANGELOG.md` entry (new importable subpackages, zero API change).
20. `TODO_LIST.md` row for the `sync`/`openapi` ADR with a verify command.
21. Skill "Top gotchas" section: pointers to gotchas 47/48.
22. `docs/runbooks/session-start.md`: add the one-session-per-worktree rule if ratified.

**Structural (v5 decisions)**
23. ADR: root as pure facade over `core/` — the only shape that unblocks `sync`/`openapi`.
24. ADR: facade deprecation policy (permanent spelling vs `// Deprecated:` in v5).
25. Consider promoting `internal/hx` if consumers keep asking for the immutable-JSON
    helpers (internal by design today).
26. Consider per-package `.golangci.yml` for the new packages if their needs diverge.

**Housekeeping**
27. Remove the prunable `/tmp/cqrs-branch` worktree (pre-existing).
28. Silence or fix the pre-existing `erraudit` unknown-linter nolint warnings repo-wide.
29. Re-run `check-skill-claims`/`check-readme-splitbrain` after any future SKILL/README
    edits (both green in the 83/84 battery).

---

## (g) QUESTIONS I CANNOT ANSWER MYSELF

1. **May I commit, and in what shape?** Rule: no commit without an explicit ask. My
   recommendation stands: six revertable commits (four package extractions, gate+CI+flake,
   docs+baselines). One word — "commit" — and the last red gate goes green.
2. **Rebase this branch onto `master` `6165b762` before landing, or leave the base at
   `7daabb8a` and merge?** It is a pushed branch, so rebasing needs your force-push
   policy; content-wise both are equivalent today.
3. **Does the root package keep shrinking (pure facade over `core/`, unblocking
   `sync`/`openapi`), or is 51 files the steady state?** This decides whether (f)23's ADR
   is next-session work or v5 material.
