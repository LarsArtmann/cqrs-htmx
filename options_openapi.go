package cqrshtmx

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
	"github.com/larsartmann/cqrs-htmx/v4/openapi"
	errorfamily "github.com/larsartmann/go-error-family"
)

// WithOpenAPI attaches OpenAPI 3.1 operation metadata to a command or query
// handler. The metadata is pure documentation — it has no runtime effect on
// dispatch — but lets the operation description travel with the handler
// definition so a spec generator or tooling can discover it.
//
// Because cqrs-htmx does not own the consumer's router (paths are registered on
// the consumer's mux), the library cannot auto-assemble a full spec from
// registered handlers. Build the spec explicitly with the openapi package, then
// serve it:
//
//	handler, err := cqrshtmx.OpenAPISpecHandler(spec)
//	if err != nil {
//	    // the spec failed to serialize — a programming error; fail fast at startup
//	}
//	mux.Handle("GET /openapi.json", handler)
//
//	app.Command("CreateItem",
//		cqrshtmx.DecodeJSON(...),
//		cqrshtmx.WithOpenAPI(openapi.Post("CreateItem").Summary("Create an item").Op()),
//	)
func WithOpenAPI(op openapi.Operation) HandlerOption {
	return func(config *handlerConfig) {
		config.openapiMeta = &op
	}
}

// OpenAPISpecHandler returns an http.HandlerFunc that serves the given spec as
// indented OpenAPI 3.1 JSON with an ETag-derived `Cache-Control: no-cache`
// (unchanged fetches are 304s; a changed spec is refetched), suitable for
// mounting at /openapi.json.
//
// The spec is serialized eagerly when this function is called, so any
// serialization error surfaces once at startup rather than on the first
// request. The returned handler holds only the immutable serialized bytes and
// ETag, so it is safe for concurrent use without synchronization.
//
//	handler, err := cqrshtmx.OpenAPISpecHandler(spec)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	mux.Handle("GET /openapi.json", handler)
func OpenAPISpecHandler(spec *openapi.Spec) (http.HandlerFunc, error) {
	if spec == nil {
		return nil, errorfamily.NewInfrastructure("cqrshtmx.openapi.nil-spec", "spec must not be nil")
	}

	return hx.SerializeToImmutableHandler(spec, "cqrshtmx.openapi.serialize", "serialize OpenAPI spec")
}
