package hx

import (
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
	etag "github.com/larsartmann/go-etag/server"
)

// JSONSerializer is the structural interface for any type with a JSON() method
// that returns pre-serialized bytes. Both the event catalog and the OpenAPI
// spec satisfy it, allowing the serialize-immutable-handler boilerplate to live
// in exactly one place.
type JSONSerializer interface {
	JSON() ([]byte, error)
}

// SerializeToImmutableHandler serializes the given JSONSerializer eagerly,
// wraps any serialization error as an Infrastructure failure, and wraps the
// result in an immutable JSON handler. Shared by the event-catalog handler and
// the OpenAPI spec handler so the
// serialize-error-return-newImmutableJSONHandler boilerplate lives in exactly
// one place.
func SerializeToImmutableHandler(s JSONSerializer, errCode, errMsg string) (http.HandlerFunc, error) {
	data, err := s.JSON()
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, errCode, errMsg)
	}

	return NewImmutableJSONHandler(data), nil
}

// NewImmutableJSONHandler wraps pre-serialized JSON bytes in an
// immutableJSONServer and returns its serve method. Shared by the event-catalog
// handler and the OpenAPI spec handler so the ETag + server construction lives
// in exactly one place.
func NewImmutableJSONHandler(data []byte) http.HandlerFunc {
	return (&immutableJSONServer{
		etag:    `"` + HashTag(data) + `"`,
		marshal: data,
	}).serve
}

// immutableJSONServer holds eagerly-serialized JSON bytes and their ETag. Both
// fields are set once at construction and never mutated, so serve is safe for
// concurrent use without synchronization.
type immutableJSONServer struct {
	etag    string
	marshal []byte
}

// ServeImmutableJSON serves an immutable JSON snapshot with ETag revalidation.
func ServeImmutableJSON(w http.ResponseWriter, r *http.Request, tag string, data []byte) {
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

	WriteAll(w, data)
}

func (s *immutableJSONServer) serve(w http.ResponseWriter, r *http.Request) {
	ServeImmutableJSON(w, r, s.etag, s.marshal)
}

// ServeJS returns the shared handler for serving JavaScript with long-lived
// caching. Used by the HTMX script/extension handlers and the offline-sync
// worker/client handlers.
func ServeJS(js []byte, tag string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", tag)

		if current, ok := etag.ParseETag(tag); ok && etag.MatchesIfNoneMatch(current, r.Header.Get("If-None-Match")) {
			w.WriteHeader(http.StatusNotModified)

			return
		}

		WriteAll(w, js)
	})
}
