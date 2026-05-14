# Modularization Proposal: cqrs-htmx

> **Status:** Phase 4 complete — self-reviewed, revised recommendation
> **Date:** 2026-05-14

---

## 1. Executive Summary

### Why Consider Modularization

`cqrs-htmx` is a single-package Go library (`github.com/larsartmann/cqrs-htmx`) with ~80 exported symbols across 10 source files. While functional and well-tested (96% coverage, 137 specs), the flat structure creates concerns:

- **API surface overload**: Consumers see all 80+ symbols at once — HTMX headers, Casbin authorization, CQRS wiring, request decoding, templ rendering, notifications, error handling, middleware, context helpers
- **Dependency overreach**: Consumers who only want HTMX response helpers must also pull in `go-cqrs-lite/core` and its transitive dependency tree
- **Coupling inside the package**: `app.go` directly references types from `authz.go`, `context.go`, `errors.go`, `options.go`, `handler.go` — the dispatch pipeline is tightly woven
- **No compile-time boundaries**: Any file can import any other file's internals (unexported symbols) — there's no architectural enforcement

### What Changes

Split the single package into **4 semi-independent sub-modules**, each with its own `go.mod`, coordinated via a `go.work` file:

| Module                 | Purpose                                                                | External Dependencies                                  |
| ---------------------- | ---------------------------------------------------------------------- | ------------------------------------------------------ |
| `cqrs-htmx` (root)     | App builder, command/query dispatch pipeline, handler options          | `go-cqrs-lite/core`, `casbin/v3`, `cockroachdb/errors` |
| `cqrs-htmx/htmx`       | HTMX request parsing, response builder, swap strategies                | **Zero external deps** (stdlib only)                   |
| `cqrs-htmx/authz`      | Enforcer interface, Authorize/Enforce/RequireAuth, AuthorizeMiddleware | `cockroachdb/errors`                                   |
| `cqrs-htmx/middleware` | ContextEnrichmentMiddleware, HTMXMiddleware, Chain                     | **Zero external deps** (stdlib only)                   |

### Expected Benefits

1. **Dependency isolation**: Consumers using only `htmx` or `middleware` sub-modules get zero transitive deps from CQRS/Casbin
2. **Smaller API surfaces**: Each module exports a focused set of symbols
3. **Compile-time boundaries**: Import cycles between modules are impossible by Go's module system
4. **Independent versioning**: Bug fixes to the HTMX response builder don't require bumping the main module
5. **Faster CI**: Only changed modules need rebuilding/testing

---

## 2. Current State Analysis

### 2.1 Module Landscape

| Property         | Value                                  |
| ---------------- | -------------------------------------- |
| Go modules       | 1 (root only)                          |
| Go version       | 1.26.2                                 |
| Packages         | 1 (`github.com/larsartmann/cqrs-htmx`) |
| Source files     | 10 (production) + 8 (test)             |
| Exported symbols | ~80                                    |
| Test coverage    | 96% (137 specs)                        |
| Lint issues      | 0                                      |

### 2.2 External Dependencies

| Dependency                                 | Type   | Used By                                                         |
| ------------------------------------------ | ------ | --------------------------------------------------------------- |
| `github.com/casbin/casbin/v3`              | Direct | `go.mod` (but never imported in source — interface duck-typing) |
| `github.com/cockroachdb/errors`            | Direct | `app.go`, `authz.go`, `errors.go`                               |
| `github.com/larsartmann/go-cqrs-lite/core` | Direct | `app.go`, `context.go`, `errors.go`, `handler.go`, `options.go` |
| `github.com/onsi/ginkgo/v2`                | Test   | All test files                                                  |
| `github.com/onsi/gomega`                   | Test   | All test files                                                  |

### 2.3 Internal Dependency Graph

