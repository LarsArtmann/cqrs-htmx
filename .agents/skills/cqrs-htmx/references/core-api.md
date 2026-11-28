# Core API reference (root module)

Import: `cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"`

This is the deep reference for the root module's `App`, `HandlerOption` values, middleware, context IDs, errors, and the HTMX `Response` builder. Load this when you need exact signatures or the full option catalogue.

## `App` and `Config`

```go
app, err := cqrshtmx.New(cfg)        // errors if both Commands and Queries are nil
app := cqrshtmx.MustNew(cfg)          // panics on error
```

```go
type Config struct {
    Commands  *command.Dispatcher           // go-cqrs-lite/command/v4
    Queries   *query.Dispatcher             // go-cqrs-lite/query/v4
    Enforcer  cqrshtmx.Enforcer             // *casbin.Enforcer satisfies it; nil = no authz
    UserIDExtractor UserIDExtractor          // func(*http.Request) (UserID, error)
    ErrorHandler    ErrorHandler             // see Errors section
    LoginRedirect   string                   // default "/login"
    BeforeDispatch  BeforeDispatchHook       // func(ctx, *http.Request) ctx
    AfterDispatch   AfterDispatchHook        // func(ctx, *http.Request, error)
    Timeout         time.Duration            // bounds command/query handlers; <= 0 = no timeout (default)
    MaxBodySize     int64                    // max request body in bytes; <= 0 = no limit (default); larger → 413
    ServerTiming    func(*http.Request) bool // opt-in Server-Timing header predicate
    IncludeRequestIDInErrors bool            // default error handler adds the request ID
    IncludeInternalDetails bool              // default error handler keeps raw 5xx detail (redacted otherwise)
    ServiceName     string                   // event source identifier carried into event metadata
}
```

`App` methods you'll use:

| Method                                                 | Purpose                                                                                        |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| `app.Command(type, opts...) http.HandlerFunc`          | Build a command-dispatch handler. `type` is `command.Type("CreateItem")`. Empty string panics. |
| `app.Query(type, opts...) http.HandlerFunc`            | Build a query-dispatch handler.                                                                |
| `app.QuerySet(set, opts...) http.HandlerFunc`          | Run a query set (parallel loaders into one props struct) and render through `Views`. See the props-seam section. |
| `app.Middleware() func(http.Handler) http.Handler`     | Context enrichment (user/correlation/request IDs + deadline propagation). Put it innermost.    |
| `app.HealthHandler() http.HandlerFunc`                 | JSON `200`/`503` health probe.                                                                 |
| `app.RecoverHandler() func(http.Handler) http.Handler` | Panic recovery using the App's configured error handler.                                       |
| `app.EventOptions(ctx) []event.Option`                 | Build event options from context (user IDs, correlation/request IDs, deadline).                |

## `HandlerOption` catalogue

`app.Command`/`app.Query` take a variadic `...HandlerOption`. They run as an ordered pipeline. The two ordering rules:

1. **Authorization first** — fail fast before reading the body.
2. **Validator after decoder** — validators read the decoded value from context.

### Authorization

```go
cqrshtmx.Authorize(resource, action)   // requires auth + Enforcer allows (resource, action)
cqrshtmx.RequireAuth()                  // requires an authenticated user, no authz check
cqrshtmx.RequestGuard(func(r *http.Request, cmdOrQuery any) error)  // custom auth (runs after decode, before dispatch)
```

`RequestGuard` supports any auth model that doesn't fit Casbin: cookie-based ownership checks,
API key validation, JWT claims. The guard receives the decoded command/query (type-assert in your function).
Return an error to abort dispatch (passed to the error handler); return nil to proceed.

### Dispatch gate (authorization at the dispatcher, ADR 0063)

Route options only cover requests. These cover every dispatch, including a
service method, a seed, a job, or a CLI:

