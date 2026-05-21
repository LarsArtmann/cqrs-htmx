# Dependency Analysis: cqrs-htmx

> **Date:** 2026-05-22
> **State:** Updated with current module structure

---

## Module Structure (4 Modules)

```
github.com/larsartmann/cqrs-htmx (root)
├── casbin/casbin/v3 v3.10.0
├── cockroachdb/errors v1.13.0
├── gorilla/csrf v1.7.3
├── larsartmann/go-cqrs-lite/core v1.4.0
├── onsi/ginkgo/v2 v2.29.0 (test)
├── onsi/gomega v1.41.0 (test)
└── golang.org/x/time v0.15.0

github.com/larsartmann/cqrs-htmx/usermgmt
├── casbin/casbin/v3 v3.10.0
├── cockroachdb/errors v1.13.0
├── larsartmann/go-branded-id v0.3.0
└── golang.org/x/crypto v0.51.0

github.com/larsartmann/cqrs-htmx/integration_test
├── cqrs-htmx (root) → replace ../
├── cqrs-htmx/usermgmt → replace ../usermgmt
└── larsartmann/go-cqrs-lite/core v1.4.0 (needs tidy: should be direct)

github.com/larsartmann/cqrs-htmx/examples/datastar-demo
├── larsartmann/go-cqrs-lite/core v1.5.0  ⚠️ version mismatch (root uses v1.4.0)
└── starfederation/datastar-go v1.2.1
```

## Root Module Internal File Dependency Graph

```
Layer 0 (leaf, zero internal deps):
  htmx.go       — stdlib only (HTMX types, constants, context, accessors)
  context.go    — go-cqrs-lite/core deps only (UserID, CorrelationID, RequestID)
  ratelimit.go  — x/time/rate only (rate limiter middleware)
  security.go   — stdlib only (security headers middleware)
  httputil.go   — stdlib only (WriteJSON helper)

Layer 1:
  logging.go  → context.go
  decoder.go  → errors.go

Layer 2 (cross-cutting cycle):
  errors.go   → htmx.go (IsHTMXRequest, headerRedirect)
              → response.go (ContentTypePlain, ContentTypeJSON)
              → csrf.go (ErrCSRFInvalid)
  csrf.go     → errors.go (ErrForbidden)
  response.go → htmx.go (IsHTMXRequest, SwapStrategy, HeaderTrue, headers)
              → notify.go (NotificationLevel, notificationDetail, defaultNotificationEvent)
              → csrf.go (defaultCSRFHeaderName)

Layer 3:
  authz.go         → errors.go, options.go, context.go
  options.go       → errors.go, response.go, decoder.go, context.go, csrf.go
  csrf_handler.go  → csrf.go, options.go
  middleware.go     → authz.go, context.go, htmx.go

Layer 4:
  notify.go → options.go

Layer 5 (entry points):
  app.go     → authz.go, context.go, errors.go, options.go, middleware.go
  handler.go → options.go, errors.go, csrf_handler.go, response.go, authz.go
```

## Coupling Analysis

| File | Internal Deps | Dependents | Level |
|------|:------------:|:----------:|-------|
| `htmx.go` | 0 | 6 | Leaf — standalone |
| `context.go` | 0 | 6 | Leaf — utility |
| `ratelimit.go` | 0 | 0 | Leaf — standalone |
| `security.go` | 0 | 0 | Leaf — standalone |
| `httputil.go` | 0 | 2 | Leaf — utility |
| `logging.go` | 1 | 0 | Low — consumed externally |
| `decoder.go` | 1 | 1 | Low — narrow |
| `notify.go` | 1 | 1 | Low — narrow |
| `csrf.go` | 1 | 3 | Medium — cycle participant |
| `response.go` | 3 | 3 | Medium — cycle participant |
| `errors.go` | 3 | 10 | **High** — most depended-upon |
| `authz.go` | 3 | 3 | Medium — handlerConfig coupling |
| `middleware.go` | 3 | 1 | Medium — consumed externally |
| `csrf_handler.go` | 2 | 1 | Medium — bridge |
| `options.go` | 5 | 4 | **Highest** — handlerConfig hub |
| `handler.go` | 5 | 1 | High — dispatch orchestrator |
| `app.go` | 5 | 0 | High — entry point |

## Key Insight

The coupling hub is `options.go` (handlerConfig). Combined with the `errors.go` ↔ `response.go` ↔ `csrf.go` cycle, no further module extraction is feasible without significant refactoring.

## Hygiene Issues

1. `integration_test/go.mod` — needs `go mod tidy` (go-cqrs-lite/core should be direct)
2. `examples/datastar-demo/go.mod` — version mismatch (core v1.5.0 vs root's v1.4.0)
3. `go.work` — only covers root + usermgmt, not integration_test
4. CI — only tests root + usermgmt, not integration_test or datastar-demo
