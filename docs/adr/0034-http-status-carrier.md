# ADR 0034: HTTPStatusCarrier — Errors That Pin Their HTTP Status

**Date:** 2026-06-30
**Status:** Accepted

## Context

`cqrs-htmx` maps errors to HTTP status codes via `MapError`, which delegates to
`event.Family.HTTPStatus()` from go-error-family (ADR-0017). This works for
errors whose status matches their family default (Rejection→400, Conflict→409,
Transient/Infrastructure→503, Corruption→500).

The problem: the **Rejection** family is overloaded. It covers 400, 401, 403,
404, and 429 — all semantically "the caller's fault, not retryable" — yet each
needs a distinct HTTP status. Until now there were two mechanisms for handling
this:

1. A hardcoded sentinel switch in root `explicitErrorStatus()` — only root's own
   sentinels.
2. A **parallel 40-line switch** in `usermgmt/http.go:errorStatus()` — duplicated
   the mapping for every usermgmt sentinel.

These two switches were a **split brain**: they would drift independently, and
adding a new sentinel in one module required the other to know about it.

Additionally, `StructuredError` (the SSE/WS error payload) was enriched with
RFC 7807 fields but still leaked raw `Detail` for 5xx errors, and the root
`errorStatus()` switch couldn't express "this Rejection is a 404" without
modifying root code for each sentinel.

## Decision

Introduce the `HTTPStatusCarrier` interface and `WithHTTPStatus` wrapper.

```go
type HTTPStatusCarrier interface {
    error
    HTTPStatus() int
}

func WithHTTPStatus(err error, status int) error
```

`MapError` now resolves status in three layers (first match wins):

1. **HTTPStatusCarrier** — an error that explicitly declares its status
2. **Explicit overrides** — auth/HTTP-semantic sentinels + panic code
3. **Family.HTTPStatus()** — the upstream default

### Why this design

- **Single source of truth:** Each sentinel carries its own status. No parallel
  switch to maintain. `usermgmt/errorStatus()` collapsed from 40 lines to a
  one-line delegation to `MapError`.
- **Family preserved:** The wrapper `Unwrap()`s to the inner `*event.Error`, so
  `event.Classify` still derives the correct family for retry/exit-code logic.
- **Identity preserved:** `errors.Is` traverses the chain, so wrapping a
  sentinel keeps it matchable.
- **Purely additive:** Errors that don't carry a status get the family default —
  zero behavior change for existing code.

### Semantic helpers in usermgmt

```go
ErrUserNotFound        = notFound("usermgmt.user_not_found", "user not found")
ErrInvalidCredentials  = unauthorized("usermgmt.invalid_credentials", ...)
ErrEmailExists         = conflict("usermgmt.email_exists", ...)
ErrAccountLocked       = withStatus("usermgmt.account_locked", ..., 429)
```

Sentinels at the family default (400/409) stay as raw `event.NewRejection`/
`event.NewConflict` — no wrapper needed.

## Additional changes in this session

1. **5xx detail redaction** (`SafeDetail`): Server-fault error messages
   (Corruption/Infrastructure/Transient, status ≥ 500) are replaced with the
   family's public-safe default message in all default error handlers. 4xx
   detail (the raw error) is unchanged. `Config.IncludeInternalDetails` opts
   back in for dev/trusted networks.

2. **StructuredError metadata**: Exposed `Message`, `Why`, `Fix` fields (RFC 7807
   extensions) sourced from go-error-family's `Family.DefaultMessage/Why/Fix`.
   Also applied 5xx redaction to the `Detail` field, closing the SSE/WS leak.

3. **ProblemDetailsErrorHandler**: New error handler emitting
   `application/problem+json` (`StructuredError` shape) — unifying the error
   contract across HTTP, SSE, and WS transports.

4. **Exported auth code constants** (`CodeUnauthorized`, `CodeForbidden`):
   Compile-time-safe cross-module error code sharing between root and usermgmt.

5. **Decode-error wrapping parity**: Command dispatch now wraps decode errors
   with context + Rejection family (matching the query path).

## Consequences

- `usermgmt.errorStatus()` is a one-liner delegating to `cqrshtmx.MapError`.
- Unknown/untyped errors return 503 (Transient fail-open) instead of 500 —
  matching upstream go-error-family design.
- 5xx error responses no longer leak internal detail by default.
- Consumers can opt into RFC 7807 problem-details via `Config.ErrorHandler`.
- Adding a new sentinel with a non-default status requires only
  `WithHTTPStatus(event.NewRejection(...), status)` — no switch changes.
