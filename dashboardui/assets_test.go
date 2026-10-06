package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestDashboard_TailwindCSSRoute locks the compiled-Tailwind asset route:
// adminui-style cache headers plus canary utilities proving the bundle was
// built from the templ-components scan (the false-green class: a
// utility-free stylesheet served with exit 0).
func TestDashboard_TailwindCSSRoute(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard-tw.css", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	for key, want := range map[string]string{
		"Content-Type":           "text/css",
		"Cache-Control":          "public, max-age=31536000, immutable",
		"X-Content-Type-Options": "nosniff",
	} {
		if got := rec.Header().Get(key); !strings.Contains(got, want) {
			t.Errorf("%s = %q, want contains %q", key, got, want)
		}
	}

	// The ETag is content-derived (FNV-1a over the served bytes), not a
	// hand-bumped version constant.
	if want := contentETag("dashboard-tw.css", rec.Body.Bytes()); rec.Header().Get("ETag") != want {
		t.Errorf("ETag = %q, want content-derived %q", rec.Header().Get("ETag"), want)
	}

	body := rec.Body.String()
	// display.StatusBadge emits these only from library .templ files; their
	// presence proves the @source scan fed Tailwind the library source. The
	// amber family comes from errorpage.ErrorPage adoption — amber was the
	// missed utility family of the M6 false-green (bundle predating the
	// component swap), so it is pinned here permanently.
	for _, want := range []string{
		"bg-green-100",
		"bg-blue-100",
		`dark\:bg-green-900`,
		":root",
		"bg-amber-50",
		"bg-amber-100",
		"border-amber-200",
		`dark\:bg-amber-900`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard-tw.css missing canary %q", want)
		}
	}
}

func TestDashboard_304OnETag(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	// Round-trip: take the ETag the server actually served, not a constant.
	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard-tw.css", nil))

	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first response carried no ETag")
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard-tw.css", nil)
	req.Header.Set("If-None-Match", etag)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304 on matching ETag", rec.Code)
	}

	// A stale ETag must re-serve the full body, not 304.
	stale := httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard-tw.css", nil)
	stale.Header.Set("If-None-Match", `"dashboardui-dashboard-tw.css-deadbeefdeadbeef"`)
	recStale := httptest.NewRecorder()
	mux.ServeHTTP(recStale, stale)

	if recStale.Code != http.StatusOK {
		t.Fatalf("stale ETag: status = %d, want 200", recStale.Code)
	}

	if recStale.Body.Len() != first.Body.Len() {
		t.Errorf("stale ETag re-served %d bytes, want full %d", recStale.Body.Len(), first.Body.Len())
	}
}

// TestDashboard_JSAssetImmutableAnd304 pins the dashboard.js route to the
// same content-hash ETag + immutable caching rule as the CSS assets (M16:
// the JS route previously had no ETag at all).
func TestDashboard_JSAssetImmutableAnd304(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard.js", nil))

	if first.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", first.Code)
	}

	if got := first.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable long-lived caching", got)
	}

	if want := contentETag("dashboard.js", first.Body.Bytes()); first.Header().Get("ETag") != want {
		t.Errorf("ETag = %q, want content-derived %q", first.Header().Get("ETag"), want)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard.js", nil)
	req.Header.Set("If-None-Match", first.Header().Get("ETag"))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304 on matching ETag", rec.Code)
	}
}

// TestContentETagChangesWithContent pins the hash property the caching rule
// relies on: same bytes → same tag, different bytes → different tag.
func TestContentETagChangesWithContent(t *testing.T) {
	t.Parallel()

	a := contentETag("asset.css", []byte("body{}"))
	if a != contentETag("asset.css", []byte("body{}")) {
		t.Error("contentETag is not deterministic for identical input")
	}

	if a == contentETag("asset.css", []byte("body{color:red}")) {
		t.Error("contentETag did not change when the content changed")
	}

	if a == contentETag("other.css", []byte("body{}")) {
		t.Error("contentETag did not change when the asset name changed")
	}
}

func TestDashboard_LayoutLinksTailwindCSS(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if body := rec.Body.String(); !strings.Contains(body, "/-/dashboard-tw.css") {
		t.Error("layout head missing dashboard-tw.css link")
	}
}

func TestNewAssetHandlerMissingAssetReturnsError(t *testing.T) {
	t.Parallel()

	if _, err := newAssetHandler(fstest.MapFS{}, "nope.css", "text/css"); err == nil {
		t.Fatal("expected an error for a missing embedded asset, got nil")
	}
}
