# catalog — Automatic API Documentation for CQRS-HTMX

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/cqrs-htmx/catalog/v3.svg)](https://pkg.go.dev/github.com/larsartmann/cqrs-htmx/catalog/v3)

Generate [OpenAPI 3.0](https://swagger.io/specification/), [AsyncAPI 3.0](https://www.asyncapi.com/), [D2](https://d2lang.com/) architecture diagrams, and [EventCatalog](https://www.eventcatalog.dev/) documentation from your Go CQRS types.

```bash
go get github.com/larsartmann/cqrs-htmx/catalog/v3
```

## Overview

Describe your commands, queries, and events once using Go generic type parameters. The catalog sub-package auto-derives JSON schemas from struct tags, generates human-readable names from type names, and serves everything as HTTP endpoints.

**Zero dependencies on cqrs-htmx root or usermgmt.** This module only depends on `go-cqrs-lite/catalog/v3`. Consumers opt in independently.

## Quick Start

```go
import (
    cataloghtmx "github.com/larsartmann/cqrs-htmx/catalog/v3"
    "github.com/larsartmann/go-cqrs-lite/catalog/v3"
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

| Function                              | Description                                |
| ------------------------------------- | ------------------------------------------ |
| `New(title, version, opts...)`        | Create a single-service catalog builder    |
| `Command[T](b, id, opts...)`          | Register a command message                 |
| `Query[T](b, id, opts...)`            | Register a query message                   |
| `Event[T](b, id, direction, opts...)` | Register an event message                  |
| `b.AddMessage(msg)`                   | Add a pre-built MessageConfig              |
| `b.Build()`                           | Produce validated `*catalog.Catalog`       |
| `b.BuildValid()`                      | Produce catalog with validation violations |

### Builder Options

| Option                  | Description                                        |
| ----------------------- | -------------------------------------------------- |
| `WithServiceID(id)`     | Override service ID (default: kebab-case of title) |
| `WithServiceName(name)` | Override service display name                      |
| `WithServiceSummary(s)` | Set service summary                                |

### Message Options

| Option                        | Description                   |
| ----------------------------- | ----------------------------- |
| `WithOperation(method, path)` | Attach HTTP endpoint metadata |
| `catalog.WithSummary(s)`      | Set message summary           |
| `catalog.WithName(n)`         | Override auto-derived name    |
| `catalog.Owners(teams...)`    | Set message owners            |
| `catalog.Labels(map)`         | Set key-value labels          |

### HTTP Handlers

| Handler                         | Format       | Content-Type                                   |
| ------------------------------- | ------------ | ---------------------------------------------- |
| `OpenAPIHandler(cat, opts...)`  | OpenAPI 3.0  | application/json (default) or application/yaml |
| `AsyncAPIHandler(cat, opts...)` | AsyncAPI 3.0 | application/json (default) or application/yaml |
| `D2Handler(cat, opts...)`       | D2 diagram   | text/plain                                     |
| `HealthCheckHandler(cat)`       | JSON status  | application/json                               |

### Serve Options

| Option                  | Description                           |
| ----------------------- | ------------------------------------- |
| `WithDescription(desc)` | Set API description                   |
| `WithBasePath(path)`    | Set OpenAPI base path (default: /api) |
| `WithFormat(f)`         | Select JSON or YAML output            |

## Schema Reflection

Struct tags are auto-derived into JSON Schema:

| Tag           | Example                    | Effect                |
| ------------- | -------------------------- | --------------------- |
| `json`        | `json:"email,omitempty"`   | Field name + required |
| `doc`         | `doc:"User email"`         | Description           |
| `description` | `description:"User email"` | Alias for `doc`       |
| `format`      | `format:"email"`           | JSON Schema format    |
| `enum`        | `enum:"active,inactive"`   | Enum values           |
| `default`     | `default:"active"`         | Default value         |

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

## Recipe: Catalog for the `usermgmt` Module

There is intentionally **no** `usermgmtcatalog.Default()` package. A pre-built
catalog would force one of two undesirable options: either the `catalog/`
module gains a dependency on `usermgmt` (breaking its zero-dependency
principle), or `usermgmt` gains a dependency on `catalog` (dragging the YAML
marshaller into every usermgmt consumer). Instead, document the events and
commands in your own application — it is roughly 20 lines.

The event payload types (`UserRegisteredPayload`, `RolesUpdatedPayload`, …) are
exported with JSON struct tags, so they reflect cleanly. The command structs
have unexported fields by design (they carry an `id.AggregateID`), so define
thin exported DTO types that mirror their HTTP request shapes:

```go
import (
    cataloghtmx "github.com/larsartmann/cqrs-htmx/catalog/v3"
    "github.com/larsartmann/go-cqrs-lite/catalog/v3"
    "github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

// Command request DTOs — exported fields with struct tags for schema reflection.
type registerUserRequest struct {
    Email       string   `json:"email"        doc:"User email address"`
    DisplayName string   `json:"display_name" doc:"Display name"`
    Roles       []string `json:"roles"        doc:"Initial roles"`
}

type changeEmailRequest struct {
    Email string `json:"email" doc:"New email address"`
}

// ...define DTOs for the remaining commands as needed...

func usermgmtCatalog() *catalog.Catalog {
    b := cataloghtmx.New("User Management", "1.0.0")

    // Commands — register the DTOs that describe the HTTP request shapes.
    cataloghtmx.Command[registerUserRequest](b, "register-user",
        cataloghtmx.WithOperation("POST", "/auth/register"))
    cataloghtmx.Command[changeEmailRequest](b, "change-email",
        cataloghtmx.WithOperation("POST", "/auth/change-email"))

    // Events — the persisted payloads are the real contract; reflect them directly.
    cataloghtmx.Event[usermgmt.UserRegisteredPayload](b, "user.registered", catalog.Sends)
    cataloghtmx.Event[usermgmt.RolesUpdatedPayload](b, "user.roles-updated", catalog.Sends)
    cataloghtmx.Event[usermgmt.EmailChangedPayload](b, "user.email-changed", catalog.Sends)
    cataloghtmx.Event[usermgmt.DisplayNameChangedPayload](b, "user.display-name-changed", catalog.Sends)
    cataloghtmx.Event[usermgmt.UserDeletedPayload](b, "user.deleted", catalog.Sends)
    cataloghtmx.Event[usermgmt.CredentialAddedPayload](b, "user.credential-added", catalog.Sends)
    cataloghtmx.Event[usermgmt.CredentialRemovedPayload](b, "user.credential-removed", catalog.Sends)
    cataloghtmx.Event[usermgmt.EmailVerifiedPayload](b, "user.email-verified", catalog.Sends)
    cataloghtmx.Event[usermgmt.TOTPEnabledPayload](b, "user.totp-enabled", catalog.Sends)
    cataloghtmx.Event[usermgmt.TOTPDisabledPayload](b, "user.totp-disabled", catalog.Sends)

    return b.Build()
}
```

This keeps the module boundaries clean while giving you a one-screen starting
point. Adjust the operations, descriptions, and DTO fields to match your
HTTP layer.

## Dependencies

| Dependency                | Purpose                                       |
| ------------------------- | --------------------------------------------- |
| `go-cqrs-lite/catalog/v3` | Catalog builder, schema reflection, exporters |
| `go-faster/yaml`          | YAML marshaling (transitive)                  |
