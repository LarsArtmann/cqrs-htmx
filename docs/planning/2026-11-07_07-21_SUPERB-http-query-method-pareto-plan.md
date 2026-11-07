# SUPERB plan — HTTP QUERY (RFC 10008) as a first-class read transport, with smart fallbacks

**Created:** 2026-11-07, 07:21 CET
**Repo:** `/home/lars/projects/cqrs-htmx`, branch `master`
**Status:** W0+W1 EXECUTED 2026-11-07 same-day (receipts in CHANGELOG [Unreleased]: middleware + three options + 415 sentinel + guide + example + parity tests; corpus baseline 55010). W2+ open — W2 gated on D3 + NV4/NV5, W3 on NV1 + D4, W4/W5 unblocked but sequenced after.

---

## 0. The thesis in one paragraph

`QUERY` (RFC 10008) is the first HTTP method whose semantics are exactly the CQRS read side: **safe + idempotent + request-body-carrying**. Today this library forces every complex read into a lie — `POST` (loses safety, retryability, cacheability, and drags CSRF along) or URL-encoded `GET` (length/encoding limits, filters leak into logs and history). The server side is already QUERY-capable by accident: `App.Query`/`QueryTyped` never inspect `r.Method` (`app.go:270`, `app.go:325`) and the JSON decoders read the body regardless of verb, so `mux.Handle("QUERY /items", app.Query(...))` works on stock Go 1.27 stdlib today (proven in-session with a live `ServeMux` + client round-trip). The work is therefore not "make it work" but "make it superb": advertise (`Accept-Query`), survive hostile infrastructure (a POST+`X-HTTP-Method-Override: QUERY` fallback ladder that rides every proxy and WAF ever shipped), meet both frontend runtimes where they actually are (htmx 2.0.10 embedded today, `hx-query` native in htmx 4; `@query()` native in current upstream DataStar, absent from our pinned 1.0.3 bundle), and describe it in OpenAPI 3.2 (which added the `query` operation type).

## 1. Non-goals

- No new router, no method-routing magic inside `App` — the consumer's mux owns patterns, exactly as today.
- No polyfill that rewrites QUERY into GET-with-params (that re-creates the URL-length problem QUERY exists to solve).
- No default flip of the embedded htmx 2.0.10 → 4.x inside v4 (one breaking window per major, ADR-0062); 4.x ships alongside.
- No change to the validation status contract (`cqrshtmx.validate.failed` → 400 stays; 422 is a v5 owner decision, not a quiet break).
- No caching middleware, no equivalent-resource/`Location` server machinery (RFC 10008 §7 equivalents) — documented as future, not built.
- `SyncPullHandler` stays GET (cursor URLs are bookmarkable by design; ADR-0056 is not a QUERY candidate).

## 2. Invariants

1. **Library principle first:** no consumer is forced onto QUERY — everything is opt-in; GET and POST behave identically forever.
2. **Fallback parity:** native QUERY, override POST, and (for small queries) GET must produce byte-identical success responses from one handler registration.
3. **Fallback honesty:** the override POST is still a POST at the CSRF layer — it carries the token and is checked. We never bypass CSRF to "make QUERY work"; QUERY's own CSRF exemption is an upstream (httputil) fix, not a local hack.
4. **Safe means safe:** nothing in the QUERY paths may dispatch commands or write; they route to `App.Query` only.
5. **No advertisement lies:** `Accept-Query`/`Allow` and the OpenAPI `query` operation must describe shipped behavior only — the decoder-driven default (W1.4) and the 3.2 auto-bump (W4.1) exist to make lying structurally impossible.
6. **Spec-truth in OpenAPI:** a `query` operation renders in an OpenAPI **3.2.0** document or not at all — never a 3.1.0 doc carrying a 3.2 field.
7. **Additive only in v4**; removals/defaults flips wait for the v5 window.

## 3. Verified fact base (everything below was checked this session)

### Server / Go

