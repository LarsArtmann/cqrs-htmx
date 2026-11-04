package eventcatalog_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
)

func sampleCatalog() *eventcatalog.Catalog {
	catalog := eventcatalog.New()
	catalog.Register(eventcatalog.EventMetadata{
		Type:          "UserRegistered",
		Aggregate:     "User",
		SchemaVersion: 2,
		Description:   "A user registered with email",
		PayloadFields: []eventcatalog.PayloadField{
			{Name: "email", Type: "string", Required: true},
			{Name: "display_name", Type: "string"},
		},
	})
	catalog.Register(eventcatalog.EventMetadata{
		Type:          "TenantCreated",
		Aggregate:     "Tenant",
		SchemaVersion: 2,
	})

	return catalog
}

func mustCatalogHandler(t *testing.T, catalog *eventcatalog.Catalog) http.HandlerFunc {
	t.Helper()

	handler, err := eventcatalog.Handler(catalog)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	return handler
}

func TestEventCatalogHandler_ServesJSON(t *testing.T) {
	handler := mustCatalogHandler(t, sampleCatalog())

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/events/catalog", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache (ETag-driven revalidation; the URL is not versioned)", cc)
	}

	etag := w.Header().Get("ETag")
	if etag == "" || etag[0] != '"' {
		t.Errorf("ETag = %q, want a quoted non-empty value", etag)
	}

	body := w.Body.String()
	for _, want := range []string{`"UserRegistered"`, `"User"`, `"TenantCreated"`, `"email"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\nbody: %s", want, body)
		}
	}
}

func TestEventCatalogHandler_304OnMatchingETag(t *testing.T) {
	handler := mustCatalogHandler(t, sampleCatalog())

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/events/catalog", nil))

	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first response has no ETag")
	}

	second := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events/catalog", nil)
	req.Header.Set("If-None-Match", etag)
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", second.Code, http.StatusNotModified)
	}

	if second.Body.Len() != 0 {
		t.Errorf("304 body should be empty, got %d bytes", second.Body.Len())
	}
}

func TestEventCatalogHandler_200OnMismatchETag(t *testing.T) {
	handler := mustCatalogHandler(t, sampleCatalog())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events/catalog", nil)
	req.Header.Set("If-None-Match", `"stale-and-wrong"`)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Body.Len() == 0 {
		t.Error("body should be non-empty on ETag mismatch")
	}
}

func TestEventCatalogHandler_NilCatalogReturnsError(t *testing.T) {
	_, err := eventcatalog.Handler(nil)
	if err == nil {
		t.Fatal("Handler(nil) should return an error")
	}
}

func TestEventCatalogHandler_ConcurrentRequestsAreSafe(t *testing.T) {
	handler := mustCatalogHandler(t, sampleCatalog())

	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/events/catalog", nil))

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
