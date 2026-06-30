# Modularization Proposal: cqrs-htmx

> **Status:** Superseded by 2026-07-01 Unix-style review — see `2026-07-01_PROPOSAL.html`
> **Date:** 2026-05-27
> **Supersedes:** PROPOSAL.md (2026-05-22), PROPOSAL.md (2026-05-14), go-modularize-assessment.md (2026-05-19)

---

## 1. Executive Summary

### What Changed Since Last Assessment (2026-05-22)

Significant dependency migrations occurred:

- **gorilla/csrf → justinas/nosurf** — CSRF library replacement
- **cockroachdb/errors → go-error-family** — error handling library migration
- **httputil extracted** — `ClientIP` delegated to `github.com/larsartmann/httputil`
- **go-cqrs-lite upgraded** — ALL modules now on v1.5.1-pre (datastar-demo migrated)
- **Go toolchain bumped** — all modules now on Go 1.26.3
- **New features added** — recovery middleware, RenderJSON, request ID correlation
- **19 production files** (up from 18) in root module, still a single Go package
- **ALL dependency versions aligned** across all 4 modules
- **Zero lint warnings** across all modules
- **datastar-demo migrated** from `command.Core`/`query.Core` → `command.BasicCommand`/`query.BasicQuery`

### Recommendation: Module Hygiene Over Splitting

**Do NOT split the root module.** The existing 4-module structure is sound. The remaining work is:

1. Align dependency versions in datastar-demo (go-cqrs-lite, go-branded-id, go-error-family)
2. Run full test suite verification
3. Update documentation to reflect current state

---

## 2. Current State Analysis (2026-05-27)

### 2.1 Module Landscape

| Module           | Path                       | Files (prod)  | Exported Symbols | Direct External Deps (prod)                                  | Internal Deps   | Replace     | go.work | State                |
| ---------------- | -------------------------- | ------------- | ---------------- | ------------------------------------------------------------ | --------------- | ----------- | ------- | -------------------- |
| Root             | `./`                       | 19            | ~160             | nosurf, go-cqrs-lite/core, go-error-family, httputil, x/time | 0               | None        | Yes     | **Clean**            |
| usermgmt         | `./usermgmt`               | 9             | ~80              | casbin/v3, go-branded-id, go-cqrs-lite/core, x/crypto        | 0               | None        | Yes     | **Clean**            |
| integration_test | `./integration_test`       | 0 (test-only) | 0                | root, usermgmt, go-cqrs-lite/core                            | root + usermgmt | Yes (→ ../) | Yes     | **Clean**            |
| datastar-demo    | `./examples/datastar-demo` | ~6            | 0                | go-cqrs-lite/core, datastar-go                               | None            | None        | No      | **Version mismatch** |

### 2.2 Classification: Workspace Mode

The project is in clean workspace mode with go.work coordinating root + usermgmt + integration_test. The previous issues (integration_test not in go.work, stale go.mod) have been resolved.

### 2.3 Root Module Internal Dependency Graph (2026-05-27)

```
Layer 0 (leaf, zero internal deps):
  htmx.go       — stdlib only (HTMX types, constants, context, accessors)
  security.go   — stdlib only (security headers middleware)
  httputil.go   — external httputil lib only (WriteJSON + ClientIP)

Layer 1 (depends on Layer 0 + external only):
  context.go    — go-cqrs-lite/core/id deps only (UserID, CorrelationID, RequestID)
  ratelimit.go  — x/time/rate + httputil.go (ClientIP)

Layer 2 (cross-cutting cycle):
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

Layer 4 (entry points):
  middleware.go → authz.go, context.go, htmx.go
  app.go       → authz.go, context.go, errors.go, options.go, middleware.go
  handler.go   → options.go, errors.go, csrf_handler.go, response.go, authz.go
```

### 2.4 Coupling Hubs

| File         | Internal Deps     | Dependents | Coupling                         |
| ------------ | ----------------- | ---------- | -------------------------------- |
| `errors.go`  | 3                 | 10+        | **Highest** — most depended-upon |
| `options.go` | 5                 | 4          | **Highest** — handlerConfig hub  |
| `context.go` | 0 (external only) | 6          | Low — utility                    |
| `htmx.go`    | 0 (stdlib only)   | 6          | Low — leaf                       |

### 2.5 Critical Cycles

```
errors.go → response.go (ContentTypePlain/ContentTypeJSON)
response.go → csrf.go (defaultCSRFHeaderName)
csrf.go → errors.go (ErrForbidden)
```

These three files form an inseparable group at the module level. This cycle **prevents** extracting any of them into a separate module.

---

## 3. Why Further Module Splitting Is Not Recommended

### 3.1 The Root Module Is a Single Cohesive Package

All 19 files serve one purpose: "HTMX-aware CQRS HTTP handler integration with auth, CSRF, and middleware." The standalone utilities (htmx, security, ratelimit) are co-located for **consumer convenience** — splitting into sub-packages would force multi-import UX for no decoupling gain.