| # | Fact | Evidence |
| --- | --- | --- |
| S1 | Go stdlib mux registers and routes `QUERY /path` patterns; client sends QUERY with a body; status 200 round-trips | proven live in-session (throwaway module, `ServeMux` + `http.NewRequest("QUERY", ...)`) |
| S2 | `App.Query`/`QueryTyped` handlers never inspect `r.Method`, and neither do the decoders: `decodeJSONBody` reads the body with no method and no Content-Type gate (empty body → zero-value T) | `app.go:270-285`, `app.go:325-339`, `decoder.go:36-53` |
| S3 | `RequireMethod(method string)` is single-method, enforced at `handler.go:194` (405) | `options_json.go:63` |
| S4 | CSRF (httputil v1.8.0, nosurf-based) exempts only GET/HEAD/OPTIONS/TRACE — a native QUERY **is challenged** despite being spec-safe | `httputil@v1.8.0/csrf.go:789` |
| S5 | `httperr` has no 415/422 mapping today (sentinels: 401/403/405/413; families 400/409/500/503) | `httperr/errors.go:97-105` |
| S6 | ETag/304 machinery exists and is root-usable: `hx.ServeImmutableJSON`, go-etag `MatchesIfNoneMatch` | `internal/hx/immutable.go:53-70` |
| S7 | `RoutePosture.Method` is a free string — QUERY rows need no type change | `route_posture.go` |
| S8 | OpenAPI builder pins `3.1.0` (`builder.go:7`), methods get/post/put/patch/delete/head/options (`builder.go:121-139`); `OpenAPIRoute.Method` is a free lowercase string | `openapi/builder.go`, `openapi_collector.go:16` |

### Clients

| # | Fact | Evidence |
| --- | --- | --- |
| C1 | htmx 4 ships `hx-query` natively ("Issues QUERY request to specified URL", cites RFC 10008); latest 4.0.1 (2026-11-01); 4.0 deliberately not npm-`latest` until 2027 | four.htmx.org/reference/attributes/hx-query (fetched 2026-11-07); htmx releases |
| C2 | Embedded htmx is 2.0.10 — no QUERY support; `ScriptHandlerWith(js, version)` exists explicitly for serving other versions ("a custom build, a beta, or htmx 4.0") | `htmx/embed.go:6`, `htmx/serve.go:19-31` |
| C3 | Current upstream DataStar documents `@query()` (signals in body, safe+idempotent, RFC 10008) with `headers`, `retry`, `openWhenHidden` options | data-star.dev/reference/actions (fetched 2026-11-07) |
| C4 | Our embedded datastar.js is upstream **1.0.3** (fork `go-datastar/static` v0.6.1) and registers only get/patch/post/put/delete fetch actions — **no `@query`**; the plugin API (`datastar.action(...)`) IS exported, so custom actions can be registered at runtime | `static@v0.6.1/static.go:22` (`Version = "1.0.3"`), action registration table in the bundle: `Le("get","GET",!1);Le("patch",...);Le("post",...);Le("put",...);Le("delete",...)`; `export{V as action,...}` |
| C5 | `ds.ReadSignals` handles QUERY correctly **by construction**: the fork routes GET/DELETE to URL-query signals and *everything else* (incl. QUERY) to the JSON body — re-verified in source, not asserted | go-datastar `inbound.go:55-68` (`isQueryMethod` = GET/DELETE only), `datastar/patch.go:169` |
| C6 | DataStar non-GET actions default `openWhenHidden: true` — QUERY keeps SSE feeds alive in background tabs | data-star.dev/reference/actions § Options |

### Specs / protocol

