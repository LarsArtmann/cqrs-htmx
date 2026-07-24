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

handler, err := cqrshtmx.EventCatalogHandler(svc.EventCatalog())
if err != nil {
    log.Fatal(err)
}
mux.Handle("GET /events/catalog", handler)
```

The handler:
- Serializes the catalog eagerly at construction (startup errors surface immediately).
- Serves immutable JSON with a 1-year `Cache-Control` and an FNV-1a ETag.
- Returns 304 on matching `If-None-Match`.

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

## Registered Events (21 total)

| Aggregate | Event Types |
| --- | --- |
| User (12) | UserRegistered, RolesUpdated (legacy), EmailChanged, DisplayNameChanged, UserDeleted, CredentialAdded, CredentialRemoved, EmailVerified, TOTPEnabled, TOTPDisabled, ExternalAccountLinked, ExternalAccountUnlinked |
| Membership (3) | MemberAdded, MemberRolesChanged, MemberRemoved |
| Tenant (4) | TenantCreated, TenantSuspended, TenantReactivated, TenantDeleted |
| Bot (2) | BotRegistered, BotDeleted |

---

## Extending the Catalog

### Adding Custom Events

If you build custom aggregates on top of cqrs-htmx, register your events:

```go
catalog := cqrshtmx.NewEventCatalog()
catalog.Register(cqrshtmx.EventMetadata{
    Type:          "OrderPlaced",
    Aggregate:     "Order",
    SchemaVersion: 1,
    Description:   "A customer placed an order",
    PayloadFields: []cqrshtmx.PayloadField{
        {Name: "customer_id", Type: "string", Required: true},
        {Name: "total", Type: "int", Required: true},
        {Name: "items", Type: "[]OrderItem", Required: true},
    },
})

handler, _ := cqrshtmx.EventCatalogHandler(catalog)
mux.Handle("GET /events/catalog", handler)
```

### Merging with the Default Catalog

To serve both usermgmt events and your custom events in one catalog:

```go
catalog := usermgmt.DefaultEventCatalog()
catalog.Register(cqrshtmx.EventMetadata{
    Type: "OrderPlaced",
    // ...
})
handler, _ := cqrshtmx.EventCatalogHandler(catalog)
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

- [Projection Health Monitoring](./projection-health-monitoring.md)
- `event_catalog.go` — The EventCatalog type
- `event_catalog_handler.go` — The HTTP handler