### 3.2 The errors↔response↔csrf Cycle Blocks Extraction

Any module boundary would cut through this cycle. Breaking it would require significant refactoring (moving ContentType constants, notification types, CSRF header names) that would be more disruptive than beneficial.

### 3.3 Library Convention — Flat Package

18+ files / ~160 symbols in a single package is within Go norms for a cohesive library. Well-named symbols with clear prefixes (CSRF*, HTMX*, Notify*, Authorize*, Enforce\*) provide sufficient organization.

### 3.4 Consumer UX Matters

The library is called "cqrs-htmx" — consumers expect `import cqrshtmx "github.com/larsartmann/cqrs-htmx"` and everything is available. Making consumers do `import "cqrs-htmx/security"` + `import "cqrs-htmx/htmx"` would be hostile.

### 3.5 Zero Mutual Imports Between Root and Usermgmt

The existing split is **perfect** — root and usermgmt are fully independent. No further seams exist that would benefit from extraction.

---

## 4. Completed Actions

### 4.1 Migrate datastar-demo to v1.5.1

Migrated `command.Core` → `command.BasicCommand`, `query.Core` → `query.BasicQuery`. Fixed struct literal field names. Fixed lint hints (`strings.Builder`, `strings.Cut`). All versions now aligned.

### 4.2 Upgrade usermgmt Dependencies

Upgraded go-cqrs-lite/core v1.5.0 → v1.5.1-pre, go 1.26.2 → 1.26.3, go-error-family v0.1.0 → v0.1.1.

### 4.3 Keep integration_test Replace Directives

The `replace` directives in integration_test/go.mod are needed for `GOWORK=off` CI builds. This is correct and should not be removed.

---

## 5. When to Revisit Modularization

- Library exceeds 10,000 LOC
- Consumer feedback about dependency overreach
- New subsystem with zero overlap (e.g., SSE/WebSocket helpers)
- Extracting `htmx` as a standalone library for non-CQRS use cases
- The ContentType/notification cycle is broken through refactoring

---

## 6. Dependency Graph (Current State)

```
                    ┌──────────────────────┐
                    │     cqrs-htmx         │  (root)
                    │  19 files, ~160 syms  │
                    │  nosurf, go-error-    │
                    │  family, httputil,    │
                    │  cqrs-lite/core,      │
                    │  x/time               │
                    └──────────┬────────────┘
                               │ (go.work)
                    ┌──────────┴────────────┐
                    │                       │
           ┌────────┴────────┐    ┌─────────┴──────────┐
           │   usermgmt      │    │  integration_test   │
           │  9 files, ~80   │    │  5 bridge tests     │
           │  casbin, crypto, │    │  imports root +     │
           │  branded-id      │    │  usermgmt           │
           │  (independent)   │    │  (replace + go.work)│
           └─────────────────┘    └─────────────────────┘

    ┌─────────────────────────┐
    │   datastar-demo         │  (standalone — doesn't use root)
    │  go-cqrs-lite v1.5.0    │  ⚠️ version mismatch
    │  datastar-go            │
    └─────────────────────────┘
```

---

## 7. Version Alignment Status

| Dependency          | Root              | Usermgmt          | Integration_test  | Datastar-demo     |
| ------------------- | ----------------- | ----------------- | ----------------- | ----------------- |
| `go-cqrs-lite/core` | v1.5.1-pre        | v1.5.1-pre        | v1.5.1-pre        | v1.5.1-pre        |
| `go-error-family`   | v0.1.1            | v0.1.1 (indirect) | v0.1.1 (indirect) | v0.1.1 (indirect) |
| `go-branded-id`     | v0.3.0 (indirect) | v0.3.0            | v0.3.0 (indirect) | v0.3.0 (indirect) |
| Go version          | 1.26.3            | 1.26.3            | 1.26.3            | 1.26.3            |

---

## 8. Task Completion Tracker (Since 2026-05-22)

| Task                            | Status  | Done In                                                         |
| ------------------------------- | ------- | --------------------------------------------------------------- |
| T1: Fix integration_test go.mod | ✅ Done | 776f101                                                         |
| T2: Upgrade datastar-demo deps  | ✅ Done | Migrated to v1.5.1 APIs (command.BasicCommand/query.BasicQuery) |
| T2b: Upgrade usermgmt deps      | ✅ Done | go-cqrs-lite → v1.5.1-pre, Go → 1.26.3                          |
| T3: Update go.work              | ✅ Done | go.work updated                                                 |
| T4: Update CI                   | ✅ Done | CI covers all 4 modules                                         |
| T5: Fix lint warnings           | ✅ Done | 0 issues                                                        |
| T6: Full test suite             | ✅ Done | All modules pass                                                |
| T7: Update AGENTS.md            | ✅ Done | Updated with new features                                       |
| T8: Update docs                 | ✅ Done | This document                                                   |
