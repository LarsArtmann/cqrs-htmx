# go-appkit Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-appkit` v0.9.0 (+ `health/` submodule v0.3.0 upstream) · Checkout: `/home/lars/projects/go-appkit`

## Verdict at a glance

| Pin                   | Latest (verified 2026-11-05) | Prod import sites                   | Test import sites                                | Adoption                                    | Score      |
| --------------------- | ---------------------------- | ----------------------------------- | ------------------------------------------------ | ------------------------------------------- | ---------- |
| v0.9.0 (setup direct) | v0.9.0 (= latest)            | 2 files (`setup/run_appkit.go`, +1) | 2 files (incl. `setup/run_appkit_test.go` bench) | **Used — the `RunWithAppkit` runtime seam** | **80/100** |

## What the library is

Production-ready HTTP service lifecycle: composes httputil (server, middleware), charmbracelet/log, and go-error-family into `Service`/`ServiceConfig` — start hooks, drain-aware shutdown, log levels/formats, metrics wiring, readiness registration, signal handling. A consumer-app convenience, which cqrs-htmx's `setup` module re-exposes as `RunWithAppkit`.

## How cqrs-htmx uses it today

`setup/run_appkit.go`: `appkit.NewService` + `ServiceConfig` with `ReadTimeout`/`WriteTimeout: appkit.NoTimeout` (lines 74-76 — the SSE-safe posture, gotcha 46), `Metrics: b.config.Metrics` (line 82 — forwarding into appkit's httputil-composed metrics), `appkit.LogLevel`/`LogLevelError` (×4/×3 — incl. bench log suppression per the benchmarking rules). The canonical bench `BenchmarkSpikeBaselineVsAppkit` measures the seam.

## Capability inventory

| #   | Capability                                                       | Status             | Evidence / reason                                                                                                                                                          |
| --- | ---------------------------------------------------------------- | ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Service composition (`NewService`, `Service`, `ServiceConfig`)   | 🟢 Fully leveraged | the whole point of `RunWithAppkit`                                                                                                                                         |
| 2   | Timeout policy                                                   | 🟢 Fully leveraged | `NoTimeout` ×3 — deliberately SSE-safe, documented in the file header                                                                                                      |
| 3   | Log level control                                                | 🟢 Fully leveraged | config forwarding + `LogLevelError` bench suppression                                                                                                                      |
| 4   | Metrics forwarding                                               | 🟢 Fully leveraged | `ServiceConfig.Metrics` ← `setup.Config` (indirectly adopts httputil metrics — see the [httputil report](httputil.md))                                                     |
| 5   | Lifecycle hooks (`Hook`, drain/shutdown hooks, `ShutdownConfig`) | 🔴 Missed (low)    | setup manages its own lifecycle (projection drain, broadcaster close); forwarding consumer drain hooks through appkit's hook contract would unify the two shutdown stories |
| 6   | Logger init (`InitLogger`, `LogFormat`, `LoggerConfig`)          | ⚪ N/A             | owned by `NewService` defaults; consumers who want charmbracelet config import appkit directly                                                                             |
| 7   | Health/readiness (`RegisterHealth`, `ReadyHandlerWithProbe`)     | ⚪ N/A             | deliberate divergence: readiness is the go-health bridge + `ProjectionReadinessCheck`; appkit's variant would duplicate                                                    |
| 8   | Signal handling (`WaitForSignal`)                                | ⚪ N/A             | exercised inside `Run`                                                                                                                                                     |
| 9   | Error mapping (`HTTPStatus`, `RegisterClassifications`)          | ⚪ N/A             | `httperr` owns mapping; classifications already registered via root's `sync.Once`                                                                                          |

## Score derivation

9 rows − 4 N/A = **5 applicable**; 4 full ×1 + 1 missed ×0 = **4 → 80/100** (band: high) — the single miss is the hook unification, recorded as low priority.

## Version currency

v0.9.0 pinned = v0.9.0 latest (verified `git ls-remote --tags origin`, 2026-11-05). The local checkout also carries the upstream `health/` submodule (v0.3.0) — orthogonal to this pin.

## Should cqrs-htmx use it (more)?

Only the hook-forwarding idea, and only if a consumer asks for unified drain hooks. The seam is otherwise exactly right: thin, SSE-safe, measured.

## How to verify

```bash
rg -oN 'appkit\.[A-Z][A-Za-z0-9]*' setup -g '*.go' | sed 's/.*://' | sort | uniq -c | sort -rn
sed -n '60,90p' setup/run_appkit.go   # NoTimeout + Metrics forwarding
git -C ~/projects/go-appkit ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- Bench evidence: `setup/run_appkit_test.go` (`BenchmarkSpikeBaselineVsAppkit`); gotcha 46 (SSE vs WriteTimeout).
