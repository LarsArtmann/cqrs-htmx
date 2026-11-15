package adminui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// seedTwoUsers registers alpha@example.com and beta@example.com so a search
// for "alpha" has a match and a non-match.
func seedTwoUsers(t *testing.T, svc *usermgmt.Service) {
	t.Helper()
	for _, s := range []struct{ id, email string }{
		{"u-search-a", "alpha@example.com"},
		{"u-search-b", "beta@example.com"},
	} {
		if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
			ID: identitymodel.SyntheticUserID(s.id), Email: s.email,
		}); err != nil {
			t.Fatalf("seed %s: %v", s.email, err)
		}
	}
}

func doSearch(t *testing.T, h http.Handler, method, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/admin/users/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestUserSearch_LadderParity pins the fallback ladder's core contract: the
// native QUERY rung, the POST fallback twin, and the POST +
// X-HTTP-Method-Override rung (rewritten to QUERY by the panel's
// MethodOverrideMiddleware) answer ONE handler and therefore byte-identical
// fragments — docs/guides/http-query-method.md.
func TestUserSearch_LadderParity(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	seedTwoUsers(t, svc)

	rungs := map[string]*httptest.ResponseRecorder{
		"native QUERY": doSearch(t, h, cqrshtmx.MethodQuery, "q=alpha", nil),
		"POST twin":    doSearch(t, h, http.MethodPost, "q=alpha", nil),
		"override POST (rung 2)": doSearch(
			t,
			h,
			http.MethodPost,
			"q=alpha",
			map[string]string{"X-HTTP-Method-Override": "QUERY"},
		),
		"override legacy X-HTTP-Method": doSearch(
			t,
			h,
			http.MethodPost,
			"q=alpha",
			map[string]string{"X-HTTP-Method": "QUERY"},
		),
	}

	golden := rungs["native QUERY"]
	if golden.Code != http.StatusOK {
		t.Fatalf("native QUERY status = %d, want 200; body=%s", golden.Code, golden.Body.String())
	}
	if !strings.Contains(golden.Body.String(), "alpha@example.com") {
		t.Errorf("search for alpha must keep alpha@example.com; body=%s", golden.Body.String())
	}
	if strings.Contains(golden.Body.String(), "beta@example.com") {
		t.Error("search for alpha must exclude beta@example.com")
	}

	for name, rung := range rungs {
		if rung.Code != golden.Code {
			t.Errorf("%s: status = %d, want %d", name, rung.Code, golden.Code)
		}
		if !bytes.Equal(rung.Body.Bytes(), golden.Body.Bytes()) {
			t.Errorf("%s: body differs from the native QUERY response (ladder parity broken)", name)
		}
	}
}

// TestUserSearch_CriteriaRideTheBody pins the body-carrying contract: filter
// and page travel as body fields exactly like the GET route's URL params.
func TestUserSearch_CriteriaRideTheBody(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	seedTwoUsers(t, svc)

	rec := doSearch(t, h, cqrshtmx.MethodQuery, "filter=unverified", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("filter-only search status = %d; body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "alpha@example.com") ||
		!strings.Contains(body, "beta@example.com") {
		t.Error("unverified filter must keep both seeded users (no verification configured)")
	}

	rec2 := doSearch(t, h, cqrshtmx.MethodQuery, "q=alpha&filter=mfa", nil)
	if body := rec2.Body.String(); strings.Contains(body, "alpha@example.com") {
		t.Error("mfa filter must exclude users without TOTP even when the name matches")
	}

	// An unknown filter value must not silently mean "no filter" on the body
	// path either (parseUserFilter refuses it back to none, which here means
	// the search still returns both users) — pinned to the GET route's
	// behavior so the rungs agree on the refuse-to-none rule.
	rec3 := doSearch(t, h, cqrshtmx.MethodQuery, "filter=nonsense", nil)
	if !strings.Contains(rec3.Body.String(), "alpha@example.com") {
		t.Error("unknown filter value must degrade to no filter, not to an empty result")
	}
}

// TestUserSearch_RefusesGarbageBodies pins the 400 paths: an oversized body
// and a non-integer page are client rejections, never silent zero-criteria
// searches (the gotcha-58 class: a mismatched body must not decode to a
// zero struct and answer as if the admin searched for everything).
func TestUserSearch_RefusesGarbageBodies(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	oversized := strings.Repeat("q=", 40_000) + "x"
	if rec := doSearch(t, h, cqrshtmx.MethodQuery, oversized, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("oversized body status = %d, want 400", rec.Code)
	}

	if rec := doSearch(t, h, cqrshtmx.MethodQuery, "page=notanumber", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("non-integer page status = %d, want 400", rec.Code)
	}
}

// TestUserSearch_UnauthenticatedGets401 pins the gate: the body-carrying
// rungs are guarded exactly like every other row — an anonymous QUERY is a
// 401, not a data leak through a new method token.
func TestUserSearch_UnauthenticatedGets401(t *testing.T) {
	t.Parallel()
	h, svc := newTestPanel(t, nil)
	seedTwoUsers(t, svc)

	for _, method := range []string{cqrshtmx.MethodQuery, http.MethodPost} {
		rec := doSearch(t, h, method, "q=alpha", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without a session: status = %d, want 401; body=%s", method, rec.Code, rec.Body.String())
		}
	}
}

// TestUserSearch_ExtensionScriptServed pins the asset posture of the panel's
// cqrs-query extension script (public row, ETag-cached, 405 for POST).
func TestUserSearch_ExtensionScriptServed(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/ext/cqrs-query.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("ext script status = %d, want 200", rec.Code)
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Error("ext script must carry an ETag (the immutable-asset posture)")
	}
	if !strings.Contains(rec.Body.String(), "cqrs-query") {
		t.Error("ext script body must be the cqrs-query extension")
	}

	recPost := httptest.NewRecorder()
	h.ServeHTTP(recPost, httptest.NewRequest(http.MethodPost, "/admin/-/ext/cqrs-query.js", nil))
	if recPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST to ext script status = %d, want 405", recPost.Code)
	}
}

// TestUserSearch_SearchInputRidesTheLadder pins the rendered wiring: the
// search form carries the cqrs-query ladder attributes (native upgrade on
// Enter, POST twin on typing) and both wire ends self-pin their target —
// the boost-wrapper trap makes an unpinned ajax element blank the page.
func TestUserSearch_SearchInputRidesTheLadder(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("users page status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`hx-ext="cqrs-query"`,
		`hx-post="/admin/users/search"`,
		`hx-query=""`,
		`hx-target="#users-table"`,
		`hx-indicator="#users-search-spinner"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("search form missing %q", want)
		}
	}
}
