# ADR-0028: Brand all ID types with go-branded-id

**Date:** 2026-06-28
**Status:** Accepted

## Context

The root module had three ID types as plain `type X string`:
- `ActorID string`
- `ImpersonatorID string`
- `SSEEventID string`

These were hand-rolled string newtypes — they gave nominal safety (Go
won't let you pass `SSEEventID` where `ActorID` is expected) but missed
everything that `go-branded-id` provides: `.Get()`, `.IsZero()`,
`.Equal()`, `BrandNamer` debug display, SQL/JSON/Text serialization.

Meanwhile, the root module already used branded types for `UserID`,
`CorrelationID`, and `RequestID` via go-cqrs-lite's `id` package
(`type Of[T] = brandid.ID[T, ulid.ULID]`). The `usermgmt` submodule used
`brandid.ID` directly for `TenantID` and `BotID`. go-branded-id was
already in the dependency tree (as `// indirect`).

The `ActorID` type was especially dangerous: root's `type ActorID string`
crossed a stringly-typed boundary with usermgmt's struct-based `ActorID`,
causing a real bug where `.String()` (unprefixed) was used where
`.PrefixedString()` was needed, silently corrupting event metadata.

## Decision

Brand all root ID types with `go-branded-id`:

```go
type actorBrand struct{}
func (actorBrand) Name() string { return "Actor" }
type ActorID = brandid.ID[actorBrand, string]

type ImpersonatorID = ActorID  // alias — an impersonator IS an actor

type sseEventBrand struct{}
func (sseEventBrand) Name() string { return "SSEEvent" }
type SSEEventID = brandid.ID[sseEventBrand, string]
```

### Key decisions

1. **`ImpersonatorID = ActorID`** — type alias, same type. An impersonator
   IS an actor. The distinction is contextual (stored under different
   context keys), not a different identity type.

2. **String-backed, not ULID-backed** — ActorID carries prefixed format
   (`"user:01JX..."`) that encodes kind. SSEEventID is an arbitrary
   server-defined string. Neither is ULIDs.

3. **`.Get()` for raw value, `.String()` for debug** — BrandNamer makes
   `.String()` return `"Actor:user:01JX..."` for debug visibility. Code
   that needs the raw value uses `.Get()`. This replaces the old
   `string(x)` cast pattern.

4. **Promoted go-branded-id to direct dependency** — was `// indirect`
   via go-cqrs-lite. Now direct since we import it ourselves.

5. **Fixed the usermgmt bridging bug** — `middleware.go` now uses
   `.PrefixedString()` (was `.String()`) so the prefix survives the
   crossing into root context. The doc example shows
   `cqrshtmx.NewActorID()` conversion.

## Consequences

### Positive

- **Compile-time safety**: `SSEEventID` can't be passed where `ActorID`
  is expected — different phantom brand types.
- **Consistency**: all ID types in the repo now use the same brandid
  infrastructure with `.Get()`, `.IsZero()`, `.Equal()`.
- **Debug visibility**: `.String()` includes the brand name
  (`"Actor:user:01JX..."`, `"SSEEvent:evt-42"`).
- **Serialization**: brandid provides JSON, SQL, Text, Binary, Gob
  serialization for free.
- **Fixed bridging bug**: usermgmt middleware now uses `.PrefixedString()`.

### Negative

- **Breaking change**: `ActorID("...")` → `NewActorID("...")`,
  `SSEEventID("...")` → `NewSSEEventID("...")`. Code that used
  `string(actorID)` must use `actorID.Get()`.
- **`.String()` semantics changed**: returns brand-prefixed form, not
  raw value. Code that needs raw uses `.Get()`.

## References

- [go-branded-id](https://github.com/larsartmann/go-branded-id) — phantom-typed ID library
- `usermgmt/id.go` — already used brandid for TenantID, BotID
- `context.go` — ActorID, ImpersonatorID definitions
- `sse_event.go` — SSEEventID definition
