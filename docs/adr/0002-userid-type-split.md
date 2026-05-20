# ADR 0002: UserID Type Split Between Root Module and usermgmt

**Date:** 2026-05-20
**Status:** Accepted

## Context

The `cqrs-htmx` root module uses `id.UserID` (ULID-backed, 26 chars, from `go-cqrs-lite/core/pkg/id`) while the `usermgmt` submodule uses `brandid.ID[userBrand, string]` (string-backed, any format). These are incompatible types.

## Decision

Keep `usermgmt.UserID` as string-backed. Do NOT switch to ULID-backed.

## Rationale

1. **Independence:** `usermgmt` is designed as a standalone submodule. Switching to ULID would create a dependency on `go-cqrs-lite/core/pkg/id`, breaking its independence.
2. **Flexibility:** String-backed allows consumers to use UUIDs, integers-as-strings, or any ID format. ULID constrains the format.
3. **Bridge pattern:** `UserIDFromRequest()` returns `string` for cqrs-htmx compatibility. `AsEnforcer()` bridges to the parent module's `Enforcer` interface. Similar patterns work for UserID conversion.

## Consequences

- Consumers integrating both modules must convert at the boundary: `usermgmt.UserID.String()` → `cqrshtmx.MustParseUserID()`.
- The two types will never be directly assignable. This is intentional.
- `GroupPolicy.User` and `GroupPolicy.Domain` remain `string` (Casbin boundary).
