package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func TestSyncWorkerHandler_ServesJS(t *testing.T) {
	h := cqrshtmx.SyncWorkerHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want text/javascript; charset=utf-8", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("body is empty")
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Error("missing ETag header")
	}
}

func TestSyncClientHandler_ServesJS(t *testing.T) {
	h := cqrshtmx.SyncClientHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want text/javascript; charset=utf-8", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("body is empty")
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Error("missing ETag header")
	}
}

func TestSyncWorkerHandler_304OnIfNoneMatch(t *testing.T) {
	h := cqrshtmx.SyncWorkerHandler()

	// First request to get the ETag
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/", nil))
	etag := rec1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first request: missing ETag")
	}

	// Second request with If-None-Match should return 304
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("If-None-Match", etag)
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotModified {
		t.Errorf("304: got %d, want 304", rec2.Code)
	}
	if rec2.Body.Len() != 0 {
		t.Errorf("304 body: got %d bytes, want 0", rec2.Body.Len())
	}
}

func TestSyncClientHandler_304OnIfNoneMatch(t *testing.T) {
	h := cqrshtmx.SyncClientHandler()

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/", nil))
	etag := rec1.Header().Get("ETag")

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("If-None-Match", etag)
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotModified {
		t.Errorf("304: got %d, want 304", rec2.Code)
	}
}

func TestSyncWorkerHandler_RejectsPost(t *testing.T) {
	h := cqrshtmx.SyncWorkerHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: got %d, want 405", rec.Code)
	}
}

func TestSyncClientHandler_RejectsPost(t *testing.T) {
	h := cqrshtmx.SyncClientHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: got %d, want 405", rec.Code)
	}
}

func TestSyncVersion(t *testing.T) {
	v := cqrshtmx.SyncVersion()
	if v == "" {
		t.Error("SyncVersion returned empty string")
	}
}

func TestSyncScriptTags(t *testing.T) {
	if tag := cqrshtmx.SyncClientScriptTag("/sync.js"); tag != `<script src="/sync.js"></script>` {
		t.Errorf("SyncClientScriptTag: got %q", tag)
	}
	if tag := cqrshtmx.SyncWorkerScriptTag("/worker.js"); tag != `<script src="/worker.js"></script>` {
		t.Errorf("SyncWorkerScriptTag: got %q", tag)
	}
}
