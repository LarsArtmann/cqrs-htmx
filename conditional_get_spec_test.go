package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// conditionalGetSpecCase describes one RFC 7232 §2.3.2 If-None-Match scenario.
// headerTemplate uses the literal token CURRENT as a placeholder for the
// representation's ETag observed on a baseline request.
type conditionalGetSpecCase struct {
	name           string
	headerTemplate string
	wantStatus     int
	wantBodyEmpty  bool
}

func conditionalGetSpecCases() []conditionalGetSpecCase {
	return []conditionalGetSpecCase{
		{
			name:           "wildcard matches any representation",
			headerTemplate: "*",
			wantStatus:     http.StatusNotModified,
			wantBodyEmpty:  true,
		},
		{
			name:           "validator list containing the current ETag",
			headerTemplate: `"stale-from-another-representation", CURRENT`,
			wantStatus:     http.StatusNotModified,
			wantBodyEmpty:  true,
		},
		{
			name:           "weak form of the current ETag",
			headerTemplate: `W/CURRENT`,
			wantStatus:     http.StatusNotModified,
			wantBodyEmpty:  true,
		},
		{
			name:           "mismatched validator serves the full body",
			headerTemplate: `"definitely-not-the-current-etag"`,
			wantStatus:     http.StatusOK,
		},
		{
			name:           "malformed-only validators are skipped leniently",
			headerTemplate: `"unterminated`,
			wantStatus:     http.StatusOK,
		},
	}
}

// runConditionalGetSpec pins RFC 7232 conditional-GET behavior for a handler
// that advertises an ETag: first a baseline GET captures the ETag, then every
// spec case replays a GET with a templated If-None-Match header.
func runConditionalGetSpec(t *testing.T, handler http.Handler, path string) {
	t.Helper()

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, path, nil))

	if first.Code != http.StatusOK {
		t.Fatalf("baseline GET status = %d, want %d", first.Code, http.StatusOK)
	}

	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("baseline GET response has no ETag")
	}

	for _, tc := range conditionalGetSpecCases() {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("If-None-Match", strings.ReplaceAll(tc.headerTemplate, "CURRENT", etag))
			handler.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("If-None-Match %q: status = %d, want %d", tc.headerTemplate, w.Code, tc.wantStatus)
			}

			if tc.wantBodyEmpty && w.Body.Len() != 0 {
				t.Errorf("If-None-Match %q: 304 body should be empty, got %d bytes", tc.headerTemplate, w.Body.Len())
			}

			if !tc.wantBodyEmpty && w.Body.Len() == 0 {
				t.Errorf("If-None-Match %q: %d body should be non-empty", tc.headerTemplate, tc.wantStatus)
			}
		})
	}
}