| # | Fact | Evidence |
| --- | --- | --- |
| P1 | RFC 10008: QUERY is safe, idempotent, cacheable (content-keyed), body-carrying; error ladder 400 (no/inconsistent Content-Type) / 415 (unsupported type) / 422 (query unprocessable) / 406; `Accept-Query` advertises formats; `OPTIONS`+`Allow` advertises the method; redirects keep QUERY (no POST→GET rewrite); conditional requests (If-None-Match → 304) apply | MDN QUERY reference (fetched 2026-11-07), rfc-editor.org/rfc/rfc10008.html |
| P2 | OpenAPI 3.2.0 adds the `query` fixed field to the Path Item Object ("A definition of a QUERY operation...") | OpenAPI-Specification `versions/3.2.0.md` Path Item Object (fetched 2026-11-07) |
| P3 | Browsers do not gate QUERY — any `fetch()`/XHR can send it; it is NOT in the CORS-safelisted method list, so cross-origin use triggers preflight (must be in `Access-Control-Allow-Methods`) | MDN QUERY § Browser compatibility ("not relevant... developers can send QUERY via fetch() (with a CORS preflight)") |
| P4 | HTML forms cannot send QUERY (fetch/XHR only) — irrelevant for htmx/DataStar, both fetch-based | MDN QUERY summary table |

### Not verified (deliberately — first task that needs them must check)

- NV1: which upstream DataStar release introduced `@query()` (the bundle bump task needs the exact tag).
- NV2: proxy/CDN/WAF behavior for QUERY (Cloudflare, nginx, Envoy, Caddy, ALB) — the guide must carry a verified table, not folklore.
- NV3: whether Huma v2 can register a QUERY-method operation (OpenAPI 3.2 awareness).
- NV4: htmx 4 extension compatibility for the embedded SSE/idiomorph extensions (htmx 4 rewrote parts of the extension surface).
- ~~NV5: whether htmx 2.0.10's internal XHR path accepts a non-standard verb cleanly (body-vs-params switch)~~ **RESOLVED 2026-11-07, static proof in the embedded bundle:** `methodsThatUseUrlParams: ["get", "delete"]` + `methodsThatUseUrlParams.indexOf(t) >= 0` decides body-vs-URL-params, so QUERY (not in the list) rides the body path exactly like POST, and the transport is `XMLHttpRequest.open(...)` which accepts arbitrary method tokens — the W2.4 extension's `htmx:configRequest` verb flip is mechanically sound. Runtime proof still owed by the W2.4 browser e2e.

- NV6: the exact htmx 4 line detail ("4.0.1, 2026-11-01") is second-hand from an assistant research pass — re-verify from the htmx releases page at W2.3 before embedding a pin.

## 4. The design

### 4.1 The fallback ladder (the "smart" in smart fallbacks)

One handler, three transports, chosen by the CLIENT, all true at the SERVER:

| Rung | Transport | When it is the right choice | Cost |
| --- | --- | --- | --- |
| 1 | native `QUERY` + JSON body | green paths: Go stdlib ✓, same-origin fetch ✓, htmx 4 `hx-query` ✓, DataStar `@query()` ✓ | unknown-method 405/501 from old proxies/WAFs (NV2) |
| 2 | `POST` + `X-HTTP-Method-Override: QUERY` + same JSON body | anywhere rung 1 is blocked; works on every proxy, CDN, and browser ever shipped | semantically a POST: CSRF token required (where mounted), no safe-method cache treatment |
| 3 | `GET` + URL params (status quo) | tiny queries, bookmarkable/shareable URLs | length/encoding limits, filters in logs/history |

Server-side this is ONE handler because `App.Query` is method-agnostic (S2); the consumer mounts it twice, or once under the new middleware. Client-side, rung 2 is a header away on every runtime we ship (htmx `hx-headers`, DataStar `headers` option — both verified C1/C3).

**The auto-downgrade (showpiece, W2):** the embedded `cqrs-query` htmx extension (works on htmx 2.0.10 AND 4.x) upgrades eligible requests to native QUERY, and on an explicit method rejection (405 without QUERY in `Allow`, 501, or a 4xx carrying `Allow` that omits QUERY) transparently re-issues as POST+override and remembers the decision per-origin in `sessionStorage` (TTL ~1h) so later requests skip the doomed rung. Safety argument: QUERY is safe+idempotent, the handler is a query handler, so the retry can never double-apply — this is the same argument RFC 10008 itself makes for retrying QUERY after connection failure.

