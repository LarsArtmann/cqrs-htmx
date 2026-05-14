# Dependency Analysis: cqrs-htmx

> **Date:** 2026-05-14
> **State:** Pre-modularization baseline

---

## Current State (Single Module)

### External Dependencies

```
github.com/larsartmann/cqrs-htmx
├── github.com/casbin/casbin/v3 v3.10.0          (test only — never imported in source)
├── github.com/cockroachdb/errors v1.13.0          (production)
├── github.com/larsartmann/go-cqrs-lite/core v1.1.0 (production)
├── github.com/onsi/ginkgo/v2 v2.28.3              (test)
├── github.com/onsi/gomega v1.40.0                  (test)
└── (22 indirect dependencies)
```

### Internal File Dependency Graph

```
app.go
├── authz.go       (Enforcer, UserIDExtractor)
├── context.go     (UserIDFromContext, ParseUserID, WithUserID)
├── errors.go      (ErrCommandsNil, ErrQueriesNil, ErrorHandler, defaultLoginRedirect)
├── options.go     (HandlerOption, buildHandlerConfig)
└── handler.go     (handleCommandDispatch, handleQueryDispatch)

handler.go
├── authz.go       (Enforce via App.executeAuthorization)
├── context.go     (UserIDFromContext)
├── errors.go      (ErrDispatchFailed, ErrDecoderMissing)
├── options.go     (handlerConfig, CommandDecoder, QueryDecoder, RenderFunc)
└── response.go    (NewResponse, applyHTMXResponse)

authz.go
├── errors.go      (ErrForbidden, ErrUnauthorized, ErrEnforcerNil, DefaultErrorHandlerWithRedirect)
├── options.go     (HandlerOption, handlerConfig)
└── htmx.go        (indirect via errors.go → headerRedirect)

errors.go
├── htmx.go        (IsHTMXRequest, headerRedirect)
└── go-cqrs-lite/core/event (classification system)

options.go
├── errors.go      (ErrDecodeFailed)
├── context.go     (UserIDFromContext)
├── response.go    (NewResponse, applyHTMXResponse)
├── htmx.go        (header constants)
└── go-cqrs-lite/core/command, query

notify.go
└── options.go     (HandlerOption, TriggerWithDetail)

middleware.go
├── context.go     (ParseUserID, WithUserID)
├── htmx.go        (parseHTMXRequest, WithHTMX)
└── authz.go       (UserIDExtractor type)

context.go
└── go-cqrs-lite/core/event, pkg/id

htmx.go
└── (stdlib only — zero internal/external deps)

response.go
├── htmx.go        (IsHTMXRequest, header constants, SwapStrategy)
└── notify.go      (defaultNotificationEvent constant)
```

### Coupling Analysis

| File            | Depends On (count)                            |                                    Dependents (count) | Coupling Level                  |
| --------------- | --------------------------------------------- | ----------------------------------------------------: | ------------------------------- |
| `htmx.go`       | 0                                             | 6 (errors, options, response, middleware, + indirect) | **Low** — leaf node             |
| `context.go`    | 2 (external)                                  |         5 (app, handler, options, middleware, errors) | Low — utility                   |
| `notify.go`     | 1 (options)                                   |                                          1 (response) | Low — narrow                    |
| `middleware.go` | 3 (context, htmx, authz)                      |                                                     0 | Low — consumed externally       |
| `response.go`   | 2 (htmx, notify)                              |                                  2 (options, handler) | Medium — core to both paths     |
| `errors.go`     | 2 (htmx, event)                               |            5 (app, handler, authz, options, coverage) | Medium — cross-cutting          |
| `authz.go`      | 2 (errors, options)                           |                          3 (app, handler, middleware) | High — depends on handlerConfig |
| `options.go`    | 4 (errors, context, response, htmx)           |                       4 (app, handler, authz, notify) | **High** — central hub          |
| `handler.go`    | 5 (authz, context, errors, options, response) |                                               1 (app) | High — dispatch orchestrator    |
| `app.go`        | 5 (authz, context, errors, options, handler)  |                                                     0 | High — entry point              |

### Key Insight

