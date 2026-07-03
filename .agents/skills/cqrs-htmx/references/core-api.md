# Core API reference (root module)

Import: `cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"`

This is the deep reference for the root module's `App`, `HandlerOption`s, middleware, context IDs, errors, and the HTMX `Response` builder. Load this when you need exact signatures or the full option catalogue.

## `App` and `Config`

```go
app, err := cqrshtmx.New(cfg)        // errors if both Commands and Queries are nil
app := cqrshtmx.MustNew(cfg)          // panics on error
```

```go
type Config struct {
    Commands  *command.Dispatcher           // go-cqrs-lite/command/v3
    Queries   *query.Dispatcher             // go-cqrs-lite/query/v3
    Enforcer  cqrshtmx.Enforcer             // *casbin.Enforcer satisfies it; nil = no authz
    UserIDExtractor UserIDExtractor          // func(*http.Request) (UserID, error)
    ErrorHandler    ErrorHandler             // see Errors section
    LoginRedirect   string                   // default "/login"
    BeforeDispatch  BeforeDispatchHook       // func(ctx, *http.Request) ctx
    AfterDispatch   AfterDispatchHook        // func(ctx, *http.Request, error)
    Timeout         time.Duration            // applies to Dispatch only, not decode/auth
    MaxBodySize     int64                    // 0 + per-handler 0 → DefaultMaxBodySize (10MB)
    ServerTiming    func(*http.Request) bool // opt-in Server-Timing header predicate
    IncludeRequestIDInErrors bool            // picks request-ID-aware error handlers
    ServiceName     string                   // labels for logs/metrics
}
```

`App` methods you'll use:

| Method                                                 | Purpose                                                                                        |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| `app.Command(type, opts...) http.HandlerFunc`          | Build a command-dispatch handler. `type` is `command.Type("CreateItem")`. Empty string panics. |
| `app.Query(type, opts...) http.HandlerFunc`            | Build a query-dispatch handler.                                                                |
| `app.Middleware() func(http.Handler) http.Handler`     | Context enrichment (user/correlation/request IDs + deadline propagation). Put it innermost.    |
| `app.HealthHandler() http.HandlerFunc`                 | JSON `200`/`503` health probe.                                                                 |
| `app.RecoverHandler() func(http.Handler) http.Handler` | Panic recovery using the App's configured error handler.                                       |
| `app.DispatchWSCommand(r, type, decoder, data) error`  | Bridge a WebSocket message into CQRS (no response writing). See realtime.md.                   |
| `app.EventOptions(ctx) []event.Option`                 | Build event options from context (user IDs, correlation/request IDs, deadline).                |

## `HandlerOption` catalogue

`app.Command`/`app.Query` take a variadic `...HandlerOption`. They run as an ordered pipeline. The two ordering rules:

1. **Authorization first** — fail fast before reading the body.
2. **Validator after decoder** — validators read the decoded value from context.

### Authorization

```go
cqrshtmx.Authorize(resource, action)   // requires auth + Enforcer allows (resource, action)
cqrshtmx.RequireAuth()                  // requires an authenticated user, no authz check
```

### Decoders (generic — body/params → command or query)

```go
cqrshtmx.DecodeJSON[T](func(T) (command.Command, error))         // JSON body
cqrshtmx.DecodeForm[T](func(T) (command.Command, error))         // application/x-www-form-urlencoded
cqrshtmx.DecodeJSONQuery[T](func(T) (query.Query, error))
cqrshtmx.DecodeFormQuery[T](func(*http.Request) (query.Query, error))  // read query params with r
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
cqrshtmx.CSRFProtect(cqrshtmx.CSRFConfig) // per-handler CSRF (instead of middleware)
cqrshtmx.DecodePagination(r)           // helper: page/page_size → query.Pagination (defaults & validation)
```

## The HTMX `Response` builder

For handlers that need fine-grained HTMX control without going through the option pipeline:

