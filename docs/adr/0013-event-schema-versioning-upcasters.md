# ADR 0013: Event Schema Versioning via Upcasters

**Date:** 2026-06-18
**Status:** Accepted

## Context

The event-sourced User aggregate persists events as JSON payloads. As the domain
model evolves, payload schemas change (fields added, renamed, removed). Old events
in the store still have the old schema, but the current code expects the new shape.

All event payload structs already embed `SchemaVersion int json:"schema_version"`,
and `currentSchemaVersion = 1` is the current version. Before this ADR, schema
evolution was not addressed — any schema change would break decoding of old events.

## Decision

Implement an **upcaster registry** pattern:

1. **`Upcaster` func type** — transforms raw JSON bytes from version N to N+1.
2. **`UpcasterRegistry`** — holds versioned upcasters per event type. Chains
   multiple upcasters to upgrade from any old version to `currentSchemaVersion`.
3. **`SetUpcasterRegistry(r)`** — configures a package-level registry used by all
   decode paths (foldUser, UserReadModel, CasbinProjection).
4. **`unmarshalPayload[T]`** — the shared decode helper now calls `applyUpcasters`
   before JSON unmarshaling. All three decode sites use this helper.

### How it works

When an event is loaded from the store:
1. `extractSchemaVersion(raw)` reads the `schema_version` field from JSON (defaults
   to 0 for pre-versioning events).
2. If version < `currentSchemaVersion`, the registry chains upcasters:
   `v0→v1→v2→...→current`.
3. Each upcaster transforms the raw bytes (e.g., renames a field, adds a default).
4. The final bytes are JSON-unmarshaled into the typed payload struct.

### Example: renaming `name` to `display_name` in v2

```go
r := usermgmt.NewUpcasterRegistry()
r.Register(usermgmt.EventUserRegistered(), 1, func(raw []byte) ([]byte, error) {
    var m map[string]any
    if err := json.Unmarshal(raw, &m); err != nil { return nil, err }
    m["schema_version"] = 2
    if old, ok := m["name"]; ok {
        m["display_name"] = old
        delete(m, "name")
    }
    return json.Marshal(m)
})
usermgmt.SetUpcasterRegistry(r)
```

### Why package-level registry?

All three decode paths (foldUser, UserReadModel, CasbinProjection) are internal
to the `usermgmt` package. A package-level registry configured at startup is the
simplest approach that avoids threading the registry through every struct and
function signature. `SetUpcasterRegistry` is called once during service init.

## Consequences

**Positive:**

- Schema evolution is now possible without breaking old events.
- Zero overhead when no upcasters are registered (fast path returns raw bytes).
- Centralized decode path — all three sites use `unmarshalPayload[T]`.
- Versioning is embedded in payloads, not in a separate version column.

**Negative:**

- Package-level global state (mitigated by RWMutex + setup-time-only convention).
- Upcasters must produce valid JSON — a malformed upcaster corrupts all events
  of that type. Tests cover error cases.
