# go-health-dashboard Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-health-dashboard` v0.13.2 · Checkout: `/home/lars/projects/go-health-dashboard`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Adoption | Score |
| --- | --- | --- | --- | --- |
| v0.13.2 (health bridge direct) | v0.13.2 (= latest) | 2 files (`health/dashboard.go`, examples) | **Used — one-call mount, by design** | **67/100 · high for the bridge's intent; one real gap (nonce/CSP)** |

## What the library is

Real-time health dashboard over go-health, DataStar-SSE powered: status banners, severity grouping, trend sparkline, stat cards, content-negotiated `/health` (browsers HTML / `Accept: application/json` JSON), ~30 `With*` options (rate limiting, metrics, nonces, webhooks, push modes, introspection, trend/timeline config). Kept in a separate module so go-health stays single-dependency.

## How cqrs-htmx uses it today

`health.NewDashboard(probe, healthdashboard.WithTitle("My App"))` — `New` ×1, `Dashboard` ×5 (type refs), `Option` ×1, `WithTitle` ×2. The bridge deliberately exposes exactly the one-call mount and hands anything deeper to direct consumers (bridge README's consumer map).

## Capability inventory (graded against what a bridge *should* surface)

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | One-call mount (`New`, `Dashboard`, routes) | 🟢 Fully leveraged | `health/dashboard.go` + README map |
| 2 | Branding (`WithTitle`) | 🟢 Fully leveraged | exposed through the bridge signature |
| 3 | CSP-safe serving (`WithNonce`) | 🔴 Missed | 0 uses — the dashboard's inline scripts under a nonce-CSP consumer need the nonce option threaded through the bridge; the repo's own CSP posture (nonce-first, `RecommendedSecurityMiddleware`) makes this the one genuine gap |
| 4 | Rate limiting (`WithRateLimit`) | ⚪ N/A | consumers wrap the mux; a library bridge should not double-rate-limit |
| 5 | Metrics (`WithMetrics`) | ⚪ N/A | probe-level metrics belong to the go-health layer |
| 6 | Push modes / trend / timeline / grouping config | ⚪ N/A | presentation depth beyond bridge intent — direct consumers import the module |
| 7 | Webhooks / introspection | ⚪ N/A | operational features of the standalone hub pattern |

## Score derivation

7 rows − 4 N/A = **3 applicable**; 2 full ×1 = **2 → 67/100**. The single miss is real but small: thread a nonce option through `health.NewDashboard`.

## Version currency

v0.13.2 pinned = v0.13.2 latest (verified `git ls-remote --tags origin`, 2026-11-05).

## Should cqrs-htmx use it (more)?

One TODO-grade item: accept and forward a nonce (or `WithNonceExtractor`) in `health.NewDashboard` so the dashboard renders correctly under the repo's own recommended nonce-CSP stack. Everything else is intentionally the consumer's dial.

## How to verify

```bash
cat health/dashboard.go
rg -oN 'healthdashboard\.[A-Z][A-Za-z0-9]*' health examples -g '*.go' | sed 's/.*://' | sort | uniq -c
git -C ~/projects/go-health-dashboard ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- Inherits the health round-2 review context ([2026-10-30](../../research/2026-10-30_samber-do-v2-health-checks-deep-review-round-2.md)).
