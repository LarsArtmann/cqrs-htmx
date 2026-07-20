// Package openapi provides a fluent, dependency-free builder for OpenAPI 3.1
// specifications. It is designed for CQRS/HTTP libraries like cqrs-htmx whose
// endpoints are registered on a consumer-owned router: because the library
// cannot introspect the consumer's paths, this package lets the consumer
// (or a code generator) declare the spec explicitly and serialize it to valid
// OpenAPI 3.1 JSON in one call.
//
// # Quick start
//
//	spec := openapi.New("My API", "1.0.0").
//		Path("/items",
//			openapi.Post("CreateItem").
//				Summary("Create a new item").
//				Tag("items").
//				JSONBody(openapi.Object(
//					openapi.Prop("name", openapi.String().MinLength(1)),
//				)).
//				Response(201, "Created"),
//		).
//		Path("/items/{id}",
//			openapi.Get("GetItem").
//				PathParam("id", openapi.String(), "the item id").
//				Response(200, "OK", openapi.JSON(
//					openapi.Object(openapi.Prop("id", openapi.String())),
//				)),
//		)
//
//	data, err := spec.JSON()
//
// # cqrs-htmx integration
//
// Attach metadata to an [cqrshtmx.App] handler with the WithOpenAPI option
// (see options_openapi.go in the root package), then assemble the spec from
// your route table. The openapi package itself has no dependency on the root
// package and can be used standalone.
package openapi
