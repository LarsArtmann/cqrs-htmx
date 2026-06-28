# Email Branded-Type Upcaster Design

> How to migrate the `Email` branded type into existing event payloads without
> breaking stored events.

**Date:** 2026-06-28 · **Status:** Design — Deferred to next major version

## Problem

The `Email` branded type (`type Email string` with `ParseEmail`/`MustParseEmail`)
exists but is not used in event payload structs. 10+ structs use raw `string`:

- `UserRegisteredPayload.Email string`
- `UserState.Email string`
- `ExternalAccount.Email string`
- `EmailChangedPayload.Email string`
- etc.

Changing these to `Email` would change JSON marshaling from `"email": "user@example.com"`
to potentially `"email": {"value": "user@example.com"}` if `Email` is a struct —
breaking existing events in all event stores.

## Constraint

**Existing events in production stores have `"email": "user@example.com"` as a JSON
string.** The Go type change must be transparent on the wire.

## Design

### Step 1: Make Email JSON-transparent (next major)

```go
type Email string

// MarshalJSON returns the raw string — identical wire format to `string`.
func (e Email) MarshalJSON() ([]byte, error) {
    return json.Marshal(string(e))
}

// UnmarshalJSON accepts a raw string — identical wire format.
func (e *Email) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return err
    }
    *e = Email(s)
    return nil
}
```

This is a **zero-wire-change migration**: the JSON output is identical whether
the field is `string` or `Email`. No upcaster needed.

### Step 2: Use Email in event payloads (next major)

```go
type UserRegisteredPayload struct {
    SchemaVersion int    `json:"schema_version"`
    Email         Email  `json:"email"` // was string — wire format unchanged
    DisplayName   string `json:"display_name"`
    Roles         []Role `json:"roles"`
}
```

### Step 3: Add validation at construction

`ParseEmail` validates RFC 5322 format. Use it at command construction sites
(`RegisterRequest.Validate`, `ImportUser.Validate`) rather than in the payload.

## Why Not an Upcaster

The `Email` type is a `type Email string` — a named type over string. Its JSON
representation is identical to `string`. There is no schema change. An upcaster
is unnecessary.

The only risk is if `Email` were changed to a struct (`type Email struct{...}`),
which would change the wire format. We explicitly avoid this.

## Related

- TODO_LIST: "Use Email branded type in domain models (MEDIUM) — BLOCKED BY EVENT SERIALIZATION"
- ADR 0013: Event Schema Versioning + Upcasters