`options.go` is the **coupling hub** — it defines `handlerConfig` which is the shared mutable state for the entire handler pipeline. Every module that produces `HandlerOption` values must know about `handlerConfig`. This makes `authz.go` and `notify.go` impossible to extract without either:

1. Moving `handlerConfig` to a shared types module, OR
2. Introducing an interface abstraction for handler configuration

---

## Proposed State (After htmx/ Extraction)

```
github.com/larsartmann/cqrs-htmx (root)
├── github.com/larsartmann/cqrs-htmx/htmx  (NEW)
├── github.com/cockroachdb/errors
├── github.com/larsartmann/go-cqrs-lite/core
└── (test deps)

github.com/larsartmann/cqrs-htmx/htmx
└── (zero external deps — stdlib only)
```

### Dependency DAG (Post-Extraction)

```
root (app, handler, options, authz, errors, context, middleware, notify)
 │
 └──→ htmx (htmx.go, response.go)
       └──→ (nothing — leaf)
```

### What Moves to `htmx/`

| Symbol                         | Type      | Notes                                |
| ------------------------------ | --------- | ------------------------------------ |
| `HTMXRequest`                  | struct    | Core HTMX request type               |
| `SwapStrategy`                 | type      | String-based swap strategy           |
| `SwapInnerHTML` ... `SwapNone` | constants | 8 swap strategy values               |
| `IsHTMXRequest`                | func      | HTMX detection                       |
| `IsBoosted`                    | func      | Boost detection                      |
| `IsHistoryRestore`             | func      | History restore detection            |
| `RenderPartial`                | func      | Partial render detection             |
| `HTMXTarget`                   | func      | Target element accessor              |
| `HTMXTrigger`                  | func      | Trigger ID accessor                  |
| `HTMXTriggerName`              | func      | Trigger name accessor                |
| `HTMXPrompt`                   | func      | Prompt accessor                      |
| `HTMXCurrentURL`               | func      | Current URL accessor                 |
| `WithHTMX`                     | func      | Context storage                      |
| `HTMXFromContext`              | func      | Context retrieval                    |
| `Response`                     | struct    | HTMX response builder                |
| `NewResponse`                  | func      | Response constructor                 |
| `Response.*` (all methods)     | methods   | 20+ fluent builder methods           |
| `defaultNotificationEvent`     | const     | "showMessage" — moves from notify.go |
| All HTMX header constants      | const     | `HX-Request`, `HX-Redirect`, etc.    |

### What Stays in Root (with re-exports)

All of the above symbols will be re-exported from the root package for backward compatibility.

| Symbol                                                   | Type                | Reason it stays                        |
| -------------------------------------------------------- | ------------------- | -------------------------------------- |
| `App`, `Config`, `New`                                   | types/func          | Core CQRS wiring                       |
| `HandlerOption`                                          | type                | Depends on `handlerConfig`             |
| `handlerConfig`                                          | struct (unexported) | Shared mutable config — coupling hub   |
| `Authorize`, `RequireAuth`                               | func                | Return `HandlerOption`                 |
| `Enforce`, `AuthorizeMiddleware`                         | func                | Authorization enforcement              |
| `Enforcer`                                               | interface           | Casbin duck-type                       |
| `UserIDExtractor`                                        | type                | Used by middleware + app               |
| `DecodeJSON/Query/Form/FormQuery`                        | func                | Generic decoders → `HandlerOption`     |
| `Render`, `RenderTempl`, `RenderTemplResult`             | func                | Rendering → `HandlerOption`            |
| `Redirect`, `Trigger`, `TriggerWithDetail`, `PushURL`    | func                | Response options → `HandlerOption`     |
| `NotifySuccess/Error/Warning/Info`                       | func                | Notification options → `HandlerOption` |
| `NotifyWithEvent`, `NotifyEventBuilder`                  | func/type           | Notification builder → `HandlerOption` |
| `ErrorHandler`, `MapError`, `DefaultErrorHandler*`       | func/type           | Error classification                   |
| `Err*` sentinels (8)                                     | var                 | CQRS error classification              |
| `UserID`, `NewUserID`, `ParseUserID`, etc.               | func/type           | Context identity                       |
| `ContextEnrichmentMiddleware`, `HTMXMiddleware`, `Chain` | func                | HTTP middleware                        |
