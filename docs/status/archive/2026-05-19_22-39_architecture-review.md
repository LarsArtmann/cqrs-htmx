# Architecture Review — cqrs-htmx

**Date:** 2026-05-19_22-39

## 1. Scalability

### Current Architecture

```
Consumer → HTTP Handler → [Middleware Chain] → App.Command/Query → [Decode → Auth → CSRF → Dispatch] → Response
```

### Scalability Assessment: **Good for a library**

- **Horizontal**: Stateless by design. All middleware and handlers take `(ResponseWriter, *Request)`. No shared state between requests.
- **Per-handler configuration**: `HandlerOption` pattern allows each route to have independent auth, validation, timeout, and CSRF settings. No global mutable state.
- **Rate limiter**: The only stateful component. Uses per-key token buckets with TTL eviction. Adequate for moderate scale; needs bounded map for extreme scale (see code review H1).

### Scalability Concerns

| Concern                               | Severity | Mitigation                                 |
| ------------------------------------- | -------- | ------------------------------------------ |
| Rate limiter unbounded map            | Medium   | Add max-keys cap or periodic cleanup       |
| CSRF per-handler Protect() allocation | Low      | Cache Protect instance per handler         |
| No circuit breaker pattern            | Low      | Consumers can add their own via middleware |

### Scalability Rating: **8/10** — Stateless by design, per-handler config, minor bounded-state concern.

---

## 2. Modularity

### Module Structure

```
cqrs-htmx/             ← Flat package, 15 files, ~100 exports
├── app.go             ← App builder (entry point)
├── handler.go         ← Dispatch pipeline
├── options.go         ← HandlerOption API
├── response.go        ← HTMX Response builder
├── authz.go           ← Authorization
├── context.go         ← Context types (UserID, CorrelationID, RequestID)
├── csrf.go            ← CSRF protection
├── errors.go          ← Error classification
├── htmx.go            ← HTMX request parsing
├── logging.go         ← Request logging
├── middleware.go       ← Middleware chain
├── notify.go          ← Notifications
├── decoder.go         ← Body decoding
├── ratelimit.go       ← Rate limiting
├── security.go        ← Security headers
└── usermgmt/          ← Sub-module (independent go.mod)
    ├── authz.go
    ├── service.go
    ├── http.go
    ├── ...
```

### Modularity Assessment: **Appropriate for library scale**

**Strengths:**

- Each file has a single concern (SRP at file level)
- `usermgmt/` correctly extracted as independent module
- `HandlerOption` pattern is composable — consumers mix and match options
- `Enforcer` interface enables adapter pattern for authorization
- `ErrorHandler` is injectable — consumers provide their own

**Weaknesses:**

- `csrf.go` (445 lines) combines config, middleware, template helpers, and per-handler protection
- `options.go` (282 lines) combines decoders, render options, validation, and response options
- No sub-packages for related but independent concerns (e.g., `htmx/` for pure HTMX helpers)

### Modularity Rating: **7/10** — Well-organized flat package, appropriate for library scale.

---

## 3. Service Orientation

### Assessment: **N/A — This is a library, not a service**

The library provides building blocks for consumers to create service-oriented architectures. It doesn't impose a service pattern.

**What it provides for SOA:**

- Middleware composition (`Chain`) enables layered service architecture
- `Enforcer` interface decouples from specific authorization backends
- `ErrorHandler` injection enables service-specific error responses
- `BeforeDispatchHook`/`AfterDispatchHook` enable cross-cutting concerns (tracing, metrics)

---

## 4. Composability

### Assessment: **Excellent**

The library is designed around composable primitives:

| Primitive                       | Composition Mechanism                                                                            |
| ------------------------------- | ------------------------------------------------------------------------------------------------ |
| `HandlerOption`                 | Variadic opts pattern — `DecodeJSON`, `Authorize`, `NotifySuccess`, `WithTimeout` compose freely |
| `Chain(mw1, mw2, ...)`          | Middleware composition left-to-right                                                             |
| `Response` builder              | Fluent API — `PushURL().Trigger().NotifySuccess()` chains                                        |
| `Enforcer` interface            | Adapter pattern — `*casbin.Enforcer`, mocks, fakes                                               |
| `ErrorHandler`                  | Strategy pattern — inject custom error handling                                                  |
| `UserIDExtractor`               | Function type — inject any auth source (JWT, session, API key)                                   |
| `RenderFunc` / `TemplComponent` | Duck-typed rendering — any template engine                                                       |
| `KeyExtractor`                  | Function type — any rate-limit key source                                                        |

**Composability Rating: 9/10** — Every extension point is an interface or function type.

---

## 5. Concrete Recommendations

### Make more composable

| # | Recommendation                                                        | Impact            | Effort |
| - | --------------------------------------------------------------------- | ----------------- | ------ |
| 1 | Split `csrf.go` into `csrf.go` + `csrf_helpers.go`                    | File readability  | Low    |
| 2 | Extract rate limiter max-keys config option                           | Production safety | Low    |
| 3 | Add `Response.RedirectWithStatus(code int)` for custom redirect codes | Composability     | Low    |
| 4 | Consider `WithDecoder(d CommandDecoder)` for fully custom decoders    | Extensibility     | Low    |

### Make more modular

| # | Recommendation                                      | Impact          | Effort |
| - | --------------------------------------------------- | --------------- | ------ |
| 5 | Deduplicate HTMX accessor pattern (8 → 1 generic)   | Maintainability | Low    |
| 6 | Deduplicate decoder pattern (4 → 1 generic)         | Maintainability | Low    |
| 7 | Deduplicate notification surface (12 → shared impl) | Maintainability | Low    |

---

## Summary

| Dimension     | Rating   | Key Insight                                           |
| ------------- | -------- | ----------------------------------------------------- |
| Scalability   | 8/10     | Stateless by design, minor bounded-state concern      |
| Modularity    | 7/10     | Well-organized flat package, appropriate for library  |
| Composability | 9/10     | Every extension point is injectable                   |
| Overall       | **8/10** | Excellent library architecture with minor duplication |
