# cqrs-htmx Self-Leverage Report (the repo auditing itself)

> Audit date: 2026-11-14 (refresh; prior 2026-11-05) · Method: library-deep-dive, mirrored inward (fleet series, [index](index.html))
> Subject: `github.com/larsartmann/cqrs-htmx/**` — 30 modules under one `go.work` (huma joined 2026-11-06); family trains v4.0.0 (huma) – v4.20.0 (usermgmt), root at v4.18.1

## Verdict at a glance

| Question                      | Answer                                                                                                                                                                                   |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does the fleet use cqrs-htmx? | It **is** cqrs-htmx — the question becomes: do the repo's own consumers (adminui, dashboardui, loginpage, setup, systemadapter, examples, e2e, integration_test) dogfood the published surface to the max? |
| Verdict                       | **Yes, 83 count / ~92 effective — exemplary dogfooding; the three new trains (QUERY ladder, bot tokens, dispatch postures) each shipped with in-repo proof the same week**                |

## How the repo exercises its own surface

- **Authorization contract (ADR-0057/0063):** the HTTP layer (`Decide`, postures, `AuthorizeRequest`) is consumed by root's own `authorizeWithRequest`, the huma adapter, `authztest`, and integration_test; gotcha 56's fix (anonymous → **401, engine never consulted**, zero-`UserID` guards at `authz.go:186`, `authz_contract.go:279`, `authz_dispatch.go:613`; pin `authz_contract_test.go:397`) landed 2026-11-08. Posture manifests are generated from ONE routeSpec table per module — adminui (`adminui/routes.go:87` → mux + `Handler.Routes` manifest, `:525`), dashboardui (`dashboardui/routes.go:114`, `Routes() :406`), setup (`setup/route_postures.go`), usermgmt (`usermgmt/command_postures.go`, 21 commands). `authztest.Suite` runs against the composed bundle (`setup/route_postures_behavior_test.go:112`); `GrantSuite`/`Suite` self-tested in `integration_test/authz_kit_selftest_test.go:118,273`.
- **HTTP QUERY ladder (ADR-0064, root v4.18.0):** `App.Query`/`QueryTyped` + `MethodOverrideMiddleware` (QUERY → POST+override fallback), `Accept-Query`, `ETagResults`, OpenAPI 3.2, the `cqrs-query.js` v1.0.0 auto-downgrade extension, and **htmx 4.0.1 embedded alongside the 2.0.11 default** (`htmx/embed4.go`, `htmx/serve4.go` `ScriptHandler4`). Dogfooded in `examples/basic` (`QUERY /api/items/search` at `main.go:329`, fallback twin `:336`, `QueryTyped` `:361`), byte-parity-pinned in `integration_test/query_method_test.go`, guide `docs/guides/http-query-method.md`, huma verdict test `huma/query_method_test.go:26`. NV4 static verdict: the 2.x extensions cannot register under htmx 4 (`defineExtension`/`createEventSource` absent) — the 2.x→4.x default flip precondition, recorded in ROADMAP q35(a) (`ROADMAP.md:61`).
- **M2M bot tokens (usermgmt v4.20.0, W0 2026-11-06):** `NewAPITokenMiddleware` (`usermgmt/api_token_middleware.go:45`) bridges the bot into the actor chain under a zero-actor guard — token-authenticated dispatch is attributed `bot:<id>`, never anonymous; `RequireBot` enforces with RFC 9457 rejections; pinned by `integration_test/actor_attribution_test.go:115` (`TestActorAttribution_BotDispatchIsNeverAnonymous`). W1/W2 (scopes, delegation, rotation) sit behind the M2M D1–D4 owner packet.
- **SSE transport:** golden-pinned envelope, `dedup.Ring` replay seam (`transport/serve.go:193`, pump at `:514`), `ReplayFiltered` fail-closed replay, heartbeats via `stream.Heartbeat` (`transport/serve.go:490`). Consumed by dashboardui, adminui, e2e; browser battery on the settled tree: **117 chromium tests green, 112 listed specs, ran-count gate green** (2026-11-14); firefox divergences (a)–(d) resolved 2026-11-08, `sync.spec.ts` stays scoped off firefox (divergence (e) open), firefox currently ADVISORY on 66 environment-class `setDownloadOptions` protocol failures.
- **ADR-0056 frontend sync:** `SyncPullHandler`/`SyncPushHandler` exercised by `examples/sync-demo` + `e2e/server/main.go:188-191` (dual backend IDs); sync assets **v1.6.0** pinned (`sync_embed.go:6`).
- **Direct imports (ADR-0062):** `htmx/`, `httperr/`, `eventcatalog/`, `internal/hx/` own their symbols; all 77 root façade aliases carry `// Deprecated:` naming their owner; the policy gate is green — `check-reexport-policy.sh --strict`: 47 alias files / 415 declarations, no growth, every re-export deprecated; `api_surface_compat_test.go` compile-pins the moved signatures; integration_test's deprecated aliases repointed 2026-11-07 (7 SA1019s → `httperr`/`identitymodel`). Internal modules import the owning packages directly (e.g. `usermgmt/es_event_catalog.go` imports `v4/eventcatalog`).
- **Event catalog:** `usermgmt.DefaultEventCatalog()` — now **27 events** across five aggregates (`usermgmt/es_event_catalog.go:27`) — served via `eventcatalog.Handler` in setup (`setup/machine.go:24`) and usermgmt; both wire formats derive from ONE spec table.
- **Projection health:** `ProjectionStatusHandler` in usermgmt (`es_projection_health.go`) + setup (`machine.go`); `ProjectionReadinessCheck` gates `AsyncStartup`.
- **adminui M05–M17 (2026-11-14):** header tenant switcher, tenant list filter/sort + blast-radius confirms, bulk import/export (size-capped, header-guarded — `adminui/handler_import.go:16`), member attribution from event metadata actor chains (`adminui/attribution.go`); routeSpec table generates mux AND manifest; `check-modules` aggregate 104/104 green.
- **Direction of travel:** the usermgmt→system/metaengine migration plan's W1 probes ran 2026-11-08 (W1.2 starvation probe GO — every go-cqrs-lite preset now declares the projections instance; `systemadapter/declarative_test.go:660` equivalence still pins the bridge), so the repo's own dogfooding surface is about to include `system.New` composition; M2M D1–D4 and migration D1–D5 packets are the open owner rows.
- **Composition:** `setup` one-call bundle + `RunWithAppkit` (SSE-safe `NoTimeout`, `setup/run_appkit.go:137`), 13 runnable examples (basic now carries the QUERY + partial-render showcases), integration_test + e2e/server as cross-module proof.