```go
resp := cqrshtmx.NewResponse(w, r)
resp.Trigger("evt").
     PushURL("/path").
     Retarget("#el").
     Reswap(cqrshtmx.SwapInnerHTML).
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
| `SecurityHeadersMiddleware` / `SecurityHeadersMiddlewareWithConfig(cfg)` | X-Content-Type-Options, X-Frame-Options, Referrer-Policy, optional CSP/HSTS. **Never auto-applied** (library principle). |
| `HTMXMiddleware`                                                         | Detects HTMX requests, stores `HTMXRequest` in context.                                                                  |
| `ContextEnrichmentMiddleware(extractor)`                                 | Generates correlation/request IDs, injects user ID. (`app.Middleware()` wraps this.)                                     |
| `CSRFMiddleware(cqrshtmx.CSRFConfig)`                                    | justinas/nosurf integration with origin checks + proxy trust.                                                            |
| `CSRFResponseHeaderMiddleware`                                           | Exposes CSRF token via a response header.                                                                                |
| `AuthorizeMiddleware(enforcer, res, act, extractor)`                     | Per-route authz.                                                                                                         |
| `RateLimiterMiddleware(RateLimitConfig)`                                 | Token-bucket, pluggable `KeyExtractor`, TTL eviction, `MaxKeys` cap.                                                     |
| `RequestLogging(fmt, sink)` / `RequestLoggingSlog(logger)`               | Structured request logging.                                                                                              |

### Recommended order (outer → inner)

```
Recovery → SecurityHeaders → [your session/auth] → CSRF → HTMX → app.Middleware() → handler
```

The cqrs-htmx trio (`CSRF → HTMX → enrichment`) is non-negotiable: enrichment depends on HTMX context, and CSRF must wrap mutations.

### CSRFConfig essentials

```go
cqrshtmx.CSRFConfig{
    CookieName:    "...", HeaderName: "...", FieldName: "...",
    TrustedProxies: []string{"127.0.0.1", "10.0.0.0/8"}, // set in production for plain-HTTP origins
    SameSite:      http.SameSiteLaxMode,
    Secure:        true,
}
```

Helper tokens for templates: `CSRFTokenHTMLMeta(tok)`, `CSRFTokenHXHeaders(tok)`, `CSRFTokenFormField(tok)`.

## Context IDs

Root IDs are **ULID-backed branded types** (via `go-cqrs-lite/id`). They are distinct from usermgmt's string-backed `UserID`.

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

The library classifies errors into **families** and maps each to an HTTP status via `MapError(err) int`:

| Family         | HTTP | Meaning                                            |
| -------------- | ---- | -------------------------------------------------- |
| Rejection      | 400  | bad user input, parse/validation failure           |
| NotFound       | 404  | resource missing                                   |
| Conflict       | 409  | state conflict (duplicate email/credential)        |
| Transient      | 503  | retryable system/external (DB I/O, OAuth provider) |
| Corruption     | 422  | stored data damage (unmarshal/upcaster failure)    |
| Infrastructure | 500  | non-retryable bug (marshal failure, nil dep)       |

Error handlers to set on `Config.ErrorHandler`:

- `DefaultErrorHandler`, `DefaultErrorHandlerWithRedirect` (text/plain, auth errors → HX-Redirect),
- `DefaultErrorHandlerWithRequestID`, `DefaultErrorHandlerWithRedirectAndRequestID`,
- `JSONErrorHandler`, `JSONErrorHandlerWithRedirect`.

### Error family rule (critical)

Inside these modules, **`errors.New` / `fmt.Errorf` (as error) / `errors.Join` are banned** in non-test code (enforced by `branching-flow errorfamily`). Use the constructors re-exported via `go-cqrs-lite/event/v3`:

```go
event.NewRejection(code, msg)        // → 400
event.NewConflict(code, msg)         // → 409
event.NewTransient(code, msg)        // → 503
event.NewCorruption(code, msg)       // → 422
event.NewInfrastructure(code, msg)   // → 500
event.WrapRejection(err, code, msg)  // classify an existing error
event.Wrapf(err, event.Classify(err), code, msg)  // preserve the inner error's own family
event.Newf(code, msg)                // build a message-string-based error
```

`fmt.Sprintf` is fine when building a **message string** (not an error object). When wrapping dispatch errors, **never force a family** — use `event.Wrapf(err, event.Classify(err), ...)` so a domain Rejection/Conflict survives.

## Pagination

```go
pq := cqrshtmx.DecodePagination(r)   // reads ?page & ?page_size, validates, applies defaults (1/20, max 100)
// pair with cqrshtmx.RenderPaginatedJSON[T]() which renders query.PaginatedResult[T]
```

Requesting a page beyond the last returns an empty page (standard REST, no silent clamping).

## Utilities

```go
cqrshtmx.WriteJSON(w, status, v)
cqrshtmx.KeyExtractorFromClientIP()     // proxy-aware rate-limit key extractor (uses httputil.ClientIP)
cqrshtmx.HTMXScriptHandler()            // embedded htmx 2.0.10 (GET/HEAD, ETag, 1y cache)
cqrshtmx.HTMXScriptHandlerWith(js, ver) // serve a custom build
cqrshtmx.HTMXScriptTag(path)            // <script src="path"></script> for templates
cqrshtmx.HTMXCDNScriptTag(version)      // CDN script tag; "" = embedded version
cqrshtmx.HTMXVersion()                  // "2.0.10"
cqrshtmx.HTMXExtensionHandler(name)     // serve embedded extension (sse/ws/idiomorph)
cqrshtmx.HTMXExtensionsHandler(names…)  // serve concatenated bundle of extensions
cqrshtmx.HTMXExtensionCDNScriptTag(name) // CDN <script> tag for one extension
cqrshtmx.HTMXExtensionVersion(name)     // "2.2.4" (sse), "2.0.4" (ws), "0.7.4" (idiomorph)
cqrshtmx.HTMXExtensionNames()           // ["idiomorph", "sse", "ws"]
cqrshtmx.KeyExtractorFromRemoteAddr()   // default rate-limit key extractor
cqrshtmx.RecommendedCSP / RecommendedHSTS // suggested header values (never auto-applied)
```
