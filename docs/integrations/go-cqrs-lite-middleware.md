# Integration: go-cqrs-lite Middleware

cqrs-htmx provides **HTTP middleware** (rate limiting, CSRF, security headers, recovery).
go-cqrs-lite provides **CQRS dispatch middleware** (retry, circuit breaker, metrics, tracing).

These are different layers — they compose, they don't conflict.

## How They Fit Together

```
HTTP Request
    │
    ▼
┌─────────────────────────┐
│  cqrs-htmx HTTP Layer   │
│  (CSRF, RateLimit,      │
│   SecurityHeaders,      │
│   Recovery, Context)    │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  cqrs-htmx App.Command  │
│  (Decode → Auth →       │
│   Dispatch)             │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  go-cqrs-lite Dispatch  │
│  Middleware             │
│  (Retry, CircuitBreaker,│
│   Metrics, Tracing,     │
│   Validation)           │
└───────────┬─────────────┘
            │
            ▼
        Handler
```

## Recommended go-cqrs-lite Middleware

Install: `go get github.com/larsartmann/go-cqrs-lite/middleware/v2`

### Retry

```go
import "github.com/larsartmann/go-cqrs-lite/middleware/v2"

cmds.Use(middleware.CommandRetry(
    middleware.WithMaxRetries(3),
    middleware.WithBackoff(time.Second),
))
```

### Circuit Breaker

```go
cmds.Use(middleware.CommandCircuitBreaker(
    middleware.WithFailureThreshold(5),
    middleware.WithResetTimeout(30 * time.Second),
))
```

### Validation

```go
cmds.Use(middleware.CommandValidation())
```

### Metrics

```go
cmds.Use(middleware.CommandMetrics())
```

### Tracing (OpenTelemetry)

```go
cmds.Use(middleware.CommandTracing(tracer))
```

## Why cqrs-htmx Doesn't Re-export These

1. **Different layer**: cqrs-htmx wraps HTTP; middleware wraps dispatch
2. **Consumer choice**: Not everyone needs retry or circuit breakers
3. **Dependency isolation**: middleware/v2 adds no deps that aren't already transitive

## cqrs-htmx Equivalents

| go-cqrs-lite Middleware | cqrs-htmx HTTP Equivalent |
|------------------------|--------------------------|
| `CommandRecovery` | `RecoveryMiddleware` / `App.RecoverHandler()` |
| `CommandMetrics` | `RequestLogging` / `RequestLoggingSlog` |
| `CommandTracing` | `BeforeDispatchHook` / `AfterDispatchHook` (wire OTel manually) |
| — | `CSRFMiddleware` (HTTP-only concern) |
| — | `RateLimiterMiddleware` (HTTP-only concern) |
| — | `SecurityHeadersMiddleware` (HTTP-only concern) |

## Summary

Use both. Configure go-cqrs-lite middleware on your dispatchers at setup time.
Configure cqrs-htmx HTTP middleware on your router. They operate at different layers.