```
app.go ──────→ authz.go (Enforcer, UserIDExtractor)
    │──────→ context.go (UserIDFromContext, ParseUserID, WithUserID)
    │──────→ errors.go (ErrCommandsNil, ErrQueriesNil, ErrorHandler)
    │──────→ options.go (HandlerOption)
    │──────→ handler.go (unexported dispatch methods)

handler.go ──→ authz.go (Enforce)
    │──────→ context.go (UserIDFromContext)
    │──────→ errors.go (ErrDispatchFailed, ErrDecoderMissing)
    │──────→ options.go (handlerConfig, decoders)
    │──────→ response.go (NewResponse, applyHTMXResponse)

authz.go ────→ errors.go (ErrForbidden, ErrUnauthorized, ErrEnforcerNil)
    │──────→ options.go (HandlerOption, handlerConfig)
    │──────→ htmx.go (headerRedirect — via errors.go chain)

errors.go ───→ htmx.go (IsHTMXRequest, headerRedirect)

options.go ──→ errors.go (ErrDecodeFailed)
    │──────→ context.go (UserIDFromContext)
    │──────→ response.go (NewResponse, applyHTMXResponse)
    │──────→ htmx.go (header constants)

notify.go ───→ options.go (HandlerOption, TriggerWithDetail)

middleware.go → context.go (ParseUserID, WithUserID)
    │──────→ htmx.go (parseHTMXRequest, WithHTMX)
    │──────→ authz.go (UserIDExtractor type)

context.go ──→ (go-cqrs-lite/core/event, id)

htmx.go ─────→ (stdlib only — standalone)

response.go ─→ htmx.go (IsHTMXRequest, header constants, SwapStrategy)
```

### 2.4 Coupling Hotspots

| Hotspot                                   | Description                                                                                                                          | Risk                                                                    |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------- |
| `handlerConfig` struct                    | Defined in `options.go`, used by `app.go`, `authz.go`, `handler.go`, `notify.go` — the shared mutable config for all handler options | High — any module boundary must handle this shared type                 |
| `headerTrue` / `headerRedirect` constants | Defined in `htmx.go`, used by `errors.go`, `options.go`, `response.go`                                                               | Medium — constants can be duplicated or shared via a types module       |
| `applyHTMXResponse` function              | Defined in `handler.go` (unexported), called from `options.go` render funcs                                                          | Medium — tight coupling between handler pipeline and response rendering |
| `UserIDExtractor` type                    | Defined in `authz.go`, used by `middleware.go` and `app.go`                                                                          | Low — type alias, easy to share                                         |
| Error classification                      | `errors.go` uses `go-cqrs-lite/core/event` classification — deeply tied to CQRS                                                      | Low — only matters if splitting errors out                              |

### 2.5 God-Package Assessment

The root package is **not a god-package** by the 15-file/30-symbol threshold, but with 10 files and ~80 exported symbols, it has a **wide API surface**. The symbols cluster into 7 natural concerns (see §3.2), suggesting the package is doing "too many things" for a single Go package.

---

## 3. Proposed Module Structure

### 3.1 Module Definitions

#### Module 1: `cqrs-htmx` (root — Core)

| Field                 | Content                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Path**              | `github.com/larsartmann/cqrs-htmx`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| **Purpose**           | App builder, CQRS dispatch pipeline, handler options, error classification                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| **Production deps**   | `cqrs-htmx/htmx`, `cqrs-htmx/authz`, `cqrs-htmx/middleware`, `go-cqrs-lite/core`, `cockroachdb/errors`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| **Test deps**         | `casbin/casbin/v3`, `onsi/ginkgo/v2`, `onsi/gomega`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| **Public API**        | `App`, `Config`, `New()`, `App.Command()`, `App.Query()`, `App.Middleware()`, `HandlerOption`, `CommandDecoder`, `QueryDecoder`, `RenderFunc`, `TemplComponent`, `DecodeJSON`, `DecodeJSONQuery`, `DecodeForm`, `DecodeFormQuery`, `Render`, `RenderTempl`, `RenderTemplResult`, `Redirect`, `Trigger`, `TriggerWithDetail`, `PushURL`, `NotifySuccess`, `NotifyError`, `NotifyWarning`, `NotifyInfo`, `NotifyWithEvent`, `NotifyEventBuilder`, `ErrorHandler`, `MapError`, `DefaultErrorHandler`, `DefaultErrorHandlerWithRedirect`, `ErrUnauthorized`, `ErrForbidden`, `ErrDecodeFailed`, `ErrDispatchFailed`, `ErrEnforcerNil`, `ErrCommandsNil`, `ErrQueriesNil`, `ErrDecoderMissing`, `UserID`, `NewUserID`, `ParseUserID`, `MustParseUserID`, `WithUserID`, `UserIDFromContext`, `EventOptionsFromContext` |
| **Internal packages** | None                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| **Files**             | `app.go`, `handler.go`, `options.go`, `notify.go`, `errors.go`, `context.go`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |

#### Module 2: `cqrs-htmx/htmx`

| Field                 | Content                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Path**              | `github.com/larsartmann/cqrs-htmx/htmx`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| **Purpose**           | HTMX request parsing, response builder, swap strategies — zero external dependencies                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| **Production deps**   | None (stdlib only)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| **Test deps**         | `onsi/ginkgo/v2`, `onsi/gomega`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| **Public API**        | `HTMXRequest`, `SwapStrategy`, `SwapInnerHTML`, `SwapOuterHTML`, `SwapBeforeBegin`, `SwapAfterBegin`, `SwapBeforeEnd`, `SwapAfterEnd`, `SwapDelete`, `SwapNone`, `IsHTMXRequest`, `IsBoosted`, `IsHistoryRestore`, `RenderPartial`, `HTMXTarget`, `HTMXTrigger`, `HTMXTriggerName`, `HTMXPrompt`, `HTMXCurrentURL`, `WithHTMX`, `HTMXFromContext`, `Response`, `NewResponse`, `Response.IsHTMX`, `Response.PushURL`, `Response.ReplaceURL`, `Response.Redirect`, `Response.Refresh`, `Response.Location`, `Response.Reswap`, `Response.Retarget`, `Response.Reselect`, `Response.Trigger`, `Response.TriggerAfterSwap`, `Response.TriggerAfterSettle`, `Response.TriggerWithDetail`, `Response.NotifySuccess`, `Response.NotifyError`, `Response.NotifyWarning`, `Response.NotifyInfo`, `Response.Apply` |
| **Internal packages** | None                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| **Files**             | `htmx.go`, `response.go`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |

#### Module 3: `cqrs-htmx/authz`

| Field                 | Content                                                                                     |
| --------------------- | ------------------------------------------------------------------------------------------- |
| **Path**              | `github.com/larsartmann/cqrs-htmx/authz`                                                    |
| **Purpose**           | Casbin-compatible authorization — Enforcer interface, policy enforcement, HTTP middleware   |
| **Production deps**   | `cqrs-htmx/htmx` (for `IsHTMXRequest` in error handling), `cockroachdb/errors`              |
| **Test deps**         | `casbin/casbin/v3`, `onsi/ginkgo/v2`, `onsi/gomega`                                         |
| **Public API**        | `Enforcer`, `UserIDExtractor`, `Authorize`, `RequireAuth`, `Enforce`, `AuthorizeMiddleware` |
| **Internal packages** | None                                                                                        |
| **Files**             | `authz.go` (split from root)                                                                |

#### Module 4: `cqrs-htmx/middleware`

| Field                 | Content                                                                                      |
| --------------------- | -------------------------------------------------------------------------------------------- |
| **Path**              | `github.com/larsartmann/cqrs-htmx/middleware`                                                |
| **Purpose**           | HTTP middleware — context enrichment, HTMX header parsing, middleware chaining               |
| **Production deps**   | `cqrs-htmx/htmx` (for `HTMXRequest` parsing), `cqrs-htmx/authz` (for `UserIDExtractor` type) |
| **Test deps**         | `onsi/ginkgo/v2`, `onsi/gomega`                                                              |
| **Public API**        | `ContextEnrichmentMiddleware`, `HTMXMiddleware`, `Chain`                                     |
| **Internal packages** | None                                                                                         |
| **Files**             | `middleware.go` (split from root)                                                            |

### 3.2 Dependency DAG

```
                    ┌─────────────┐
                    │ cqrs-htmx   │  (root — core)
                    │ (app, opts, │
                    │  dispatch)  │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
      ┌───────────┐ ┌───────────┐ ┌──────────────┐
      │   htmx    │ │   authz   │ │  middleware   │
      │ (request, │ │ (casbin,  │ │ (enrich,     │
      │  response)│ │  enforce) │ │  chain, htmx)│
      └───────────┘ └─────┬─────┘ └──────┬───────┘
                         │               │
                         │    ┌──────────┘
                         │    │
                         ▼    ▼
                      ┌───────────┐
                      │   htmx    │  (shared leaf)
                      └───────────┘
```

**DAG verification**: All arrows point downward. No cycles. The `htmx` module is a leaf with zero internal deps.

### 3.3 Replace / Workspace Strategy

