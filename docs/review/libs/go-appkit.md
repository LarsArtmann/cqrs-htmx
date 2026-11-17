# go-appkit Utilization Report

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-appkit` v0.12.0 (root module; 12 submodule tag trains upstream — none consumed) · Checkout: `/home/lars/projects/go-appkit`

## Verdict at a glance

| Pin                                                   | Latest (verified 2026-11-14) | Prod import sites                                  | Test import sites                       | Adoption                                    | Score       |
| ----------------------------------------------------- | ---------------------------- | -------------------------------------------------- | --------------------------------------- | ------------------------------------------- | ----------- |
| v0.12.0 (setup direct; examples+integration indirect) | v0.12.0 (= latest)           | 2 files (`setup/run_appkit.go`, `setup/config.go`) | 3 files (bench + hooks + metrics tests) | **Used — the `RunWithAppkit` runtime seam** | **100/100** |

## What the library is

Production-ready HTTP service lifecycle: composes httputil (server, middleware), charmbracelet/log, and go-error-family into `Service`/`ServiceConfig` — start hooks, drain-aware shutdown, log levels/formats, metrics + version wiring, readiness probe, signal handling. A consumer-app convenience, which cqrs-htmx's `setup` module re-exposes as `RunWithAppkit`. Upstream has since grown 12 submodule trains (`config/` v0.1.1, `cqrs/` v0.9.0, `errorpages/` v0.1.3, `flightrecorder/` v0.2.0, `flightrecorderhealth/` v0.1.7, `health/` v0.3.2, `httpx/` v0.2.0, `otel/` v0.4.0, `preset/` v0.1.1, `realtime/` v0.2.0, `security/` v0.2.0, `systemd/` v0.2.0) — cqrs-htmx consumes **none** (see § Version currency).

## How cqrs-htmx uses it today

`setup/run_appkit.go:134-150` — `appkitServiceConfig` now forwards the full consumer surface: `ReadTimeout`/`WriteTimeout: appkit.NoTimeout` (137, 139 — SSE-safe posture, gotcha 46) with the `ReadHeaderTimeout`/`IdleTimeout` reaping pair kept, `ShutdownTimeout` + `DrainDelay` (default 2s, line 121 — the drain phase `Run` deliberately lacks), `ReadyCheck: b.bundleReadyCheck()` (143), `Metrics` (145), `Version` (146), and the three lifecycle-hook seams `StartHooks`/`DrainHooks`/`ShutdownHooks` ← `Config.Appkit*Hooks` (147-149; fields at `setup/config.go:364-378`, `[]appkit.Hook`). `Metrics *appkit.MetricsConfig` and `Version` are declared consumer-facing at `setup/config.go:348-358`. Hook ordering (start → drain → shutdown, shutdown before bundle `Close` tears down the SSE hub) is contract-pinned by `setup/run_appkit_hooks_test.go` (A22/B93). The canonical bench `BenchmarkSpikeBaselineVsAppkit` (`setup/run_appkit_test.go:369`, `appkit.LogLevelError` at 408) remains the AGENTS.md:116 pattern. `RunWithAppkit` is the documented production drain path (round45 T13, closed won't-fix 2026-10-31); plain `Run`/`RunHandler` keep their own SSE-safe no-timeout posture (`setup/run.go:38-40`).

## Capability inventory

| #   | Capability                                                       | Status             | Evidence / reason                                                                                                                                                                                                                                                |
| --- | ---------------------------------------------------------------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Service composition (`NewService`, `Service`, `ServiceConfig`)   | 🟢 Fully leveraged | the whole point of `RunWithAppkit`                                                                                                                                                                                                                               |
| 2   | Timeout policy                                                   | 🟢 Fully leveraged | `NoTimeout` ×2 — deliberately SSE-safe, documented in the file header (lines 20-22)                                                                                                                                                                              |
| 3   | Log level control                                                | 🟢 Fully leveraged | config forwarding + `LogLevelError` bench suppression                                                                                                                                                                                                            |
| 4   | Metrics forwarding (`MetricsConfig`)                             | 🟢 Fully leveraged | `ServiceConfig.Metrics` ← `setup.Config.Metrics` (indirectly adopts httputil metrics — see the [httputil report](httputil.md))                                                                                                                                   |
| 5   | Lifecycle hooks (`Hook`, drain/shutdown hooks, `ShutdownConfig`) | 🟢 Fully leveraged | **landed since the 2026-11-05 audit**: `AppkitStartHooks`/`AppkitDrainHooks`/`AppkitShutdownHooks` forwarded (run_appkit.go:147-149), order + before-Close contract tested                                                                                       |
| 6   | Logger init (`InitLogger`, `LogFormat`, `LoggerConfig`)          | ⚪ N/A             | owned by `NewService` defaults; consumers who want charmbracelet config import appkit directly                                                                                                                                                                   |
| 7   | Health/readiness (`ReadyCheck`, `RegisterHealth`)                | 🟢 Fully leveraged | was "deliberate divergence", now composition: appkit's `ReadyCheck` runs the full bundle readiness set (projection drain gate, shared-hub check, `Config.HealthChecks`); `RegisterHealth: nil` keeps appkit's default `/health*` — one composition, two surfaces |
| 8   | Signal handling (`WaitForSignal`)                                | ⚪ N/A             | exercised inside the service; setup drives `ctx.Done()` itself                                                                                                                                                                                                   |
| 9   | Error mapping (`HTTPStatus`, `RegisterClassifications`)          | ⚪ N/A             | `httperr` owns mapping; errorfamily wraps the seam's own failures                                                                                                                                                                                                |
| 10  | Version/build info (`ServiceConfig.Version`, `/version`)         | 🟢 Fully leveraged | existed at v0.9.0 but unlisted prior audit; now `Config.Version` → `GET /version` + `appkit_build_info` label (config.go:350-358, run_appkit.go:146)                                                                                                             |

## Score derivation

10 rows − 3 N/A = **7 applicable**; 7 full ×1 = **7 → 100/100** (band: max) — up from 80: the single miss (hook forwarding) landed with a contract test, readiness moved from N/A-divergence to deliberate composition, and the previously-unlisted Version surface is wired.

## Version currency

v0.12.0 pinned = v0.12.0 latest (verified `git ls-remote --tags origin`, 2026-11-14; v0.12.0 tagged 2026-11-07). **No submodule consumed**: `rg 'go-appkit/' --type go` → 0 hits repo-wide; every go.mod requires only the root module (`setup/go.mod:16` direct; `examples/setup-demo`, `examples/async-startup-demo`, `integration_test` indirect at v0.12.0).

Functional delta v0.9.0→v0.12.0: the root `ServiceConfig` seam setup consumes is unchanged (Version/Hooks/DrainDelay predate v0.9.0); the wave is satellite + operational — the new `preset/` module (one-call env-switch boot), `systemd/` godoc examples + quiet-run gates, `health/` v0.3.1-v0.3.2, hook errors keep their error-family classification, plus toolchain floor go 1.27.3, nightly govulncheck, and release-scripting. Mapping to cqrs-htmx needs: `realtime/` (SSE hub) would duplicate setup's own broadcaster/SSE mount; `errorpages/` duplicates httperr + dashboard error UX; `preset/` env-driven boot conflicts with setup's explicit `Config` model — none qualify under the library principle (opt-in only, setup optional). The sweep was mechanical; the bench seam is unaffected.

## Should cqrs-htmx use it (more)?

No expansion. The prior report's one open idea (hook forwarding) has since landed; the seam is now complete: thin, SSE-safe, drain-aware, hook-forwarding, measured. Stay on the root module tag; the submodule trains stay unconsumed by design.

## How to verify

```bash
rg -oN 'appkit\.[A-Z][A-Za-z0-9]*' setup -g '*.go' | sed 's/.*://' | sort | uniq -c | sort -rn
sed -n '129,151p' setup/run_appkit.go   # NoTimeout + ReadyCheck + Metrics/Version/hooks forwarding
rg -n 'go-appkit/' --type go .           # 0 hits — no submodule consumption
git -C ~/projects/go-appkit ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- Bench evidence: `setup/run_appkit_test.go` (`BenchmarkSpikeBaselineVsAppkit`); gotcha 46 (SSE vs WriteTimeout).
- Hook contract: `setup/run_appkit_hooks_test.go` (A22/B93); drain posture decision: `docs/decisions/2026-10-31_round45-m01-decision-batch.md` (T13).
- 2026-11-05 audit (score 80, pin v0.9.0): superseded by this pass — hook-forwarding gap closed in-repo, pin swept 2026-11-07.
