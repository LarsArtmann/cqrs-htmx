# ADR 0018: Unify UserID — usermgmt Adopts id.UserID

**Date:** 2026-06-22
**Status:** Accepted
**Supersedes:** ADR-0002

## Context

`cqrs-htmx` had two incompatible types named `UserID`:

- **Root module:** `cqrshtmx.UserID = id.UserID = Of[UserMarker]` — backed by
  `ulid.ULID` (binary), from `go-cqrs-lite/id/v4`.
- **usermgmt submodule:** `usermgmt.UserID = brandid.ID[userBrand, string]` —
  backed by `string`, from `go-branded-id`.

ADR-0002 originally justified the split for "independence" — usermgmt was
designed as a standalone submodule without go-cqrs-lite dependencies.

**That rationale is now moot.** `usermgmt/go.mod` already depends on
`go-cqrs-lite/id/v4 v4.0.0` (added for `id.AggregateID` in the event sourcing
read model). The "independence" that ADR-0002 protected no longer exists.

The split forced manual `.String()` → `NewUserID()` conversion at every
HTTP→domain boundary, with zero compile-time safety. Two types named `UserID`
that silently don't interoperate is worse than one type — it creates false
confidence.

## Decision

**Make `usermgmt.UserID` a type alias of `id.UserID`.**

```go
// usermgmt/id.go — AFTER
type UserID = id.UserID  // = Of[UserMarker] = brandid.ID[UserMarker, ulid.ULID]
```

This unifies the two types into one. The `userBrand` phantom type is removed
in favor of `id.UserMarker` (inherited via the alias).

### Constructor Migration

- `usermgmt.NewUserID(s string) UserID` → panics on invalid ULID (test convenience).
- `usermgmt.ParseUserID(s string) (UserID, error)` → returns error on invalid ULID.
- `usermgmt.MustParseUserID(s string) UserID` → panics on invalid ULID.

### `.Get()` Semantics Change

Before: `UserID.Get()` returned `string`.
After: `UserID.Get()` returns `ulid.ULID`.

All call sites that used `.Get()` for string context (SQL, Casbin, logging)
now use `.Get().String()` which returns the 26-char ULID string.

## Consequences

- **Breaking change:** any consumer passing non-ULID strings to
  `NewUserID` (e.g., `"alice"`) must switch to valid ULIDs.
- The `userBrand` phantom type is removed. Any code referencing it directly
  (unlikely — it was internal) must switch to `id.UserMarker`.
- `TenantID` and `BotID` remain `brandid.ID[xBrand, string]` — they are NOT
  changed by this ADR. Only `UserID` is unified.
