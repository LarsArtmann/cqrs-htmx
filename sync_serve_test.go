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

func TestSyncClientScriptTag(t *testing.T) {
	tag := cqrshtmx.SyncClientScriptTag("/sync-client.js")

	if tag != `<script src="/sync-client.js"></script>` {
		t.Errorf("SyncClientScriptTag: got %q", tag)
	}
}

func TestSyncClientScriptTag_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty path", "", `<script src=""></script>`},
		{"with query params", "/sync-client.js?v=2", `<script src="/sync-client.js?v=2"></script>`},
		{"with fragment", "/sync-client.js#section", `<script src="/sync-client.js#section"></script>`},
		{"relative path", "sync-client.js", `<script src="sync-client.js"></script>`},
		{
			"full URL",
			"https://cdn.example.com/sync-client.js",
			`<script src="https://cdn.example.com/sync-client.js"></script>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cqrshtmx.SyncClientScriptTag(tt.path)

			if got != tt.want {
				t.Errorf("SyncClientScriptTag(%q): got %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestSyncWorkerHandlerWith_ServesCustomJS(t *testing.T) {
	customJS := []byte("// custom worker")
	h := cqrshtmx.SyncWorkerHandlerWith(customJS, "2.0.0")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}

	if rec.Body.String() != "// custom worker" {
		t.Errorf("body: got %q, want // custom worker", rec.Body.String())
	}

	wantETag := `"cqrshtmx-sync-worker-2.0.0"`
	if etag := rec.Header().Get("ETag"); etag != wantETag {
		t.Errorf("ETag: got %q, want %q", etag, wantETag)
	}
}

func TestSyncClientHandlerWith_ServesCustomJS(t *testing.T) {
	customJS := []byte("// custom client")
	h := cqrshtmx.SyncClientHandlerWith(customJS, "3.0.0")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}

	if rec.Body.String() != "// custom client" {
		t.Errorf("body: got %q, want // custom client", rec.Body.String())
	}

	wantETag := `"cqrshtmx-sync-client-3.0.0"`
	if etag := rec.Header().Get("ETag"); etag != wantETag {
		t.Errorf("ETag: got %q, want %q", etag, wantETag)
	}
}

func TestSyncWorkerHandlerWith_304OnIfNoneMatch(t *testing.T) {
	h := cqrshtmx.SyncWorkerHandlerWith([]byte("// v2"), "2.0.0")
	etag := `"cqrshtmx-sync-worker-2.0.0"`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", etag)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("304: got %d, want 304", rec.Code)
	}
}
