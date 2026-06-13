# ADR 0005: go-cqrs-lite v2.3.0 API Adoption

**Date:** 2026-06-13
**Status:** Accepted

## Context

`go-cqrs-lite` released per-module `v2.3.0` tags in 2026-06. The bump introduced four API additions that we want to standardize on:

1. **`command.Type.IsZero()` / `query.Type.IsZero()`** — type-level zero-value detection, replacing the previous string-comparison approach.
2. **`event.FromContext(ctx)`** — propagates a `context.Context`'s deadline and other transport values into an `event.Option`.
3. **`command.RegisterTyped[T]`** — generic typed command handler registration, eliminating manual `interface{}` casts in handlers.
4. **`query.RegisterTyped[T]` / `query.DispatchTyped[T]`** — generic typed query handler registration and dispatch.

Prior to v2.3.0 we either re-implemented the same logic (IsZero comparisons) or lived with untyped handler signatures that required `cmd.(MyCmd)` assertions. Per-module v2.3.0 tags also let us drop the `replace` directives in our `go.mod` files.

## Decision

Adopt all four v2.3.0 APIs across the root module, the `usermgmt` submodule, the `integration_test` module, and the `datastar-demo` example.

### 1. `IsZero()` for command/query type validation

- `app.go`: `app.Command("")` and `app.Query("")` panic via `command.Type.IsZero()` / `query.Type.IsZero()`.
- Replaces the previous `t == ""` check that was fragile to non-empty zero values.
- Coverage: `app_test.go` exercises both panic paths.

### 2. `event.FromContext` for deadline propagation

- `context.go`: `EventOptionsFromContext` now appends `event.FromContext(ctx)` whenever `ctx.Deadline()` is set.
- This means downstream events emitted from a request with a deadline (set by `Config.Timeout`) automatically inherit that deadline.
- No new tests required — `event.FromContext` is upstream-tested and our integration of it is trivial.

### 3. `command.RegisterTyped[T]` and `query.RegisterTyped[T]`

- `examples/datastar-demo/`: CreateTodo, ToggleTodo, DeleteTodo handlers use `command.RegisterTyped`.
- `examples/datastar-demo/`: ListTodos uses `query.RegisterTyped` + `query.DispatchTyped[[]Todo]`.
- The root module's `example_test.go` demonstrates `command.RegisterTyped`; the typed query variant is now also demonstrated in `coverage_test.go` (see P1-#5 of the 2026-06-13 status report).

### 4. Drop `replace` directives

- All `go.mod` files (root, usermgmt, integration_test, datastar-demo) declare `v2.3.0` directly.
- `go.work` still orchestrates the local multi-module layout for local development.
- `GOWORK=off` is required only for submodule-isolated CI commands, and is documented in `AGENTS.md` (see P2-#18).

## Rationale

### Why adopt `IsZero()` and not just `== ""`?

- The previous `t == ""` check was correct for ULID-backed types but was a leaky implementation detail — it assumed `Type` is always a string. `IsZero()` is defined on the type itself, so the check survives future type changes.

### Why use `event.FromContext` instead of manual `WithDeadline`?

- Manual deadline capture in our code would have to be repeated for every event-emission site. `FromContext` is a single `event.Option` that captures everything the dispatcher might need: deadline, cancellation, and any future transport values upstream adds.

### Why `RegisterTyped` instead of `Register`?

- Type safety: handlers are declared as `func(ctx, T) error` instead of `func(ctx, any) error`. No more `cmd.(MyCmd)` casts.
- Compile-time registration: a handler registered with the wrong type fails at registration, not at dispatch.
- Performance: `RegisterTyped` uses direct function calls; `Register` boxes through `any` and forces a type assertion at dispatch.

### Why drop `replace` directives?

- Local `replace` blocks masked upstream breakage and made our `go.mod` files non-portable to other consumers.
- Per-module `v2.3.0` tags are the upstream-blessed way to consume v2.3.0 — no `replace` needed.

## Consequences

- All `MustNew` / `MustParse[T]` calls were replaced with `New` / `Parse[T]`. The removed APIs are not in our public surface (we wrap them in `MustParseUserID` etc.).
- `EventOptionsFromContext` now produces up to four options. Callers iterating over the result must be aware that `FromContext` is added last; ordering of `event.Option`s is not significant in v2.3.0.
- `datastar-demo` is the canonical example of typed command/query dispatch. The root `example_test.go` is being augmented with a typed-query example to expose the pattern to library consumers who do not import `datastar-demo`.
- `usermgmt` does not directly use `RegisterTyped` — it provides a higher-level `Service` that already abstracts command dispatch. The pattern is showcased at the library surface, not the service surface.

## References

- `docs/status/2026-06-13_10-46_comprehensive-status-update.md` — "go-cqrs-lite v2.3.0 Adoption" row
- `docs/status/2026-06-12_22-00_v230-full-adoption-complete.md` — the session that completed the adoption
- `app.go` — `IsZero()` validation in `Command()`/`Query()`
- `context.go` — `EventOptionsFromContext` with `event.FromContext`
- `examples/datastar-demo/handlers.go` — typed command/query registration
- `go.work` — multi-module orchestration
