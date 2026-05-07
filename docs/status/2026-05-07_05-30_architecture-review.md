# Architecture Review — cqrs-htmx

**Date:** 2026-05-07 | **Reviewer:** Senior Staff Architect

## Scalability & Modularity Assessment

### Overall Architecture: 8/10

This is a well-designed integration library. The flat package structure is appropriate for a library (not an application). The API surface is small, composable, and framework-agnostic.

### Strengths

1. **Framework-agnostic**: Pure `net/http` — works with Chi, Gin, Echo, or stdlib mux. No framework lock-in.
2. **Option pattern**: `HandlerOption` pattern is idiomatic Go. Composable, extensible, and testable.
3. **Duck-typed templ integration**: `TemplComponent` interface matches `templ.Component` without importing templ. Zero coupling to a specific template engine.
4. **CQRS error classification**: Automatic mapping from domain errors to HTTP status codes. Clean integration with `go-cqrs-lite` event classification.
5. **HTMX dual-path**: Accessors work with or without `HTMXMiddleware`. Graceful degradation.
6. **Small public API**: 10 source files, ~1100 LOC. Easy to understand, hard to misuse.

### Weaknesses

1. **Mutable globals**: `LoginRedirect` and `NotificationEvent` are package-level `var`. Multiple `App` instances with different configs will race. Should be per-App configuration.
2. **No interface for Casbin**: `*casbin.Enforcer` is a concrete type. Hard to test or mock without a real enforcer. Should accept an interface with `Enforce(sub, res, act) (bool, error)`.
3. **No timeout propagation**: The `context.Context` flows to `command.Dispatch`/`query.Dispatch`, but the library doesn't set any deadline. Consumers must handle this externally.
4. **No observability hooks**: No logging, metrics, or tracing middleware. No way to hook into the dispatch pipeline for monitoring.

### Service Orientation Assessment

For a **library**, this is appropriately structured. It's not a service — it's a SDK that consumers import. The key question is: **can consumers extend it without forking?**

| Extension Point      | Status                                                        |
| -------------------- | ------------------------------------------------------------- |
| Custom error handler | Configurable via `Config.ErrorHandler`                        |
| Custom decoder       | Any `HandlerOption` that sets `commandDecoder`/`queryDecoder` |
| Custom renderer      | `Render(func)` / `RenderTempl` / `RenderTemplResult`          |
| Custom authorization | `AuthorizeMiddleware` or custom `HandlerOption`               |
| Custom response      | `Response` builder is public and composable                   |
| Custom notifications | Override `NotificationEvent` var                              |

Missing extension points:

- No middleware hooks in the dispatch pipeline (pre/post dispatch)
- No way to add custom response headers without a custom render function
- No plugin system for adding new HandlerOption categories

### Composability Assessment

**High composability.** The option pattern, middleware chain, and response builder all compose well.

```
app.Command("CreateUser",
    cqrshtmx.Authorize("users", "create"),     // auth
    cqrshtmx.DecodeJSON(mapper),               // decode
    cqrshtmx.NotifySuccess("Created"),         // notification
    cqrshtmx.PushURL("/users"),                // HTMX
)
```

This declarative style is excellent. Each concern is a single option.

### Recommendations (Priority Order)

1. **Move globals to per-App config**: `LoginRedirect` and `NotificationEvent` should be fields on `App`, accessed through `handlerConfig` or `Response`. Breaking change — requires major version bump.

2. **Extract Casbin interface**: Define `Enforcer interface { Enforce(...) (bool, error) }` instead of using `*casbin.Enforcer` directly. Non-breaking (backward compatible with structural typing).

3. **Add dispatch lifecycle hooks**: `OnBeforeDispatch` / `OnAfterDispatch` options for logging, metrics, and tracing. Non-breaking addition.

4. **Reduce handleCommandDispatch complexity**: Extract the response-finalization logic into a separate function. The empty `else if` block is a code smell.

5. **Fix XSS surface**: Sanitize error messages in `DefaultErrorHandler` before writing to response.

6. **Add request validation middleware**: Optional schema validation in the decode pipeline.

## Modularity Scorecard

| Dimension          | Score | Notes                                               |
| ------------------ | ----- | --------------------------------------------------- |
| API surface        | 9/10  | Small, clean, well-documented                       |
| Framework coupling | 10/10 | Pure net/http                                       |
| Testability        | 8/10  | Good coverage, concrete Casbin makes mocking harder |
| Extensibility      | 7/10  | Options pattern is great, but no pipeline hooks     |
| Type safety        | 9/10  | Generics for decoders, duck-typing for templ        |
| Error handling     | 8/10  | Good classification, XSS concern                    |
| Documentation      | 9/10  | Excellent README, godoc comments, examples          |
| File organization  | 8/10  | Flat package appropriate for library size           |
