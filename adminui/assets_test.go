package adminui

import (
	"net/http/httptest"
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
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/-/admin-tw.css", nil))
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if got := rec.Header().Get("ETag"); got != assetETag {
		t.Errorf("unexpected ETag header: %q", got)
	}
}
