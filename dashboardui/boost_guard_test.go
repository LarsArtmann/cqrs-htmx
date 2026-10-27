package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// ajaxTagRE matches a rendered element that issues an htmx ajax request. The
// word boundary also catches the data-hx-* spelling.
var ajaxTagRE = regexp.MustCompile(`(?s)<[a-zA-Z][^>]*?\bhx-(?:get|post|put|delete)=[^>]*>`)

// TestRenderedPages_AjaxElementsPinTargetAndSelect guards the 2026-10-18
// boost-inheritance class: the layout wrapper carries
// data-hx-target="#main-content" + data-hx-select="#main-content" for boosted
// navigation, and htmx resolves BOTH attributes through the ancestor chain
// for every ajax element. An unpinned element inside the wrapper selects
// #main-content out of a partial response that has none and swaps an EMPTY
// fragment into #main-content — the page blanks within one poll interval
// (the overview-blanking bug). Every ajax element must therefore pin its own
// hx-target and hx-select ("unset" is htmx's documented inheritance
// neutralizer). The guard walks every GET route so any NEW page with an
// unpinned ajax element fails here instead of in production.
func TestRenderedPages_AjaxElementsPinTargetAndSelect(t *testing.T) {
	t.Parallel()

	d := mustTestDashboardWithConfig(t, Config{})

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	seen := map[string]bool{}

	for _, route := range d.Routes() {
		if route.Method != http.MethodGet {
			continue
		}

		// Wildcards get placeholder values: invalid IDs render error pages,
		// which carry no ajax elements and pass trivially.
		path := "/dashboard" + strings.NewReplacer("{id}", "x", "{type}", "x").Replace(route.Pattern)
		if seen[path] {
			continue
		}

		seen[path] = true

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		for _, tag := range ajaxTagRE.FindAllString(rec.Body.String(), -1) {
			if !strings.Contains(tag, "hx-target=") || !strings.Contains(tag, "hx-select=") {
				t.Errorf("%s: ajax element lacks its own hx-target/hx-select pins "+
					"(boost inheritance would blank #main-content): %s", path, tag)
			}
		}
	}

	if len(seen) == 0 {
		t.Fatal("no GET routes scanned — the guard must cover every rendered page")
	}
}
