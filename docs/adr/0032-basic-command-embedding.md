# ADR-0032: Embed `command.BasicCommand` in all usermgmt commands

**Date:** 2026-06-29
**Status:** Accepted

## Context

A critical bug was discovered in usermgmt's event-sourced commands: **7 of 20
command constructors returned a zero-value `cmdID`**, silently breaking:

- **Idempotency dedup** — the `IdempotencyStore` deduplicates by `cmd.ID()`. A
  zero ID means every retry is treated as a new command, so duplicate commands
  are processed.
- **Watermill message UUIDs** — the event bus derives the message UUID from
  `cmd.ID()`. A zero UUID can cause routing issues and makes tracing impossible.

### Root cause

The affected constructors created a `*command.BasicCommand` but forgot to call
`id.NewCommandID()`. The pattern before the fix was:

```go
type RegisterUserCmd struct {
    aggregateID id.AggregateID
    cmdID       id.CommandID  // ← zero-value, never set
    email       string
    // ...
}

func (c *RegisterUserCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *RegisterUserCmd) ID() id.CommandID            { return c.cmdID }
func (c *RegisterUserCmd) Type() command.Type          { return cmdRegisterUser }
```

Each command type manually declared three methods (`AggregateID`, `ID`, `Type`)
and stored three fields (`aggregateID`, `cmdID`, plus its domain fields). This
is ~60 lines of boilerplate per command across 20 commands. The `cmdID` field
was easy to forget — the struct compiled fine, the constructor ran without
error, and the zero-value `cmdID` silently propagated.

### The 7 affected constructors

1. `RegisterUserCmd`
2. `LinkExternalAccountCmd`
3. `UnlinkExternalAccountCmd`
4. `AddMemberCmd`
5. `UpdateMemberRolesCmd`
6. `RemoveMemberCmd`
7. `RegisterBotCmd`

## Decision

**Embed `*command.BasicCommand` in every command struct.** This structurally
eliminates the bug class: `BasicCommand`'s constructor (`command.New`) mints the
command ID and validates inputs. The embedding promotes `Type()`, `AggregateID()`,
and `ID()` methods automatically — no manual method declarations, no manual
`cmdID` field.

### Pattern

```go
type RegisterUserCmd struct {
    *command.BasicCommand              // ← promotes Type(), AggregateID(), ID()
    email       string
    displayName string
    roles       []Role
}

func NewRegisterUserCmd(
    aggID id.AggregateID, email, displayName string, roles []Role,
) *RegisterUserCmd {
    return &RegisterUserCmd{
        BasicCommand: mustCommand(cmdRegisterUser, aggID),
        email:        email,
        displayName:  displayName,
        roles:        roles,
    }
}
```

The `mustCommand` helper centralizes construction with fail-fast validation:

```go
func mustCommand(cmdType command.Type, aggID id.AggregateID) *command.BasicCommand {
    base, err := command.New(cmdType, aggID)
    if err != nil {
        panic(event.Wrapf(err, event.Infrastructure,
            "usermgmt.command.create_failed",
            "create %s command", cmdType))
    }
    return base
}
```

### Why `panic` is correct here

The only error cases from `command.New` are:

- **Empty command type** — a compile-time constant is wrong (programming bug).
- **Zero aggregate ID** — the constructor was called without a valid ID (programming bug).

Neither can happen at runtime in correct code. A `panic` (fail-fast) is the
correct response — returning an error from a constructor with signature
`(*Cmd, error)` would force every call site to handle an impossible error,
polluting the domain API with infrastructure concerns. The panic surface is
limited to construction time (startup or request handler setup), never during
event replay or normal dispatch.

### Why not a `(*Cmd, error)` constructor signature?

Considered but rejected:

1. **Every call site already has a valid `aggID`** — it comes from the service
   layer which generates it via `id.NewAggregateID()`. A zero aggID reaching a
   constructor is a bug in the caller, not a user-input error.
2. **Domain API pollution** — 20 constructors returning `(*Cmd, error)` adds
   error-handling boilerplate at every call site for a condition that never
   occurs in practice. The domain API should reflect domain errors (conflicts,
   rejections), not infrastructure construction failures.
3. **The `must` prefix convention** — Go's `template.Must`, `regexp.MustCompile`,
   and Go's own `panic` on impossible states establish this pattern. Our
   `mustCommand` follows the same convention.

## Consequences

### Positive

- **Structurally impossible to forget cmdID** — `BasicCommand`'s constructor
  always mints a valid ID. No manual field to forget.
- **~60 lines of boilerplate removed** — 3 method declarations × 20 commands
  eliminated. Method promotion handles it automatically.
- **Consistent command identity** — all commands now use the same `ID()`,
  `Type()`, `AggregateID()` implementation from `BasicCommand`. No risk of
  subtle per-command variation.
- **Regression test guard** — `es_command_id_test.go` asserts all 20
  constructors produce non-zero IDs, plus uniqueness across batches.

### Negative

- **`panic` on construction failure** — not idiomatic for library code that
  might be called with dynamic input. Mitigated by the fact that the only
  inputs are a compile-time constant and a generated ID.

- **Pointer embedding** — `*command.BasicCommand` is a pointer, so commands are
  no longer purely value types. This is acceptable because commands are
  short-lived (created per request, dispatched, discarded) and never compared
  by value.

### Alternatives considered

1. **Return `(*Cmd, error)` from constructors** — rejected (see above).
2. **Use `command.MustNew` directly in constructors** — `command.New` doesn't
   have a `MustNew` variant, and adding one upstream just to avoid a local
   helper adds unnecessary coupling. The local `mustCommand` is clearer.
3. **Code generation** — generating command structs from a schema would
   eliminate the boilerplate entirely, but adds a build dependency (go generate)
   for marginal benefit over the embedding pattern.

## Related

- [ADR-0026](0026-command-idempotency-store.md) — The idempotency store that
  relies on `cmd.ID()` for dedup.
- [go-cqrs-lite command/v4](https://github.com/LarsArtmann/go-cqrs-lite) —
  `BasicCommand` struct and `command.New` constructor.
