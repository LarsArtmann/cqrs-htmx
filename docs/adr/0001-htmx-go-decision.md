# cqrs-htmx vs angelofallars/htmx-go

## Verdict: No. Do not add as a dependency.

## Why Not

| Aspect                   | `angelofallars/htmx-go`                  | `cqrs-htmx` (current)                                                                               |
| ------------------------ | ---------------------------------------- | --------------------------------------------------------------------------------------------------- |
| **Type**                 | Standalone HTMX helper                   | CQRS-integrated framework                                                                           |
| **Response builder**     | `NewResponse().Write(w)`                 | `NewResponse(w, r).Apply()` — integrates with redirect sanitization, CSRF, request-aware `IsHTMX()` |
| **Templ integration**    | Direct `templ` import                    | Duck-typed `TemplComponent` — no forced dependency                                                  |
| **Triggers**             | `TriggerObject`, swap modifiers          | `TriggerWithDetail`, `NotifyWithEvent` builder                                                      |
| **Swap strategies**      | Fluent modifiers (`After()`, `Scroll()`) | Basic constants                                                                                     |
| **HTMX request parsing** | Standalone functions                     | Context-cached via `HTMXMiddleware` + fallback                                                      |

### Adding this dependency creates problems

1. **This is a library, not an app.** Adding dependencies forces them on every consumer. We currently don't depend on `templ` directly (duck-typing), which is a feature. `htmx-go` likely imports it.

2. **API mismatch.** `htmx-go`'s `Response.Write(w)` writes headers immediately. Our `Apply()` returns a `bool` for redirect handling and integrates with the CQRS handler lifecycle — you'd need an adapter layer, not a drop-in replacement.

3. **Overlap is ~95%.** We already have headers, swap strategies, trigger serialization, request detection, notification levels, and `RenderTemplResult`. The only missing features are swap modifiers (`swap:1s`, `scroll:bottom`) and `TriggerObject` — trivial to add ourselves.

4. **Loss of control.** Our `Response` has CQRS-specific features (redirect URL sanitization, `CSRFToken`, notification methods, HTMX-aware `Content-Type`). Wrapping `htmx-go` would be more code, not less.

## What To Do Instead

If you want swap modifiers or `TriggerObject` features, port them into `response.go` directly. The pattern is straightforward and keeps the dependency graph clean.

## TL;DR

We'd gain ~5% functionality at the cost of an extra dependency, a `templ` import, and API friction. Not worth it for a library.