```go
gate, err := cqrshtmx.NewDispatchGate(checker, opts...)   // nil checker = error
cqrshtmx.RegisterCommand(gate, cmdDisp, "CreateItem", posture, why, handler)  // registers + declares
cqrshtmx.RegisterQuery(gate, qryDisp, "ListItems", posture, why, handler)
gate.Declare(posture)          // declaration only, for raw registrations
gate.CommandMiddleware()       // install OUTERMOST on the dispatcher
gate.QueryMiddleware()
gate.Validate()                // boot check; call after wiring
gate.Manifest()                // []DispatchPosture, sorted

cqrshtmx.AuthorizeCommand(checker, opts...)  // declaration-free: every command gated
cqrshtmx.AuthorizeQuery(checker, opts...)    // same for queries

// Gate options
cqrshtmx.AuditOnly()                    // observe denials, let dispatch through (one wave)
cqrshtmx.OnDecision(func(DispatchOutcome))  // metrics / OTel seam, allowed + denied
```

Postures are the route vocabulary: `PosturePublic`, `PostureAuthenticated`,
`PostureRoleGated`. Objects are `DispatchObject(kind, name)` =
`command/<Name>` / `query/<Name>`; actions are `ActionExecute` / `ActionRead`.
An undeclared type is denied (fail-closed) and named. A command that implements
`PolicyContext` overlays its own domain/resource/action.

### Decoders (generic — body/params → command or query)

```go
cqrshtmx.DecodeJSON[T](func(T) (command.Command, error))         // JSON body
cqrshtmx.DecodeForm[T](func(T) (command.Command, error))         // application/x-www-form-urlencoded
cqrshtmx.DecodeJSONQuery[T](func(T) (query.Query, error))        // JSON body → query
cqrshtmx.DecodeFormQuery[T](func(T) (query.Query, error))        // form data → query

// *http.Request-aware variants (for cookie/header access in the mapper):
cqrshtmx.DecodeJSONWithRequest[T](func(r *http.Request, body T) (command.Command, error))
cqrshtmx.DecodeFormWithRequest[T](func(r *http.Request, body T) (command.Command, error))
cqrshtmx.DecodeJSONQueryWithRequest[T](func(r *http.Request, body T) (query.Query, error))
cqrshtmx.DecodeFormQueryWithRequest[T](func(r *http.Request, body T) (query.Query, error))
```

### Validation (must follow the decoder)

```go
cqrshtmx.ValidateCommand(func(command.Command) error)
cqrshtmx.ValidateQuery(func(query.Query) error)
```

### Response / side effects

```go
cqrshtmx.Render(func(any, *http.Request) error)       // full-control renderer
cqrshtmx.RenderTempl(component)                        // render a templ.Component (duck-typed)
cqrshtmx.RenderTemplResult[T](func(T) cqrshtmx.TemplComponent) // map query result → templ
cqrshtmx.RenderJSON[T]()                               // JSON 200
cqrshtmx.RenderJSONStatus[T](status)                   // JSON with explicit status
cqrshtmx.RenderPaginatedJSON[T]()                      // query.PaginatedResult[T] → JSON
cqrshtmx.RenderHTML(htmlString)                         // static HTML response (text/html)
cqrshtmx.RenderPartialOrFull[T](partial, full)          // HTMX partial vs full page (templ, generic typed)
cqrshtmx.RenderPartialOrFullFunc(partial, full)          // HTMX partial vs full page (non-generic RenderFunc)
cqrshtmx.RenderIf(check, match, noMatch)                 // custom predicate render selector
cqrshtmx.Views[T](cqrshtmx.Page(fn), cqrshtmx.Region("#id", fn), cqrshtmx.OOB("id", fn)) // one props struct → page / region / OOB, selected from the request
cqrshtmx.ETagResults()                                   // AFTER a renderer: strong FNV ETag over rendered bytes, 304 on If-None-Match
cqrshtmx.Redirect(url)                                 // HTMX-aware (HX-Redirect for HTMX reqs)
cqrshtmx.PushURL(url)                                  // HTMX address-bar update
cqrshtmx.Trigger(event) / cqrshtmx.TriggerWithDetail(event, detail)  // HX-Trigger
cqrshtmx.WithSuccessStatus(code)
```

### Notifications (HX-Trigger toast-style events)

