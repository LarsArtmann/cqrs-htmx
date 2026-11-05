package eventcatalog

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Handler returns an http.HandlerFunc that serves the given
// event catalog as indented JSON with an ETag-driven `Cache-Control: no-cache`
// (every unchanged fetch is a 304; a changed catalog is refetched — see
// [hx.ServeImmutableJSON]), mirroring [OpenAPISpecHandler].
//
// The catalog is serialized eagerly when this function is called, so any
// serialization error surfaces once at startup. The returned handler holds
// only the immutable serialized bytes and ETag — safe for concurrent use.
//
//	catalog := cqrshtmx.New()
//	// ... register events ...
//	handler, err := cqrshtmx.Handler(catalog)
//	if err != nil { log.Fatal(err) }
//	mux.Handle("GET /events/catalog", handler)
func Handler(catalog *Catalog) (http.HandlerFunc, error) {
	if catalog == nil {
		return nil, errorfamily.NewInfrastructure("cqrshtmx.event_catalog.nil", "catalog must not be nil")
	}

	return hx.SerializeToImmutableHandler(catalog, "cqrshtmx.event_catalog.serialize", "serialize event catalog")
}