### 4.2 Server surface (root, additive)

```go
// 1. The fallback seam — wrap the mux (INSIDE the middleware trio; see the
// ordering table below — the security layers must validate the WIRE method):
mux := http.NewServeMux()
mux.Handle("QUERY /contacts", app.Query("SearchContacts", cqrshtmx.DecodeJSONQuery(...)))

srv := cqrshtmx.Chain(
    cqrshtmx.RecommendedSecurityMiddleware(),
    sessionMW,
    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
    cqrshtmx.HTMXMiddleware,
    app.Middleware(), // enrichment sees QUERY, not the wire POST
)(cqrshtmx.MethodOverrideMiddleware(mux)) // POST + override header -> r.Method="QUERY" pre-routing
```

**Placement is a security decision, not a style choice.** The rewrite must sit AFTER CSRF and BEFORE the mux:

| Placement | What CSRF validates | Today (QUERY challenged, S4) | After W5.1 (QUERY safe-listed) |
 | --- | --- | --- | --- |
 | **Inside CSRF (wraps the mux) — chosen** | the wire method (POST) | override POST demands the token | override POST still demands the token — behavior never changes |
 | Outside CSRF (outermost — rejected) | the logical method (QUERY) | override POST demands the token (accidentally correct) | override POST silently stops demanding the token — fallback auth becomes coupled to an upstream fix |

The rejected placement "works" today by accident and breaks the day W5.1 lands; the chosen one keeps the fallback rung permanently as strict as a POST, which is what it is on the wire. Native QUERY requests pass through untouched either way (challenged today per S4, exempt after W5.1 — the desired endgame for a safe method).

- **`cqrshtmx.MethodOverrideMiddleware(next http.Handler) http.Handler`** — rewrites `POST` + `X-HTTP-Method-Override: QUERY` (and the legacy `X-HTTP-Method` spelling) to `r.Method = "QUERY"` via `r.Clone` BEFORE the mux sees it, so a single `QUERY /path` pattern serves both rungs (the dual registration above becomes optional). Overriding into a non-safe target (e.g. `Override: DELETE`) is ignored — this middleware is for read fallbacks only, by construction.
- **`cqrshtmx.AcceptQuery(types ...string) HandlerOption`** — overrides the `Accept-Query` advertisement (default: the decoder already advertises itself — see below); contributes `QUERY` to the `Allow` header on 405s so clients can discover rung 1 per RFC 10008 §5.3. The DEFAULT source of truth is the decoder, not this option: `DecodeJSONQuery`/`DecodeJSONQueryTyped` set `Accept-Query: application/json` on the responses they produce, so the advertisement can never drift from what the handler actually decodes (a hand-set header option would re-create exactly that split brain). `AcceptQuery` exists only to ADD types (e.g. a consumer decoder that also accepts `application/sql`).
- **`cqrshtmx.RequireAnyMethod(methods ...string) HandlerOption`** — the multi-method generalization of `RequireMethod` (S3) for consumers who mount one handler under several patterns without method-scoped mux patterns (Chi/Gin).
- **`cqrshtmx.RequireContentType(mt string) HandlerOption`** — 400 on missing/lying Content-Type and 415 on a known-but-wrong type, per RFC 10008's media-type ladder (P1). Backed by a new `httperr.ErrUnsupportedMediaType` sentinel → 415 (S5 closes). Runs after authorize, before decode. **Documented divergence:** RFC 10008 says servers MUST reject a missing/inconsistent Content-Type; our default is lenient (S2 — `decodeJSONBody` never looks), because the library principle forbids enforcing a default consumers might disagree with. `RequireContentType` is the opt-in that makes a route RFC-pure; the guide states this plainly instead of letting consumers believe the default is spec-exact.
- **`cqrshtmx.ETagResults() HandlerOption`** — opt-in conditional-query support: FNV-1a the rendered bytes, set `ETag`, answer `If-None-Match` with 304 (machinery exists, S6). Explicitly opt-in because hashing every response is a CPU trade the consumer should choose.

