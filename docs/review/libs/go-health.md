# go-health Utilization Report

> Audit date: 2026-11-22 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-health` v0.10.0 (alpha-stable) · Checkout: `/home/lars/projects/go-health`

## Verdict at a glance

| Pin                            | Latest (verified 2026-11-22) | Prod import sites                               | Test import sites | Adoption                                  | Score                                                   |
| ------------------------------ | ---------------------------- | ----------------------------------------------- | ----------------- | ----------------------------------------- | ------------------------------------------------------- |
| v0.10.0 (health bridge direct) | v0.10.0 (= latest)           | 4 files (`health/`, examples, integration_test) | 5 files           | **Used — via the `health` bridge module** | **65/100 count · 9/10 bridge quality (round-2 review)** |

## What the library is

Kubernetes three-probe SDK (liveness/readiness/startup) for `samber/do` v2 containers: one `Probe` type with critical/non-critical classification, background caching, shutdown awareness, recorder merging, detailed checks (`duration_ns`), batteries of common checks, metrics, middleware, federation, and K8s wiring.

## How cqrs-htmx uses it today

The `cqrs-htmx/health` bridge consumes (symbol census, re-run 2026-11-22 — counts byte-identical to the 2026-11-14 census; zero bridge-code drift since): `Probe` ×10, `WithHealthRecorder` ×8, `New` ×7, `CheckDetail` ×6, `WithCriticalServices` ×5, `DefaultRoutes` ×5, `StatusPass`/`Warn`/`Fail` ×4/3/2, `HealthRecorder` ×4, `WithVersion` ×3, `DetailedHealthRecorder` ×3, `NewWithDetailedCheck` ×2, `ChainRecorders` ×2 (the deprecated `RecorderChain` alias at `health/probe.go:96-104` forwards here — kept per M01 verdict #2), `WithRefreshInterval`, `Option`, `NewChecks` + `CheckFunc` ×1 (test fixture only, `dashboard_test.go:23`). `health.NewProbe(svc, ...)` maps projection health → probe checks (`NewWithDetailedCheck`, `probe.go:54`, puts `duration_ns` on the wire); `health.Recorder` merges with samber/do injector checks and also implements `DetailedHealthRecorder` (`probe.go:72,135`); `NewDashboard` (see the [dashboard report](go-health-dashboard.md)) now also forwards the CSP-nonce options. The bridge's own README carries the "which surface for which consumer" map plus the nonce-CSP pairing section (`health/README.md:75-88`) and the refused-recorder note (`:24`). Sharp edge: `gohealth.WithHealthRecorder` passed to `NewProbe` is refused at construction (recorders belong to `New`). Adjacent, not go-health: the root module's built-in `/health` gate under `AsyncStartup` (`setup/mount.go:142` appends `cqrshtmx.ProjectionReadinessCheck`; 503-while-draining per `setup/config.go:511-513`, AGENTS.md:70) is the zero-dependency alternative surface the README map distinguishes.

Overlap note: go-appkit now ships its own `health/` submodule (latest still `health/v0.3.2`, verified 2026-11-22) — zero consumption anywhere in this repo; `setup/config.go:219-224` keeps the stacks from colliding (`HealthPath: "-"` opts out of the built-in gate when a consumer's own stack — the go-health bridge, or appkit's `/health/*` via `RunWithAppkit` — owns probing).

## Capability inventory

| #   | Capability                                                                                              | Status             | Evidence / reason                                                                                                                     |
| --- | ------------------------------------------------------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Probe construction & options (`New`, `WithCriticalServices`, `WithVersion`, `WithRefreshInterval`)      | 🟢 Fully leveraged | bridge + examples + integration tests                                                                                                 |
| 2   | Detailed checks (`NewWithDetailedCheck`, `CheckDetail`, `duration_ns` on the wire)                      | 🟢 Fully leveraged | per-projection checks with timing                                                                                                     |
| 3   | Recorder composition (`WithHealthRecorder`, `ChainRecorders`, `DetailedHealthRecorder`)                 | 🟢 Fully leveraged | merge with samber/do injector checks (auditlog "one probe" pattern)                                                                   |
| 4   | Route wiring (`DefaultRoutes`, `RegisterRoutes`)                                                        | 🟢 Fully leveraged | kubelet JSON probes                                                                                                                   |
| 5   | Status vocabulary (`StatusPass/Warn/Fail`)                                                              | 🟢 Fully leveraged | projection states map through it                                                                                                      |
| 6   | Shutdown awareness (probe lifecycle with service Close)                                                 | 🟢 Fully leveraged | bridge stops with the service                                                                                                         |
| 7   | Batteries (`NewChecks` common checks)                                                                   | 🔴 Missed (low)    | the only use is a test fixture (`dashboard_test.go:23`); batteries (HTTP-ping etc.) could serve consumer dependency checks — thin fit |
| 8   | Metrics integration                                                                                     | 🟡 Partially used  | go-health metrics section unexplored; observability demos use go-cqrs-lite middleware metrics instead                                 |
| 9   | Probe tuning depth (`WithTimeout`, `WithLiveThrottle`, `WithShutdownGracePeriod`, `WithEvaluationHook`) | 🔴 Missed (low)    | defaults proven adequate; hooks could power audit trails                                                                              |
| 10  | `VersionHandler`                                                                                        | 🔴 Missed (low)    | `/version` endpoint trivially derivable from `WithVersion` metadata                                                                   |
| 11  | `SanitizeResponse`                                                                                      | ⚪ N/A             | responses are constructed by our recorders, not reflected                                                                             |
| 12  | Federation (remote probes hub)                                                                          | ⚪ N/A             | that is the standalone `health-hub` deployment pattern, not a library bridge concern                                                  |

## Score derivation

12 rows − 2 N/A = **10 applicable**; 6 full ×1 + 1 partial ×0.5 = **6.5 → 65/100** (band: moderate by count). The bridge itself was independently reviewed at **9/10** (round-2 deep review, 2026-10-30, verdict line 33: "9/10, up from 8/10"; DO-1→DO-6 table clean) and tests at **100.0%** of statements (`go test -cover` in `health/`, re-run 2026-11-22; gate threshold 90 — `scripts/checks/coverage-thresholds.tsv:22`). The unexplored remainder is option depth, not correctness.

## Version currency

v0.10.0 pinned = v0.10.0 latest (verified `git ls-remote --tags origin`, 2026-11-22; still no upstream tag movement since the 2026-11-05 audit — v0.10.0 was cut 2026-11-04). Library HEAD `3ca6644` (2026-11-20) is docs + auto-commit churn only, and the CHANGELOG `[Unreleased]` section is empty ("Nothing yet" ×4) — no new upstream capability to inventory. Alpha-stable contract; three-probe surface declared stable upstream.

## Should cqrs-htmx use it (more)?

Only opportunistically: `WithEvaluationHook` (audit integration) and the metrics section are the two with real consumer value; both belong in a future bridge round, not urgently.

## How to verify

```bash
# symbol census over the bridge + examples (top rows: WithHealthRecorder 8,
# New 7, CheckDetail 6, Probe 10 — matches the census paragraph above)
rg -oN 'gohealth\.[A-Z][A-Za-z0-9]*' health examples -g '*.go' | sed 's/.*://' | sort | uniq -c | sort -rn
rg -n 'refused' health/README.md | head -1                    # :24 — WithHealthRecorder refused-at-NewProbe note
(cd health && go test -cover ./...)                       # 100.0% (threshold 90, coverage-thresholds.tsv:22)
git -C ~/projects/go-health ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1   # v0.10.0
rg -n 'AsyncStartup' AGENTS.md | head -1                 # :70 — the /health-under-AsyncStartup line
rg -n 'health' scripts/checks/coverage-thresholds.tsv    # :22 — health 90
```

## Prior reports

- 2026-11-14 (prior state of this file): 65/100 · no capability-row flips — census re-run identical, upstream `[Unreleased]` empty at HEAD `3ca6644` (2026-11-20, docs-only); receipt drift only (AGENTS.md :88→:70, coverage-gate row :21→:22); a 2026-11-17 interim auto-commit realigned table widths without content change.
- [2026-10-08](../../research/2026-10-08_samber-do-v2-health-checks-deep-review.md) and [2026-10-30 round 2](../../research/2026-10-30_samber-do-v2-health-checks-deep-review-round-2.md) deep reviews; M01 decision batch [2026-10-31](../../decisions/2026-10-31_round45-m01-decision-batch.md) (verdict #2: keep the deprecated `RecorderChain` alias, migrate docs to `gohealth.ChainRecorders`).
