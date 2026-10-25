package cqrshtmx

import (
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
	etag "github.com/larsartmann/go-etag/server"
)

// jsonSerializer is the structural interface for any type with a JSON()
// method that returns pre-serialized bytes. Both *EventCatalog and
// *openapi.Spec satisfy it, allowing the serialize-immutable-handler
// boilerplate to live in exactly one place.
type jsonSerializer interface {
	JSON() ([]byte, error)
}

// serializeToImmutableHandler serializes the given jsonSerializer eagerly,
// wraps any serialization error as an Infrastructure failure, and wraps the
// result in an immutableJSONHandler. Shared by EventCatalogHandler and
// OpenAPISpecHandler so the serialize-error-return-newImmutableJSONHandler
// boilerplate lives in exactly one place.
func serializeToImmutableHandler(s jsonSerializer, errCode, errMsg string) (http.HandlerFunc, error) {
	data, err := s.JSON()
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, errCode, errMsg)
	}

	return newImmutableJSONHandler(data), nil
}

// EventCatalogHandler returns an http.HandlerFunc that serves the given
// event catalog as indented JSON with an ETag-driven `Cache-Control: no-cache`
// (every unchanged fetch is a 304; a changed catalog is refetched — see
// serveImmutableJSON), mirroring [OpenAPISpecHandler].
//
// The catalog is serialized eagerly when this function is called, so any
// serialization error surfaces once at startup. The returned handler holds
// only the immutable serialized bytes and ETag — safe for concurrent use.
//
//	catalog := cqrshtmx.NewEventCatalog()
//	// ... register events ...
//	handler, err := cqrshtmx.EventCatalogHandler(catalog)
//	if err != nil { log.Fatal(err) }
//	mux.Handle("GET /events/catalog", handler)
func EventCatalogHandler(catalog *EventCatalog) (http.HandlerFunc, error) {
	if catalog == nil {
		return nil, errorfamily.NewInfrastructure("cqrshtmx.event_catalog.nil", "catalog must not be nil")
	}

	return serializeToImmutableHandler(catalog, "cqrshtmx.event_catalog.serialize", "serialize event catalog")
}

// newImmutableJSONHandler wraps pre-serialized JSON bytes in an
// immutableJSONServer and returns its serve method. Shared by
// EventCatalogHandler and OpenAPISpecHandler so the ETag + server construction
// lives in exactly one place.
func newImmutableJSONHandler(data []byte) http.HandlerFunc {
	return (&immutableJSONServer{
		etag:    `"` + hashTag(data) + `"`,
		marshal: data,
	}).serve
}

// immutableJSONServer holds eagerly-serialized JSON bytes and their ETag. Both
// fields are set once at construction and never mutated, so serve is safe for
// concurrent use without synchronization. Shared by EventCatalogHandler and
// OpenAPISpecHandler.
type immutableJSONServer struct {
	etag    string
	marshal []byte
}

func serveImmutableJSON(w http.ResponseWriter, r *http.Request, tag string, data []byte) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	// The bytes are an immutable snapshot, but the URL is not versioned: a new
	// process or an edited catalog/spec changes the content under the same URL.
	// "no-cache" forces revalidation while the ETag keeps every unchanged fetch
	// a 304 — a 1-year immutable directive would pin clients to stale specs.
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", tag)

	if current, ok := etag.ParseETag(tag); ok && etag.MatchesIfNoneMatch(current, r.Header.Get("If-None-Match")) {
		w.WriteHeader(http.StatusNotModified)

		return
	}

	writeAll(w, data)
}

func (s *immutableJSONServer) serve(w http.ResponseWriter, r *http.Request) {
	serveImmutableJSON(w, r, s.etag, s.marshal)
}