Naming follows the existing vocabulary (`*Middleware`, `Require*` options); all four are additive v4 API, no alias files (ADR-0062 — these are new symbols in the owning package, root).

### 4.3 HTMX integration

**Today (embedded 2.0.10), zero new code required:** `hx-post` + `hx-headers='{"X-HTTP-Method-Override":"QUERY"}'` rides the fallback rung through `MethodOverrideMiddleware`; CSRF token flows through the standard `hx-headers` trio (skill gotcha 4). This is the documented default recipe until htmx 4 is the embedded default.

**htmx 4 alongside (W2):** embed `htmx/htmx4.min.js` + `htmx.ScriptHandler4()` + `htmx.Version4()` — default handler keeps serving 2.0.10 (invariant 7); consumers opt into 4.x per-page and get native `hx-query`. The existing `ScriptHandlerWith` already proves the serving seam (C2). NV4 (extension compat) gates whether `ExtSSE`/`ExtIdiomorph` are served alongside 4.x or marked 2.x-only.

**The `cqrs-query` extension (W2, showpiece):** `<div hx-ext="cqrs-query">` upgrades child `hx-post`/`hx-get` elements marked `data-cqrs-query` to native QUERY with the htmx JSON body encoder, auto-downgrades per §4.1, ~1 KB, CSP-safe (no inline handlers), served via the existing extension machinery (`htmx/extensions/`) with ETag + immutable cache. NV5 gates whether the htmx-2 path needs the `json-enc` companion or ships standalone.

### 4.4 DataStar integration

**Server: nothing to change.** `ds.ReadSignals` is method-agnostic (C5) — `mux.Handle("QUERY /search", searchHandler)` + `@query('/search')` is already the correct wiring once the client bundle knows the action. The SSE response builder answering a QUERY request works unchanged (content-type-driven patching, C5) and QUERY benefits from `openWhenHidden: true` defaults (C6) — a QUERY-driven dashboard feed survives background tabs, which GET-SSE closes.

**Bundle gap (W3):** our pinned upstream 1.0.3 lacks `@query()` (C4). The fork's upgrade process is documented (replace bundle, bump `static.Version`, run wire-format goldens — `go-datastar` `docs/static-js.md`), so W3 is: identify the release that added `@query()` (NV1), bump the fork, re-run goldens, sweep `datastar` module pins. **Interim, documented not shipped:** a five-line consumer snippet registering a custom action via the exported plugin API (`datastar.action('query', ...)` → fetch with method QUERY, signals JSON body — the shape `@post` already uses), plus the universal `@post` + override-header recipe (headers option verified, C3). Shipping shim JS in the `datastar` module would change that module's no-JS-assets shape for a gap the bundle bump closes anyway — rejected (verschlimmbessern).

### 4.5 OpenAPI (W4)

`openapi.Query(operationID)` constructor emitting a `query` Path Item operation (P2), with the rendered spec version **auto-bumping to `3.2.0` the first time a query operation is registered** — a 3.1.0 document carrying a `query` field is silently tool-invalid (invariant 6), so the builder refuses to produce the lie. README documents the bump and its tooling implication (3.1-only parsers). `WithOpenAPI` needs no change (`Method` is a free string, S8).

### 4.6 Docs, examples, gates

- **Guide** `docs/guides/http-query-method.md`: the ladder, the per-transport recipes (curl, htmx 2, htmx 4, DataStar, raw fetch), CORS preflight note (P3), CSRF posture (S4 + upstream fix), proxy table (NV2 — verified-only rows), caching/Vary note (P1). Corpus baseline bumps in the same commit.
- **Example:** `examples/basic` gains `QUERY /api/items` + the override twin + a demo page section — the `/demo/items` showcase pattern, not a new module.
- **Tests:** `integration_test` parity suite (native QUERY ≡ override POST ≡ GET for small queries, byte-identical bodies), 405/`Allow`/`Accept-Query` goldens, `MethodOverrideMiddleware` routing tests, CSRF-interaction test (override POST is checked, native QUERY documented-challenged until upstream fix).
- **Postures:** QUERY rows in the example manifest pins where routes are declared (S7).

