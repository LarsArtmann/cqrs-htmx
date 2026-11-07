# Status — root-package gates, a conflict-marker hook, and a tree that kept deleting my work

> **ANNOTATED 2026-11-07** — ~~every open item in this report is resolved~~ **ARCHIVED 2026-11-07 (train-execution).**
> The next session in the same arc is [`2026-11-06_23-52_receipt-battery-dep-sweep-coverage-recovery-and-owner-gated-train_status.md`](2026-11-06_23-52_receipt-battery-dep-sweep-coverage-recovery-and-owner-gated-train_status.md), which Its gates landed and were receipted by the successor; the conflict-marker gate it shipped is now also fixture-shipped (578c8a9d); owner-gated residue carried by TODO.
> Nothing here is a live work item: it is kept as the session's evidence.

**Written:** 2026-11-06 21:31 CET (`date`) · **Session window:** ≈11:00–21:31, with a ~10 h gap 11:40–21:28
**Branch:** `master` · **HEAD:** `686e3cad` (auto-commit daemon) · **ahead of origin:** 15 commits (not pushed)

Scope of this report: only what this session did and observed. No new research.

---

## 0. One paragraph

This session was handed a 26-coarse-task / 76-fine-task plan whose 1 % was a release train. It executed
**four** of that plan's tasks end to end (the `streamid-identity` fix, the conflict-marker gate wired into the
pre-commit hook, a root-size ratchet, and a generated root-cluster table), built the fifth (the docs-gates
matrix wrapper) but did not wire it, and **did not run the train** — because the tree turned out to be shared
with three concurrent writers: the auto-commit daemon (every ~2–3 min), a sibling Crush session issuing
"commit all files" prompts, and two live `git-town sync` runs (one of which pushed tags). That concurrency is
not a nuisance, it is a **data-loss mechanism**: Git Town stashes WIP and restores from the index, so my
uncommitted edits were erased twice, silently, and one of my "verification" runs read the erased file. The
work that survived is committed and green; the train remains blocked by a rule I am not willing to break
(never tag with a foreign diff on the tree), and 15 commits now sit local because pushing without an explicit
ask is also not mine to do.

---

## a) FULLY DONE (verified, with receipts)

### a1. `streamid-identity` — the one red gate no tag could fix (plan C11)

- **Defect:** `dashboardui/core/command_fates.go:198` read `evt.StreamID().String()` into `CommandFate.CommandID`,
  which the struct documents as the command's identity.
- **Why it was a real bug, not a lint nit:** `id.ParseStreamID` strips only a `StreamMarker:` prefix, so a
  lifecycle stream's **bare** value is literally the command's display identity (`CommandID:<ulid>`) — exactly
  what upstream `commandlifecycle` records in the payload (`CommandKey(cmd.ID().String())`). The display form
  therefore produced `StreamMarker:CommandID:<ulid>`: the same command printed two different ways **in one row
  set**, depending on whether the payload carried an ID.
- **Fix:** `.String()` → `.Get()`, plus a comment stating the derivation instead of restating the symptom.
- **Test hardened rather than adjusted:** the fixture now builds values of type `id.CommandID` and derives streams with
  `id.StreamIDFrom` (the same call `LifecycleStreamRef` makes), so payload-sourced and fallback rows are asserted
  to be *the same string*. Before, the fixture fed `StreamID.String()` into the payload — unrealistic data that
  hid the inconsistency the fix removes.
- **Receipts:** `check-streamid-identity.sh` → `OK: all 12 … sites match the pinned allowlist (8 files, 12 pinned)`;
  its self-test 5/5; `go test ./...` in `dashboardui` green (3 pkgs); committed in `0f3eeb20`
  (`git show 0f3eeb20 --stat` lists both files).

### a2. Conflict-marker gate: `--staged` mode + pre-commit wiring (plan C20)

- **New mode:** `check-conflict-markers.sh [--staged]` scans the **index** via `git grep --cached`, so a marker
  that was staged and then edited out of the working tree still blocks the commit, and a marker that only exists
  unstaged does not. Exit contract unchanged: 0 clean · 1 markers · 2 the scan verified nothing.
- **Fixture coverage:** 3 new cases (dirty index + repaired tree, clean index + dirty tree, unknown argument) →
  **12/12** passing.
- **Wiring:** a 4th `MANUAL EDIT` block in **both** `.githooks/pre-commit` and `scripts/hooks/pre-commit.template`
  (byte-identical, `diff` clean), header updated 3 → 4 blocks, `bash -n` clean, `shellcheck -x` clean.