```go
cqrshtmx.NotifySuccess(msg) / NotifyError(msg) / NotifyWarning(msg) / NotifyInfo(msg)
cqrshtmx.NotifyWithEvent(name).Success(msg)  // custom event name
```

### Per-handler config

```go
cqrshtmx.WithTimeout(d)               // overrides Config.Timeout for this handler
cqrshtmx.WithMaxBodySize(n)            // overrides Config.MaxBodySize
cqrshtmx.RequireMethod(method)         // e.g. only allow POST
cqrshtmx.OnError(func(w, r, err))      // custom per-handler error handling
cqrshtmx.CSRFProtect(httputil.CSRFConfig) // per-handler CSRF (instead of middleware)
cqrshtmx.DecodePagination(r)           // helper: page/page_size → query.Pagination (defaults & validation)
```

## The HTMX `Response` builder

For handlers that need fine-grained HTMX control without going through the option pipeline:

```go
resp := cqrshtmx.NewResponse(w, r)
resp.Trigger("evt").
     PushURL("/path").
     Retarget("#el").
     Reswap(htmx.SwapInnerHTML).
     NotifySuccess("done").
     Redirect("/x").       // sanitizes for HTMX vs non-HTMX
     Refresh().
     Location("/x").
     ReplaceURL("/x").
     Reselect("#el").
     TriggerAfterSwap("evt").
     TriggerAfterSettle("evt").
     TriggerWithDetail(name, detail).
     CSRFToken(tok).
     Apply()               // writes headers — call once
```

Swap strategy constants: `SwapInnerHTML`, `SwapOuterHTML`, `SwapBeforeBegin`, `SwapAfterBegin`, `SwapBeforeEnd`, `SwapAfterEnd`, `SwapDelete`, `SwapNone`.

## Middleware catalogue

`cqrshtmx.Chain(mw1, mw2, ...)` composes middleware (outer first). Provided middleware:

| Middleware                                                               | Purpose                                                                                                                  |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------ |
| `RecoveryMiddleware`                                                     | Package-level panic recovery (uses `DefaultErrorHandler`).                                                               |
| `app.RecoverHandler()`                                                   | Panic recovery using the App's error handler.                                                                            |
| `httputil.SecurityHeaders(httputil.SecurityHeadersConfig{...})` | X-Content-Type-Options, X-Frame-Options, Referrer-Policy, optional CSP/HSTS. **Never auto-applied** (library principle). The root names `SecurityHeadersMiddleware`/`SecurityHeadersMiddlewareWithConfig` are deprecated v5-removal wrappers. |
| `HTMXMiddleware`                                                         | Detects HTMX requests, stores `htmx.Request` in context.                                                                 |
| `ContextEnrichmentMiddleware(extractor)`                                 | Generates correlation/request IDs, injects user ID. (`app.Middleware()` wraps this.)                                     |
| `httputil.CSRFMiddleware(httputil.CSRFConfig)`                           | justinas/nosurf integration with origin checks + proxy trust. The root alias `cqrshtmx.CSRFMiddleware` is deprecated.   |
| `httputil.CSRFResponseHeaderMiddleware`                                  | Repeats the request's CSRF token in the `httputil.DefaultCSRFHeaderName` response header, so `hx-headers` can echo it.  |
| `AuthorizeMiddleware(enforcer, res, act, extractor)`                     | Per-route authz.                                                                                                         |
| `httputil.KeyedRateLimiterMiddleware(httputil.KeyedRateLimiterConfig)`  | Keyed token bucket (`Limit`, `Window`, `Burst`, `KeyExtractor`, `TTL`); start from `httputil.DefaultKeyedRateLimiterConfig()`. |
| `RequestLogging(fmt, sink)` / `RequestLoggingSlog(logger)`               | Structured request logging.                                                                                              |

### Recommended order (outer → inner)

```
Recovery → SecurityHeaders → [your session/auth] → CSRF → HTMX → app.Middleware() → handler
```

The cqrs-htmx trio (`CSRF → HTMX → enrichment`) is non-negotiable: enrichment depends on HTMX context, and CSRF must wrap mutations.

