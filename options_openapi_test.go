package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/openapi"
)

// sampleSpec builds a minimal but realistic OpenAPI spec for handler tests.
func sampleSpec() *openapi.Spec {
	return openapi.New("Test API", "1.0.0").
		Path("/items",
			openapi.Get("ListItems").
				Summary("List all items").
				Response(http.StatusOK, "OK"),
			openapi.Post("CreateItem").
				JSONBody(openapi.Object(openapi.PropReq("name", openapi.String()))).
				Response(http.StatusCreated, "Created"),
		).
		Schema("Error", openapi.ErrorSchema())
}

// mustHandler builds the spec handler once (serializing eagerly), failing the
// test if the spec is invalid. Every test below shares the same serialized
// bytes, mirroring how a consumer mounts /openapi.json at startup.
func mustHandler(t *testing.T, spec *openapi.Spec) http.HandlerFunc {
	t.Helper()

	handler, err := cqrshtmx.OpenAPISpecHandler(spec)
	if err != nil {
		t.Fatalf("OpenAPISpecHandler returned error: %v", err)
	}

	return handler
}

func TestOpenAPISpecHandler_ServesJSON(t *testing.T) {
	handler := mustHandler(t, sampleSpec())

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want immutable", cc)
	}

	etag := w.Header().Get("ETag")
	if etag == "" || etag[0] != '"' {
		t.Errorf("ETag = %q, want a quoted non-empty value", etag)
	}

	body := w.Body.String()
	if body == "" {
		t.Fatal("body is empty")
	}

	for _, want := range []string{`"openapi": "3.1.0"`, `"title": "Test API"`, `"/items"`, `"ListItems"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\nbody: %s", want, body)
		}
	}
}

func TestOpenAPISpecHandler_304OnMatchingETag(t *testing.T) {
	handler := mustHandler(t, sampleSpec())

	// First request to learn the ETag.
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first response has no ETag")
	}

	// Second request with matching If-None-Match must return 304.
	second := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	req.Header.Set("If-None-Match", etag)
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", second.Code, http.StatusNotModified)
	}

	if second.Body.Len() != 0 {
		t.Errorf("304 body should be empty, got %d bytes", second.Body.Len())
	}
}

func TestOpenAPISpecHandler_200OnMismatchETag(t *testing.T) {
	handler := mustHandler(t, sampleSpec())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	req.Header.Set("If-None-Match", `"stale-and-wrong"`)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Body.Len() == 0 {
		t.Error("body should be non-empty on ETag mismatch")
	}
}

func TestOpenAPISpecHandler_StableETagAcrossRequests(t *testing.T) {
	handler := mustHandler(t, sampleSpec())

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

	if first.Header().Get("ETag") != second.Header().Get("ETag") {
		t.Errorf("ETag changed between requests: %q vs %q",
			first.Header().Get("ETag"), second.Header().Get("ETag"))
	}

	if first.Body.String() != second.Body.String() {
		t.Error("body changed between requests for the same spec")
	}
}

// TestOpenAPISpecHandler_ConcurrentRequestsAreSafe hammers the handler from
// many goroutines. Run under `go test -race` to confirm that the eagerly
// serialized, immutable handler has no data race.
func TestOpenAPISpecHandler_ConcurrentRequestsAreSafe(t *testing.T) {
	handler := mustHandler(t, sampleSpec())

	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}

			if w.Body.Len() == 0 {
				t.Error("concurrent request returned empty body")
			}
		}()
	}

	wg.Wait()
}
