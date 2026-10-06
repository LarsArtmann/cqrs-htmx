package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNewAssetHandlerMissingAssetReturnsError(t *testing.T) {
	t.Parallel()

	if _, err := newAssetHandler(fstest.MapFS{}, "nope.js", "text/javascript"); err == nil {
		t.Fatal("expected an error for a missing embedded asset, got nil")
	}
}

func TestNewAssetHandlerServesEmbeddedAsset(t *testing.T) {
	t.Parallel()

	handler, err := newAssetHandler(assetsFS, "admin-tw.css", "text/css; charset=utf-8")
	if err != nil {
		t.Fatalf("expected the real embedded asset to resolve, got: %v", err)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/-/admin-tw.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	wantTag := contentETag("admin-tw.css", rec.Body.Bytes())
	if got := rec.Header().Get("ETag"); got != wantTag {
		t.Errorf("ETag = %q, want content-derived %q", got, wantTag)
	}

	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable long-lived caching", got)
	}

	// The same ETag must answer If-None-Match with a 304 (ServeContent path).
	rec304 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/-/admin-tw.css", nil)
	req.Header.Set("If-None-Match", wantTag)
	handler.ServeHTTP(rec304, req)
	if rec304.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match with the served ETag: got %d, want 304", rec304.Code)
	}
}
