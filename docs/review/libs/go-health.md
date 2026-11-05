# go-health Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-health` v0.10.0 (alpha-stable) · Checkout: `/home/lars/projects/go-health`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Test import sites | Adoption | Score |
| --- | --- | --- | --- | --- | --- |
| v0.10.0 (health bridge direct) | v0.10.0 (= latest) | 4 files (`health/`, examples, integration_test) | 4 files | **Used — via the `health` bridge module** | **65/100 count · 9/10 bridge quality (round-2 review)** |

## What the library is

Kubernetes three-probe SDK (liveness/readiness/startup) for `samber/do` v2 containers: one `Probe` type with critical/non-critical classification, background caching, shutdown awareness, recorder merging, detailed checks (`duration_ns`), batteries of common checks, metrics, middleware, federation, and K8s wiring.

## How cqrs-htmx uses it today

The `cqrs-htmx/health` bridge consumes (symbol census): `Probe` ×10, `WithHealthRecorder` ×8, `New` ×7, `CheckDetail` ×6, `WithCriticalServices` ×5, `DefaultRoutes` ×5, `StatusPass/Warn/Fail`, `HealthRecorder`, `WithVersion` ×3, `DetailedHealthRecorder`, `NewWithDetailedCheck` ×2, `ChainRecorders` (the deprecated `RecorderChain` alias forwards here), `WithRefreshInterval`, `Option`. `health.NewProbe(svc, ...)` maps projection health → probe checks; `NewDashboard` (see the [dashboard report](go-health-dashboard.md)) mounts the human view. The bridge's own README carries the "which surface for which consumer" map. Note: `gohealth.WithHealthRecorder` passed to `NewProbe` is refused at construction (recorders belong to `New`, not the projection probe) — a documented sharp edge.

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | Probe construction & options (`New`, `WithCriticalServices`, `WithVersion`, `WithRefreshInterval`) | 🟢 Fully leveraged | bridge + examples + integration tests |
| 2 | Detailed checks (`NewWithDetailedCheck`, `CheckDetail`, `duration_ns` on the wire) | 🟢 Fully leveraged | per-projection checks with timing |
| 3 | Recorder composition (`WithHealthRecorder`, `ChainRecorders`, `DetailedHealthRecorder`) | 🟢 Fully leveraged | merge with samber/do injector checks (auditlog "one probe" pattern) |
| 4 | Route wiring (`DefaultRoutes`, `RegisterRoutes`) | 🟢 Fully leveraged | kubelet JSON probes |
| 5 | Status vocabulary (`StatusPass/Warn/Fail`) | 🟢 Fully leveraged | projection states map through it |
| 6 | Shutdown awareness (probe lifecycle with service Close) | 🟢 Fully leveraged | bridge stops with the service |
| 7 | Batteries (`NewChecks` common checks) | 🔴 Missed (low) | all bridge checks are projection-specific; batteries (HTTP-ping etc.) could serve consumer dependency checks — thin fit |
| 8 | Metrics integration | 🟡 Partially used | go-health metrics section unexplored; observability demos use go-cqrs-lite middleware metrics instead |
| 9 | Probe tuning depth (`WithTimeout`, `WithLiveThrottle`, `WithShutdownGracePeriod`, `WithEvaluationHook`) | 🔴 Missed (low) | defaults proven adequate; hooks could power audit trails |
| 10 | `VersionHandler` | 🔴 Missed (low) | `/version` endpoint trivially derivable from `WithVersion` metadata |
| 11 | `SanitizeResponse` | ⚪ N/A | responses are constructed by our recorders, not reflected |
| 12 | Federation (remote probes hub) | ⚪ N/A | that is the standalone `health-hub` deployment pattern, not a library bridge concern |

## Score derivation

12 rows − 2 N/A = **10 applicable**; 6 full ×1 + 1 partial ×0.5 = **6.5 → 65/100** (band: moderate by count). The bridge itself was independently reviewed at **9/10** (round-2 deep review, 2026-10-30, DO-1→DO-6 clean) — the unexplored remainder is option depth, not correctness.

## Version currency

v0.10.0 pinned = v0.10.0 latest (verified `git ls-remote --tags origin`, 2026-11-05). Alpha-stable contract; three-probe surface declared stable upstream.

## Should cqrs-htmx use it (more)?

Only opportunistically: `WithEvaluationHook` (audit integration) and the metrics section are the two with real consumer value; both belong in a future bridge round, not urgently.

## How to verify

```bash
rg -oN 'gohealth\.[A-Z][A-Za-z0-9]*' health examples -g '*.go' | sed 's/.*://' | sort | uniq -c | sort -rn
rg -n 'WithHealthRecorder' health/README.md | head -2   # refused-at-New note
git -C ~/projects/go-health ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- [2026-10-08](../../research/2026-10-08_samber-do-v2-health-checks-deep-review.md) and [2026-10-30 round 2](../../research/2026-10-30_samber-do-v2-health-checks-deep-review-round-2.md) deep reviews; M01 decision batch [2026-10-31](../../decisions/2026-10-31_round45-m01-decision-batch.md).
