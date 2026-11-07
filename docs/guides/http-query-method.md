# The HTTP QUERY Method (RFC 10008)

`QUERY` is the first HTTP method whose semantics are exactly the CQRS read side: **safe** (like GET), **idempotent**, **cacheable**, and **request-body-carrying** (like POST). Complex read criteria — large filters, nested structures, SQL/JSONPath-shaped queries — travel as a request body instead of being squeezed into URL parameters, and the request can be retried after a connection failure without any risk of applying it twice.

This guide shows how cqrs-htmx serves the same query handler over three transports and how clients pick the best one they can send.

> Design record: `docs/planning/2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md` (verified fact base + the open upstream items). Runnable proof: `examples/basic` (`QUERY /api/items/search` + fallback twin) and `integration_test/query_method_test.go`.

## The fallback ladder

One handler, three transports. The CLIENT chooses the rung; the SERVER answers identically from a single handler registration.

| Rung | Transport | Use when | Trade-off |
| --- | --- | --- | --- |
| 1 | `QUERY` + JSON body | green path: Go stdlib, same-origin fetch/XHR, htmx 4 `hx-query`, DataStar `@query()` | old proxies/WAFs may answer 405/501 to unknown methods |
| 2 | `POST` + `X-HTTP-Method-Override: QUERY` header + same JSON body | rung 1 blocked anywhere on the path | is a POST at the security layers: carries the CSRF token, no safe-method cache treatment |
| 3 | `GET` + URL params | small, bookmarkable queries | length/encoding limits, filters land in logs and history |

`App.Query` never inspects the request method and neither do the decoders, so the method truth lives in your mux pattern — one handler serves every rung.

## Server: mount one handler, three rungs

```go
mux := http.NewServeMux()

// Rung 1 + rung 2 in one pattern: MethodOverrideMiddleware rewrites the
// fallback POST (X-HTTP-Method-Override: QUERY) into a QUERY request
// BEFORE routing, so a single QUERY pattern serves both.
mux.Handle("QUERY /contacts", app.Query("SearchContacts",
    cqrshtmx.DecodeJSONQuery(func(req searchRequest) (query.Query, error) {
        return &searchContactsQuery{Filter: req.Filter}, nil
    }),
    cqrshtmx.RenderJSON[[]contact](),
))

// Rung 3 for URL-sized queries (separate handler, same dispatch type).
mux.Handle("GET /contacts", app.Query("SearchContacts",
    cqrshtmx.DecodeFormQueryWithRequest(func(r *http.Request, _ struct{}) (query.Query, error) {
        return &searchContactsQuery{Filter: r.URL.Query().Get("filter")}, nil
    }),
    cqrshtmx.RenderJSON[[]contact](),
))
```

### MethodOverrideMiddleware placement is a security decision

Wrap the mux with it, INSIDE your middleware stack (after CSRF), never outermost:

```go
srv := cqrshtmx.Chain(
    cqrshtmx.RecommendedSecurityMiddleware(),
    sessionMW,
    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(cqrshtmx.MethodOverrideMiddleware(mux))
```

Why inside: the security layers must validate the **wire** method. A fallback POST is token-checked as a POST — today and forever, regardless of how CSRF safe-method lists treat QUERY later. Mounted outermost, the rewrite would put the logical method in front of CSRF: accidentally strict today, and silently token-free the day QUERY joins a safe-method exemption upstream.

Only `QUERY` is honored as an override target. An `X-HTTP-Method-Override: DELETE` is ignored — this middleware exists for safe read fallbacks, never to smuggle unsafe verbs past method-based routing. The rewrite clones the request; the caller's `*http.Request` is never mutated.

### Method guard options

- `RequireMethod(m)` — single method, unchanged.
- `RequireAnyMethod("QUERY", http.MethodPost)` — the multi-method guard for handlers mounted without method-scoped mux patterns (Chi, Gin). A mismatch answers 405 with an `Allow` header listing the configured methods (RFC 9110; also how RFC 10008 clients discover QUERY support).

### Content-Type enforcement (opt-in)

