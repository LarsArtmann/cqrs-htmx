# Status — SUPERB round executed (20/20 tasks), preflight refuses the foreign wave

**Date:** 2026-10-09 ~04:15 CEST · **Branch:** master (local ahead of origin; push gated on the GCL wave, see TODO push-window row) · **Plan:** [`docs/planning/2026-10-08_20-00_SUPERB-train-trust-and-cadence-pareto-plan.md`](../planning/2026-10-08_20-00_SUPERB-train-trust-and-cadence-pareto-plan.md) (annotated in place)

## a) Executive summary

All 20 plan tasks executed with receipts. The round's product is FOUR new atomic gates (train-preflight, hermetic consumer catch, nightly train-lag report, checker/fixer equivalence meta-test), honest pre-commit budgets, bump-dep hardening, the CHANGELOG rotated into v4.13.1–3, and a 50-row semantic repair of the archived-report damage class. The final verification produced the round's best moment: the FIRST live train-preflight run refused a push with precise intelligence (foreign in-flight GCL wave + 42 lags), and the T12 erraudit sweep caught a fresh context_loss in same-day sync code within hours.

## b) Gates shipped (all atomic: checker + fixture self-test + flake app + check-modules stage + CI step)

| Gate | Purpose | Evidence |
| ---- | ------- | -------- |
| `.#train-preflight` (T02) | quiescence → surprise → lint → hermetic battery → strict train, gotcha-2 guards | self-test 6/6; first live run REFUSED correctly (§d) |
| `.#check-train-consumers-hermetic` (T04) | gotcha-30 mechanized; GOWORK=off pipeline per dependency-changed module; release-checklist step before `verify-tag --push` | self-test 5/5; retro-run at the pre-v4.13.3 worktree RED on setup (22 other modules green) — the catch is proven |
| `.#train-lag-report` + nightly workflow (T08) | advisory lag table + fix recipe; never red on lag; broken gates propagate | self-test 5/5; scheduled 03:17 UTC, no token (public repos) |
| `test-status-table-equivalence` (T05) | checker+fixer agree on 7 adversarial fixtures via the shared `scripts/lib/status_table.py` | 7/7; the split-brain class is pinned, not just aligned |

## c) Fixes at root cause

- **T01 pre-commit budgets** (`7505cb85`): `--budget 150s` + 2m step timeouts, zero steps scoped out — the 13/13 `--no-verify` day is closed; hook rc=0 proven on a scratch commit.
- **T03 vendorHash**: `nix build .#benchstat` rc=0 → third misattribution receipt; hash untouched.
- **T06 CSP sweep**: 7 files, zero stragglers (agents-notes receipt).
- **T09 bump-dep**: `--message` placeholders, auto tag-cache refresh (gotcha 27), daemon-amend hints; self-test 7/7.
- **T12 erraudit**: sweep found a fresh `context_loss` in same-day sync code → fixed with `WithContextAny("after_id", …)`; gate TOTAL=0.
- **T13 semantic repair**: 50 rows / 10 files beyond the planned 7 — bare-`~~` ID cells dropped, columns realigned to separator width (3 passes; the 2-dash malformed separator defeated naive detection twice).
- **T14 CHANGELOG rotation**: 37 entries bucketed into [v4.13.1]/[v4.13.2]/[v4.13.3] by tag window; [Unreleased] keeps genuinely unshipped content.
- **T16 litter**: workspace-build temp → /tmp with absolute use-path rewrite; daemon-capture class dead; gate green 28 modules.
- **T20 row gate**: mixed tables now reported as `file:line` anchors.

## d) Final verification — honest state

- All new gates + fixture self-tests GREEN (listed above); row/annotation/freshness/links gates GREEN (469 files).
- `check-modules --report`: 26/27 modules green; **systemadapter red via its pre-tag local replace** to the mid-migration local `go-cqrs-lite/systemscenario` (foreign in-flight work; vet+hermetic+isolation legs all trace to it).
- **CI run 37933181002 red on their push**: 12 lint jobs from the rolled runner image (Go 1.27.2 export-data v5 > golangci v2.13.2 ceiling) → FIXED by `af14058a` (root directive pinned 1.27.1); the test job's red is the foreign replace. T19's same-day correction is in agents-notes.
- **First live `.#train-preflight` refused (rc=1)**: lint+tests (the foreign module) + train lag 42 (the same GCL wave). Alignment must wait for the upstream publish (gotcha-4); the push-window TODO row carries the full sequence.

## e) Owner gates / decisions opened

- **D16** GitHub-Releases per tag: default DEFER, recommendation = root tags only. **D17** gosec `@master` float: pin at next CI touch. Both in TODO_LIST.
- **T18 upstream ask** (annotate directive-wave tags carrying code migrations): owner-channel draft; `signing/v4.4.0` tag-level WithEncoding VERIFIED at source.

## f) Next tasks (ranked)

1. GCL wave publishes systemscenario → align 42 lags (the nightly report prints the recipe) → preflight → push → `gh run watch --exit-status`.
2. D16/D17 owner calls.
3. Coverage-gate re-run on a quiet tree (tonight's run was blocked by the foreign module; 11 visible modules all above threshold).
4. Upstream filing of the tag-annotation ask (after D-lane decision).

> ANNOTATED 2026-10-09: executed inline; see the plan's EXECUTED blockquote for the commit range.
