package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		"Cache-Control":          "public, max-age=86400",
		"X-Content-Type-Options": "nosniff",
		"ETag":                   assetETag,
	} {
		if got := rec.Header().Get(key); !strings.Contains(got, want) {
			t.Errorf("%s = %q, want contains %q", key, got, want)
		}
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

	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard-tw.css", nil)
	req.Header.Set("If-None-Match", assetETag)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304 on matching ETag", rec.Code)
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
