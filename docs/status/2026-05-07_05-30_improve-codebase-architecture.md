# Improve Codebase Architecture — cqrs-htmx

**Date:** 2026-05-07 | **Type:** Deepening opportunities analysis

## Glossary

- **Module** — anything with an interface and implementation (function, struct, package)
- **Depth** — leverage at the interface: lots of behavior behind a small interface
- **Seam** — where an interface lives; a place behavior can be altered without editing in place
- **Adapter** — a concrete thing satisfying an interface at a seam

## Deepening Opportunities

### 1. Per-App Configuration (vs. Mutable Globals)

**Files:** `errors.go`, `notify.go`, `app.go`, `options.go`, `response.go`

**Problem:** `LoginRedirect` and `NotificationEvent` are package-level `var`. Creating two Apps with different configs silently races. `Config.LoginRedirect` mutates the global inside `New()`, which is a surprising side effect in a constructor.

**Solution:** Store both on `App`. Thread through `handlerConfig` to response helpers. Keep package-level vars as defaults but never mutate them from `New()`.

**Benefits:** Eliminates race conditions. Makes `App` truly independent. Callers can run multiple Apps in the same process (e.g., multi-tenant setups). Improves locality — all config lives on one struct.

**ADR conflict:** None. The README documents per-App configuration as the intended pattern.

### 2. Casbin Interface Abstraction

**Files:** `authz.go`, `app.go`

**Problem:** `*casbin.Enforcer` is a concrete type. Tests must create a real Casbin enforcer with model + policy. This adds ~15 lines of boilerplate per test file. Consumers who want a different auth backend (OPA, Ory, custom) cannot adapt without wrapping the entire `App`.

**Solution:** Define `Authorizer interface { Enforce(subject, resource, action string) (bool, error) }`. `*casbin.Enforcer` already satisfies this via structural typing. `App` stores the interface instead of the concrete type.

**Benefits:** Testability — tests can provide a mock authorizer with zero setup. Extensibility — consumers can use any auth backend. Locality — auth concerns stay behind one seam.

**ADR conflict:** None. The library already uses duck-typing for `TemplComponent`. This extends the same pattern to auth.

### 3. Dispatch Pipeline Hooks

**Files:** `handler.go`, `options.go`

**Problem:** The dispatch pipeline has no extension points. Consumers cannot add logging, metrics, or tracing without wrapping the entire handler. The pipeline is: authorize → decode → dispatch → respond, but there's no way to hook into this without rewriting `handleCommandDispatch`.

**Solution:** Add `OnBeforeDispatch(func(r *http.Request, cmdType string))` and `OnAfterDispatch(func(r *http.Request, cmdType string, err error))` as `HandlerOption`s. These fire before and after the CQRS dispatch call.

**Benefits:** Observability without wrapping. Non-breaking addition. Follows the existing option pattern. Enables logging, metrics, tracing as cross-cutting concerns.

**ADR conflict:** None. Additive change.

### 4. Response Finalization Extraction

**Files:** `handler.go`

**Problem:** `handleCommandDispatch` had complexity 11 (fixed to ≤10 by extracting `executePreDispatchChecks` and `applyCommandResponse` in a prior session). The query handler `handleQueryDispatch` still has the same response-finalization logic duplicated.

**Solution:** Unify response finalization into a shared `applyResponse(w, r, cfg, hasRender bool)` function that both command and query dispatchers call.

**Benefits:** Eliminates duplication. Single place to understand response behavior. Reduces cognitive load.

**ADR conflict:** None. Refactoring within existing seams.