### CSRFConfig essentials

```go
httputil.CSRFConfig{
    CookieName:    "...", HeaderName: "...", FieldName: "...",
    TrustedProxies: []string{"127.0.0.1", "10.0.0.0/8"}, // set in production for plain-HTTP origins
    SameSite:      http.SameSiteLaxMode,
    Secure:        true,
}
```

Helper tokens for templates take the request and return the token it carries: `httputil.CSRFTokenHTMLMeta(r)`, `httputil.CSRFTokenHXHeaders(r)`, `httputil.CSRFTokenFormField(r)`.

## Context IDs

Root IDs are **ULID-backed branded types** (via `go-cqrs-lite/id`). `cqrshtmx.UserID`, `identitymodel.UserID` and `usermgmt.UserID` are one type (`id.UserID`), so a user ID crosses these modules without conversion.

```go
cqrshtmx.NewUserID() / NewCorrelationID() / NewRequestID()
cqrshtmx.ParseUserID(s) / MustParseUserID(s)           // and Correlation/Request variants
cqrshtmx.WithUserID(ctx, id) / UserIDFromContext(ctx)
cqrshtmx.WithCorrelationID(ctx, id) / CorrelationIDFromContext(ctx)
cqrshtmx.WithRequestID(ctx, id)
cqrshtmx.EventOptionsFromContext(ctx) []event.Option    // carry user/correlation/request IDs into events
cqrshtmx.IsAuthenticated(r) bool
```

Context keys are collision-free empty-struct sentinels. Invalid correlation IDs in headers are logged at debug and dropped; invalid request IDs silently generate a fresh one.

## Errors and error→HTTP mapping

The library classifies errors into **families** and maps each to an HTTP status via `MapError(err) int`. First match wins:

1. an error that carries its own status (`httperr.WithHTTPStatus(err, 404)`, ADR-0034) — the way to answer 404 or any status a family cannot express;
2. the auth/HTTP sentinels (`httperr.ErrUnauthorized` 401, `ErrForbidden` 403, `ErrRequestTooLarge` 413, `ErrMethodNotAllowed` 405, `ErrUnsupportedMediaType` 415) and the panic code (500);
3. the family:

| Family         | HTTP | Meaning                                            |
| -------------- | ---- | -------------------------------------------------- |
| Rejection      | 400  | bad user input, parse/validation failure           |
| Conflict       | 409  | state conflict (duplicate email/credential)        |
| Transient      | 503  | retryable system/external (DB I/O, OAuth provider) |
| Corruption     | 500  | stored data damage (unmarshal/upcaster failure)    |
| Infrastructure | 503  | non-retryable bug (marshal failure, nil dep)       |

A nil error maps to 500; an untyped error maps to 503 (fail open as Transient). There is no `NotFound` family — use `httperr.WithHTTPStatus`.

Error handlers to set on `Config.ErrorHandler`:

- `DefaultErrorHandler`, `DefaultErrorHandlerWithRedirect` (text/plain, auth errors → HX-Redirect),
- `DefaultErrorHandlerWithRequestID`, `DefaultErrorHandlerWithRedirectAndRequestID`,
- `JSONErrorHandler`, `JSONErrorHandlerWithRedirect`.

`JSONErrorHandler` writes `{"error": "...", "status": NNN}` (plus `"code"` and `"request_id"` when available). The `"code"` field carries the machine-readable error code from the innermost errorfamily error (e.g., `"battle.exists"`), enabling programmatic error handling on the client.

### Error family rule (critical)

Inside these modules, **`errors.New` / `fmt.Errorf` (as error) / `errors.Join` are banned** in non-test code (enforced by `branching-flow errorfamily`). Every constructor lives in `github.com/larsartmann/go-error-family` (imported as `errorfamily`); `go-cqrs-lite/event/v4` only re-exposes the family constants (`event.Rejection`, `event.Conflict`, ...):

