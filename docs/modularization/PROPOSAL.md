# Modularization Proposal: cqrs-htmx

> **Status:** Phase 3 complete — updated assessment with 2026-05-22 state
> **Date:** 2026-05-22
> **Supersedes:** PROPOSAL.md (2026-05-14), go-modularize-assessment.md (2026-05-19)

---

## 1. Executive Summary

### What Changed Since Last Assessment

The project has grown from 15 to 18 production files, added CSRF protection, rate limiting, security headers, request logging, and request decoding. The internal coupling has deepened — specifically:

- `response.go` now depends on `notify.go` (notification types) and `csrf.go` (CSRF header names)
- `errors.go` now depends on `response.go` (ContentType constants) and `csrf.go` (ErrCSRFInvalid)
- This creates a cycle: `errors.go` → `response.go` → `csrf.go` → `errors.go`

The module count grew from 2 to 4, with `integration_test/` and `examples/datastar-demo/` added.

### Recommendation: Module Hygiene Over Splitting

**Do NOT extract `htmx/` as a separate go.mod.** The coupling has deepened since the original proposal, and the extraction is no longer clean. Instead, focus on:

1. Fix module hygiene (integration_test, go.work, CI)
2. Resolve the datastar-demo ownership question
3. Fix existing lint warnings

---

## 2. Current State Analysis (2026-05-22)

### 2.1 Module Landscape

| Module           | Path                       | Files (prod) | Exported Symbols | Direct External Deps                                                   | Internal Deps                | Replace     | go.work | State                 |
| ---------------- | -------------------------- | ------------ | ---------------- | ---------------------------------------------------------------------- | ---------------------------- | ----------- | ------- | --------------------- |
| Root             | `./`                       | 18           | 151              | casbin/v3, cockroachdb/errors, gorilla/csrf, go-cqrs-lite/core, x/time | 0                            | None        | Yes     | Clean but large       |
| usermgmt         | `./usermgmt`               | 12           | 95               | casbin/v3, cockroachdb/errors, go-branded-id, x/crypto                 | 0                            | None        | Yes     | Clean                 |
| integration_test | `./integration_test`       | 2            | 0                | root, usermgmt, go-cqrs-lite/core                                      | root + usermgmt              | Yes (→ ../) | No      | **Needs go mod tidy** |
| datastar-demo    | `./examples/datastar-demo` | 3            | 0                | go-cqrs-lite/core v1.5.0, datastar-go                                  | **None (doesn't use root!)** | None        | No      | **Version mismatch**  |

### 2.2 Classification: Partial Split

The root + usermgmt split is clean and well-executed. However:

- integration_test has stale go.mod (needs tidy)
- datastar-demo doesn't import the root library at all — it's a standalone go-cqrs-lite + datastar example
- go.work only covers root + usermgmt, not integration_test or datastar-demo
- CI only tests root + usermgmt, not integration_test or datastar-demo

### 2.3 Root Module Internal Dependency Graph (2026-05-22)

```
Layer 0 (leaf, zero internal deps):
  htmx.go, context.go, ratelimit.go, security.go, httputil.go

Layer 1 (depends on Layer 0 only):
  logging.go → context.go
  decoder.go → errors.go

Layer 2 (cross-cutting, coupled):
  errors.go    → htmx.go, response.go, csrf.go
  csrf.go      → errors.go
  response.go  → htmx.go, notify.go, csrf.go

Layer 3 (depends on Layer 2):
  authz.go         → errors.go, options.go, context.go
  options.go       → errors.go, response.go, decoder.go, context.go, csrf.go
  csrf_handler.go  → csrf.go, options.go
  middleware.go     → authz.go, context.go, htmx.go

Layer 4 (depends on Layer 3):
  notify.go → options.go

Layer 5 (entry points):
  app.go     → authz.go, context.go, errors.go, options.go, middleware.go
  handler.go → options.go, errors.go, csrf_handler.go, response.go, authz.go
```

### 2.4 Critical Cycles

```
errors.go → response.go (ContentTypePlain/ContentTypeJSON)
response.go → csrf.go (defaultCSRFHeaderName)
csrf.go → errors.go (ErrForbidden)
```

These three files form an inseparable group at the module level.

### 2.5 Coupling Hubs

| File         | Depends On        | Dependents | Coupling                                            |
| ------------ | ----------------- | ---------- | --------------------------------------------------- |
| `options.go` | 5 files           | 4 files    | **Highest** — handlerConfig is shared mutable state |
| `errors.go`  | 3 files           | 10 files   | **Highest** — most depended-upon                    |
| `context.go` | 0 (external only) | 6 files    | Low — utility                                       |
| `htmx.go`    | 0 (stdlib only)   | 6 files    | Low — leaf                                          |

---

## 3. Why Further Module Splitting Is Not Recommended

### 3.1 The htmx/ Extraction Is No Longer Clean

The original 2026-05-14 proposal identified htmx/ (htmx.go + response.go) as the only clean extraction. Since then:

- `response.go:145-147` uses `NotificationLevel`, `notificationDetail()`, `defaultNotificationEvent` from `notify.go`
- `response.go:158` uses `defaultCSRFHeaderName` from `csrf.go`
- `errors.go:156,179` uses `ContentTypePlain`, `ContentTypeJSON` from `response.go`

To extract htmx/, you would need to:

1. Move notification types to htmx/ (breaks notify.go)
2. Remove CSRFToken() from Response (breaking API change)
3. Move ContentType constants to httputil.go (creates cross-module constant sharing)
4. Break the errors.go ↔ csrf.go ↔ response.go cycle

This is more disruptive than beneficial for a library of this size.

### 3.2 Library, Not Application

The library is called "cqrs-htmx" — CQRS is the core purpose. Consumers who import it expect CQRS + HTMX together. The "HTMX-only" use case is an edge case for a library explicitly named for CQRS integration.

### 3.3 Flat Package = Go Convention for Libraries

18 files / 151 symbols in a single package is within Go norms for a cohesive library. Well-named symbols with clear prefixes (CSRF*, HTMX*, Notify*, Authorize*, Enforce\*) provide sufficient organization.

### 3.4 New Files Are Independent Concerns

The recently added files (ratelimit.go, security.go, logging.go, decoder.go) are leaf nodes with minimal coupling. They could theoretically form sub-modules, but each is a single file (125-268 lines) — too small for a separate go.mod.

---

## 4. Proposed Actions: Module Hygiene

### 4.1 Fix integration_test Module

- Run `go mod tidy` to fix stale go.mod (go-cqrs-lite/core should be direct)
- Verify tests pass

### 4.2 Resolve datastar-demo Ownership

**Problem:** datastar-demo doesn't import cqrs-htmx at all. It's a standalone go-cqrs-lite + datastar example with version mismatches (core v1.5.0 vs root's v1.4.0, go-branded-id v0.1.0 vs v0.3.0).

**Options:**

- A) Update to import cqrs-htmx and demonstrate library usage
- B) Upgrade deps to match root module versions
- C) Move to go-cqrs-lite repo (where it belongs)