RFC 10008 says a server must reject a `QUERY` whose `Content-Type` is missing/inconsistent (400) or unsupported (415). cqrs-htmx's default is deliberately lenient — decoders accept any bytes they can parse, per the library principle of never enforcing defaults consumers might disagree with. `RequireContentType("application/json")` opts a route into the strict ladder: missing type on a body-carrying request → 400, different type → 415 (`httperr.ErrUnsupportedMediaType`), `; charset=utf-8` parameters ignored, bodyless requests always allowed.

### Discovery: Accept-Query

The JSON body-query decoders (`DecodeJSONQuery`, `DecodeJSONQueryTyped`) automatically set `Accept-Query: application/json` on responses — including error responses — because the decoder is the component that knows what it accepts; the advertisement cannot drift from reality. A URL-form decoder advertises nothing (it reads no body). `AcceptQuery("application/sql", ...)` appends further types for consumer decoders that accept them.

## Client recipes

### curl

```bash
# rung 1
curl -X QUERY localhost:8096/api/items/search \
  -H 'Content-Type: application/json' -d '{"filter":"widget"}'

# rung 2 (identical response)
curl -X POST localhost:8096/api/items/search \
  -H 'Content-Type: application/json' \
  -H 'X-HTTP-Method-Override: QUERY' \
  -d '{"filter":"widget"}'
```

### htmx 2.x (the embedded default)

No native `hx-query` — use rung 2; the override header rides `hx-headers`:

```html
<button hx-post="/contacts/search"
        hx-headers='{"X-HTTP-Method-Override": "QUERY"}'
        hx-ext="json-enc"
        hx-target="#results">
  Search
</button>
```

(With `MethodOverrideMiddleware` the POST lands on your `QUERY` pattern. The CSRF token flows through your existing `hx-headers` wiring.)

### htmx 4

Native `hx-query`: `<button hx-query="/contacts/search">`. The library embeds htmx 4.0.1 alongside the 2.0.10 default — serve it with `htmx.ScriptHandler4()` (ETag + immutable caching, same contract as the default handler); the default flips to 4.x only in v5.

### DataStar

Current upstream DataStar documents `@query()` (signals in the request body). The bundle embedded by the `datastar` module predates it, so today use rung 2 — `@post` sends signals as a JSON body, which `ds.ReadSignals` decodes regardless of method:

```html
<button data-on-click="@post('/contacts/search', {
  headers: {'X-HTTP-Method-Override': 'QUERY'}
})">Search</button>
```

`ds.ReadSignals` handles a native QUERY request correctly by construction (the fork routes GET/DELETE to URL-query signals and every other method to the JSON body).

### Raw fetch

```js
const res = await fetch("/contacts/search", {
  method: "QUERY",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ filter: "widget" }),
});
```

## Posture notes

- **CSRF:** httputil's CSRF middleware (the one cqrs-htmx re-exposes) currently challenges native QUERY — its safe-method exemption covers GET/HEAD/OPTIONS/TRACE only, and QUERY is spec-safe. Until the exemption is fixed upstream, either mount QUERY routes outside CSRF or send the token; rung 2 always carries it. The fallback-POST rule above does not change either way.
- **CORS:** QUERY is not in the CORS-safelisted method list — cross-origin use triggers a preflight, and `QUERY` must appear in `Access-Control-Allow-Methods`. Same-origin htmx/DataStar/fetch are unaffected.
- **HTML forms** cannot send QUERY (fetch/XHR only) — irrelevant for htmx and DataStar, both fetch-based.
- **Proxies/WAFs:** QUERY support across CDNs and corporate proxies is still uneven. Rung 2 exists precisely for the paths where rung 1 is refused. (A verified proxy-support table is tracked as plan item W5.2 — this guide deliberately carries no folklore.)
- **Caching:** QUERY responses are cacheable, but the cache key must include the request content; servers signal content-dependence with `Vary` (e.g. `Vary: Content-Type`). Conditional QUERY (`If-None-Match` → 304) is on the roadmap as an opt-in handler option (`ETagResults`, plan W4.2) — not shipped yet.
- **Errors:** decode/validation failures keep the library-wide 400 contract (`cqrshtmx.validate.failed`); 415 is `httperr.ErrUnsupportedMediaType` via `RequireContentType`; RFC 10008's 422 (query semantically unprocessable) is deliberately not introduced in v4.
