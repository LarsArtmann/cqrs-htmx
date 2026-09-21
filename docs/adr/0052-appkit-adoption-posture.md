# ADR 0052: go-appkit adoption posture — opt-in `RunWithAppkit`, default flip deferred to v5

**Date:** 2026-09-22 (verdict consolidated; spike 2026-08-16, follow-up assessment 2026-09-20)
**Status:** Accepted
**Related:** comparison report `/home/lars/projects/docs/review/2026-08-16_setup-vs-go-appkit-comparison.md` §4/§6/§8, adopt report `docs/status/archived/2026-08-16_01-38_appkit-spike-adopt.md`, `setup/run_appkit.go`

> Historical note: earlier documents referred to this decision as "ADR-001
> (appkit)" — a label that predates this file and never resolved to an ADR in
> either repo. This ADR is the durable record; those references now point here.

## Context

`setup.Bundle` has two serve paths: `Run`/`RunHandler` (stdlib `httputil.NewServer`,
SSE-safe timeouts, quiet logging) and the opt-in `RunWithAppkit` (go-appkit v0.5.x:
metrics/version endpoints, shutdown-phase logging, `testkit.Serve`). A 2026-08-16
spike adopted Option A — keep both paths, appkit opt-in — and a 2026-09-20
assessment examined the default-flip follow-ups.

## Decision

**Option A stands: appkit stays opt-in via `RunWithAppkit`. The default flip
(folding appkit into `RunHandler`) is deferred to v5 at the earliest.** Per-item
verdicts from the assessment:

| Follow-up | Verdict | Rationale |
| --- | --- | --- |
| (b) Fold into `RunHandler` | **Defer to v5** | Silently changes the default path's behavior: ~2s drain delay, generic middleware wrapping, `/health/*` reshaped. Needs (c)+(d) settled first; stays behind the DataStar ADR-first sequencing. |
| (c) `/health` dedup + LB guidance | **Document, don't automate** | appkit serves `/health`, `/health/live`, `/health/ready` while the bundle's session-gated `HealthPath` mount coexists by design. `HealthPath: "-"` is the existing opt-out; `Mount()` cannot know which run path a consumer will call. |
| (d) Stacked-chain dedup | **Not safe blindly** | The bundle's inner `RecommendedSecurityMiddleware` owns CSP NONCE generation (appkit's SecurityHeaders do not). The layering (generic outer, domain-aware inner) is documented in `run_appkit.go`. |
| (e) Logging posture | **No action** | The 2.8× bench delta is isolated by `LogLevelError` in the bench; an appkit LogLevel config seam is marginal (`RunHandler` stays the quiet path). |
| (f) Expose `Addr()` | **Consumer-demand-gated** | Small (listener captured in appkit's Start) but racy to expose post-Start. Same gate as DataStar Tier 4. |

## Consequences

- `Run`/`RunHandler` remain the documented default for SSE-safe quiet serving;
  `RunWithAppkit` remains the escape hatch for consumers wanting appkit's
  metrics/version/observability surface (pinned at go-appkit v0.5.x, no replace).
- The flip may be revisited for v5 only once (c) and (d) have landed as
  documentation/behavior the default path can inherit without silent changes.
