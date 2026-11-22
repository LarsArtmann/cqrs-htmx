# go-appkit Utilization Report

> Audit date: 2026-11-22 (refreshes the 2026-11-14 audit) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-appkit` v0.12.0 (root module; 12 submodule tag trains upstream — none consumed) · Checkout: `/home/lars/projects/go-appkit`

## Verdict at a glance

| Pin                                                   | Latest (verified 2026-11-22)          | Prod import sites                                  | Test import sites                       | Adoption                                    | Score                                      |
| ----------------------------------------------------- | ------------------------------------- | -------------------------------------------------- | --------------------------------------- | ------------------------------------------- | ------------------------------------------ |
| v0.12.0 (setup direct; examples+integration indirect) | **v0.13.0** — held, receipted (below) | 2 files (`setup/run_appkit.go`, `setup/config.go`) | 3 files (bench + hooks + metrics tests) | **Used — the `RunWithAppkit` runtime seam** | **100/100 · 1 behind (deliberate, receipted)** |

## What the library is

Production-ready HTTP service lifecycle: composes httputil (server, middleware), charmbracelet/log, and go-error-family into `Service`/`ServiceConfig` — start hooks, drain-aware shutdown, log levels/formats, metrics + version wiring, readiness probe, signal handling. A consumer-app convenience, which cqrs-htmx's `setup` module re-exposes as `RunWithAppkit`. Upstream has since grown 12 submodule trains (`config/`, `cqrs/`, `errorpages/`, `flightrecorder/`, `flightrecorderhealth/`, `health/`, `httpx/`, `otel/`, `preset/`, `realtime/`, `security/`, `systemd/`) — cqrs-htmx consumes **none** (see § Version currency). v0.13.0 added two more root seams — `ServiceConfig.Listener` (listener injection) and `ServiceConfig.LogHandler` (any `slog.Handler`) — both deferred by the hold, both N/A rows below.

## How cqrs-htmx uses it today

`setup/run_appkit.go:129-152` — `appkitServiceConfig` forwards the full consumer surface: `ReadTimeout`/`WriteTimeout: appkit.NoTimeout` (:137, :139 — SSE-safe posture, gotcha 46) with the `ReadHeaderTimeout`/`IdleTimeout` reaping pair kept, `ShutdownTimeout` (:141) + `DrainDelay` (default 2s, const `appkitDefaultDrainDelay` at :121 — the drain phase `Run` deliberately lacks), `ReadyCheck: b.bundleReadyCheck()` (:143), `Metrics` (:145), `Version` (:146), and the three lifecycle-hook seams `StartHooks`/`DrainHooks`/`ShutdownHooks` ← `Config.Appkit*Hooks` (:147-149; fields at `setup/config.go:358-378`, `[]appkit.Hook`). `Metrics *appkit.MetricsConfig` (:348) and `Version` (:350-356) are declared consumer-facing in `setup/config.go`. Hook ordering (start → drain → shutdown, shutdown before bundle `Close` tears down the SSE hub) is contract-pinned by `setup/run_appkit_hooks_test.go` (A22/B93). The canonical bench `BenchmarkSpikeBaselineVsAppkit` (`setup/run_appkit_test.go:369`, `appkit.LogLevelError` at :408) remains the AGENTS.md:121 pattern. `RunWithAppkit` is the documented production drain path (round45 T13, closed won't-fix 2026-10-31); plain `Run`/`RunHandler` keep their own SSE-safe no-timeout posture (`setup/run.go:38-40`). All 2026-11-14 claims re-verified against the 2026-11-22 tree — the seam is byte-equivalent; only the enclosing function moved from :134 to :129.

## Capability inventory

| #   | Capability                                                       | Status             | Evidence / reason                                                                                                                                                                                                                                                |
| --- | ---------------------------------------------------------------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Service composition (`NewService`, `Service`, `ServiceConfig`)   | 🟢 Fully leveraged | the whole point of `RunWithAppkit`                                                                                                                                                                                                                               |
| 2   | Timeout policy                                                   | 🟢 Fully leveraged | `NoTimeout` ×2 — deliberately SSE-safe, documented in the file header (lines 20-22)                                                                                                                                                                              |
| 3   | Log level control                                                | 🟢 Fully leveraged | config forwarding + `LogLevelError` bench suppression                                                                                                                                                                                                           |
| 4   | Metrics forwarding (`MetricsConfig`)                             | 🟢 Fully leveraged | `ServiceConfig.Metrics` ← `setup.Config.Metrics` (indirectly adopts httputil metrics — see the [httputil report](httputil.md))                                                                                                                                    |
| 5   | Lifecycle hooks (`Hook`, drain/shutdown hooks, `ShutdownConfig`) | 🟢 Fully leveraged | `AppkitStartHooks`/`AppkitDrainHooks`/`AppkitShutdownHooks` forwarded (run_appkit.go:147-149), order + before-Close contract tested (A22/B93) — re-verified this pass                                                                                             |
| 6   | Logger init (`InitLogger`, `LogFormat`, `LoggerConfig`)          | ⚪ N/A             | owned by `NewService` defaults; consumers who want charmbracelet config import appkit directly                                                                                                                                                                   |
| 7   | Health/readiness (`ReadyCheck`, `RegisterHealth`)                | 🟢 Fully leveraged | appkit's `ReadyCheck` runs the full bundle readiness set (projection drain gate, shared-hub check, `Config.HealthChecks`); `RegisterHealth: nil` keeps appkit's default `/health*` — one composition, two surfaces — re-verified this pass                        |
| 8   | Signal handling (`WaitForSignal`)                                | ⚪ N/A             | exercised inside the service; setup drives `ctx.Done()` itself                                                                                                                                                                                                   |
| 9   | Error mapping (`HTTPStatus`, `RegisterClassifications`)          | ⚪ N/A             | `httperr` owns mapping; errorfamily wraps the seam's own failures                                                                                                                                                                                                |
| 10  | Version/build info (`ServiceConfig.Version`, `/version`)         | 🟢 Fully leveraged | `Config.Version` → `GET /version` + `appkit_build_info` label (config.go:350-356, run_appkit.go:146)                                                                                                                                                             |
| 11  | Listener injection (`ServiceConfig.Listener`, v0.13.0)           | ⚪ N/A             | deferred by the receipted toolchain-floor hold (TODO row 1); no unix-socket/systemd-socket-activation surface exists in cqrs-htmx today — the seam was demanded by OTHER fleet consumers (go-compliance, nsfw-classifier reviews)                                  |
| 12  | Log-handler seam (`ServiceConfig.LogHandler`, v0.13.0)           | ⚪ N/A             | same hold; setup's charmbracelet forwarding (row 3) is unchanged and sufficient — no fleet log-lane decision has fired                                                                                                                                            |

## Score derivation

12 rows − 5 N/A (rows 6, 8, 9 — structurally owned elsewhere; rows 11, 12 — new-in-v0.13.0 seams deferred by the receipted hold) = **7 applicable**; 7 full ×1 = **7 → 100/100** (band: max at the pinned tag). Capability score unchanged from 2026-11-14 — the seam itself did not move; the version posture did, and it carries in the suffix: **1 behind, deliberate, receipted** (not a miss — see Version currency).

## Version currency

**v0.12.0 pinned; v0.13.0 = latest** (verified `git ls-remote --tags origin`, 2026-11-22; v0.13.0 tagged 2026-11-21, library HEAD `10a17ab`). This is the **only lag in the fleet** — treadmill worst-lag 1, release-train strict-lag shows exactly the 4 consumer entries (setup direct at `setup/go.mod:16`; `examples/setup-demo:49`, `examples/async-startup-demo:45`, `integration_test:95` indirect) — and it is a **receipted deliberate hold, not a miss**: v0.13.0's go.mod declares **go 1.27.4** while the fleet toolchain floor is 1.27.3 (ADR-0003 nixpkgs rev; nixpkgs-unstable HEAD `680275f0`, probed 2026-11-21 ~17:20, still ships 1.27.3 — the one-train recipe's first step has no target). The decision, including the **survived premature sweep** (a 2026-11-21 17:05 pass re-moved appkit to v0.13.0 + the four directives to 1.27.4 against the flake's 1.27.3 toolchain and was re-healed same evening), is documented in full in TODO_LIST row 1 (the fleet-floor row) with both dated addenda; the fleet floor and the appkit train are the SAME decision. When nixpkgs ships 1.27.4, ONE train does all of it: flake rev, go.work + four go.mods to 1.27.4, appkit to v0.13.0, CI pins — and strict-lag returns to 0.

**What the hold defers** (CHANGELOG [0.13.0]): the **listener-injection seam** (`ServiceConfig.Listener net.Listener` — serve a pre-bound listener: unix-socket IPC, systemd socket activation; ownership transfers at `Start`) and the **log-handler seam** (`ServiceConfig.LogHandler slog.Handler` — every line through any handler; `LogLevel`/`LogFormat` ignored in that mode), plus README adoption surfaces. Neither changes the v0.12.0 surface setup consumes — rows 11-12 are the entire exposure. Tag-scoped nuance: the 1.27.4 directive is a property of the v0.13.0 TAG — library master (10 commits past the tag) already relaxed it to `go 1.27` (`git show v0.13.0:go.mod` vs HEAD), so the next upstream release may unblock the sweep without the fleet floor moving; until then the hold stands.

**No submodule consumed**: `rg 'go-appkit/' --type go` → 0 hits repo-wide. Functional posture v0.12.0 (unchanged from the 2026-11-14 report): the root `ServiceConfig` seam setup consumes is stable; the satellite modules (realtime, errorpages, preset, …) stay unconsumed by design — none qualify under the library principle (opt-in only, setup optional).

## Should cqrs-htmx use it (more)?

No change to the seam: thin, SSE-safe, drain-aware, hook-forwarding, measured — at max at the pinned tag. When the floor moves (TODO row 1's one-train recipe), the v0.13.0 sweep is mechanical; the Listener seam is the first deferred capability with a plausible consumer need here (systemd socket activation), the LogHandler seam stays optional until a fleet log-lane decision fires.

## How to verify

```bash
rg -oN 'appkit\.[A-Z][A-Za-z0-9]*' setup -g '*.go' | sed 's/.*://' | sort | uniq -c | sort -rn
# 8 appkit.Hook · 5 appkit.LogLevel · 3 each ServiceConfig/NoTimeout/MetricsConfig/LogLevelError · 1 each Service/NewService
sed -n '129,152p' setup/run_appkit.go   # NoTimeout ×2 (:137,139) + ReadyCheck (:143) + Metrics/Version (:145,146) + hooks (:147-149)
rg -n 'go-appkit/' --type go .           # 0 hits — no submodule consumption
rg -n 'go-appkit v0\.' setup/go.mod examples/setup-demo/go.mod examples/async-startup-demo/go.mod integration_test/go.mod   # v0.12.0 ×4 (setup direct, rest indirect)
git -C ~/projects/go-appkit ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1   # <sha>	refs/tags/v0.13.0 (held — see TODO row 1)
git -C ~/projects/go-appkit show v0.13.0:go.mod | rg '^go '   # go 1.27.4 — the hold's reason (library master already relaxed to go 1.27)
```

## Prior reports

- 2026-11-14 (prior state of this file): 100/100 at v0.12.0 = latest; this pass records the receipted v0.13.0 hold (toolchain floor, TODO row 1) and adds 2 N/A rows for the deferred seams — capability score unchanged.
- Bench evidence: `setup/run_appkit_test.go` (`BenchmarkSpikeBaselineVsAppkit`); gotcha 46 (SSE vs WriteTimeout).
- Hook contract: `setup/run_appkit_hooks_test.go` (A22/B93); drain posture decision: `docs/decisions/2026-10-31_round45-m01-decision-batch.md` (T13).
- 2026-11-05 audit (score 80, pin v0.9.0): superseded 2026-11-14 — hook-forwarding gap closed in-repo, pin swept 2026-11-07.
