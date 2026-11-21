# Master CI red window — six jobs triaged, five root causes fixed, three landed

> **ANNOTATED 2026-11-21** — ~~this report~~ **SUPERSEDED** by [`2026-11-21_08-45_master-ci-heal-composite-bill-fast-slice-and-harvest_status.md`](../2026-11-21_08-45_master-ci-heal-composite-bill-fast-slice-and-harvest_status.md), which the 08-45 report receipts the same master-CI root-cause classes healed (corpus/tail ratchet re-trues, the cqrs-lint shadow-binary class retired, the lint-leg verdict) and adds gotcha 68 for the transient class the triage named. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-21 (round-48).**

**Date:** 2026-11-18 06:53 CET · **Session:** single-thread continuation of the 07-20 handoff, resumed under the owner's blanket GET-SHIT-DONE directive · **Module count:** 30 (`go.work`)

**Mission as given:** execute the whole handoff todo list; the first item (`push-and-watch` on the `b8a0a00d` train) exploded into a six-job CI red window that the handoff believed green. This session: triage it, fix the root causes, verify.

## Scope line

**Verified here (fresh commands this session):**

- `gh run list --workflow=ci.yml`: main CI red back to 2026-11-14 (twelve consecutive failures listed; the handoff's "master CI GREEN" claim was the `train-lag` workflow, not `ci.yml` — 44010050578 green on 79a24ef6 was train-lag).
- Step-level attribution (`gh run view --json jobs`): the six failing jobs on `b8a0a00d` decompose into **six distinct root causes**, not one.
- The working tree had moved far past the handoff snapshot: 22+ daemon commits (the BuildFlow-incident session's waves: C027/C009 inline suppressions, the `errorfamily` alias, per-module `.cqrs-lint.json` disables), then a separate overnight session (22:25, cqrs-lint v4.22.2 train). Liveness checked via `ps` before editing: no sibling session running now.
- `/tmp/ci-repro2/` worktree clone: the exact CI layout (three sibling clones + the literal `..`-path sed spelling) **reproduces** the ambiguous import on `e2e/server`'s member build and goes green with the canonical spelling — first reproduction of a red the 07-20 session called not-reproducible.
- Full usermgmt suite green hermetically (19.5s) after the SQL fix; `TestSQLEventStore -count=5 -race` green.

## a) FULLY DONE

| Item | What landed | Receipt |
| --- | --- | --- |
| CI red-window triage | Six jobs → six causes: lint(systemadapter)=goimports (already healed by the sibling's alias edit — verified `golangci-lint run` action-parity env = 0 issues), security=GO-2026-6859, e2e=ambiguous import, test=no-such-table, checks=42 MB `server` binary tracked, module-architecture=root-files drift | run 44031218741 step list |
| Security job pin | `setup-go` pinned `go-version: "1.27.4"` with the GO-2026-6859 rationale and a revert condition; go.work deliberately stays 1.27.3 (check-go-toolchain forbids directive > flake toolchain under `GOTOOLCHAIN=local`); AGENTS row trued; TODO P1 revert row added | `10a117c6` |
| e2e ambiguous-import fix | The CI sed spelled siblings `<workspace>/../<sibling>`; go treats the `..`-spelling as a second module instance, so nested module `event/v4/eventtest` resolved twice (event/v4 v4.15.2 + eventtest v0.4.0). Sed now uses `$(dirname "${GITHUB_WORKSPACE}")/` (canonical). Repro-verified before landing | `1a98b4b4` + /tmp/ci-repro2 |
| SQL "no such table" root cause + fix | Bare `:memory:` = per-connection databases under `database/sql` pooling: the auto-migrate DDL landed on one connection, the casbin-projection drain queried another → "no such table: events" (3 consecutive CI reds). `newTestSQLiteStore` now uses a named shared-cache DSN + keeper conn. **Gotcha 41's real mechanism, now fixed at source instead of rerun-once** | daemon capture `afa3c518` |
| `.buildflow.yml` restored | The daemon had captured an unsanctioned deletion (`8a438530`, authorship unverified per the 21-19 verdict report); restored from `983dbd1d` per gotcha-35 restore-first doctrine. Un-reds check-root-files + check-buildflow-skips (both verified green) | `5f0c3354` (daemon capture) |
| Binary-guard offender found and swept | A 42 MB compiled `server` binary sat tracked at repo ROOT in `b8a0a00d` (mode 100755, 42302495 bytes — the exact CI FAIL line); already gone at HEAD (a later daemon commit removed it). The guard worked exactly as designed | `git ls-tree -r -l b8a0a00d` |
| Full usermgmt suite | Hermetic `-count=1` green after the SQL fix; scoped SQL-store tests `-count=5 -race` green | this session |

## b) PARTIALLY DONE

| Item | Works now | Still open | Blocker | Effort |
| --- | --- | --- | --- | --- |
| CI healing | 5 of 6 causes fixed + committed locally (4 commits ahead of origin) | **Nothing pushed, no CI re-watch** — the fixed window is unproven on the runner | waiting on this report's delivery (owner said report-then-wait) | S after instruction |
| The 42 MB blob | Identified; tree clean | Git HISTORY still carries 42 MB of binary in the pack (and possibly other incidental binaries from the same window) | history rewrite = owner call (destructive) | M |
| AGENTS cqrs-lint row | Documents v4.22.0 train closure | Upstream moved to v4.22.2 overnight (the 22-25 session); row may lag one train | none — truing is minutes | S |
| Pre-commit hook red window | Documented twice this session | The hook now ALSO fails on BuildFlow's TIMING-regression verdict (govalid-generate 2 ms→8 s = pure box-load noise at load 58–228) — a new structural-red class beyond the handoff's known debt (2026-11-18 correction: MISATTRIBUTED — verified advisory-only: the verdict has never gated the exit code, see BuildFlow regression_verdict.go since 73e61349e 2026-09-02 whose run verdict must never depend on the timing query, and a live probe under load ~100 printed dozens of regression lines and exited 0; the real blockers were the pre-existing structural debt and gotcha-51-class step failures) | RESOLVED — no hook change needed; §g② answered empirically | S |

## c) NOT STARTED

The handoff's remaining backlog is untouched by this session (correctly — CI healing came first): PR #101 CSS-Freshness re-run verdict; admin-demo + systemadapter CHANGELOG entries; §g③ e2e-flake annotation; K2 idempotency default; K4 waves; f.32 dashboardui QUERY ladder; f.33 merge-loop note; §f.44 TODO_LIST harvest; the two candidate gotchas; close-out push + watch.

## d) TOTALLY FUCKED UP

- **Nothing destructive landed.** One mid-session edit error: my first ci.yml edit swallowed the neighboring "Require tags check" comment block (bad old_string scoping) — caught on the follow-up view and restored in the next edit, before commit.
- **Theory-before-repro cost real time** on the ambiguous import: ~6 dead-end hypotheses (zip contents, tag shapes, go.mod replaces) before running the decisive /tmp repro. The repro took three commands. The 07-20 session's "not reproducible" verdict came from a repro with CANONICAL paths — comparing the sed spelling would have settled it immediately.
- **Wrong job→step attribution early in triage**: my first log extraction lacked job-prefix filtering and I briefly attributed the ambiguous import to module-architecture, whose real failing step was root-files. The step-level `gh run view --json jobs` query should have been step one.
- **Near-miss**: I almost bumped go.work to 1.27.4 for the vuln before reading `check-go-toolchain.sh` — that gate exists precisely to block that bump under `GOTOOLCHAIN=local` (the 1.26.5-era trap). Read-the-gate-first caught it; the landed fix (security-job-only pin) is strictly better.
- **Two edit-tool staleness round trips** on ci.yml/TODO_LIST.md (churned by sibling waves; I had only rg'd them). View-before-edit is the rule; I paid the tax twice.

## e) WHAT WE SHOULD IMPROVE

1. **Repro-first discipline**: for any CI-only red, build the /tmp CI-layout clone BEFORE theorizing. This session's only mystery that fell to a repro fell in minutes; the ones I theorized about cost an hour.
2. **Step-level CI queries first**: `gh run view --json jobs --jq` (job → failed STEPS) before any log grep. The log-failed dump interleaves jobs and invites misattribution.
3. **Handoff claims need re-verification at read time**: "master CI GREEN" was a different workflow. Verify workflow NAME + sha before trusting a green claim.
4. **The hook's regression-verdict needs a noise floor**: timing-regression verdicts must not block commits under box load (see §g).
5. **In-flight edits from quiesced sessions**: the overnight session left a half-written ledger line ("→ measured after this row") that the daemon then committed; completing foreign ritual lines should be a same-session rule, not a next-session cleanup.
6. **Build guard for build outputs**: a 42 MB `go build -o server` product entered the tree via a heuristic daemon commit. The binary guard caught it in CI — a local pre-commit staged-mode guard should have caught it first (the staged-mode leg exists; it was bypassed by the daemon's commit path — same hole as gotcha 35's class).
7. **Corpus/tail baseline ritual**: this report bumps both baselines same-commit (done below); the overnight session's baseline flow shows the ritual works but leaves trails when sessions are interrupted mid-ritual.

## f) Up to 50 next items (priority order; the authoritative backlog remains TODO_LIST.md + the 07-20 report §f)

1. **Push the 4-commit train + `push-and-watch`** (§f.1 of the 07-20 report) — the CI heal is unproven until the runner agrees.
2. Watch the e2e job's post-Warm-cache legs on the new train (the Warm-cache step was the first failure; later legs unverified).
3. Amend gotcha 41: root cause found (`:memory:` per-connection DBs under pooling), fix landed, rerun-once rule now obsolete for this class.
4. New gotcha: CI sed must spell sibling replaces canonically — `..`-spelling = second module instance = ambiguous import (the 373209a7-class gets a sibling entry + repro recipe).
5. New gotcha: heuristic daemon commits bypass the staged-mode binary guard — a 42 MB `server` binary entered the tree; guard the daemon path (content-scan in the daemon's pre-commit, or a deny-list in check-large-files --all wired to check-modules).
6. Decide the 42 MB blob history cleanup (owner; §g).
7. Decide the pre-commit regression-verdict noise policy (owner; §g).
8. PR #101: observe the CSS-Freshness re-run verdict + maintainer review state (carried from handoff item 16).
9. admin-demo CHANGELOG entry for the page rule + posture tests (handoff item 15).
10. systemadapter CHANGELOG line for the tenant-deletion cascade fix (handoff item 15).
11. §g③ decision: annotate the e2e CI flake known-flaky or keep fail-loud (handoff item, owner-gated).
12. K2: the go-idempotency v0.5.0 train (23 modules lagging, verified via the hook's train-lag list) + the default-on decision (owner-gated; no recorded recommendation).
13. K4 under the D18 precedent: tick + execute the M2M D1–D4 packet (recommendations pre-staged at docs/proposals/2026-11-07_m2m-owner-decisions-d1-d4.md).
14. K4: the migration D1–D5 packet (plan-recorded default exists).
15. K4: the adminui D1–D3 packet (recommendations pre-staged 2026-11-14).
16. f.32: dashboardui QUERY ladder (stretch, carried).
17. f.33: the merge-loop note — do it or strike it (handoff item 19; skipped silently last session).
18. §f.44: TODO_LIST harvest of the landed 14-37 §f items.
19. Candidate gotcha: the fixture-mirrors-bug class (T04: fixtures seeded display-form IDs, realigning two wrong forms to green over buggy production code).
20. Candidate gotcha: pathspec commits trip the BuildFlow pre-commit hook's nix-fmt roundtrip under concurrency (documented in the 07-20 report, not yet in gotchas).
21. True the AGENTS cqrs-lint row to v4.22.2 (overnight train).
22. Verify the 22-25 session's gate-escalation claim against this session's cqrs-lint gate run (binary floor v4.22.0 vs v4.22.2).
23. Full chromium e2e battery locally once the train pushes (the e2e job's Warm-cache fix needs its battery green too).
24. erraudit strict ×2 re-true after the overnight sessions' new code (the window counter may have grown again).
25. check-modules composite locally before the push (the composite has not run this session; individual gates ran).
26. Sweep `git log --all --diff-filter=A` for OTHER incidental binaries captured by the daemon in the red window (the `server` find suggests a class).
27. Consider a `.gitignore` entry for root-level Go build outputs (`/server`, `/cqrs-htmx`, `/demo`) to kill the class at source.
28. Post-push: annotate the 07-20 report's "master CI GREEN" correction (it was train-lag, not ci.yml).
29. Re-verify `docs/status/TAIL_BASELINE`=7 vs live tail after this report lands (same-commit bump done).
30. The `docs/status/README.md` foreign ledger-line completion rides this commit (attributed in the message).
31. Fleet nixpkgs rev watch: bump the ADR-0003 rev when nixpkgs ships go ≥ 1.27.4, then revert the security pin (TODO P1 row landed).
32. go-idempotency v0.5.0 sweep decision: ride the next family train or defer to K2's outcome.
33. Check whether the e2e job needs the same sibling-clone treatment for `check-workspace-build` parity (its filtered gate ignores path replaces — verify that stays true after the sed change).
34. Verify the canonical-sed fix under the ACTUAL runner path shape (`/home/runner/work/cqrs-htmx/cqrs-htmx` vs my /tmp layout) — the dirname form is layout-independent by construction, but the first CI run proves it.
35. Backfill the harness-cheatsheet with the /tmp CI-layout repro recipe (it is now a proven triage instrument).
36. The 07-20 report §g list: this report adds §g answers only where the owner gave them; re-ask unanswered ones in its §g.
37. `nix run .#check-module-layers` + `check-reexport-policy --strict` on the post-wave tree (the sibling's 33-file suppression wave touched many modules; layering unchanged per its report, but the composite never ran).
38. Branching-flow ratchet re-pin check: the overnight + this session added files; confirm no NEW analyzer findings past the 212 baseline.
39. Confirm `check-dep-budgets` still green after the overnight module sweeps (budget 22 pinned by the 07-20 session).
40. After the train greens: harvest this session into TODO_LIST (close the CI-heal thread, open the watch thread).
41. Idle-box bench re-pin opportunity: box load is volatile (58→228); the bench baseline must NOT be re-pinned under load (standing rule) — schedule for a quiet window.
42. The 22-25 report's "417 md files" count vs the corpus gate's 417 — consistent; keep the ledger monotone on the next report.
43. Verify the sibling's C027/C009 inline suppressions survive the cqrs-lint v4.22.2 gate (directive grammar stability across the train).
44. Update CURRENT_FOCUS.md to point at the watch thread (post-push, same commit as the harvest).
45. Consider quarantining `TestSQLEventStore_UnsupportedDialect`'s bare `:memory:` open (line ~218) to the shared DSN too — single-op today, but the class should have one shape.
46. Read the overnight 22-25 report before executing K4 (its gate-escalation may change the wave's gate set).
47. Sweeps for the fleet-presentation plan (18-29 plan, TODO P1 top row) remain untouched by this session — they need a quiet tree window.
48. The `crush-daily` scheduler fires on this box; verify no scheduled session collides with the next push window (gotcha 47's two-agents class).
49. Post-green CI: close the loop in AGENTS gotchas 22-class if the 1.27.4 toolchain changes any gate verdict (formatter fingerprint drift class).
50. The 07-20 report's §f list still owns items not enumerated here; that file plus TODO_LIST.md remain canonical — this list is the session's delta only.

## g) Questions for the owner (max 3)

1. **The 42 MB `server` binary** tracked at repo root in `b8a0a00d` (removed from HEAD by a later daemon capture, still in pack history): do you know which session/command built it (shape = `go build -o server` at root), and do you want a git-history rewrite (filter-repo) to reclaim the pack, or accept the bloat and just add the `.gitignore`/guard hardening?
2. **The pre-commit hook's regression-verdict red**: BuildFlow's timing-regression verdict (steps "slower than baseline", e.g. govalid-generate 2 ms→8 s at box load 58+) now blocks EVERY commit under load on top of the known structural debt. Demote the regression verdict to advisory in pre-commit mode (I can do that in `.githooks/pre-commit` + receipt), or keep it blocking? (2026-11-18 ANSWERED — premise false: the verdict is ALREADY advisory-only and never gates; verified in BuildFlow source (regression_verdict.go, advisory since its creation 2026-09-02) and live (probe under load ~100: regression lines printed, hook exit 0). No demotion to perform; the hook failures under load were genuine step failures accompanied by the advisory line, and this misread is exactly gotcha-60's comment-as-contract class applied to tool output.)
3. **§g③ + K2/K4 scope**: with the blanket GET-SHIT-DONE directive — do the pre-staged M2M D1–D4 and adminui D1–D3 recommendations tick under the D18 precedent so I execute those waves (multi-session), and should the e2e flake (failed-then-green on 9d159fdc) be annotated known-flaky or stay fail-loud?

---

*Corpus ledger: 61,812 → measured after this row (this report + the ledger line below; baselines bumped same-commit).*