- **Proof (the part that matters):** in a throwaway clone with a marker staged and the tree repaired,
  `git commit` → **rc=1**, output `✗ conflict-markers: 1 committed conflict-marker line(s): docs/CORPUS_BASELINE:2`.
- **Discovery worth keeping:** this repo sets `core.hooksPath=.githooks`, so `.git/hooks/pre-commit` is **dead
  configuration**. Two of my proof runs exercised a stale hook before I checked `git config --get core.hooksPath`.

### a3. Root-size ratchet — the ONE source of a number docs quoted by hand (plan C17)

- `scripts/checks/check-root-size.sh`: measures top-level production `.go` (excludes `_test.go` and `_templ.go`),
  ratchets **downward** against `scripts/checks/root-size.baseline`, and offers `--print` (the canonical sentence
  for docs), `--advisory`, `--update-baseline`. Missing/malformed baseline, empty root, or an unmeasurable line
  count all exit **2** — never a green.
- Baseline boots to `54 8547`; `--print` → `54 production files / 8,547 LOC`.
- Fixture self-test **11/11** (at-baseline, LOC growth, file growth, shrink + lower-baseline hint, advisory,
  missing/malformed baseline, trailing field, `--update-baseline`, production-only, empty root).
- Wired: 2 flake apps (`.#check-root-size`, `.#test-check-root-size`) + 2 `check-modules` stages + 2 CI-parity
  entries + 2 CI steps. Verified from the flake, not just the script.
- **Docs trued:** `TODO_LIST.md` rows 21/22 said `53 production files / 7,594 LOC`; both now quote the gate
  (`54 / 8,547`) and the row names the gate as the source. `check-todo-hygiene` green afterwards.

### a4. Root-cluster table: generated, not hand-summed (plan C18)

- `scripts/checks/root-clusters.tsv` — a **partition manifest**, 54 rows, one per production file, 11 clusters;
  verified against `ls *.go | grep -v _test` with `comm` (0 unmapped, 0 stale, 0 duplicates).
- `scripts/checks/check-root-clusters.sh`: proves the partition (unmapped / stale / duplicate / malformed rows),
  measures per-cluster LOC from disk every run, **reconciles its total against `root-size.baseline`** so the two
  gates cannot disagree, and `--emit` prints the markdown table docs paste.
- Fixture self-test **12/12**.
- Wired: 2 flake apps + 2 `check-modules` stages + 2 CI-parity entries + 2 CI steps.
- **The re-export plan's §1.2 is now that output**, and the hand sum's three defects are recorded in place rather
  than deleted: the missing **`dispatch gate` 702-LOC row**, aliases at **563** LOC (not 336 — the
  `// Deprecated:` markers grew them), and `leaf leftovers` at 30 LOC over one file (not `~150` over three).
- Total now reconciles to the byte: **54 files / 8,547 LOC**.

### a5. A false green I planted and then caught

The clusters gate originally considered "baseline file missing" a **pass** (`reconcile` initialised to 0). Fixture
case 8 failed on it. Fixed to rc 2 with the reasoning written into the code, pinned by the case.

### a6. A red gate I caused and then caught

`check-root-clusters.sh` had an unused `read` variable → **SC2034**, which made the repo-wide
`check-shellcheck.sh` gate **RED** (138 scripts) — after it had been green at 133 scripts earlier in the session
when my scripts were not yet in scope. Fixed by *using* the variable (trailing-field strictness on the baseline,
matching the size gate) plus a 12th fixture case. Re-verified: `✓ shellcheck: 138 script(s) … clean`.

### a7. `docs-gates-battery.sh` — one matrix, honest self-skip (plan C21, first half)

Resolution order `docs-gates` on PATH → `nix run github:Artmann-Technologies/docs-gates` → **SKIP (rc 0, reason
printed)**; `--strict` turns a FAIL verdict into rc 1; rc 2 when the battery itself breaks (a failed run is not a
findings verdict). All three modes exercised. Committed.

### a8. Environment archaeology (this is why the train did not run)

Three concurrent writers, all observed directly:

| Actor | Evidence | Hazard |
| --- | --- | --- |
| Auto-commit daemon | 20+ `chore: auto-commit N changed file(s)` commits, every 2–3 min | commits whatever is in the tree, including half-written docs; the commit message never describes the change |
| Sibling Crush session | `pgrep`: `crush run git status & git diff --cached + git commit … just git commit all files!!!` | `git add -A` semantics; owns `AGENTS.md`, `FEATURES.md`, `docs/status/*`, `dashboardui/*`, new `huma/` module |
| `git town sync` (twice) | `pgrep`: `/run/current-system/sw/bin/git-town sync`, plus a child `git push --tags` | **stashes WIP and restores from the index** — my uncommitted edits were erased at 11:12:06 and again at 11:14:16 |

**The operating rule I derived the hard way:** *stage immediately after writing*. An index copy survives the
restore; a worktree-only edit does not. My `--staged` gate and self-test were lost twice and only landed once I
ran `git add` in the same shell invocation as the write.

### a9. Verification sweep at 21:28–21:31 (all re-run just now, on a clean tree)

| Gate | Result |
| --- | --- |
| `GOEXPERIMENT=jsonv2 go build ./...` (root) | rc 0 |
| `GOEXPERIMENT=jsonv2 go vet ./...` (root) | rc 0 |
| `go test ./...` (`dashboardui`) | green, 3 pkgs |
| `check-root-size.sh` | green — 54 files / 8,547 LOC |
| `check-root-clusters.sh` | green — partition, 11 clusters |
| `check-conflict-markers.sh` (+ self-test) | green — 1731 tracked files; 12/12 |
| `check-streamid-identity.sh` | green — 12 sites / 8 pinned files |
| `check-todo-hygiene.sh` | green — 79 rows, 0 dead |
| `check-docs-corpus-budget.sh --ratchet` | green — 52,970 lines / 350 md files |
| `check-docs-tail-budget.sh --ratchet` | green — TAIL_BASELINE 19 |
| `check-docs-links.sh` | green — 1,430 links |
| `check-shellcheck.sh` | green — 138 scripts |
| `nix run .#check-root-size` / `.#test-check-root-size` / `.#check-root-clusters` / `.#test-check-root-clusters` | all green from the flake |

---

## b) PARTIALLY DONE

| # | Item | State | What is missing |
| --- | --- | --- | --- |
| b1 | Plan C21 (docs-gates matrix in CI) | Wrapper exists and works; **not** a flake app, **not** in `check-modules`, **not** a CI step, **no** self-test; the "check-modules membership" decision is unrecorded | 4 small wirings + a 5-case fixture self-test + one decision line |
| b2 | Plan C12/C13 (the receipt battery) | Only `go build`, `go vet`, and two package suites were run | `check-modules` end to end (92 stages), `coverage-gate`, `erraudit-strict`, `cqrs-lint`, `branching-flow`, `upstream-treadmill` — none run |
| b3 | Plan C01 (quiescence proof) | Measured (load, `pgrep`, `git status`) and written down | the window never became quiet: load hit **173** at 21:28 and a sync was live; no quiet window was ever available |
| b4 | Plan C14 (`docs-gates audit-all`) | Measured precisely: **409** unharvested report-item lines vs the pinned `audit_max: 322` | the drain-or-re-pin decision (owner Q2 below); the pin was not touched |
| b5 | Plan C15 (archive pass) | Measured: **19** unarchived reports against a cadence threshold of 15; tail baseline 19 against a budget of 3 | the ritual itself (`check-rows` → `git mv` → banner → manifest → lower baseline). `docs/status/*` is the sibling's active workspace, so a bulk move would have been destroyed or would have destroyed their work |
| b6 | Plan C16 (two `FEATURES.md` rows) | Precisely described in `TODO_LIST.md` (no inventory row for ADR-0062 or ADR-0063; nearest is `FEATURES.md:200` = ADR-0055) | the edit — `FEATURES.md` is one of the files the sibling holds |
| b7 | Plan C19 (split the 15 over-1,500-char TODO rows) | Not attempted | `TODO_LIST.md` is actively edited by the sibling (a live uncommitted diff on its header exists right now) |
| b8 | Plan C22 (corpus ledger as a table) | Not attempted | `docs/status/README.md`, same contention |
| b9 | Root-size honesty | The gate prints the number; the docs quote it | no gate greps the docs for `N production files / M LOC` — the split-brain killer is not built (see e7) |

---

## c) NOT STARTED