**Recommended: `go.work` at repo root.**

```go
// go.work
go 1.26.2

use (
    .
    ./htmx
    ./authz
    ./middleware
)
```

Rationale:

- 4 modules in same repo — `go.work` is cleaner than per-module `replace` directives
- Each module's `go.mod` stays clean — no `replace` directives
- `go.work` is auto-ignored by consumers using published versions
- `go mod tidy` works both with and without the workspace

### 3.4 Test Dependency Isolation

| Module                 | Production go.mod                                                                                      | Test-only deps                            |
| ---------------------- | ------------------------------------------------------------------------------------------------------ | ----------------------------------------- |
| `cqrs-htmx` (root)     | `go-cqrs-lite/core`, `cockroachdb/errors`, `cqrs-htmx/htmx`, `cqrs-htmx/authz`, `cqrs-htmx/middleware` | `casbin/casbin/v3`, `ginkgo/v2`, `gomega` |
| `cqrs-htmx/htmx`       | _(empty — stdlib only)_                                                                                | `ginkgo/v2`, `gomega`                     |
| `cqrs-htmx/authz`      | `cockroachdb/errors`, `cqrs-htmx/htmx`                                                                 | `casbin/casbin/v3`, `ginkgo/v2`, `gomega` |
| `cqrs-htmx/middleware` | `cqrs-htmx/htmx`, `cqrs-htmx/authz`                                                                    | `ginkgo/v2`, `gomega`                     |

**Test helpers strategy**: Shared test types (`testCreateUserCmd`, `bddTemplComponent`, `newTestEnforcer()`) remain in the root module's `_test.go` files. Sub-module tests define their own local helpers. No shared `testhelpers` module — the test surface is small enough to justify localized helpers.

### 3.5 Interface Extraction

| Module       | Interface Role                                                       | Implementation                             |
| ------------ | -------------------------------------------------------------------- | ------------------------------------------ |
| `htmx`       | Pure types and utilities — no interfaces needed                      | N/A                                        |
| `authz`      | `Enforcer` interface already exists — stays here                     | `*casbin.Enforcer` satisfies it externally |
| `middleware` | `UserIDExtractor` type stays in `authz` (referenced by `middleware`) | Consumers provide extractor                |
| `root`       | `TemplComponent` duck-types `templ.Component` — stays here           | Consumer's templ components satisfy it     |

No additional interface extraction needed — the existing design already follows the interface/impl pattern.

### 3.6 Versioning Strategy

**Recommended: Independent semver per module.**

| Module                 | Tag Format          | Initial Version       |
| ---------------------- | ------------------- | --------------------- |
| `cqrs-htmx` (root)     | `v1.2.0`            | Continue from current |
| `cqrs-htmx/htmx`       | `htmx/v0.1.0`       | New — start at v0     |
| `cqrs-htmx/authz`      | `authz/v0.1.0`      | New — start at v0     |
| `cqrs-htmx/middleware` | `middleware/v0.1.0` | New — start at v0     |

Rationale:

- This is a published library with external consumers
- Sub-modules are new extractions — v0 signals instability
- Root module continues its existing version trajectory
- Each sub-module can stabilize at its own pace

### 3.7 Migration Strategy (Ordered Steps)

1. **Create `htmx/` sub-module** — Extract `htmx.go` + `response.go` (zero external deps, leaf node)
2. **Create `authz/` sub-module** — Extract `authz.go` (depends only on `htmx` + `cockroachdb/errors`)
3. **Create `middleware/` sub-module** — Extract `middleware.go` (depends on `htmx` + `authz`)
4. **Wire root module** — Update `app.go`, `handler.go`, `options.go`, `notify.go`, `errors.go`, `context.go` to import from sub-modules
5. **Create `go.work`** — Coordinate all modules
6. **Migrate tests** — Split test files per module, create localized helpers
7. **Update CI/build** — Verify `go work sync`, `go build ./...`, `go test ./...`
8. **Update documentation** — README, AGENTS.md, FEATURES.md

### 3.8 Risk Assessment

