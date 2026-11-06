# ADR 0061: Audience-Aware Error Detail

**Status:** Accepted
**Date:** 2026-11-06
**Related:** [ADR 0057](0057-authorization-contract.md) (audience is orthogonal to authorization — WHO reads vs MAY they act), the fleet library-utilization audit ([`docs/review/libs/index.html`](../review/libs/index.html)) finding that errorfamily's `DefaultMessage` redaction vocabulary was unused by this library

## Context

`SafeDetail(err, status, includeInternal)` took a **boolean** for whether to
include internal detail. Every caller faced the same impossible choice:

- `includeInternal: true` — operators get actionable diagnostics, but anonymous
  readers of the same 5xx body see hostnames, ports, and query text.
- `includeInternal: false` — the public is safe, but the on-call engineer reads
  "internal error" and reaches for the logs before knowing anything.

The boolean encodes the wrong question. Server-fault redaction is not a
deployment property (a "dev" flag) — the **same running server** answers
anonymous attackers and authenticated operators. The missing variable is
*who is reading this response*, which the HTTP layer knows per-request (session,
role, internal-network origin) and the error-mapping layer does not.

go-error-family already ships the vocabulary: every family carries a
`DefaultMessage` (generic, public-safe) distinct from the real error text.
This library never used it — the boolean could not express "redact for THEM,
not for ME".

## Decision

1. **`httperr.Audience`** — a two-value enum (`AudiencePublic`,
   `AudienceOperator`). The **zero value is public**: a caller that never
   thinks about audiences leaks nothing. Fail-safe by construction.
2. **`SafeDetailFor(err, status, audience)`** — the audience-aware
   `SafeDetail`:
   - `< 500` → `err.Error()` for BOTH audiences (client-fault text describes
     the caller's own input; redacting it helps no one).
   - `>= 500` → operator gets `err.Error()`; public gets the family's
     `DefaultMessage`.
3. **`ProblemDetailsAudienceErrorHandler(resolver, loginRedirect)`** — the
   `Config.ErrorHandler` shape. The resolver answers "who is reading?" from
   the request (session/role/IP — consumer policy, the library stays
   auth-agnostic). **A nil resolver fails public** — the fail-safe default
   again.
4. **Operator 5xx bodies carry machine context**: `NewStructuredErrorForAudience`
   attaches `ErrorContext()` (the errorfamily key/value pairs — pool sizes,
   shard names) to the RFC 9457 payload for operators only. Public bodies keep
   `Context: null`.
5. **All pre-existing constructors default to `AudiencePublic`** — zero
   behavior change for existing consumers; the new surface is strictly opt-in.
   `SafeDetail`'s boolean maps `includeInternal ⇄ AudienceOperator`.

## Consequences

- The redaction question becomes per-request, consumer-owned policy wired in
  one constructor — no middleware, no error-type changes.
- The re-export surface (`cqrshtmx.Audience`, `SafeDetailFor`,
  `ProblemDetailsAudienceErrorHandler`, …) follows the established
  subpackage-alias pattern and is compile-pinned by
  `api_surface_compat_test.go`.
- v5 disposition: `SafeDetail`'s boolean form (and its
  `includeInternal`-style peers) are candidates for deprecation once the
  audience form has a release train behind it; tracked in the v007/v5
  disposition ADR-0051 family.
- Non-goal: response-body redaction is NOT an authorization mechanism. It is
  a courtesy layer over logging and metrics; secrets must never enter error
  text in the first place.