1. **The release train (plan C03–C09).** Preconditions on the build side are now **met** (`go build` and `go vet`
   both rc 0 at 21:31). It is still blocked on: a foreign diff that is never absent, a daemon that commits every
   2–3 minutes (so "no uncommitted `go.mod`" cannot be certified at the instant of tagging), and irreversibility
   (a poisoned module-proxy tag is permanent). **I did not tag anything.**
2. `check-modules` end to end (never run this session).
3. `coverage-gate`, `erraudit-strict`, `cqrs-lint`, `branching-flow`, `upstream-treadmill`.
4. Consumer waves (browser-history + 5 others) and the upstream go-cqrs-lite ask.
5. Pushing the 15 local commits.
6. The `huma/` module's gates — sibling-owned, not mine to wire.
7. Bench re-pin, `b.RunParallel` sub-bench, allocation ratchet.

---

## d) TOTALLY FUCKED UP

**d1. I lost an edit twice and did not notice.** `check-conflict-markers.sh --staged` was reverted at **11:12:06**
and again at **11:14:16** by the Git Town stash/restore cycle. Worse: my first post-write "verification" ran the
*old* file and printed `9/9`, while I was reasoning about a 12-case test. Only `grep -c run_staged` → `0` exposed
it. **Rule earned:** after writing, grep the artifact for a token from each new branch. "Applied 1 edit" is the
tool's belief, not the file's state.

**d2. My first hook proof proved the wrong thing.** I copied the patched hook to `.git/hooks/pre-commit` in a
scratch clone and read the resulting `rc=1` as success — while `core.hooksPath=.githooks` meant git never ran that
file. The refusal I "observed" came from a *different gate* in the *old* hook; I even noted that my
`grep 'conflict-markers'` printed nothing and moved on. **Rule earned:** a verdict counts only when you read
*which* gate produced it.

**d3. I made the repo's shellcheck gate red and only found it 10 hours later** (d-a6 above). The gate was green at
133 scripts when my new scripts were not yet scanned; SC2034 landed in a file I had already called "verified".

**d4. I abandoned a background job for 10 hours.** Job `25E` was blocked in a `nix run` fallback while I moved on;
I only killed it at 21:28. A stuck fallback should be killed or bounded at creation, not remembered.

**d5. I left 15 commits unpushed.** They contain the four gate deliveries *and* the sibling's work. If the tree is
stashed again, the only copy is local. I did not push because pushing is not mine to decide — but I also did not
raise it until this report.

**d6. I twice fell back from `edit` to whole-file `write` after "modified since last read".** Defensible under the
clobber race, but it traded read-before-edit discipline for speed, and it rewrote files I had only partially
re-read.

**d7. `nix run .#check-modules` "flake does not provide attribute" is still unresolved knowledge.** The working
path is `nix eval --raw .#apps.x86_64-linux.check-modules.program`; I never re-proved the invocation I planned to
use in C12, so the next session will rediscover it.

---

## e) WHAT WE SHOULD IMPROVE

- **e1. Write the concurrency rule into `AGENTS.md`**: *stage immediately; the tree is restored from the index by
  Git Town and committed every 2–3 min by the daemon; an uncommitted edit can vanish.* Today that knowledge lives
  only in this report, so the next session re-learns it. (Blocked on sibling ownership of `AGENTS.md` — see Q3.)
- **e2. Add "grep a token per new branch" to the gate-authoring checklist** (d1/d2 are the same failure class:
  trusting a claim over the artifact).
- **e3. Make the size number self-enforcing**, not merely printed: a gate that greps `TODO_LIST.md` and
  `docs/planning/*.md` for `N production files / M LOC` and fails on mismatch. Then the prose cannot drift from
  the baseline even when a human forgets to re-copy.
- **e4. Promote the two judgment columns into the manifest** (`root-clusters.tsv` gains `deps`, `blocker`), so
  `--emit` produces the *whole* §1.2 table and there is nothing left to hand-maintain.
- **e5. Give `root-size.baseline` a header** (date + the exact command) so a human reading the file needs no
  script to interpret it.
- **e6. Mechanize cadence's own remedy.** It already prints the three archive steps; the repo already has
  `.#archive-status`. Chain them, or cadence will red every session forever.
- **e7. Prefer "measure, then quote" over "quote, then hope"** as a review habit: three of this session's five
  defects (root total, cluster table, alias LOC) were numbers a human had typed.
- **e8. Treat a sibling session as an environment, not an incident**: check `pgrep -af 'git-town|crush|buildflow'`
  *before* planning, not after losing an edit.

---