## 5. Pareto layers

| Layer | Effort | Content | Outcome share |
| --- | --- | --- | --- |
| **1% (~1 h)** | Guide + examples/basic QUERY/override routes + parity tests | consumers can ship QUERY TODAY on stock API | **51%** — the server truth (S1/S2) becomes documented, tested surface |
| **4% (~4 h)** | `MethodOverrideMiddleware`, `AcceptQuery`, `RequireAnyMethod`, `RequireContentType` + 415 sentinel, tests | the fallback ladder is one import | **64%** |
| **20% (~2 days)** | htmx4 embed + `cqrs-query` extension + browser e2e; DataStar bundle bump (fork) + guide §; `openapi.Query()` + 3.2 auto-bump; `ETagResults` | "superb": both frontend runtimes native, discovery, spec-truth | **80%** |
| **80% (~1 week)** | upstream legs (httputil CSRF safe-list, proxy table verification), huma QUERY verdict, route-posture/example sweep, ADR-0064 recording the ladder as the library contract, v5 flags (htmx 4 default, 422) | durability — the design stops depending on this session | 100% |

## 6. Waves

### W0 — truth on disk (docs + example; no new API) — **EXECUTED 2026-11-07**

| # | Task | Done when |
| --- | --- | --- |
| W0.1 | Guide skeleton with the verified fact base (§3) and the three rung recipes | **DONE** — `docs/guides/http-query-method.md` (documents the shipped W1 API, not the hand-rolled patterns; corpus baseline bumped same-commit) |
| W0.2 | `examples/basic`: `QUERY /api/items/search` + `POST /api/items/search` override twin on one handler; demo page button uses the override recipe | **DONE** — routes + README curl recipes; live byte-parity verified on a running server (`PARITY-OK`) |
| W0.3 | `integration_test/query_method_test.go`: parity + 405/Allow assertions | **DONE** — 4 tests green under `-race`; published-API-only (hermetic isolation stays green) |
| W0.4 | TODO_LIST row pointing here | **DONE** — P1 row, todo-hygiene green |

### W1 — the fallback ladder as API (root, additive) — **EXECUTED 2026-11-07**

| # | Task | Done when |
| --- | --- | --- |
| W1.1 | `MethodOverrideMiddleware` (header rewrite pre-routing, safe-target-only, `r.Clone`) + unit tests incl. a mux-routing proof AND the placement proof (mounted inside CSRF: an override POST without a token is refused; with the token, passes) | **DONE** — `method_override.go` + 7 tests incl. `TestMethodOverride_WireMethodSeenByOuterLayers` (the wire-POST proof; the CSRF-token refusal itself rides the httputil middleware and stays documented in the guide — see erratum E5) |
| W1.2 | `httperr.ErrUnsupportedMediaType` (415) + `MapError` wiring + sentinel table row | **DONE** — `httperr/errors.go`, maps via `explicitErrorStatus` |
| W1.3 | `RequireContentType` option (400 missing/lying, 415 wrong) | **DONE** — `options_json.go` `checkRequiredContentType`; charset params ignored; bodyless always allowed; ladder test green |
| W1.4 | Decoder-driven `Accept-Query` (JSON query decoders advertise `application/json` on responses) + `AcceptQuery` add-only override + `Allow` enrichment on 405 | **DONE** — advertisement set on success AND error responses; form decoders advertise nothing (test-pinned) |
| W1.5 | `RequireAnyMethod` (generalizes `RequireMethod`, which stays) | **DONE** — 405 + `Allow` header; legacy single-method message byte-compatible |
| W1.6 | Guide § fallback updated from hand-rolled to shipped API; skill snippet fences | **DONE** — guide documents the shipped API; skill gains the QUERY subsection