| Risk                                                                                                     | Likelihood | Impact | Mitigation                                                                |
| -------------------------------------------------------------------------------------------------------- | ---------- | ------ | ------------------------------------------------------------------------- |
| **Consumer breaking change** — import paths change from `cqrshtmx.IsHTMXRequest` to `htmx.IsHTMXRequest` | High       | High   | Root module re-exports all symbols for backward compatibility during v1.x |
| **Circular dependency** — `authz` → `htmx` and `htmx` → `authz`                                          | Low        | High   | DAG verified; `htmx` has zero deps                                        |
| **`handlerConfig` shared type** — needs to be in root since all handler options modify it                | Medium     | Medium | Keep `handlerConfig` in root module; sub-modules don't reference it       |
| **Test complexity** — shared test helpers across modules                                                 | Medium     | Low    | Localize helpers per module; small test surface                           |
| **CI slowdown** — multiple `go.mod` files                                                                | Low        | Low    | `go.work` handles this transparently                                      |
| **Versioning confusion** — which version bump for which change?                                          | Medium     | Medium | Document tagging conventions; automate with CI checks                     |

### 3.9 Build System Impact

| File                | Change Required                                                       |
| ------------------- | --------------------------------------------------------------------- |
| `go.work`           | New file — coordinates all 4 modules                                  |
| `htmx/go.mod`       | New — stdlib only                                                     |
| `authz/go.mod`      | New — `cockroachdb/errors` + `cqrs-htmx/htmx`                         |
| `middleware/go.mod` | New — `cqrs-htmx/htmx` + `cqrs-htmx/authz`                            |
| `go.mod` (root)     | Add sub-module deps, remove some direct deps that move to sub-modules |
| `.golangci.yml`     | Update paths if per-module linting desired                            |
| CI/CD               | Add per-module build/test steps                                       |

---

## 4. Critical Decision: Backward Compatibility

The single biggest risk is **breaking existing consumers** who import `github.com/larsartmann/cqrs-htmx` and use symbols like `IsHTMXRequest`, `NewResponse`, `Enforce`, etc.

### Strategy: Root Re-exports

The root module (`github.com/larsartmann/cqrs-htmx`) **re-exports** all public symbols from sub-modules:

```go
// app.go (root)
package cqrshtmx

// Re-export from sub-modules for backward compatibility
type HTMXRequest = htmx.HTMXRequest
type Response = htmx.Response
type Enforcer = authz.Enforcer
// ... etc
```

This means:

- **Existing consumers**: Zero changes — all imports still work
- **New consumers**: Can import sub-modules directly for smaller dependency trees
- **Deprecation path**: Mark re-exports as deprecated in v2, remove in v3

### Cost of Re-exports

- Root module's go.mod still depends on everything (transitively through sub-modules)
- Consumers who only want `htmx` features can now `import "github.com/larsartmann/cqrs-htmx/htmx"` to avoid CQRS/Casbin deps
- The re-export pattern is idiomatic Go (used by `io/fs`, `net/http`, etc.)

---

## 5. Alternative Considered and Dismissed

### Alternative A: Package-only split (no separate go.mod files)

Split into sub-packages (`/htmx`, `/authz`, `/middleware`) within the same module.

**Dismissed because:**

- No dependency isolation — `go.mod` still lists all transitive deps
- No compile-time boundary enforcement beyond Go's package-level visibility
- No independent versioning
- The main benefit (smaller API surface per package) is achievable, but the dependency overreach problem remains

### Alternative B: Full library split (separate repos)

Split into separate GitHub repos: `cqrs-htmx-htmx`, `cqrs-htmx-authz`, etc.

**Dismissed because:**

- Massive coordination overhead for a small library
- CI complexity — cross-repo dependency management
- Premature for a library with ~10 source files
- Consumer friction — multiple imports from different repos

### Alternative C: No split — status quo

Keep the single flat package.

**Valid arguments for this:**

- Small codebase (10 files, ~1500 LOC)
- Already well-organized by file naming
- No actual import cycles or build problems
- Go's single-package design works fine at this scale

**Why we still recommend splitting:**

- The dependency overreach problem is real — consumers who just want HTMX response helpers should not need go-cqrs-lite
- The API surface (80+ symbols) is large for a single import
- The split creates natural architectural boundaries that prevent future coupling
- The independent versioning benefit will grow as the library matures

---

## 6. Self-Review Findings (Phase 4)

### 6.1 What I Forgot

1. **`Authorize` and `RequireAuth` return `HandlerOption`** — these functions are defined in `authz.go` but return `HandlerOption` (defined in `options.go`) and modify `handlerConfig` (also in `options.go`). This means `authz` cannot be a separate module unless `handlerConfig` and `HandlerOption` are also in `authz` or a shared types module. **This is a hard coupling that invalidates the original `authz` module proposal.**