## Capability inventory

| #   | Surface                                                               | Status                 | Evidence / reason                                                                                                                                                   |
| --- | --------------------------------------------------------------------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Handler builder + HandlerOption pipeline                              | 🟢 Fully leveraged     | every module; ordering rules tested                                                                                                                                 |
| 2   | Authorization contract (`Decide`, postures, authztest; 401-anonymous) | 🟢 Fully leveraged     | HTTP layer + proof kit + gotcha-56 pins; manifests from one routeSpec table per module                                                                             |
| 3   | SSE `transport/` surface                                              | 🟢 Fully leveraged     | golden-pinned; dedup.Ring seam; 3 consumers; 117-test chromium battery                                                                                              |
| 4   | Frontend sync (ADR-0056)                                              | 🟢 Fully leveraged     | sync-demo + e2e + assets tests (v1.6.0)                                                                                                                             |
| 5   | Event catalog                                                         | 🟢 Fully leveraged     | 27 events, 2 modules, direct-imported per ADR-0062                                                                                                                  |
| 6   | Projection health/readiness                                           | 🟢 Fully leveraged     | 2 modules + health bridge                                                                                                                                           |
| 7   | Typed endpoints (`CommandTyped`/`QueryTyped`)                         | 🟡 Partially dogfooded | examples only (3 demos; `QueryTyped` now in basic) — internal UIs mount direct handlers (legitimate); typed path example-proven                                     |
| 8   | Partial/full rendering (`RenderPartialOrFull`, `RenderIf`, `OOBHTML`) | 🟡 Partially dogfooded | ⬆ was 🔴: the `/demo/items` showcase landed 2026-11-06 in `examples/basic` (`main.go:391`); internal UIs still hand-roll their fragment contracts (table-fragment + polled-region patterns) |
| 9   | OpenAPI builder (`WithOpenAPI`, `OpenAPISpecHandler`)                 | 🟡 Partially dogfooded | sync-demo serves a spec in a shipped binary (`main.go:134`); the `WithOpenAPI` HandlerOption is test-proven only; internal UIs publish no specs                      |
| 10  | Deprecated re-export hygiene / direct imports (ADR-0062)              | 🟢 Fully leveraged     | 0 internal façade uses; policy gate green (47/415 baselined); compile-pins; v5 deletion mechanized                                                                   |
| 11  | setup composition root                                                | 🟢 Fully leveraged     | the product itself                                                                                                                                                  |
| 12  | Examples breadth                                                      | 🟢 Fully leveraged     | 13 demos incl. middleware/observability/sync/datastar/system/QUERY/partial-render proofs                                                                            |
| 13  | HTTP QUERY ladder (ADR-0064; htmx 4.0.1 + cqrs-query ext)             | 🟢 Fully leveraged     | example + fallback twin + integration byte-parity + guide + huma verdict test; shipped in root v4.18.0                                                              |
| 14  | M2M bot-token surface (`NewAPITokenMiddleware`, `RequireBot`)         | 🟡 Partially dogfooded | W0 landed + attribution pinned in integration_test; W1/W2 (scopes, delegation, rotation) owner-gated on M2M D1–D4                                                    |
| 15  | Dispatch-boundary postures (ADR-0063, `DispatchGate`)                 | 🟡 Partially dogfooded | usermgmt declares 21 command postures + gate rules + `authztest.CommandSuite`; consumer adoption rides the C28 waves + upstream go-cqrs-lite ask                      |