## f) NEXT — 50 things, in the order I would do them

| # | Task | Why |
| --- | --- | --- |
| 1 | Decide Q1 (train authorization / single-writer rule) | every other plan item is downstream of it |
| 2 | Push the 15 local commits (on approval) | they are the only copy of four deliveries |
| ~~3~~ | ~~Wire `docs-gates-battery.sh` into flake apps + CI (advisory)~~ done at 22:0x — flake apps `check-docs-gates-battery`/`test-check-docs-gates-battery`, two check-modules stages + CI-parity rows, advisory `ci.yml` step with the membership comment; the parity self-test caught my stage env-prefix bug, fixed | ~~finishes C21~~ |
| ~~4~~ | ~~Add a 5-case self-test for the battery wrapper (stub binary: PASS / FAIL / strict / skip / broken)~~ done at 22:0x — shipped 9 cases (green matrix, advisory WARNING, strict exit 1, broken battery exit 2, crash not promotable, honest SKIP, non-executable binary, unknown arg, --json passthrough), 9/9 green, shellcheck-clean | ~~a wrapper with no fixture is a claim~~ |
| ~~5~~ | ~~Record the `check-modules` membership decision for the battery (advisory, CI-only, with the reason)~~ done at 22:0x — the `ci.yml` step comment carries it: advisory + CI-only because the `audit_max` pin is local policy and the binary is not a flake input | ~~C21 second half~~ |
| ~~6~~ | ~~Run `check-modules` end to end and write the residual red list~~ done at 22:0x — baseline run 13/100 red, all triaged: 2 my wiring bugs fixed, 1 corpus race cleared, reexport 17→19 adjudicated, branching re-pinned 203→210, route-postures = API-vs-tag (train-gated); final re-run after the receipt battery | ~~C12~~ |
| ~~7~~ | ~~Run `coverage-gate`, compare against `coverage-thresholds.tsv`~~ done at 23:17 — root leg GREEN at 90.4% (floor 90) after writing focused tests for the uncovered delegation shims (`coverage_delegation_test.go`, `htmx/request_coverage_test.go`, httperr accessor Describe; 89.6% → 90.4%); adminui now MEASURES green (67.4% ≥ 66%) after the templ-components v1.26.0 → v1.27.0 bump (CSS bundles rebuilt same-change, class sets exact); the 4 remaining unmeasured modules (usermgmt/dashboardui/setup/systemadapter) are the API-vs-tag hermetic class — train-gated. The test sweep also caught a real doc-vs-code split brain: `WithSyncPullLimit`'s comment claimed ceiling semantics the handler never enforced — doc fixed to the actual default-override contract, test pins it | ~~C13~~ |
| ~~8~~ | ~~Run `erraudit-strict`, compare against TOTAL=303~~ done at 22:1x — TOTAL=319, advisory posture; growth is the sibling sessions' new code | ~~C13~~ |
| ~~9~~ | ~~Run `check-cqrs-lint` per module~~ done at 22:4x — 10/14; the 4 reds (root/usermgmt/dashboardui/systemadapter) are the API-vs-tag hermetic-load class, NOT findings (TODO row 49 carries the full receipt incl. the ambient-toolchain trap and the 4.20.1 master-binary drift) | ~~C13~~ |
| ~~10~~ | ~~Run `check-branching-flow` against the SARIF baseline~~ done at 22:2x — re-pinned 203 → 210 under existing verdicts (+17 drift-sync), amendment in `docs/analysis/triage-decisions.md` | ~~C13~~ |
| ~~11~~ | ~~Run `check-upstream-treadmill --strict-lag 0`~~ done at 22:4x — worst lag 0 after sweeping the six lagged pins via `go get`+tidy (bump-dep is a no-op on indirect MVS floors); workspace build+vet+targeted tests green | ~~C13~~ |
| 12 | Re-run the go-appkit `bump-dep` sweep and expect 4/4 PASS after the train | proves the tag-lag diagnosis |
| 13 | Execute the train (identity-model → root → usermgmt → UI → setup) with sweeps between steps | C03–C09 |
| 14 | Re-run the 8 tag-gap legs and record before/after | C10 |
| 15 | Decide `audit-all`: drain 409 → 322 or owner-approved re-pin | Q2 |
| 16 | Harvest the newest report's 4 freshness items (`BotScopesChanged`, `ListBots`, Huma subject parity, `coverage-gate`) | freshness is red right now |
| 17 | Archive pass down to a tail of 3 (19 reports live) | cadence is red right now |
| 18 | Run `check-rows --strict` **before** any `git mv` (the ritual order that failed twice) | prevents broken links |
| 19 | Add the two `FEATURES.md` rows (ADR-0062, ADR-0063) — when the sibling frees the file | C16 |
| 20 | Build the docs-claim gate for the root-size sentence | e3 |
| 21 | Add `deps`/`blocker` columns to `root-clusters.tsv` and re-emit §1.2 | e4 |
| 22 | Split the 15 over-1,500-char TODO rows into short rows + anchors | C19 |
| 23 | Convert the corpus ledger paragraph to a table | C22 |
| 24 | Write the concurrency rule into `AGENTS.md` (stage-immediately, foreign-diff ban, `core.hooksPath=.githooks`) | e1 |
| 25 | Add a `check-concurrency` advisory? (warn when a sync/daemon is live) — decide first whether it is noise | new hazard class |
| 26 | Wire the conflict gate into a `post-merge`/`post-checkout` hook too (catches a stash-pop conflict at birth) | the class has bitten twice |
| 27 | Record `core.hooksPath=.githooks` in AGENTS gotchas (two of my proofs used a dead hook) | d2 |
| 28 | Re-pin the bench baseline on an idle machine | needs quiet |
| 29 | Add `b.RunParallel` sub-bench for the dispatch path | throughput unmeasured |
| 30 | Add the allocation ratchet for the three dispatch paths | allocation set is the only load-independent metric |
| 31 | File the upstream go-cqrs-lite ask (dispatch middleware) and link ADR-0063 | C24 |
| 32 | Consumer wave 1: browser-history (`GrantRoles` replaces `reconcileSuperAdmins` + dispatch gate declared) | C25 |
| 33–37 | Consumers 2–6 (Swetty, crm, InboxClean, github-local-sync, webphone) — one receipt each | C26 |
| 38 | Prove the `nix run .#check-modules` invocation and add it to the runbooks | d7 |
| 39 | Add a `gate-authoring` checklist to `docs/` (grep-the-token, plant-one-failure-per-case, rc=2 floor) | e2 |
| 40 | Re-check the `huma/` module's gates once the sibling lands it | unverified surface |
| 41 | True the `shellcheck` count claim in AGENTS (133 vs 138 scripts) | drift by design |
| 42 | Make `check-root-size` also report test-file LOC for context (prod-only is the ratchet) | reviewer clarity |
| 43 | Decide whether the `dispatch gate` cluster deserves its own extraction candidate (702 LOC, 4 deps) | it is the largest unplanned cluster |
| 44 | Add a `--json` mode to `check-root-size`/`check-root-clusters` for dashboarding | machine consumption |
| 45 | Re-run the conflict gate self-test in CI against a fixture that *stages* through a real index (not just `git add`) | closes the last simulation gap |
| 46 | Add the 11 pushed-tag audit (`verify-remote-tags`) to the pre-push hook's output | peeled-ref drift is invisible locally |
| 47 | Note in the release runbook that a fresh tag reads unpublished until the cache TTL (gotcha 27) was *not* re-exercised this session | honesty about untested knowledge |
| 48 | Clean the `/tmp/hookproof-*` residue class (my scratch clone now lives in the trash) | hygiene |
| 49 | Reconcile the 5 correctly-failing battery gates with the "all five PASS" claim in AGENTS | AGENTS is stale |
| 50 | Write the next session's opening three commands (load, `pgrep`, `git status`) into the runbook | e8 |

---

## g) Questions I cannot answer myself

1. **Train under concurrent writers.** The build side is finally verifiable (`go build`/`go vet` rc 0), but another
   session, the auto-commit daemon, and `git town sync` are live. Do you authorize tagging under those conditions
   (accepting that a stash mid-train can poison a tag), or is "one writer in the tree" a hard precondition? I did
   not tag, because a bad tag is irreversible.
2. **`audit-all` pin.** 409 unharvested item lines against `audit_max: 322`. Drain downward, or re-pin to 409 —
   and should archived reports keep draining at all, given `audit-all` replays them (verified)?
3. **Pushing and file ownership.** May I push the 15 unpushed commits (they include your sibling's work), and may
   I edit `AGENTS.md` / `FEATURES.md` / `docs/status/*` while the sibling session holds them — or should every such
   finding keep routing through `TODO_LIST.md` as this session did?