### W2 — HTMX lane

| # | Task | Done when |
| --- | --- | --- |
| W2.1 | Resolve NV5 (htmx 2.0.10 non-standard verb path) with a minimal browser probe | **DONE 2026-11-07 (static)** — `methodsThatUseUrlParams` proof recorded in §3 NV5; runtime confirmation folded into W2.4's e2e |
| W2.2 | Resolve NV4 (htmx 4 extension compat for `ExtSSE`/`ExtIdiomorph`) | verdict recorded; 4.x embed scope decided |
| W2.3 | `htmx/htmx4.min.js` + `ScriptHandler4()` + `Version4()` (default unchanged) | served with ETag; version pin + banner comments; `check-templates`-style gate if needed |
| W2.4 | `cqrs-query` extension (upgrade + auto-downgrade + sessionStorage memory, ~1 KB, CSP-safe) | served via extension machinery; browser e2e (e2e/server) proves upgrade, downgrade-on-405, and memory |
| W2.5 | Guide § htmx (2.x recipe, 4.x recipe, extension) + example page section | example demo works in the e2e browser run |

### W3 — DataStar lane

| # | Task | Done when |
| --- | --- | --- |
| W3.1 | Resolve NV1 (upstream release adding `@query()`) | tag recorded here |
| W3.2 | Fork bundle bump (external repo `go-datastar/static`) + wire-format goldens | fork tests green; upstream treadmill sweep to the new pin |
| W3.3 | `datastar` module: README § QUERY (`@query()` recipe + interim custom-action snippet + override recipe) | `ds.ReadSignals`-on-QUERY e2e in datastar-demo or integration_test |
| W3.4 | Guide § DataStar (incl. `openWhenHidden` synergy, C6) | docs-links green |

### W4 — OpenAPI + conditional queries

| # | Task | Done when |
| --- | --- | --- |
| W4.1 | `openapi.Query()` — four touchpoints: `PathItem.Query` field, `hasMethod` case, `operations()` map entry, the constructor — + spec-version auto-bump to 3.2.0 on first query op + tests + README tooling note | a rendered spec with a query op declares `openapi: 3.2.0` — test-pinned; `PathItem` JSON shows the `query` key |
| W4.2 | `ETagResults()` option (FNV over rendered bytes; If-None-Match → 304) | conditional test: second identical QUERY → 304, empty body |
| W4.3 | huma verdict (NV3): QUERY-method registration support | verdict + README gap-note or recipe |
| W4.4 | ADR-0064: the fallback ladder as library contract (rungs, safety argument, CSRF posture) | ADR Accepted; guide links it |

### W5 — upstream and durability

| # | Task | Done when |
| --- | --- | --- |
| W5.1 | httputil ask + patch: QUERY joins the CSRF safe list (S4; it is spec-safe) | merged upstream + pinned; CSRF-interaction test flips to "native QUERY exempt" |
| W5.2 | Proxy support table (NV2): verified rows only (Cloudflare, nginx, Envoy, Caddy, ALB) | guide table cites a source per row; unknown stays "unknown" |
| W5.3 | v5 flags recorded: htmx 4 default flip, 422 validation status, htmx-2 extension retirement | ROADMAP v5 section rows |

## 7. Verification receipts (per wave)

- W0/W1: `GOEXPERIMENT=jsonv2 go test ./... -race` (root + integration_test), `nix run .#check-modules` (skill fences, docs links, corpus with bumped baseline), `nix run .#check-todo-hygiene`.
- W2: e2e browser battery for the extension; `htmx` package tests; banner/version checks.
- W3: datastar module suite + fork goldens + `nix run .#check-upstream-treadmill` (worst lag 0 after sweep).
- W4: openapi package tests; coverage gate row unchanged (module thresholds already single-sourced).
- W5: CSRF test flip + treadmill; docs-gates battery still 5/5 or the audit pin policy respected.

## 8. Owner decisions

