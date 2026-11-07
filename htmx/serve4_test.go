package htmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// The embedded htmx 4 surface (QUERY plan W2.3): ScriptHandler4 serves the
// 4.0.1 bundle with the same caching contract as the 2.0.10 default, and the
// bundle genuinely carries native QUERY support (the reason it exists).
func TestScriptHandler4_ServesBundleWithQuerySupport(t *testing.T) {
	t.Parallel()

	handler := htmx.ScriptHandler4()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/htmx4.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Fatalf("content-type = %q", ct)
	}

	if etag := rec.Header().Get("ETag"); !strings.Contains(etag, htmx.HTMX4Version) {
		t.Fatalf("ETag = %q, want it to name %s", etag, htmx.HTMX4Version)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "hx-query") || !strings.Contains(body, `"query"`) {
		t.Fatal("embedded htmx 4 bundle lacks native QUERY support — the pin drifted; " +
			"re-verify the bundle source before shipping")
	}

	if rec.Body.Len() <= 1024 {
		t.Fatalf("bundle suspiciously small: %d bytes", rec.Body.Len())
	}
}

func TestScriptHandler4_RevalidatesAndGuardsMethods(t *testing.T) {
	t.Parallel()

	handler := htmx.ScriptHandler4()

	get := httptest.NewRequest(http.MethodGet, "/htmx4.js", nil)
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, get)

	tag := first.Header().Get("ETag")

	reval := httptest.NewRequest(http.MethodGet, "/htmx4.js", nil)
	reval.Header.Set("If-None-Match", tag)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, reval)

	if second.Code != http.StatusNotModified {
		t.Fatalf("revalidation status = %d, want 304", second.Code)
	}

	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/htmx4.js", nil))

	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", post.Code)
	}
}

func TestVersion4_Exposed(t *testing.T) {
	t.Parallel()

	if htmx.HTMX4Version != "4.0.1" {
		t.Fatalf("HTMX4Version = %q", htmx.HTMX4Version)
	}
}
