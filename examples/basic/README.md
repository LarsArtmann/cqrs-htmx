# cqrs-htmx Basic Example

A minimal onboarding example showing the core cqrs-htmx library usage.

## What it demonstrates

- **App builder** — `cqrshtmx.Config{Commands: disp, Queries: disp}`
- **Command handler** — POST with JSON decode via mapper function
- **Query handler** — GET with JSON result rendering
- **SSE live updates** — Broadcaster fan-out on command success
- **Embedded HTMX** — Self-hosted HTMX v2.0.10 JS

## Run

```bash
cd examples/basic
go run .
```

Open http://localhost:8096

## Key patterns

### Command with custom type

```go
type createItemCmd struct {
    typ   command.Type
    aggID id.AggregateID
    Name  string
}

app.Command("CreateItem",
    cqrshtmx.DecodeJSON(func(req createItemRequest) (command.Command, error) {
        return &createItemCmd{typ: "CreateItem", aggID: id.NewAggregateID(), Name: req.Name}, nil
    }),
    cqrshtmx.WithSuccessStatus(201),
)
```

### Query with JSON rendering

```go
app.Query("ListItems",
    cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
        return &listItemsQuery{}, nil
    }),
    cqrshtmx.RenderJSON[[]item](),
)
```

### SSE live updates

```go
broadcaster.Broadcast(cqrshtmx.SSEEvent{Event: "itemCreated", Data: jsonStr})
```