| # | Question | Default if unanswered |
| --- | --- | --- |
| D1 | Flip embedded htmx default to 4.x in v5? | yes at v5 (W5.3 records it), no earlier |
| D2 | Move validation failures to 422 for QUERY (RFC-ladder-pure) at the cost of the 400 contract? | no — 400 stays in v4; 422 is a v5 decision |
| D3 | Ship the `cqrs-query` extension for htmx 2, or docs-only override recipe until 4.x default? | ship it (the auto-downgrade is the differentiator), gated on W2.1 |
| D4 | Bundle-bump cadence for `go-datastar/static` (ride upstream minors?) | ride releases that add actions we document (here: `@query`) |
| D5 | `MethodOverrideMiddleware` home: root (CQRS-coupled, safe-targets-only variant, ships in v4) vs upstream httputil offer (it is generic HTTP plumbing; httputil v1.8.0 has none — verified) | root first, offer upstream in W5 if it proves general (the SSE-deadline-fix pattern) |

## 9. Errata / honest notes

### 9.1 Re-review pass 2026-11-07 (same day, after the "is it superb?" challenge)

Four defects found by re-verifying the plan's own claims against source (a review's verdict is a claim, not a fact):

1. **The middleware placement instruction was wrong.** §4.2 originally said "OUTERMOST" — which puts the rewrite before CSRF and couples fallback authentication to the W5.1 upstream CSRF fix (accidentally strict today, silently token-free the day QUERY is safe-listed). Corrected to inside-CSRF placement with the ordering table; W1.1's done-when now includes the placement proof.
2. **`AcceptQuery` overpromised validation magic.** "Refuses types the shipped decoders cannot decode" had no mechanism (decoders are closures; the option sees strings). Replaced with the decoder-driven advertisement — the component that knows the media type is the one that advertises it, killing the drift split brain by construction.
3. **C5 was asserted, not verified.** `ReadSignals` method-agnosticism was claimed from the signature alone. Source read now: `inbound.go:55-68` routes GET/DELETE to URL-query and everything else to the body, so QUERY reads the JSON body — correct by construction (deliberate fork design). Claim now cites source.
4. **C1's version detail was second-hand** ("4.0.1, 2026-11-01" from an assistant research pass, not the releases page) — promoted to NV6, verified at W2.3.

Also sharpened: S2 now cites the decoder (`decoder.go:36-53`, no Content-Type gate — which makes the RFC's MUST-reject media-type ladder an opt-in divergence, now stated in `RequireContentType`), and W4.1 names all four `PathItem` touchpoints (typed struct, not a map — `openapi/types.go:40-92`).

**Provenance note:** this file was twice nearly lost the same day — a sibling session's `Git Town WIP` stash (`stash@{0}`, gotcha 47 strike three) captured the uncommitted plan + TODO row + baseline bump before the daemon could commit them; recovered read-only via `git restore --source=stash@{0}` (the stash was never popped — it holds the sibling's work too) and the amendments were applied on top.

### 9.2 Original errata

- **E5 (execution pass, 2026-11-07):** the W1.1 done-when promised a CSRF-token refusal test, but that behavior belongs to httputil's middleware, not ours — the shipped placement proof (`TestMethodOverride_WireMethodSeenByOuterLayers`) pins the load-bearing half (outer layers see the wire POST); the token-refusal half is exercised by the CSRF test battery in httputil and stays documented in the guide's posture section. Promising it here conflated layers.

- The in-session Go stdlib proof (S1) used the ambient toolchain, not `nix develop` — same 1.27 floor, but W0.3 re-proves inside the repo test battery.
- MDN's QUERY page carries no browser-compat table ("not relevant" — browsers don't gate it, P3); any "browser support" claim in the guide must cite fetch-spec method-token rules, not a version matrix.
- The `@post`+override DataStar recipe sends the CSRF-relevant POST with `credentials` per DataStar defaults — the guide's CSRF section must state whether DataStar fetches include cookies by default (verify when writing W3.4; do not assume).
