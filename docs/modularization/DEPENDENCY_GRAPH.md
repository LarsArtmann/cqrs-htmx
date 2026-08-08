# Dependency Analysis: cqrs-htmx

> **Date:** 2026-06-08
> **State:** Updated with current module structure

---

## Module Structure (4 Modules)

```
github.com/larsartmann/cqrs-htmx/v2 (root)
├── justinas/nosurf v1.2.0
├── larsartmann/go-cqrs-lite/command/v2 v2.2.0
├── larsartmann/go-cqrs-lite/event/v2 v2.2.0
├── larsartmann/go-cqrs-lite/id/v2 v2.2.0
├── larsartmann/go-cqrs-lite/query/v2 v2.2.0
├── larsartmann/go-error-family v0.3.0
├── larsartmann/httputil v0.1.0
├── onsi/ginkgo/v2 v2.29.0 (test)
├── onsi/gomega v1.41.0 (test)
└── golang.org/x/time v0.15.0

github.com/larsartmann/cqrs-htmx/usermgmt/v2
├── casbin/casbin/v3 v3.10.0
├── larsartmann/go-branded-id v0.3.0
├── larsartmann/go-cqrs-lite/event/v2 v2.2.0
├── larsartmann/go-error-family v0.3.0 (indirect)
└── golang.org/x/crypto v0.52.0

github.com/larsartmann/cqrs-htmx/integration_test
├── cqrs-htmx/v2 v2.0.0 → replace ../
├── cqrs-htmx/usermgmt/v2 v2.0.0 → replace ../usermgmt
└── larsartmann/go-cqrs-lite/command/v2 v2.2.0

github.com/larsartmann/cqrs-htmx/examples/datastar-demo
├── larsartmann/go-cqrs-lite/command/v2 v2.2.0
├── larsartmann/go-cqrs-lite/id/v2 v2.2.0
├── larsartmann/go-cqrs-lite/query/v2 v2.2.0
├── starfederation/datastar-go v1.2.2
├── larsartmann/go-branded-id v0.3.0 (indirect)
└── larsartmann/go-error-family v0.3.0 (indirect)
```

## Cross-Module Dependency DAG

```
           ┌───────────────────┐
           │   External Deps   │
           │  (go-cqrs-lite,   │
           │   casbin, etc.)   │
           └────────┬──────────┘
                    │
     ┌──────────────┼──────────────┐
     │              │              │
┌────┴────┐   ┌────┴────┐   ┌─────┴──────┐
│  root   │   │usermgmt │   │datastar-   │
│ (1 pkg) │   │ (1 pkg) │   │  demo      │
│21 files │   │10 files │   │ (standalone│
└────┬────┘   └────┬────┘   │  example)  │
     │              │        └────────────┘
     └──────┬───────┘
            │
     ┌──────┴──────┐
     │integration_ │
     │   test      │
     │ (bridge)    │
     └─────────────┘
```

**Key property:** root ↔ usermgmt have **zero mutual imports**. Clean DAG.

## Root Module Internal File Dependency Graph

```
Layer 0 (leaf, zero internal deps):
  htmx.go       — stdlib only (HTMX types, constants, context, accessors)
  security.go   — stdlib only (security headers middleware)
  httputil.go   — external httputil lib only (WriteJSON + ClientIP)
  ws.go         — stdlib + encoding/json only (WSMessage, ParseWSMessage, WSOOBHTML)

Layer 1 (depends on Layer 0 + external only):
  context.go    — go-cqrs-lite/id/v2 + go-cqrs-lite/event/v2 (UserID, CorrelationID, RequestID)
  ratelimit.go  — x/time/rate + external httputil (ClientIP)

Layer 2 (cross-cutting cycle — inseparable):
  errors.go   → htmx.go (IsHTMXRequest, headerRedirect)
              → response.go (ContentTypePlain, ContentTypeJSON)
              → csrf.go (ErrCSRFInvalid)
  csrf.go     → errors.go (ErrForbidden)
  response.go → htmx.go (IsHTMXRequest, SwapStrategy, HeaderTrue, headers)
              → notify.go (NotificationLevel, notificationDetail, defaultNotificationEvent)
              → csrf.go (defaultCSRFHeaderName)

Layer 3 (depends on Layer 2):
  logging.go       → context.go
  decoder.go       → errors.go
  authz.go         → errors.go, options.go, context.go
  options.go       → errors.go, response.go, decoder.go, context.go, csrf.go
  csrf_handler.go  → csrf.go, options.go
  csrf_helpers.go  → csrf.go
  notify.go        → options.go
  recovery.go      → errors.go
  sse.go           → app.go (AfterDispatchHook)

Layer 4 (entry points):
  middleware.go → authz.go, context.go, htmx.go
  app.go       → authz.go, context.go, errors.go, options.go, middleware.go
  handler.go   → options.go, errors.go, csrf_handler.go, response.go, authz.go
```

