# catalog — Automatic API Documentation for CQRS-HTMX

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/cqrs-htmx/catalog/v2.svg)](https://pkg.go.dev/github.com/larsartmann/cqrs-htmx/catalog/v2)

Generate [OpenAPI 3.0](https://swagger.io/specification/), [AsyncAPI 3.0](https://www.asyncapi.com/), [D2](https://d2lang.com/) architecture diagrams, and [EventCatalog](https://www.eventcatalog.dev/) documentation from your Go CQRS types.

```bash
go get github.com/larsartmann/cqrs-htmx/catalog/v2
```

## Overview

Describe your commands, queries, and events once using Go generic type parameters. The catalog sub-package auto-derives JSON schemas from struct tags, generates human-readable names from type names, and serves everything as HTTP endpoints.

**Zero dependencies on cqrs-htmx root or usermgmt.** This module only depends on `go-cqrs-lite/catalog/v2`. Consumers opt in independently.

## Quick Start

```go
import (
    cataloghtmx "github.com/larsartmann/cqrs-htmx/catalog/v2"
    "github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

type RegisterUserCmd struct {
    Email       string `json:"email" doc:"User email address"`
    DisplayName string `json:"display_name" doc:"Display name"`
}

type UserRegisteredEvent struct {
    UserID string `json:"user_id" doc:"The new user ID"`
    Email  string `json:"email" doc:"User email"`
}

type GetUserQuery struct {
    ID string `json:"id" doc:"User ID to look up"`
}

func setupDocs() *catalog.Catalog {
    b := cataloghtmx.New("User Service", "1.0.0")

    cataloghtmx.Command[RegisterUserCmd](b, "register-user",
        cataloghtmx.WithOperation("POST", "/api/users"),
    )

    cataloghtmx.Event[UserRegisteredEvent](b, "user.registered", catalog.Sends)

    cataloghtmx.Query[GetUserQuery](b, "get-user",
        cataloghtmx.WithOperation("GET", "/api/users/{id}"),
    )

    return b.Build()
}
```

## Serving Documentation

```go
cat := setupDocs()

mux := http.NewServeMux()
mux.Handle("/docs/openapi.json", cataloghtmx.OpenAPIHandler(cat))
mux.Handle("/docs/asyncapi.json", cataloghtmx.AsyncAPIHandler(cat))
mux.Handle("/docs/diagram.d2", cataloghtmx.D2Handler(cat))
```

### YAML Output

```go
handler := cataloghtmx.OpenAPIHandler(cat,
    cataloghtmx.WithFormat(cataloghtmx.FormatYAML),
)
```

### Custom Base Path

```go
handler := cataloghtmx.OpenAPIHandler(cat,
    cataloghtmx.WithBasePath("/v2/api"),
)
```

## API Reference

### Builder

| Function | Description |
|----------|-------------|
| `New(title, version, opts...)` | Create a single-service catalog builder |
| `Command[T](b, id, opts...)` | Register a command message |
| `Query[T](b, id, opts...)` | Register a query message |
| `Event[T](b, id, direction, opts...)` | Register an event message |
| `b.AddMessage(msg)` | Add a pre-built MessageConfig |
| `b.Build()` | Produce validated `*catalog.Catalog` |
| `b.BuildValid()` | Produce catalog with validation violations |

### Builder Options

| Option | Description |
|--------|-------------|
| `WithServiceID(id)` | Override service ID (default: kebab-case of title) |
| `WithServiceName(name)` | Override service display name |
| `WithServiceSummary(s)` | Set service summary |

### Message Options

| Option | Description |
|--------|-------------|
| `WithOperation(method, path)` | Attach HTTP endpoint metadata |
| `catalog.WithSummary(s)` | Set message summary |
| `catalog.WithName(n)` | Override auto-derived name |
| `catalog.Owners(teams...)` | Set message owners |
| `catalog.Labels(map)` | Set key-value labels |

### HTTP Handlers

| Handler | Format | Content-Type |
|---------|--------|-------------|
| `OpenAPIHandler(cat, opts...)` | OpenAPI 3.0 | application/json (default) or application/yaml |
| `AsyncAPIHandler(cat, opts...)` | AsyncAPI 3.0 | application/json (default) or application/yaml |
| `D2Handler(cat, opts...)` | D2 diagram | text/plain |
| `EventCatalogHandler(cat, dir)` | EventCatalog MDX | application/zip |
| `HealthCheckHandler(cat)` | JSON status | application/json |

### Serve Options

| Option | Description |
|--------|-------------|
| `WithDescription(desc)` | Set API description |
| `WithBasePath(path)` | Set OpenAPI base path (default: /api) |
| `WithFormat(f)` | Select JSON or YAML output |

## Schema Reflection

Struct tags are auto-derived into JSON Schema:

| Tag | Example | Effect |
|-----|---------|--------|
| `json` | `json:"email,omitempty"` | Field name + required |
| `doc` | `doc:"User email"` | Description |
| `description` | `description:"User email"` | Alias for `doc` |
| `format` | `format:"email"` | JSON Schema format |
| `enum` | `enum:"active,inactive"` | Enum values |
| `default` | `default:"active"` | Default value |

## EventCatalog Generation

For production EventCatalog deployments, generate files at startup:

```go
cat := setupDocs()
err := cataloghtmx.GenerateEventCatalog(cat, "./docs/eventcatalog")
if err != nil {
    log.Fatal(err)
}
```

Then serve the generated directory with [EventCatalog CLI](https://www.eventcatalog.dev/docs/guides/quick-start).

## Advanced: Multi-Service Catalogs

Access the underlying `catalog.Builder` for multi-service setups:

```go
b := cataloghtmx.New("Platform", "1.0.0")

// Use the inner builder for advanced features
inner := b.InnerBuilder()
inner.AddDomain("identity", "Identity", "1.0.0", "User identity", "user-svc")
inner.AddChannel(/* ... */)
inner.AddDataStore(/* ... */)

cat := b.Build()
```

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `go-cqrs-lite/catalog/v2` | Catalog builder, schema reflection, exporters |
| `go-faster/yaml` | YAML marshaling (transitive) |
