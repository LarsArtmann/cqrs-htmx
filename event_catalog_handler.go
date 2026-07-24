package cqrshtmx

import (
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// EventCatalogHandler returns an http.HandlerFunc that serves the given
// event catalog as indented JSON with a 1-year immutable Cache-Control and
// an FNV-1a ETag, mirroring [OpenAPISpecHandler].
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

	data, err := catalog.JSON()
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"cqrshtmx.event_catalog.serialize",
			"serialize event catalog")
	}

	server := &eventCatalogServer{
		etag:    `"` + hashTag(data) + `"`,
		marshal: data,
	}

	return server.serve, nil
}

type eventCatalogServer struct {
	etag    string
	marshal []byte
}

func (s *eventCatalogServer) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", s.etag)

	if match := r.Header.Get("If-None-Match"); match != "" && match == s.etag {
		w.WriteHeader(http.StatusNotModified)

		return
	}

	_, _ = w.Write(s.marshal)
}
