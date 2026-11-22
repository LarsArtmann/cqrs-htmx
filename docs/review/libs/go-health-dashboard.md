# go-health-dashboard Utilization Report

> Audit date: 2026-11-22 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-health-dashboard` v0.13.3 · Checkout: `/home/lars/projects/go-health-dashboard`

## Verdict at a glance

| Pin                            | Latest (verified 2026-11-22) | Prod import sites                                                  | Adoption                             | Score                                         |
| ------------------------------ | ---------------------------- | ------------------------------------------------------------------ | ------------------------------------ | --------------------------------------------- |
| v0.13.3 (health bridge direct) | v0.13.3 (= latest)           | 3 files (`health/dashboard.go`, `e2e/server/bridges.go`, examples) | **Used — one-call mount, by design** | **100/100 · nonce/CSP gap closed 2026-11-06** |

## What the library is

Real-time health dashboard over go-health, DataStar-SSE powered: status banners, severity grouping, trend sparkline, stat cards, content-negotiated `/health` (browsers HTML / `Accept: application/json` JSON), ~30 `With*` options (rate limiting, metrics, nonces, webhooks, push modes, introspection, trend/timeline config). Kept in a separate module so go-health stays single-dependency.

## How cqrs-htmx uses it today

`health.NewDashboard(probe, opts...)` forwards options verbatim (doc names `WithTitle`, `WithBasePath`, `WithNonceExtractor` — `health/dashboard.go:11-17`); census (re-run 2026-11-22, identical to 2026-11-14): `Dashboard` ×5, `New` ×1, `Option` ×1, `WithTitle` ×2, `WithNonce` ×1, `WithNonceExtractor` ×1, `RecommendedCSP` ×1 (the last three from the nonce test). Beyond the bridge: `e2e/server/bridges.go:13,68` mounts the full dashboard in the e2e server — the fullest option consumer in the repo: `WithRoutes(Routes{…})` pins the whole `/bridges/health/*` route set (`:78-91`), `WithEmbeddedDatastarSDK()` swaps the CDN script tag for a same-origin one so CI stays hermetic (`:92`), then `RegisterRoutes` (`:94`) + `Start` (`:100`, the SSE pusher without which the page stays on "Reconnecting") — and `examples/samber-do-demo/container.go:242-247` wires it in the demo. The bridge README documents the nonce-CSP pairing (`health/README.md:75-88`): `WithNonceExtractor(httputil.NonceFromRequest)` under `cqrshtmx.RecommendedSecurityMiddleware()`, header via `healthdashboard.RecommendedCSP(nonce)`.

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
| 8   | Auth & ops wave (`WithOIDC`, `WithPerIPRateLimit`, `WithRequestLogging`, SSE lifecycle metrics — all `[Unreleased]` upstream) | ⚪ N/A | hub/operational posture beyond bridge intent — auth and limits belong to the consumer's middleware; the pass-through bridge signature delivers all of them free at the next pin bump, zero bridge changes |

## Score derivation

8 rows − 5 N/A = **3 applicable**; 3 full ×1 = **3 → 100/100**. The 2026-11-05 score of 67 (2/3) rested entirely on the nonce miss; the 2026-11-06 session closed it with the pass-through bridge, the pinned test, and the README pairing section. 100 is honest because the bridge's declared surface is deliberately this small — one-call mount, branding, CSP-safe serving — and every option class a bridge should forward is forwarded. Row 8 (added 2026-11-22 for the upstream `[Unreleased]` ops/OIDC wave) is N/A-class like rows 4-7, so the derivation is unchanged.

## Version currency

v0.13.3 pinned (`health/go.mod:9`, `e2e/server/go.mod:23`, `examples/samber-do-demo/go.mod:16`) = v0.13.3 latest (verified `git ls-remote --tags origin`, 2026-11-22). Delta `v0.13.2..v0.13.3` (~42 commits): almost all chore auto-commits and docs/planning; the consumer-relevant substance is `e3a0e9d` (fix: connection pill visibility and **root data-nonce stamping**), plus `d15fcb6` (NixOS module for the health-hub federation dashboard — not consumed here). Nonce-fix dependency finding: `WithNonce`/`WithNonceExtractor` are **not new in v0.13.3** — they ship in `v0.13.2:options.go:375-390` (introduced `a22ef06`, 2026-08-09), so the bridge fix works against v0.13.2 and was never sweep-dependent; the bump merely also delivers the nonce-stamping rendering fix.

Upstream drift since the tag (HEAD `19af9cf`, 2026-11-20): the CHANGELOG `[Unreleased]` section carries a substantial ops wave — **native OIDC login** (`dashboard.WithOIDC`, PKCE + JWKS + group gate + HMAC session cookie, `HEALTH_HUB_OIDC_*` env, NixOS submodule; CHANGELOG.md:20), SSE connection lifecycle metrics and a pusher watchdog gauge, rate-limit observability + `WithPerIPRateLimit` (:45), and `WithRequestLogging` (:48), plus a NixOS OIDC secret-reading fix. None of it is consumable at the v0.13.3 pin and none belongs in the bridge (row 8); when a future sweep bumps the pin, the pass-through `NewDashboard` forwards every new option with zero bridge edits.

## Should cqrs-htmx use it (more)?

No — the audit's one TODO-grade item (thread a nonce through `health.NewDashboard`) is done. Everything else is intentionally the consumer's dial.

## How to verify

```bash
cat health/dashboard.go                                     # NewDashboard forwards opts verbatim (:16-17)
# census over bridge + examples: Dashboard 5, WithTitle 2, WithNonce 1,
# WithNonceExtractor 1, RecommendedCSP 1, New 1, Option 1
rg -oN 'healthdashboard\.[A-Z][A-Za-z0-9]*' health examples -g '*.go' | sed 's/.*://' | sort | uniq -c
rg -n 'WithRoutes|WithEmbeddedDatastarSDK' e2e/server/bridges.go   # :78, :92 — fullest option consumer
# substance of the pinned bump: e3a0e9d (nonce stamping), d15fcb6 (NixOS)
git -C ~/projects/go-health-dashboard log --oneline v0.13.2..v0.13.3 | grep -v 'auto-commit' | grep -v chore
git -C ~/projects/go-health-dashboard ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1   # v0.13.3
# upstream [Unreleased] ops wave: WithOIDC :20, WithPerIPRateLimit :45, WithRequestLogging :48
rg -n 'WithOIDC|WithPerIPRateLimit|WithRequestLogging' ~/projects/go-health-dashboard/CHANGELOG.md | head -3
```

## Prior reports

- 2026-11-14 (prior state of this file): 100/100 · no applicable-row flips — added N/A row 8 for the upstream `[Unreleased]` ops/OIDC wave (`WithOIDC`, per-IP rate limit, request logging, SSE metrics at HEAD `19af9cf`, 2026-11-20); e2e mount receipts refreshed (`WithRoutes` + `WithEmbeddedDatastarSDK` were already in the 2026-11-14 tree, under-cited); a 2026-11-17 interim auto-commit realigned table widths without content change.
- Inherits the health round-2 review context ([2026-10-30](../../research/2026-10-30_samber-do-v2-health-checks-deep-review-round-2.md)).
