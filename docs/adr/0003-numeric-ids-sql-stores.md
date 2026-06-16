# ADR 0003: Numeric IDs for SQL Store Backends

**Date:** 2026-05-23
**Status:** Superseded — UserStore interface removed; usermgmt is now event-sourced (see ADR 0006). Numeric ID strategy remains valid for future SQL event store implementations.

## Context

The `usermgmt.UserID` is string-backed (`brandid.ID[userBrand, string]`), designed for flexibility. Future SQL store backends may prefer auto-incrementing `int64` primary keys for performance and storage efficiency.

## Decision

Defer until a concrete SQL store backend is implemented. When that happens, use `brandid.ID[Brand, int64]` for the SQL-specific store types while keeping the public `UserID` as string-backed for API compatibility.

## Rationale

1. `go-branded-id` supports any underlying type via `brandid.ID[Brand, T]`.
2. The store interface is the natural boundary for type conversion.
3. Premature optimization — no SQL store exists yet.

## Consequences

- No action needed now.
- When SQL store is built, conversion happens at the store boundary (int64 ↔ string).
