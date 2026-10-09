# Status: samber/do × health SUPERB execution — phases 2–5 complete, push reconciling

> **Date:** 2026-10-09 18:10 CEST
> **Input:** [`docs/planning/2026-10-09_16-52_SUPERB-samber-do-health-truth-and-composability-pareto-plan.md`](../planning/2026-10-09_16-52_SUPERB-samber-do-health-truth-and-composability-pareto-plan.md) (annotated in place) · Phase-1 status [`2026-10-09_17-19`](2026-10-09_17-19_SUPERB-execution-phase1-status.md)
> **Predecessors:** review [`2026-10-08_samber-do-v2-health-checks-deep-review.md`](../research/2026-10-08_samber-do-v2-health-checks-deep-review.md) (now with Outcome Annex) · self-critique [`2026-10-09_16-21`](2026-10-09_16-21_samber-do-health-deep-review-session.md)

## a) What is DONE (this session, phases 2–5 + boot proofs)

| Item | State | Evidence |
| --- | --- | --- |
| T5 root state export (`ProjectionStatus*` + `ProjectionDrainReady`) | DONE | commit `0012e2dd`; root projection suite green; root lint 0 issues; CHANGELOG receipt |
| T4 health `RecorderChain` + duration_ns + `NewWithDetailedCheck` ctor (implements #31) | DONE | commit `f0e6f7e5`; health suite green `-race`; health lint 0 issues; CHANGELOG receipt; demo suite green |
| T6 integration proofs (lifecycle E2E + plugin-is-recorder) | DONE | commit `8669f538`; hermetic (GOWORK=off) test green + lint 0 issues — exactly CI's mode |
| T7 docs surface map + cross-links (health/auditlog/setup/root READMEs) | DONE | commit `ff1abd0b`; `NewChecks` package + route names verified against go-health v0.5.1 source |
| T8 drain posture (RunWithAppkit = production drain path) | DONE | commit `ff1abd0b`; claims verified against `run_appkit.go` (ReadyCheck, DrainDelay, /health/live) first |
| T9 harvest (TODO_LIST row + AGENTS gotcha 37 + plan annotation + research Outcome Annex) | DONE | commit `ff1abd0b`; F12 repro embedded durably in the research report |
| Boot-smoke test + PORT env override (beyond plan) | DONE | commit `b6277605`; `TestBoot_ProcessSmoke` boots the compiled binary on an ephemeral port, polls /readyz to 200; `buildRouter` shared by main + tests |
| Full hermetic battery (`nix run .#test`) | DONE | 13/14 modules ok; the 1 red is systemadapter's pre-adjudicated foreign replace (gotcha 36, `undefined: system.WithClock`) |
| Upstream filings (F11/F12) | OPEN — owner decision | queued in TODO_LIST (filing = external action; repro + mechanism captured) |

## b) DEVIATIONS from the plan (both deliberate, both gotcha-30-class)

1. **f16 (delete health's status mirrors) DEFERRED to the health train.** The plan ordered T5→T4 with mirror deletion, but health's go.mod requires root v4.13.3 (published, lacks `ProjectionStatus*`) — consuming the unpublished constants would break hermetic builds that workspace mode masks. The mirrors now carry a switch-on-train comment; TODO_LIST health-train rider owns the deletion + root-require bump.
2. **f25's chain proof in integration_test DEFERRED with it.** CI runs integration_test `GOWORK=off` against the PUBLISHED health tag, so a `RecorderChain` test there cannot pass until the train. The plugin-is-recorder + lifecycle halves (published API only) landed instead; the chain unit-proofs live in health's own suite.

## c) F12 CORRECTED (research-grade)

The handoff's "OverrideNamed on an already-invoked do.As alias is nondeterministic" was directionally right, mechanically wrong. Source-verified root cause: `do.InvokeAs` resolves by SCANNING the service map for the first interface-satisfying service — with ≥2 candidates the pick follows Go map iteration order, nondeterministic per process run (observed 15–28/200 leak across 5 runs, single-goroutine). Safe pattern (deterministic, verified 400/400): `AsNamed` + `InvokeNamed` + alias-typed `OverrideNamed`. Full repro + both variants in the research report's Outcome Annex; rule codified as AGENTS gotcha 37.

## d) Environment events this session

- **Concurrent sessions active throughout:** a go-health v0.5.1 bump sweep (committed `06b6ec5d`), an httputil-train session, and a go-cqrs-lite schema/systemscenario wave (the battery's single red). No collisions: my files never overlapped theirs.
- **Pre-commit red the whole session** on the foreign identity-model exhaustruct findings (`external_account.go:23`, `fold.go:190` — their status report lists them as their tasks 5-7). All my commits used the documented `--no-verify` fallback with per-module independent verification (tests + lint green named in each message).
- **Daemon interleaving hit on every phase** (T4, T6, docs, boot): recovered each time by soft-reset folding into one coherent commit; final chain is 5 clean commits.
- **A concurrent session PUSHED mid-flight** (origin/master moved to `3ffb168d`, my pre-amend T5 shape) — local and origin have diverged cleanly: origin holds my T5 content in daemon shape, local holds the 5-commit verified chain (+761 lines, all mine in the delta).

## e) ~~Push state (authorized, sequenced)~~

> **ANNOTATED 2026-10-09 18:25:** DONE — pushed as `ae7a25ee` through the strict pre-push gates (release-train strict: 841 requires published, 0 drift). A concurrent session had pushed my pre-amend T5 shape mid-flight (`3ffb168d`); reconciled by merge taking ours (origin's delta was strictly my own intermediate content; post-merge tree verified byte-identical to the pre-merge HEAD). CI run 37957309392: every job my commits touches is GREEN (root, health, integration_test, samber-do-demo, examples, coverage, docs gates). The 8 red jobs are all pre-existing foreign classes: systemadapter filesystem replace ×4 (test / module-architecture / mod-tidy / lint — gotcha 36, drops with the GCL systemscenario wave), identity-model + usermgmt exhaustruct (their tasks 5-7), dashboardui exhaustruct_v5 panic (their filed question 3), govulncheck stdlib CVEs (owner-gated fleet toolchain question). No NEW red introduced.

## f) Rubric re-score (T14)

| Dimension | Was | Now | Why |
| --- | --- | --- | --- |
| Service orientation | 4.5 | 4.5 | unchanged structurally |
| Composability | 4.0 | 4.5 | RecorderChain kills the one-recorder ceiling; "one probe" composition is literal API |
| Resilience | 4.0 | 4.0 | drain posture documented; MarkDraining still owner-gated (T13) |
| Self-health | 3.5 | 4.0 | demo truth + durations on the wire + boot proofs; capped by F8 code drift (T12) and Run's no-flip drain (T13) |
| **Overall** | **8/10** | **8.5/10** | the flagship demo is no longer health theater; remaining gaps are owner-gated or train-time |

## g) OPEN (owner decisions)

1. **T12** — unify projection-health error codes (health `cqrshtmx.health.*` → root `projection.*`) in the next minor train; wire-visible, needs owner OK.
2. **T13** — `Bundle.MarkDraining()` vs keeping the RunWithAppkit docs posture.
3. **Upstream filings** — samber/do (InvokeAs interface-scan) and/or go-health (method-less RegisterRoutes): file, or keep internal?

## h) Next mechanical step

Rebase + push per §e once the tree is quiet (`nix run .#wait-tree-quiet`); then the health train rider when the owner cuts the next train.

*Point-in-time snapshot per [`docs/status/README.md`](README.md). Items resolved later get inline strikethrough + a dated `> ANNOTATED` blockquote, never a rewrite.*
