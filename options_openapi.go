package cqrshtmx

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/openapi"
	errorfamily "github.com/larsartmann/go-error-family"
)

// WithOpenAPI attaches OpenAPI 3.1 operation metadata to a command or query
// handler. The metadata is pure documentation — it has no runtime effect on
// dispatch — but lets the operation description travel with the handler
// definition so a spec generator or tooling can discover it.
//
// Because cqrs-htmx does not own the consumer's router (paths are registered on
// the consumer's mux), the library cannot auto-assemble a full spec. Build the
// spec with the openapi package, then serve it:
//
//	spec := openapi.New("My API", "1.0.0").
//		Path("/items",
//			openapi.Post("CreateItem").
//				JSONBody(openapi.Object(openapi.PropReq("name", openapi.String()))).
//				Response(201, "Created"),
//		)
//	mux.Handle("GET /openapi.json", cqrshtmx.OpenAPISpecHandler(spec))
//
//	app.Command("CreateItem",
//		cqrshtmx.DecodeJSON(...),
//		cqrshtmx.WithOpenAPI(openapi.Post("CreateItem").Summary("Create an item").Op()),
//	)
func WithOpenAPI(op openapi.Operation) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.openapiMeta = &op
	}
}

// OpenAPISpecHandler returns an http.HandlerFunc that serves the given spec as
// indented OpenAPI 3.1 JSON with a long immutable cache header, suitable for
// mounting at /openapi.json. The spec is serialized once on the first request
// and cached; subsequent requests return the cached bytes with a 1-year
// Cache-Control and an ETag derived from the content.
//
//	mux.Handle("GET /openapi.json", cqrshtmx.OpenAPISpecHandler(spec))
func OpenAPISpecHandler(spec *openapi.Spec) http.HandlerFunc {
	server := &openAPISpecServer{spec: spec, cached: false, etag: "", marshal: nil}

	return server.serve
}

// openAPISpecServer holds the lazily-serialized spec bytes and ETag.
type openAPISpecServer struct {
	spec    *openapi.Spec
	cached  bool
	etag    string
	marshal []byte
}

func (s *openAPISpecServer) serve(w http.ResponseWriter, r *http.Request) {
	if err := s.ensureSerialized(); err != nil {
		http.Error(w, "failed to serialize OpenAPI spec", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", s.etag)

	if match := r.Header.Get("If-None-Match"); match != "" && match == s.etag {
		w.WriteHeader(http.StatusNotModified)

		return
	}

	_, _ = w.Write(s.marshal)
}

func (s *openAPISpecServer) ensureSerialized() error {
	if s.cached {
		return nil
	}

	data, err := s.spec.JSON()
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "cqrshtmx.openapi.serialize", "serialize OpenAPI spec")
	}

	s.marshal = data
	s.etag = `"` + hashTag(data) + `"`
	s.cached = true

	return nil
}

// hashTag derives a short, stable cache tag from the spec bytes via FNV-1a. It
// is not cryptographically significant — it only needs to change when the
// content changes (for ETag invalidation).
func hashTag(data []byte) string {
	const fnvOffset uint64 = 14695981039346656037
	const fnvPrime uint64 = 1099511628211

	hash := fnvOffset

	for _, b := range data {
		hash ^= uint64(b)
		hash *= fnvPrime
	}

	return uint64ToHex(hash)
}

func uint64ToHex(n uint64) string {
	const hexDigits = "0123456789abcdef"

	var buf [16]byte

	for i := 15; i >= 0; i-- {
		buf[i] = hexDigits[n&0xf]
		n >>= 4
	}

	return string(buf[:])
}
