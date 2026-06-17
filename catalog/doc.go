// Package cataloghtmx bridges go-cqrs-lite's catalog module with cqrs-htmx
// to provide automatic API documentation generation from Go CQRS types.
//
// The package offers a fluent Builder that wraps catalog.Builder and
// adds HTTP handler functions for serving documentation in four formats:
//
//   - OpenAPI 3.0 (JSON/YAML)
//   - AsyncAPI 3.0 (JSON/YAML)
//   - D2 architecture diagrams
//   - EventCatalog MDX file trees
//
// Consumers describe their command, query, and event types once using
// generic type parameters. Schemas, names, and directions are auto-derived
// via reflection. The resulting catalog can be served as HTTP endpoints
// for Swagger UI, AsyncAPI Studio, D2 renderers, or EventCatalog.
//
// Build panics on validation failure (duplicate IDs, empty service ID).
// Use BuildValid to receive violations instead — useful when the catalog
// is assembled from multiple sources and you want to report all problems.
//
// Example:
//
//	b := cataloghtmx.New("User Service", "1.0.0")
//	cataloghtmx.Command[RegisterUserCmd](b, "register-user",
//	    cataloghtmx.WithOperation("POST", "/api/users"))
//	cataloghtmx.Event[UserRegisteredEvent](b, "user.registered", catalog.Sends)
//	cataloghtmx.Query[GetUserQuery](b, "get-user",
//	    cataloghtmx.WithOperation("GET", "/api/users/{id}"))
//
//	// Build panics on invalid catalogs (fail-fast at startup).
//	cat := b.Build()
//
//	// BuildValid returns violations instead — use when assembling from
//	// untrusted or partially-known sources.
//	if _, violations := b.BuildValid(); len(violations) > 0 {
//	    log.Printf("catalog violations: %v", violations)
//	}
//
//	mux.Handle("/docs/openapi.json", cataloghtmx.OpenAPIHandler(cat))
//	mux.Handle("/docs/asyncapi.json", cataloghtmx.AsyncAPIHandler(cat))
//	mux.Handle("/docs/diagram.d2", cataloghtmx.D2Handler(cat))
package cataloghtmx
