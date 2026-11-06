# Event Catalog Guide

> How to serve and use the event catalog — a discoverable schema for all event types published by usermgmt.

---

## What Is the Event Catalog?

Events are part of the public API surface for any consumer building custom projections. The event catalog makes this contract explicit: it lists every event type, which aggregate owns it, the payload schema version, and the payload fields.

This implements the **Published Language** pattern from DDD: a shared, well-documented vocabulary that all integration partners can rely on.

---

## Serving the Catalog

Mount the catalog handler at an HTTP endpoint:

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
defer svc.Close()

handler, err := eventcatalog.Handler(svc.EventCatalog())
if err != nil {
    log.Fatal(err)
}
mux.Handle("GET /events/catalog", handler)
```

The handler:

- Serializes the catalog eagerly at construction (startup errors surface immediately).
- Serves immutable JSON with a 1-year `Cache-Control` and an FNV-1a ETag.
- Returns 304 on matching `If-None-Match`.

> **Not the doc-generation suite.** This catalog is schema metadata only. For
> full API documentation (OpenAPI/AsyncAPI specs, D2 diagrams, EventCatalog MDX)
> generated from Go types, use [`go-cqrs-lite/catalog/v4`](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/catalog/v4)
> — see [`examples/catalog-demo`](../../examples/catalog-demo/) for the executable
> recipe and [See Also](#see-also) for the distinction.

---

## Catalog Format

The catalog is a JSON array of event metadata:

```json
[
  {
    "type": "UserRegistered",
    "aggregate": "User",
    "schema_version": 2,
    "description": "A user registered with an email address and display name",
    "payload_fields": [
      {"name": "schema_version", "type": "int", "required": true},
      {"name": "email", "type": "string", "required": true},
      {"name": "display_name", "type": "string"},
      {"name": "roles", "type": "[]Role"}
    ]
  },
  ...
]
```

---

## Registered Events (27 total)

| Aggregate      | Event Types                                                                                                                                                                                                        |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| User (13)      | UserRegistered, RolesUpdated (legacy), EmailChanged, DisplayNameChanged, UserDeleted, CredentialAdded, CredentialRemoved, CredentialUsed, EmailVerified, TOTPEnabled, TOTPDisabled, ExternalAccountLinked, ExternalAccountUnlinked |
| Membership (3) | MemberAdded, MemberRolesChanged, MemberRemoved                                                                                                                                                                     |
| Tenant (4)     | TenantCreated, TenantSuspended, TenantReactivated, TenantDeleted                                                                                                                                                   |
| Bot (2)        | BotRegistered, BotDeleted                                                                                                                                                                                          |
| Auth (5)       | LoginSucceeded, LoginFailed, RegistrationClosed, SessionStarted, SessionEnded                                                                                                                                      |

The inventory is drift-gated: `TestEventCatalog_InventoryMatchesEventConstants`
(usermgmt) fails when an identity-model event constant and the catalog table
drift apart, and `catalog.AssertEventInventory` pins the same inventory in the
architecture format (see below).

---

## Extending the Catalog

### Adding Custom Events

If you build custom aggregates on top of cqrs-htmx, register your events:

```go
catalog := eventcatalog.New()
catalog.Register(eventcatalog.EventMetadata{
    Type:          "OrderPlaced",
    Aggregate:     "Order",
    SchemaVersion: 1,
    Description:   "A customer placed an order",
    PayloadFields: []eventcatalog.PayloadField{
        {Name: "customer_id", Type: "string", Required: true},
        {Name: "total", Type: "int", Required: true},
        {Name: "items", Type: "[]OrderItem", Required: true},
    },
})

handler, _ := eventcatalog.Handler(catalog)
mux.Handle("GET /events/catalog", handler)
```

### Merging with the Default Catalog

To serve both usermgmt events and your custom events in one catalog:

```go
catalog := usermgmt.DefaultEventCatalog()
catalog.Register(eventcatalog.EventMetadata{
    Type: "OrderPlaced",
    // ...
})
handler, _ := eventcatalog.Handler(catalog)
```

---

## How Consumers Use the Catalog

### Building Custom Projections

A consumer building a projection needs to know:

1. What events exist (the `type` field).
2. What aggregate owns them (the `aggregate` field).
3. What the payload looks like (the `payload_fields` array).
4. What schema version is current (the `schema_version` field).

The catalog provides all of this without the consumer needing to read the source code.

### Integration Testing

Fetch the catalog at test time to verify your projection handles all event types:

```go
resp, _ := http.Get("http://localhost:8080/events/catalog")
var events []EventMetadata
json.NewDecoder(resp.Body).Decode(&events)
for _, e := range events {
    t.Run("handle_"+e.Type, func(t *testing.T) {
        // Verify your projection handles this event type
    })
}
```

---

## See Also

- **Two catalogs — do not conflate:**
  1. **`eventcatalog.Catalog`** (this guide) — dependency-free, immutable-JSON
     schema surface for the 27 identity events (Published Language).
  2. **[`go-cqrs-lite/catalog/v4`](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/catalog/v4)**
     — the heavier doc-generation suite (OpenAPI/AsyncAPI 3.0, D2 diagrams,
     EventCatalog MDX from Go types), pulling templ/yaml into the graph. Not
     embedded by `setup`; wiring it is a consumer-side composition.

     The bridge: **`usermgmt.ArchitectureCatalog()`** derives the same 27-event
     published language as a `catalog.Catalog` (one table feeds both wire
     formats — they cannot drift), and **`systemadapter.ArchitectureCatalog(ctx)`**
     derives the full 21-command + 27-event inventory from `DomainConfig()`
     through `systemcatalog` (ephemeral in-memory boot, nothing starts).
- [catalog/v4 integration map](../research/2026-10-12_catalog-v4-integration-map.md) —
  who depends on which catalog, and why `setup` ships only the light one
- [`examples/catalog-demo`](../../examples/catalog-demo/) — executable guide:
  `simple` builder + `docserver` serving OpenAPI/AsyncAPI/D2
- [`integration_test/catalog_test.go`](../../integration_test/catalog_test.go) —
  contract test proving the composition over a live App
- [Projection Health Monitoring](./projection-health-monitoring.md)
- `eventcatalog/catalog.go` — the `Catalog` type (deprecated root alias `cqrshtmx.EventCatalog`)
- `eventcatalog/handler.go` — the HTTP handler (deprecated root alias `cqrshtmx.EventCatalogHandler`)
