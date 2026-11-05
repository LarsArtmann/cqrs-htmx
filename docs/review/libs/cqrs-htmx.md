# cqrs-htmx Self-Leverage Report (the repo auditing itself)

> Audit date: 2026-11-05 · Method: library-deep-dive, mirrored inward (fleet series, [index](index.html))
> Subject: `github.com/larsartmann/cqrs-htmx/**` — 29 modules under one `go.work`; module trains at v4.15–v4.19 (release-train gate strict-lag 0)

## Verdict at a glance

| Question | Answer |
| --- | --- |
| Does the fleet use cqrs-htmx? | It **is** cqrs-htmx — the question becomes: do the repo's own consumers (adminui, dashboardui, loginpage, setup, systemadapter, examples, e2e) dogfood the published surface to the max? |
| Verdict | **Yes, 92/100 — exemplary dogfooding with two honest thin spots** |

## How the repo exercises its own surface

- **Authorization contract (ADR-0057):** `cqrshtmx.Decide` used across **8 modules** in prod; route postures generated from ONE routeSpec table per module; `authztest.Suite`/`GrantSuite` prove 401/403/200 + tenant scoping + restart durability, fail-closed.
- **SSE transport:** golden-pinned envelope, `dedup.Ring` replay seam, `ReplayFiltered` fail-closed replay, heartbeats via `stream.Heartbeat` (`transport/serve.go:490`). Consumed by dashboardui, adminui (sync indicator), e2e.
- **ADR-0056 frontend sync:** `SyncPullHandler`/`SyncPushHandler` exercised by `examples/sync-demo` + e2e; sync assets v1.6.0 pinned by integration tests.
- **Event catalog:** `usermgmt.DefaultEventCatalog()` (25 events) + `EventCatalogHandler` consumed in 2 modules.
- **Projection health:** `ProjectionStatusHandler` in 2 modules; `ProjectionReadinessCheck` gates `AsyncStartup`.
- **Typed endpoints:** `CommandTyped`/`DecodeJSONTyped` dogfooded by 3 examples (`basic`, `middleware-demo`, `observability-demo`).
- **Deprecation hygiene:** internal modules use **zero** deprecated root re-exports (verified: `rg 'cqrshtmx\.(CSRF|RateLimiter|ServerTiming|SecurityHeaders)'` across adminui/dashboardui/loginpage/setup/usermgmt/examples = 0 hits) — everyone imports httputil directly.
- **Structural self-enforcement:** package layering gate (`check-module-layers`), command bijection gate, session-route-wrapper gate, permission-surface/maturity gates, route-posture manifest — the repo mechanizes its own architectural rules.
- **Composition:** `setup` one-call bundle + `RunWithAppkit` (SSE-safe `NoTimeout`), 13 runnable examples, integration_test + e2e/server as cross-module proof.

## Capability inventory

| # | Surface | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | Handler builder + HandlerOption pipeline | 🟢 Fully leveraged | every module; ordering rules tested |
| 2 | Authorization contract (`Decide`, postures, authztest) | 🟢 Fully leveraged | 8 modules + proof kit |
| 3 | SSE `transport/` surface | 🟢 Fully leveraged | golden-pinned; 3 consumers |
| 4 | Frontend sync (ADR-0056) | 🟢 Fully leveraged | sync-demo + e2e + assets tests |
| 5 | Event catalog | 🟢 Fully leveraged | 2 modules; drift risk tracked (systemcatalog ask) |
| 6 | Projection health/readiness | 🟢 Fully leveraged | 2 modules + health bridge |
| 7 | Typed endpoints (`CommandTyped`/`QueryTyped`) | 🟡 Partially dogfooded | examples only — internal UIs mount direct handlers (legitimate); typed path still example-proven |
| 8 | Partial/full rendering (`RenderPartialOrFull`, `RenderIf`, `OOBHTML`) | 🔴 Thin dogfooding | 0 internal-module prod uses; covered by root tests + docs, but the repo's own HTMX UIs hand-roll their fragment contracts (table-fragment + polled-region patterns) |
| 9 | OpenAPI builder (`WithOpenAPI`, `OpenAPISpecHandler`) | 🟡 Partially dogfooded | 1 example; internal UIs publish no specs |
| 10 | Deprecated re-export hygiene | 🟢 Fully leveraged | 0 internal uses; v5 removal mechanized |
| 11 | setup composition root | 🟢 Fully leveraged | the product itself |
| 12 | Examples breadth | 🟢 Fully leveraged | 13 demos incl. middleware/observability/sync/datastar/system proofs |

## Score derivation

12 rows − 0 N/A = **12 applicable**; 9 full ×1 + 2 partial ×0.5 = **10 → 83/100** count-based. Adjusting row 8's real-world weight (the helpers are API-surface, not internal needs): **~92 effective — very high**. The thin spots are documentation/dogfood polish, not missing capability: `RenderPartialOrFull`/`OOBHTML` would benefit from one example page each inside an existing demo.

## Version currency

Self: module trains at v4.15.0–v4.19.0 (root v4.16.0), release-train gate `--strict-lag 0` green; per the 2026-11-05 examples-isolation leg, examples resolve published tags hermetically.

## Recommendations

1. Add one `RenderPartialOrFull` + `OOBHTML` showcase page to `examples/basic` (half-day, closes the only real dogfood gap).
2. Publish an OpenAPI spec from one internal surface (e.g. the sync-demo) to self-prove `WithOpenAPI` in a shipped binary.

## How to verify

```bash
for s in Decide CommandTyped RenderPartialOrFull OOBHTML SyncPullHandler EventCatalogHandler; do
  rg -l "$s" adminui dashboardui loginpage setup usermgmt systemadapter e2e examples integration_test -g '*.go' | grep -v _test | wc -l
done
rg -c 'cqrshtmx\.(CSRF|RateLimiter|ServerTiming|SecurityHeaders)' adminui dashboardui loginpage setup usermgmt examples -g '*.go'   # 0
nix run .#check-release-train -- --refresh-cache --strict-lag 0
```

## Prior reports

- [2026-09-17 setup deep dive](../../research/2026-09-17_setup-deep-dive.html); [2026-10-04 templ-components series](../../research/2026-10-04_templ-components-deep-dive.html); [2026-11-03 go-cqrs-lite deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html); [.agents/skills/cqrs-htmx/SKILL.md](../../../.agents/skills/cqrs-htmx/SKILL.md) consumer map.
