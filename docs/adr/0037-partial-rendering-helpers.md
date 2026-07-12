# ADR 0037: Partial Rendering Helpers — Eliminating HTMX Boilerplate

**Date:** 2026-07-12
**Status:** Accepted

## Context

Every HTMX handler in every consumer repeats the same branching pattern:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    data := loadData()
    if r.Header.Get("HX-Request") == "true" {
        renderPartial(w, r, fragment(data))
        return
    }
    renderPage(w, r, fullPage(data))
}
```

This pattern appears in:

- **adminui** (`handler_users.go`): user list with search — HTMX gets the table fragment, full navigation gets the page
- **Every consumer app**: any page that supports both HTMX-driven partial updates and direct URL navigation
- **The library's own README examples**: the partial-vs-full branching was the #1 source of boilerplate

The problems with the status quo:

1. **Raw header access** — `r.Header.Get("HX-Request") == "true"` bypasses the library's `HTMXMiddleware`, which enriches the request into an `HTMXRequest` struct with typed accessors. The library already has `RenderPartial(r)` which correctly handles history-restore requests (`HX-History-Restore-Request`), but consumers keep writing the raw header check.
2. **No composable abstraction** — consumers can't express "partial for HTMX, full otherwise" as a single `HandlerOption`. They must write a handler function with branching logic.
3. **OOB swap wrapping locked to WebSocket** — `WSOOBHTML` wraps HTML with `hx-swap-oob` attributes, but the same wrapping is needed for HTTP responses. The logic was duplicated (or inaccessible) for non-WS consumers.

## Decision

Add five functions across two files that eliminate this boilerplate at three levels of abstraction.

### Layer 1: Composable primitive (`RenderIf`)

```go
func RenderIf(check func(*http.Request) bool, match, noMatch RenderFunc) HandlerOption
```

The most general selector. Takes any predicate and two `RenderFunc`s. All other helpers compose on top of this. Use when the branching decision isn't partial-vs-full (e.g. target-based selection).

### Layer 2: Partial-vs-full selectors

```go
func RenderPartialOrFullFunc(partial, full RenderFunc) HandlerOption
func RenderPartialOrFull[T any](partial, full func(T) TemplComponent) HandlerOption
```

`RenderPartialOrFullFunc` is the non-generic version — delegates to `RenderIf(RenderPartial, partial, full)`. For consumers using html/template, raw string building, or JSON.

`RenderPartialOrFull[T]` is the generic typed version — matches the existing `RenderTemplResult[T]` pattern. Both mappers receive the typed query result. Uses `RenderPartial(r)` for selection, so history-restore requests correctly render the full page.

### Layer 3: Standalone helper (`RenderTemplComponent`)

```go
func RenderTemplComponent(w http.ResponseWriter, r *http.Request, partial, full TemplComponent) error
```

For non-CQRS routes that don't go through `App.Query` — `net/http` handlers, middleware-injected routes, etc. Same partial-vs-full selection via `RenderPartial(r)`.

### OOB swap extraction (`OOBHTML`)

```go
func OOBHTML(id, html string, swapStrategy ...SwapStrategy) string
```

General OOB swap wrapper extracted from `WSOOBHTML`. `WSOOBHTML` is now a 1-line delegate (`return OOBHTML(id, html, swapStrategy...)`), fully backward compatible.

### Content-Type design decision

The HTML-specific helpers (`RenderPartialOrFull[T]`, `RenderTemplComponent`) set `Content-Type: text/html; charset=utf-8` because they know they produce HTML via templ.

The generic helpers (`RenderIf`, `RenderPartialOrFullFunc`) do **not** set Content-Type. This is intentional: `RenderPartialOrFullFunc` is documented for "html/template, raw string building, or any non-templ rendering" — some of those produce JSON or plain text. The user's `RenderFunc` owns the Content-Type in those cases.

This mirrors the existing pattern: `RenderTempl` and `RenderHTML` (which know they produce HTML) set Content-Type; `RenderJSON` sets `application/json`. The content type follows the helper's contract, not a global default.

## Alternatives Considered

### Single `RenderPartialOrFull` without `RenderIf`

Rejected. Without the composable primitive, `RenderPartialOrFullFunc` would inline the branching logic and `RenderIf` consumers would have no building block. The layered design means `RenderPartialOrFullFunc` is a 1-liner: `return RenderIf(RenderPartial, partial, full)`.

### Auto-detect Content-Type from the rendered output

Rejected. Sniffing Content-Type from bytes is fragile (HTML vs XHTML vs plain text) and would override explicit Content-Type headers set by the consumer's `RenderFunc`. The helper's type signature (`TemplComponent` → HTML) is a stronger signal than runtime byte inspection.

### Make `RenderTemplComponent` a `HandlerOption`

Rejected. `HandlerOption` only works inside `App.Query`/`App.Command` handlers. Non-CQRS routes (`net/http` handlers, static file servers with HTMX awareness, adminui routes) need a plain function. Making it a `HandlerOption` would force consumers to create an `App` instance just to render a partial.

### Put everything in `options_render.go`

Partially adopted. `RenderIf`, `RenderPartialOrFullFunc`, and `RenderPartialOrFull[T]` live in `options_render.go` because they are `HandlerOption`s. `RenderTemplComponent` and `OOBHTML` live in `partial.go` because they are standalone helpers, not `HandlerOption`s.

## Consequences

- **Consumers write less boilerplate**: the 5-line if-else branching collapses to a single `HandlerOption` or function call.
- **`RenderPartial(r)` adoption**: consumers are nudged toward the library's typed accessor instead of raw header checks. adminui was refactored as the first internal consumer.
- **`WSOOBHTML` is frozen**: it delegates to `OOBHTML` and will not gain new features. New OOB wrapping logic goes in `OOBHTML`. `WSOOBHTML` stays for backward compatibility.
- **No breaking changes**: all five functions are additive. `WSOOBHTML` has the same signature and output.
