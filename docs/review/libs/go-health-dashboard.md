# go-health-dashboard Utilization Report

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-health-dashboard` v0.13.3 · Checkout: `/home/lars/projects/go-health-dashboard`

## Verdict at a glance

| Pin                            | Latest (verified 2026-11-14) | Prod import sites                                                  | Adoption                             | Score                                         |
| ------------------------------ | ---------------------------- | ------------------------------------------------------------------ | ------------------------------------ | --------------------------------------------- |
| v0.13.3 (health bridge direct) | v0.13.3 (= latest)           | 3 files (`health/dashboard.go`, `e2e/server/bridges.go`, examples) | **Used — one-call mount, by design** | **100/100 · nonce/CSP gap closed 2026-11-06** |

## What the library is

Real-time health dashboard over go-health, DataStar-SSE powered: status banners, severity grouping, trend sparkline, stat cards, content-negotiated `/health` (browsers HTML / `Accept: application/json` JSON), ~30 `With*` options (rate limiting, metrics, nonces, webhooks, push modes, introspection, trend/timeline config). Kept in a separate module so go-health stays single-dependency.

## How cqrs-htmx uses it today

`health.NewDashboard(probe, opts...)` forwards options verbatim (doc names `WithTitle`, `WithBasePath`, `WithNonceExtractor` — `health/dashboard.go:11-17`); census: `Dashboard` ×5, `New` ×1, `Option` ×1, `WithTitle` ×2, `WithNonce` ×1, `WithNonceExtractor` ×1, `RecommendedCSP` ×1 (the last three from the nonce test). Beyond the bridge: `e2e/server/bridges.go:13,60` mounts the full dashboard in the e2e server, and `examples/samber-do-demo/container.go` wires it in the demo. The bridge README documents the nonce-CSP pairing (`health/README.md:75-88`): `WithNonceExtractor(httputil.NonceFromRequest)` under `cqrshtmx.RecommendedSecurityMiddleware()`, header via `healthdashboard.RecommendedCSP(nonce)`.

## Capability inventory (graded against what a bridge _should_ surface)

| #   | Capability                                          | Status                                    | Evidence / reason                                                                                                                                                                                                            |
| --- | --------------------------------------------------- | ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | One-call mount (`New`, `Dashboard`, routes)         | 🟢 Fully leveraged                        | `health/dashboard.go` + README map + e2e mount (`e2e/server/bridges.go:60`)                                                                                                                                                  |
| 2   | Branding (`WithTitle`)                              | 🟢 Fully leveraged                        | exposed through the bridge signature                                                                                                                                                                                         |
| 3   | CSP-safe serving (`WithNonce`/`WithNonceExtractor`) | 🟢 Fully leveraged (was 🔴 at 2026-11-05) | bridge passes options through (`dashboard.go:11`); `TestNewDashboard_ForwardsCSPNonceOptions` pins fixed-nonce (`:28-31`), per-request extraction (`:50-53`), and the `RecommendedCSP` header pairing (`:44`); README :75-88 |
| 4   | Rate limiting (`WithRateLimit`)                     | ⚪ N/A                                    | consumers wrap the mux; a library bridge should not double-rate-limit                                                                                                                                                        |
| 5   | Metrics (`WithMetrics`)                             | ⚪ N/A                                    | probe-level metrics belong to the go-health layer                                                                                                                                                                            |
| 6   | Push modes / trend / timeline / grouping config     | ⚪ N/A                                    | presentation depth beyond bridge intent — direct consumers import the module                                                                                                                                                 |
| 7   | Webhooks / introspection                            | ⚪ N/A                                    | operational features of the standalone hub pattern                                                                                                                                                                           |

## Score derivation

7 rows − 4 N/A = **3 applicable**; 3 full ×1 = **3 → 100/100**. The 2026-11-05 score of 67 (2/3) rested entirely on the nonce miss; the 2026-11-06 session closed it with the pass-through bridge, the pinned test, and the README pairing section. 100 is honest because the bridge's declared surface is deliberately this small — one-call mount, branding, CSP-safe serving — and every option class a bridge should forward is forwarded.

## Version currency

v0.13.3 pinned (`health/go.mod:9`, `e2e/server/go.mod:23`) = v0.13.3 latest, cut 2026-11-12 (verified `git ls-remote --tags origin`, 2026-11-14). Delta `v0.13.2..v0.13.3` (~42 commits): almost all chore auto-commits and docs/planning; the consumer-relevant substance is `e3a0e9d` (fix: connection pill visibility and **root data-nonce stamping**), plus `d15fcb6` (NixOS module for the health-hub federation dashboard — not consumed here). Nonce-fix dependency finding: `WithNonce`/`WithNonceExtractor` are **not new in v0.13.3** — they ship in `v0.13.2:options.go:375-390` (introduced `a22ef06`, 2026-08-09), so the bridge fix works against v0.13.2 and was never sweep-dependent; the bump merely also delivers the nonce-stamping rendering fix.

## Should cqrs-htmx use it (more)?

No — the audit's one TODO-grade item (thread a nonce through `health.NewDashboard`) is done. Everything else is intentionally the consumer's dial.

## How to verify

```bash
cat health/dashboard.go
rg -oN 'healthdashboard\.[A-Z][A-Za-z0-9]*' health examples -g '*.go' | sed 's/.*://' | sort | uniq -c
git -C ~/projects/go-health-dashboard log --oneline v0.13.2..v0.13.3 | grep -v 'auto-commit' | grep -v chore
git -C ~/projects/go-health-dashboard ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- Inherits the health round-2 review context ([2026-10-30](../../research/2026-10-30_samber-do-v2-health-checks-deep-review-round-2.md)).
