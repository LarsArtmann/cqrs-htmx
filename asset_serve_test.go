package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func TestServeAsset_HeadersAndConditionalGet(t *testing.T) {
	body := []byte("body { color: rebeccapurple }\n")
	handler := cqrshtmx.ServeAsset("app.css", "text/css; charset=utf-8", body)

	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/-/app.css", nil)
	handler.ServeHTTP(first, firstReq)

	if first.Code != http.StatusOK {
		t.Fatalf("baseline GET status = %d, want %d", first.Code, http.StatusOK)
	}

	if got, want := first.Header().Get("Content-Type"), "text/css; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	if got := first.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable long-lived caching", got)
	}

	if got := first.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}

	wantTag := cqrshtmx.AssetETag("app.css", body)
	if got := first.Header().Get("ETag"); got != wantTag {
		t.Errorf("ETag = %q, want content-derived %q", got, wantTag)
	}

	// The full If-None-Match contract (fresh tag 304s, stale re-serves, weak
	// and list forms) is pinned by the shared spec used by the other
	// immutable handlers.
	runConditionalGetSpec(t, handler, "/-/app.css")
}

func TestAssetETag_Derivation(t *testing.T) {
	css := []byte("body{}")

	tag := cqrshtmx.AssetETag("app.css", css)
	if !strings.HasPrefix(tag, `"`) || !strings.HasSuffix(tag, `"`) {
		t.Errorf("AssetETag = %q, want a quoted strong validator per RFC 7232", tag)
	}

	if tag != cqrshtmx.AssetETag("app.css", css) {
		t.Error("AssetETag is not deterministic for identical input")
	}

	if tag == cqrshtmx.AssetETag("app.css", []byte("body{color:red}")) {
		t.Error("AssetETag did not change when the content changed")
	}

	if tag == cqrshtmx.AssetETag("other.css", css) {
		t.Error("AssetETag did not change when the asset name changed")
	}
}