2. **`AuthorizeMiddleware` uses `DefaultErrorHandlerWithRedirect`** — defined in `errors.go`. The authz module would depend on the error handler from root, creating a circular dependency.

3. **`response.go` uses `defaultNotificationEvent` from `notify.go`** — `response.go`'s `triggerNotification` method references `defaultNotificationEvent`. If `response.go` moves to `htmx/`, it can't reference `notify.go` which stays in root.

4. **`errors.go` uses `headerRedirect` from `htmx.go`** — the constant is unexported. If `htmx.go` moves to a sub-module, `errors.go` can't access it. Options: (a) export it, (b) duplicate it, (c) keep errors in the htmx module.

### 6.2 Revised Assessment: The Coupling Is Deeper Than Expected

After re-reading every file, the actual coupling chain is:

```
authz.go → options.go (handlerConfig, HandlerOption) → root
authz.go → errors.go (sentinels, DefaultErrorHandlerWithRedirect) → root
errors.go → htmx.go (headerRedirect constant) → htmx
notify.go → options.go (HandlerOption, TriggerWithDetail) → root
response.go → notify.go (defaultNotificationEvent) → notify
options.go → response.go (NewResponse, applyHTMXResponse) → response
```

The original 4-module split assumed `authz` could be cleanly extracted. **It cannot** — `Authorize()` and `RequireAuth()` are `HandlerOption` constructors, tightly bound to the handler config pattern. Similarly, `errors.go` is deeply entangled with both CQRS classification and HTMX headers.

### 6.3 Revised Module Structure

Given the coupling analysis, the **only clean extraction** is the `htmx` module. The rest must stay together:

| Module             | Purpose                                                                                | External Dependencies                                       |
| ------------------ | -------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| `cqrs-htmx` (root) | App builder, dispatch pipeline, handler options, authz, errors, context, notifications | `go-cqrs-lite/core`, `cockroachdb/errors`, `cqrs-htmx/htmx` |
| `cqrs-htmx/htmx`   | HTMX request parsing, response builder, swap strategies                                | **Zero external deps** (stdlib only)                        |

**What changed from original proposal:**

- `authz/` module: **Removed** — `Authorize`/`RequireAuth` are `HandlerOption` constructors that cannot be separated from the handler config
- `middleware/` module: **Removed** — `ContextEnrichmentMiddleware` uses `ParseUserID`/`WithUserID` (root) and `UserIDExtractor` (root). `HTMXMiddleware` uses `parseHTMXRequest`/`WithHTMX` (htmx). Too coupled for a clean split.
- Only `htmx/` survives — it's the only module with zero coupling to the root

### 6.4 Resolved Issue: `response.go` → `notify.go` Coupling

`response.go` uses `defaultNotificationEvent` (constant from `notify.go`). Solution: move `defaultNotificationEvent` to `response.go` or `htmx.go` where it's used. The notification event name is an HTMX concern, not a CQRS concern. `notify.go`'s exported `DefaultNotificationEvent` var can remain in root (deprecated) while the actual constant moves to `htmx/`.

### 6.5 What Could Still Be Improved

1. **Package-only split** (within same module): Even without separate go.mod files, sub-packages like `htmx/`, `authz/`, `middleware/` would improve API organization. This is Alternative A from §5, but now recognized as a **better first step** than full module extraction.
2. **`handlerConfig` abstraction**: If we introduced a proper `HandlerConfig` interface, `authz` could define auth options without depending on the concrete config struct. This is a pre-requisite for any future authz module extraction.
3. **Test file organization**: The test files are heavily cross-referencing. A package split would require careful test migration.

### 6.6 Final Recommendation

**Start with `htmx/` module extraction only.** This is the highest-value, lowest-risk change:

- Zero coupling to root — clean DAG
- Biggest dependency win — consumers who only want HTMX helpers get zero transitive deps
- Clean test migration — `htmx_test.go` tests only `htmx.go`, `response.go` is tested by `htmx_test.go` too
- Backward compatible — root re-exports all HTMX symbols

The `authz` and `middleware` extractions should be **deferred** until the handler config abstraction is improved. Package-only splits (sub-packages within same module) can be explored as an intermediate step.
