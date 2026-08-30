# ADR 0047: Re-export Layer Retirement Plan

## Status: Accepted

## Context

cqrs-htmx has two deprecated re-export layers that exist solely for backward
compatibility:

1. **Root httputil/SSE re-exports** (~57 deprecated symbols across
   `csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go`,
   `security.go`, `sse_event.go`, `sse_store.go`). These alias
   `httputil.*` and `sse.*` symbols so consumers can write
   `cqrshtmx.CSRFMiddleware` instead of importing httputil directly.

2. **usermgmt identity-model re-exports** (~161 deprecated symbols). These
   alias identity-model types/constants/fold functions so consumers can write
   `usermgmt.UserCreatedPayload` instead of importing identity-model directly.

All symbols carry `// Deprecated:` markers pointing to the direct-import
alternative. All internal consumers (root, adminui, dashboardui, examples,
tests) already use direct imports. The re-exports exist ONLY for external
consumers who haven't migrated yet.

Removing these aliases is a **breaking change** (consumers' code stops
compiling), so it must be bundled with a major version bump.

## Decision

Remove both re-export layers in **v5.0.0**, following this sequence:

### Phase 1: Pre-removal verification (before cutting v5)

1. Grep the entire codebase for any remaining `cqrshtmx.CSRF*`,
   `cqrshtmx.SecurityHeaders*`, `cqrshtmx.RateLimit*`, `cqrshtmx.ServerTiming*`,
   `cqrshtmx.SSE*` references in non-test production code — verify zero.
2. Grep for `usermgmt.*Payload`, `usermgmt.*Cmd`, `usermgmt.Fold*` in non-test
   production code — verify zero.
3. Ensure the v4-to-v5 migration guide covers every removed symbol with a
   Before/After recipe.

### Phase 2: Removal (in the v5 branch)

4. Delete `csrf_reexport.go`, `ratelimit_reexport.go`,
   `server_timing_reexport.go`, `sse_event.go`, `sse_store.go`.
5. Remove the deprecated aliases from `security.go` (keep the domain-aware
   `RecoveryMiddleware`, `LoggingMiddleware`, `ContextEnrichmentMiddleware` —
   those are NOT re-exports).
6. Delete the usermgmt re-export files (type aliases, var aliases, constructor
   wrappers).
7. Remove `httputil` and `go-sse` from root `go.mod` if no longer directly
   imported (they may still be needed for `Broadcaster`, `JournalSSEStore`,
   `ServeSSE`, `EventToSSEMapper` which are genuinely coupled types that stay).
8. Run `go build ./...` and fix any remaining references.
9. Run the full test suite.

### Phase 3: Release

10. Change module paths from `/v4` to `/v5` in ALL `go.mod` files.
11. Update ALL import paths across the codebase.
12. Tag `v5.0.0`.

### Symbols that STAY (not re-exports)

These root-module types are genuinely coupled to cqrs-htmx and must NOT be
removed:

- `Broadcaster` — integrates CQRS `AfterDispatchHook` with SSE broadcasting.
- `JournalSSEStore` — bridges go-cqrs-lite journal to `sse.EventStore`.
- `ServeSSE` — convenience lifecycle handler.
- `EventToSSEMapper` — maps domain events to SSE events.
- `RecoveryMiddleware`, `LoggingMiddleware`, `ContextEnrichmentMiddleware` —
  domain-aware reimplementations (not httputil aliases).
- `CSRFProtect` — cqrs-htmx-native `HandlerOption` (not a re-export).
- `OOBHTML` — HTMX out-of-band HTML helper (not an SSE re-export).

## Consequences

- v5.0.0 is a breaking change: consumers must add `httputil` and/or
  `go-sse` and/or `identity-model` direct imports.
- The v4-to-v5 migration guide (`docs/migrations/v4-to-v5.md`) must be
  complete before the release.
- The root module becomes leaner (fewer transitive deps exposed).
- usermgmt becomes a thinner layer (truly just infrastructure over
  identity-model, not a facade).

## See Also

- [v5 Removal Inventory](../guides/v5-removal-inventory.md) — the full v5 removal list with per-item criteria (all classes, not just re-exports)