## Score derivation

15 rows − 0 N/A = **15 applicable**; 10 full ×1 + 5 partial ×0.5 = **12.5 → 83/100** count-based. The count held at 83 while absolute leverage grew: the QUERY ladder entered as a new full row, row 8 moved 🔴→🟡 (showcase shipped), and three new honest partials (14, 15, and the still-examples-only typed row) entered the denominator. Effective: rows 7/14/15 are partial **by design** — consumer adoption lives in external repos (guardrail 7) or behind filed owner packets (M2M D1–D4, migration D1–D5), while the published surfaces themselves are fully exercised in-repo — so they count near-full: **~92 effective — very high**. The genuine thin spots remain rows 8/9: internal UIs hand-roll fragments and publish no OpenAPI specs.

## Version currency

Family trains advanced past the 2026-11-05 audit (root was v4.16.0): **root v4.17.0/v4.18.0/v4.18.1** (2026-11-07 — decomposition exports, QUERY ladder, Go 1.27.3 floor; v4.18.1 is the pin every dependent module resolves), **usermgmt v4.20.0** (bot-token surface), **identity-model v4.17.0**, **loginpage v4.18.0** (2026-11-11), **systemadapter v4.13.0**, **dashboardui/adminui/setup v4.17.0**, **huma v4.0.0** (new module). Release-train gate on family requires: **0 unpublished, 0 lag** (strict gate green at the 2026-11-14 M17 alignment, 104/104 checks); a same-day upstream publish (go-cqrs-lite metaengine v4.24.0) currently lags 5 requires — the gate prints the one-line sweep recipe; an upstream treadmill item, not a family lag. Examples resolve published tags hermetically.

## Recommendations

1. Adopt the QUERY ladder in one shipped internal panel (e.g. a dashboardui filter) so RFC 10008 reads prove out in a product surface, not only examples/integration.
2. When the M2M D1–D4 / migration D1–D5 ticks land, sweep the consumer waves (bot scopes, machines on SSE/sync/Huma) — the in-repo surface is published and pinned, ready for them.
3. Seed the 4-era extension answer (ExtSSE/ExtIdiomorph replacements) that ROADMAP q35(a) names as the htmx 2.x→4.x default-flip precondition — the one piece of the QUERY future with no in-tree seed.
4. Carry-over: publish an OpenAPI spec from one internal UI to self-prove `WithOpenAPI`'s per-route option form (the builder form is already shipped by sync-demo).

## How to verify

```bash
rg -n 'RenderPartialOrFull|OOBHTML' examples/basic/main.go                       # /demo/items showcase (row 8) — main.go:391-399
rg -n 'QUERY /api/items/search|QueryTyped' examples/basic/main.go                # QUERY ladder dogfood (row 13) — :329, :361
rg -l 'NewAPITokenMiddleware|RequireBot' usermgmt integration_test -g '*.go'     # bot tokens (row 14) — api_token_middleware.go + actor_attribution_test.go
rg -n 'func CommandPostures' usermgmt/command_postures.go                        # dispatch postures (row 15) — 21 commands
rg -c 'cqrshtmx\.(CSRF|RateLimiter|ServerTiming|SecurityHeaders)' adminui dashboardui loginpage setup usermgmt examples -g '*.go'   # no output = 0 façade uses
bash scripts/checks/check-reexport-policy.sh --strict                            # ✓ ADR-0062 holds (47 alias files / 415 declarations)
nix run .#check-release-train -- --refresh-cache --strict-lag 0                  # family 0 unpublished/0 lag; a same-day upstream metaengine publish prints its sweep recipe
```

## Prior reports

- Prior self-audit 2026-11-05 (superseded by this refresh): 12 rows, 83 count / ~92 effective — both of its recommendations have since landed (partial-render showcase in `examples/basic`; the OpenAPI builder-form receipt was already the sync-demo).
- [2026-09-17 setup deep dive](../../research/2026-09-17_setup-deep-dive.html); [2026-10-04 templ-components series](../../research/2026-10-04_templ-components-deep-dive.html); [2026-11-03 go-cqrs-lite deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html); [.agents/skills/cqrs-htmx/SKILL.md](../../../.agents/skills/cqrs-htmx/SKILL.md) consumer map.
