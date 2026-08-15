# Status Report: appkit Spike (ADR-001 P3 / plan task M18) — ADOPT

**Date:** 2026-08-16 01:38 (Sunday)
**Session scope:** The M18 spike from go-appkit's SUPERB ecosystem plan (`docs/planning/2026-08-15_19-27_SUPERB-ECOSYSTEM-EXECUTION.md`), validating ADR-001 "appkit-as-foundation" ([go-appkit design-decisions §11](https://github.com/larsartmann/go-appkit/blob/master/docs/planning/design-decisions.md)). Branch: `spike/appkit-server` (uncommitted until this report ships with it). Everything below was verified by running the spike tests/benchmark on this machine this session.

**Verdict: ADOPT (ADR-001 Option A confirmed).** All three prerequisites landed additively; zero regressions found; the two feared regressions (SSE write-timeout kills, loss of projection-aware readiness) are both disproven by passing tests.

---

## What the spike is

`setup/run_appkit.go` adds `(*Bundle).RunWithAppkit` — `RunHandler` with the server layer swapped from `httputil.Server` to `appkit.Service`. Everything else is identical: the bundle's own middleware chain, `Mount`, Close-on-every-exit, the 5s/60s/30s ReadHeader/Idle/Shutdown timeout trio with Read/Write deliberately absent (SSE), and `context.WithoutCancel` for the shutdown budget. New behavior (the uplift ADR-001 wants):

- `ReadTimeout`/`WriteTimeout` set to `appkit.NoTimeout` — the P1 prerequisite shipped in go-appkit core (`1e19ef5`) that makes deadline-free SSE explicit instead of relying on zero-value accidents.
- `ReadyCheck: b.projectionReadyCheck()` — appkit's `/health/ready` now answers from `cqrshtmx.ProjectionReadinessCheck` (503 while any projection drains or has failed) via the P2/M08 `ReadyCheck()` composition.
- `DrainDelay: 2s` — readiness flips to 503 first, load balancers get a beat, then connections close. `RunHandler` has no drain phase at all.
- appkit's generic stack (Recovery → RequestID → Logging → SecurityHeaders) wraps the bundle's domain chain; health endpoints stay appkit's own so the drain probe governs them.

The `require` + `replace => ../../go-appkit` in `setup/go.mod` is spike-only and must be repointed at a published appkit tag before merge (see follow-ups).

## Evidence (all from this session's runs, `-race -count=1`)

| Check                                                                                                      | Result                                                                           |
| ---------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| M18.3 — SSE header flush through full stack (appkit outer + bundle inner, first event 400ms after headers) | **PASS** — headers observed ≪400ms, event body intact end-to-end                 |
| M18.2 — `/health/ready` projection-aware (polls to 200) + clean drain/shutdown on cancel                   | **PASS** — 2.10s wall including the 2s drain, zero goroutine leaks under `-race` |
| Response parity — ordinary handler through both stacks                                                     | **PASS** — 200 + body unchanged                                                  |
| Full `setup` module suite (`go test -race -count=1 ./...`)                                                 | **ok** — 8.16s                                                                   |
| `go vet` + `go build`                                                                                      | clean                                                                            |
| M18.4 — benchmark (same ping handler, both paths, drain excluded from timing)                              | baseline-httputil **16178 ns/op** vs appkit-service **45049 ns/op**              |

### Reading the benchmark honestly

~2.8x ns/op sounds alarming; it is not. The delta (~29µs/request) is dominated by appkit's Logging middleware emitting a formatted INFO line per request to the test process's stderr — the baseline path logs nothing per request. That is a real cost (it is also the observability uplift being adopted), it shrinks to noise with any real handler doing I/O, and it can be tuned later (sampling, level) without API change. Single-connection throughput at 45µs/op is still ~22k req/s. The benchmark is explicitly a smoke comparison, not a production load profile.

## What adoption requires next (follow-ups)

1. **Publish the dependency:** appkit needs a tagged release carrying `NoTimeout` + `ReadyCheck` (core v0.3.0 or similar); then drop the `replace` in `setup/go.mod` and point at the tag. Until then this branch cannot merge.
2. **Swap `setup.Run`/`RunHandler` internals** to construct `appkit.Service` behind the existing signatures (the ADR-001 decision proper — `RunWithAppkit` becomes the implementation, not a sibling). Public API of cqrs-htmx unchanged.
3. **Decide the default middleware posture:** appkit's Logging at INFO per request is verbose for the test/bench path; consider wiring the bundle's logger level or a sampling option.
4. **`Addr()` uplift:** with appkit, `setup` can finally expose the live listener address (`Addr(":0")` pattern) — small, additive, previously impossible via `httputil.Server`.

Harvested into `TODO_LIST.md` (P3 — blocked on the upstream tag).

## Not done / out of scope

- No merge: the branch stays `spike/appkit-server` until the appkit tag exists (push approval for go-appkit's 20 commits + 3 tags is still pending with the user).
- `feat/transport-package` (`ac743f30`) untouched — separate concern.
- The benchmark numbers are single-machine, single-connection, one run each; directional, not citable.