**Recommendation:** Option B — upgrade deps. The demo is valuable as a go-cqrs-lite example, and restructuring it to use cqrs-htmx would change its nature entirely. Keep it as a companion example.

### 4.3 Update go.work

Add integration_test to go.work for consistent local development:

```go
go 1.26.2

use (
    .
    ./usermgmt
    ./integration_test
)
```

Do NOT add datastar-demo — it doesn't import any sibling modules, so go.work provides no benefit.

### 4.4 Update CI

Add integration_test and datastar-demo to CI pipeline:

- Build and test integration_test
- Build datastar-demo (no tests to run — it's a main package)

### 4.5 Fix Lint Warnings

10 existing lint warnings (revive missing docs, errcheck, forcetypeassert, recvcheck, exhaustruct, noctx) should be fixed before declaring the modularization complete.

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
                    │  18 files, 151 syms   │
                    │  casbin, errors,      │
                    │  csrf, cqrs-lite,     │
                    │  x/time               │
                    └──────────┬────────────┘
                               │ (go.work)
                    ┌──────────┴────────────┐
                    │                       │
           ┌────────┴────────┐    ┌─────────┴──────────┐
           │   usermgmt      │    │  integration_test   │
           │  12 files, 95   │    │  4 tests            │
           │  casbin, crypto, │    │  imports root +     │
           │  branded-id      │    │  usermgmt          │
           │  (independent)   │    │  (replace directive)│
           └─────────────────┘    └─────────────────────┘

    ┌─────────────────────────┐
    │   datastar-demo         │  (standalone — doesn't use root)
    │  go-cqrs-lite v1.5.0    │
    │  datastar-go            │
    │  (version mismatch)     │
    └─────────────────────────┘
```

---

## 7. Self-Review Findings (Phase 4)

### 7.1 What I Forgot

1. **datastar-demo version mismatch** — go-cqrs-lite/core v1.5.0 vs root's v1.4.0. The demo could break if v1.5.0 has breaking changes.
2. **CI doesn't test integration_test** — these cross-module tests could regress without detection.
3. **integration_test uses replace directives while root uses go.work** — inconsistent strategy.

### 7.2 What Could Be Improved

1. **Consistent replace strategy** — integration_test should use go.work (if added) or replace directives, not mix.
2. **datastar-demo should at minimum have matching dep versions** — even if it doesn't import root.
3. **The 151-symbol API surface** could benefit from sub-package organization (not sub-modules) in a future iteration.

### 7.3 Decision: Hygiene First

The right next step is fixing module hygiene, not adding complexity. The current structure is sound.