```go
errorfamily.NewRejection(code, msg)        // → 400
errorfamily.NewConflict(code, msg)         // → 409
errorfamily.NewTransient(code, msg)        // → 503
errorfamily.NewCorruption(code, msg)       // → 500
errorfamily.NewInfrastructure(code, msg)   // → 503
errorfamily.WrapInfrastructure(err, code, msg)  // classify an existing error
errorfamily.Wrapf(err, event.Conflict, code, msg) // wrap with an explicit family, formatted message
errorfamily.Wrapf(err, errorfamily.Classify(err), code, msg) // preserve the inner error's own family
errorfamily.Newf(event.Rejection, code, format, args...)     // formatted message
```

`fmt.Sprintf` is fine when building a **message string** (not an error object). When wrapping dispatch errors, **never force a family** — wrap with `errorfamily.Classify(err)` so a domain Rejection/Conflict survives.

## Pagination

```go
pq := cqrshtmx.DecodePagination(r)   // reads ?page & ?page_size, validates, applies defaults (1/20, max 100)
// pair with cqrshtmx.RenderPaginatedJSON[T]() which renders query.PaginatedResult[T]
```

Requesting a page beyond the last returns an empty page (standard REST, no silent clamping).

## Query sets and view sets (one props seam)

A real HTMX page is three views over one view model. Declare the reads as
loaders and the shapes as views; the request selects one:

```go
type tenantsProps struct {
    Rows   []Tenant
    Counts map[identitymodel.TenantID]int
    Nav    PageMeta // PageOnly: a region swap never pays for it
}

set := cqrshtmx.NewQuerySet(
    cqrshtmx.Load("ListTenants", listTenants,
        func(p *tenantsProps, rows []Tenant) { p.Rows = rows }),
    cqrshtmx.Load("TenantMemberCounts", countMembers,
        func(p *tenantsProps, c map[identitymodel.TenantID]int) { p.Counts = c }),
    cqrshtmx.Load("PageNav", loadNav,
        func(p *tenantsProps, nav PageMeta) { p.Nav = nav }).PageOnly(),
)

mux.Handle("GET /tenants", app.QuerySet(set,
    cqrshtmx.Views(
        cqrshtmx.Region("#tenant-rows", tenantsRows),
        cqrshtmx.Region("#tenant-other", tenantsOther).Default(),
        cqrshtmx.Page(tenantsPage),
        cqrshtmx.OOB("tenant-count", tenantCount),
    ),
    cqrshtmx.OnLoadError(func(r *http.Request, rep cqrshtmx.LoadReport) { /* visible degradation */ }),
    cqrshtmx.OnLoad(func(r *http.Request, rep cqrshtmx.LoadReport) {
        metrics.ObserveLoad(r.URL.Path, len(rep.Ran), len(rep.Skipped)) // every non-aborted load
    }),
))
```