## Coupling Analysis

| File              | Internal Deps | Dependents | Level                           |
| ----------------- | :-----------: | :--------: | ------------------------------- |
| `htmx.go`         |       0       |     7      | Leaf — standalone               |
| `security.go`     |       0       |     0      | Leaf — standalone               |
| `httputil.go`     |       0       |     2      | Leaf — utility                  |
| `ws.go`           |       0       |     0      | Leaf — protocol helpers         |
| `context.go`      |       0       |     6      | Leaf — utility                  |
| `ratelimit.go`    |       1       |     0      | Low — standalone middleware     |
| `logging.go`      |       1       |     0      | Low — consumed externally       |
| `decoder.go`      |       1       |     1      | Low — narrow                    |
| `notify.go`       |       1       |     1      | Low — narrow                    |
| `recovery.go`     |       1       |     0      | Low — standalone middleware     |
| `csrf_helpers.go` |       1       |     0      | Low — template helpers          |
| `csrf.go`         |       1       |     3      | Medium — cycle participant      |
| `response.go`     |       3       |     3      | Medium — cycle participant      |
| `authz.go`        |       3       |     3      | Medium — handlerConfig coupling |
| `csrf_handler.go` |       2       |     1      | Medium — bridge                 |
| `middleware.go`   |       3       |     1      | Medium — consumed externally    |
| `sse.go`          |       1       |     0      | Low — SSE building blocks       |
| `errors.go`       |       3       |    10+     | **High** — most depended-upon   |
| `options.go`      |       5       |     4      | **Highest** — handlerConfig hub |
| `handler.go`      |       5       |     1      | High — dispatch orchestrator    |
| `app.go`          |       5       |     1      | High — entry point              |

## External Dependency Mapping

| External Dep              | Root Files Using It                                                       | Type          |
| ------------------------- | ------------------------------------------------------------------------- | ------------- |
| `go-cqrs-lite/command/v2` | app.go, handler.go, options.go                                            | Production    |
| `go-cqrs-lite/event/v2`   | app.go, authz.go, context.go, csrf.go, errors.go, options.go, recovery.go | Production    |
| `go-cqrs-lite/id/v2`      | context.go                                                                | Production    |
| `go-cqrs-lite/query/v2`   | app.go, handler.go, options.go                                            | Production    |
| `justinas/nosurf`         | csrf.go, csrf_handler.go                                                  | Production    |
| `go-error-family`         | errors.go                                                                 | Production    |
| `larsartmann/httputil`    | httputil.go, ratelimit.go                                                 | Production    |
| `golang.org/x/time`       | ratelimit.go                                                              | Production    |
| `casbin/casbin/v3`        | bdd_test.go, testing_test.go, integration_test.go                         | **Test-only** |
| `onsi/ginkgo/v2`          | All `_test.go` files                                                      | Test-only     |
| `onsi/gomega`             | All `_test.go` files                                                      | Test-only     |

**Note:** casbin/casbin/v3 appears in root's direct `require` but is only used in test code. Root's production code defines an `Enforcer` interface that matches casbin's signature without importing casbin directly. This is standard Go module behavior — test deps live in the same `require` block.

**Note:** `go-cqrs-lite/dispatcher/v2` and `go-cqrs-lite/codec/v2` appear as indirect deps in `go.mod` but are never directly imported by any `.go` file — they are transitive only.

## Hygiene Issues

| Issue                              | Status   | Notes                                                          |
| ---------------------------------- | -------- | -------------------------------------------------------------- |
| integration_test/go.mod needs tidy | ✅ Fixed | Commit 776f101                                                 |
| integration_test not in go.work    | ✅ Fixed | go.work updated                                                |
| CI doesn't test all modules        | ✅ Fixed | CI covers all 4 modules                                        |
| Lint warnings                      | ✅ Fixed | 0 issues                                                       |
| datastar-demo version mismatch     | ✅ Fixed | Migrated to v2.2.0 APIs                                        |
| datastar-demo Go version           | ✅ Fixed | Upgraded to 1.26.3                                             |
| usermgmt version alignment         | ✅ Fixed | go-cqrs-lite → v2.2.0, Go → 1.26.3, go-error-family → v0.3.0   |
| datastar-demo go-branded-id        | ✅ Fixed | Upgraded from v0.1.0 to v0.3.0                                 |
| go-cqrs-lite v2 migration          | ✅ Fixed | All modules migrated from core/v1.5.1-pre to v2.2.0 submodules |
| SSE + WebSocket support            | ✅ Done  | sse.go, ws.go added (21 root files total)                      |
| usermgmt events                    | ✅ Done  | events.go added (10 usermgmt files total)                      |
