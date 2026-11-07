# cqrs-htmx Basic Example

A minimal onboarding example showing the core cqrs-htmx library usage.

## What it demonstrates

- **App builder** — `cqrshtmx.Config{Commands: disp, Queries: disp}`
- **Command handler** — POST the page's form with form decode via mapper function (`cqrshtmx.DecodeForm`; JSON decode stays for API-shaped endpoints like `/api/audit` and the search twin)
- **Query handler** — GET with JSON result rendering
- **HTTP QUERY method** — `QUERY /api/items/search` (RFC 10008: safe, idempotent, body-carrying read) with a `POST /api/items/search` fallback twin — one handler, both transports
- **Actor metadata** — POST `/api/audit` propagates the acting user from the request context into command metadata (`cqrshtmx.CommandOptionsFromContext`) and reads it back — the audit-trail pattern
- **Partial-render showcase** — GET `/demo/items` serves the full page for direct visits and just the fragment for HTMX (`cqrshtmx.RenderPartialOrFull`, templ), refreshes the `#item-count` badge out-of-band (`cqrshtmx.OOBHTML`), and GET `/demo/items/count` answers the bare count when `HX-Target` is `#item-count` (`cqrshtmx.RenderIf` custom predicate)
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
    cqrshtmx.DecodeForm(func(req createItemRequest) (command.Command, error) {
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
broadcaster.Broadcast(sse.Event{Event: "itemCreated", Data: jsonStr})
```

### HTTP QUERY method (RFC 10008)

The same search served three ways — native `QUERY` with a JSON body, the `POST` fallback twin (identical handler), and plain `GET` with URL params for tiny filters:

```bash
curl -X QUERY localhost:8096/api/items/search -H 'Content-Type: application/json' \
  -d '{"filter":"demo"}'
curl -X POST localhost:8096/api/items/search -H 'Content-Type: application/json' \
  -d '{"filter":"demo"}'
```

Both return the identical JSON — `App.Query` is method-agnostic, so the mux pattern carries the method. See `docs/guides/http-query-method.md` in the repo root module for the full fallback ladder (including `MethodOverrideMiddleware`, which lets one `QUERY` route serve the POST twin via `X-HTTP-Method-Override`).
