# ADR 0017: Reconcile HTTP Status Mapping with go-error-family Upstream

**Date:** 2026-06-22
**Status:** Accepted

## Context

`cqrs-htmx/errors.go` contained a local `familyStatus()` function that mapped
CQRS error families to HTTP status codes. This mapping **contradicted** the
authoritative mapping in `go-error-family/family.go` for two families:

| Family             | go-error-family (upstream)    | cqrs-htmx (local)             | Match? |
| ------------------ | ----------------------------- | ----------------------------- | ------ |
| Rejection          | 400 Bad Request               | 400 Bad Request               | ✅     |
| Conflict           | 409 Conflict                  | 409 Conflict                  | ✅     |
| Transient          | 503 Service Unavailable       | 503 Service Unavailable       | ✅     |
| **Corruption**     | **500 Internal Server Error** | **422 Unprocessable Entity**  | ❌     |
| **Infrastructure** | **503 Service Unavailable**   | **500 Internal Server Error** | ❌     |

The upstream library provides `Family.HTTPStatus()` — a method specifically
designed for this purpose. The local function was both a **duplication** and a
**contradiction**, meaning consumers received different HTTP codes depending on
which mapping they consulted.

## Decision

**Delegate to `event.Family.HTTPStatus()` from upstream.** Remove the local
`familyStatus()` switch entirely. This eliminates duplication, resolves the
contradiction, and ensures cqrs-htmx tracks upstream automatically.

### Rationale

1. **Single source of truth.** The upstream `HTTPStatus()` method is the
   canonical mapping. It already encodes the semantic reasoning for each code.
2. **Corruption = 500 is correct.** Corruption means the source of truth is
   damaged (unparseable payload, schema break). This is a server-side data
   integrity failure, not a client input problem. 422 implies "your input is
   semantically wrong" — that's `Rejection`, not `Corruption`.
3. **Infrastructure = 503 is correct.** Infrastructure means the system cannot
   serve (closed, nil deps, startup failure). 503 Service Unavailable tells
   clients and load balancers to retry later — the correct semantic.

### Breaking Change Impact

This changes two HTTP status codes:

- `Corruption` errors: 422 → **500**
- `Infrastructure` errors: 500 → **503**

Consumer count is low and controlled. Any client relying on 422 for corruption
should interpret 500 as the same severity class (server-side failure). Any
client relying on 500 for infrastructure should interpret 503 as the same
severity class (system unavailable, retry later).

## Consequences

- cqrs-htmx's `MapError()` now tracks upstream automatically — no manual sync.
- `familyTitle()` in `structured_error.go` updated to derive titles from
  `http.StatusText(status)` for all families, eliminating another hardcoded
  mapping.
- Tests updated to expect the upstream codes.