- **Loaders are named functions, not queries.** `LoadFunc[V] = func(ctx) (V, error)`; dispatched queries are one adapter (`LoadQuery` + `FromDispatcher`) that keeps the `query/<Name>` posture. A loader never needs a dispatcher, so consumers that read a service directly use the same API.
- **Parallel, then one apply pass.** Loaders run concurrently; the props struct is written by a single goroutine after the join, so no field can race and `PageOnly` skipping is trivially correct.
- **Fail loud, degrade explicitly.** A non-optional failure aborts with the first failing loader in DECLARATION order (never completion order — a flake class avoided by design). `Optional()` is the only degradation path, and it is visible through `LoadReport` / `OnLoadError`.
- **`PageOnly()`** loaders are skipped for targeted HTMX requests, per `LoadScopeFor(r)` — the same rule `Views` uses to pick a fragment.
- **`Views[T]`** requires exactly one `Page` and at most one `Default` region (validated at registration; a zero-value `View`, a `Page` or `OOB` marked `Default`, or a view without a component panics at registration, every problem collected into one message). Selection: boosted/history-restore → Page; matching `HX-Target` → that region; partial with no match → the `Default` region, else an `unroutable_view` rejection (400) — never a silent full-page fallback.
- **The refusal is free.** Inside a QuerySet handler the selection runs BEFORE the loaders: an unroutable request is answered without a single read, and the rejection names the target and lists the shapes that would have matched (`routable: page, region #tenant-rows`).
- **Two load hooks.** `OnLoad(func(r *http.Request, rep LoadReport))` receives the account (`Ran`, `Skipped`, `Failures`) of every load that did not abort — a degraded run included, after `OnLoadError`; an aborted load never reaches it. `OnLoadError` replaces the default WARN log, so a metrics-only `OnLoad` cannot silence a degrading read.
- **Response parity.** `Trigger`, `TriggerWithDetail`, `Notify*`, `PushURL` and `Redirect` apply exactly as on `App.Query`: headers first, then the renderer; a `Redirect` for a non-HTMX client is terminal (303, nothing rendered).
- **A nil result applies as the zero value.** A loader returning a nil pointer or nil interface is not skipped and is not an error; any other type mismatch is a classified `cqrshtmx.queryset.apply_failed` error.
- **Two `App.Query` behaviours deliberately do not apply** (there is no request body to decode): no `Accept-Query` header is advertised, and `RequestGuard` never runs — gate the reads instead.
- **`OOB` views** render only on fragment responses, so a direct visit cannot leak an `hx-swap-oob` element into the document.
- **`QuerySetHandler(set, opts...)`** is the App-less form for a bare mux (no dispatch pipeline, no authz gate — your middleware owns those).
- **`ETagResults()`** composes after `Views`: the tag hashes the rendered bytes, so it is 304-correct across all three shapes.
- **Cost:** a page or region render is 5 allocs/op (identical to `RenderPartialOrFull`); each out-of-band view adds 8; the loader scheduler's floor is 8 for one trivial loader. Measured by `benchmark_views_test.go`.
- **Availability:** unreleased until the next root tag (planned `v4.20.0`). Until that tag exists, only in-repo code can call it — examples resolve published tags, so a demo cannot use it early.

## Utilities

```go
cqrshtmx.WriteJSON(w, status, v)
cqrshtmx.RenderTemplComponent(w, r, partial, full) // standalone partial-vs-full (non-CQRS routes)
cqrshtmx.OOBHTML(id, html, swapStrategy...)        // wrap HTML with hx-swap-oob (string form)
cqrshtmx.OOBTempl(id, component, swapStrategy...)  // same wire bytes, component form (byte-identical to OOBHTML)
cqrshtmx.LoadScopeFor(r)                           // page vs fragment scope (the PageOnly-loader rule; Views uses it too)
httputil.KeyExtractorFromClientIP()     // proxy-aware rate-limit key extractor (uses httputil.ClientIP)
htmx.ScriptHandler()            // embedded htmx 2.0.11 (GET/HEAD, ETag, 1y cache)
htmx.ScriptHandler4()           // embedded htmx 4.0.1 (native hx-query)
htmx.ScriptHandlerWith(js, ver) // serve a custom build
htmx.ScriptTag(path)            // <script src="path"></script> for templates
htmx.CDNScriptTag(version)      // CDN script tag; "" = embedded version
htmx.Version()                  // "2.0.11"
htmx.ExtensionHandler(name)     // serve one embedded extension: sse, idiomorph, cqrs-query
htmx.ExtensionsHandler(names…)  // serve concatenated bundle of extensions
htmx.ExtensionCDNScriptTag(name) // CDN <script> tag for one extension
htmx.ExtensionVersion(name)     // "2.2.4" (sse), "0.8.0" (idiomorph), "1.0.0" (cqrs-query)
htmx.ExtensionNames()           // ["cqrs-query", "idiomorph", "sse"]
httputil.KeyExtractorFromRemoteAddr()   // default rate-limit key extractor
httputil.DefaultKeyedRateLimiterConfig() // keyed limiter defaults (the root RateLimiter* names left with the httputil move)
httputil.SecurityHeaderSkip             // sentinel "-" to suppress a default security header
httputil.RecommendedCSP / httputil.RecommendedHSTS // suggested header values (never auto-applied)
cqrshtmx.SSEEventConnected / SSEEventHeartbeat // standard SSE event name constants
httputil.CSRFTestToken(mw)              // test helper: extracts a valid CSRF token from a middleware chain
```
