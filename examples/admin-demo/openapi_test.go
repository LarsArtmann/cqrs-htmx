package main

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"encoding/json/jsontext"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// newDirectoryTestServer boots the demo's API surface against a seeded
// service: alpha/beta users registered, one session token minted for alpha.
func newDirectoryTestServer(t *testing.T) (http.Handler, string) {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	var token string
	for _, s := range []struct{ email string }{{"alpha@acme.dev"}, {"beta@acme.dev"}} {
		resp, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
			Email: s.email, DisplayName: strings.ToUpper(s.email[:1]) + s.email[1:5],
		})
		if err != nil {
			t.Fatalf("register %s: %v", s.email, err)
		}
		if s.email == "alpha@acme.dev" && resp.Session != nil {
			token = resp.Session.Token
		}
	}

	app, spec := newDirectoryApp(svc)
	specHandler, err := cqrshtmx.OpenAPISpecHandler(spec)
	if err != nil {
		t.Fatalf("OpenAPISpecHandler: %v", err)
	}

	sessionMW := usermgmt.NewSessionMiddleware(svc, cookieName)
	mux := http.NewServeMux()
	mux.Handle(cqrshtmx.MethodQuery+" /api/users/search", sessionMW(requireSession(directorySearchHandler(app, false))))
	mux.Handle(http.MethodPost+" /api/users/search", sessionMW(requireSession(directorySearchHandler(app, false))))
	mux.Handle("GET /openapi.json", specHandler)

	return mux, token
}

func directoryRequest(t *testing.T, h http.Handler, method, body, token string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/users/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

// TestDirectorySearch_LadderParityAndFilter pins the machine story: the
// native QUERY rung and the POST twin answer byte-identically, the filter
// narrows the directory, and the projection is bounded and machine-shaped.
func TestDirectorySearch_LadderParityAndFilter(t *testing.T) {
	t.Parallel()
	h, token := newDirectoryTestServer(t)

	queryRec := directoryRequest(t, h, cqrshtmx.MethodQuery, `{"q":"alpha"}`, token, nil)
	if queryRec.Code != http.StatusOK {
		t.Fatalf("native QUERY: status = %d, want 200; body=%s", queryRec.Code, queryRec.Body.String())
	}
	if ct := queryRec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}

	var result directoryResult
	if err := json.Unmarshal(queryRec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Total != 1 || len(result.Users) != 1 || result.Users[0].Email != "alpha@acme.dev" {
		t.Errorf("q=alpha must match exactly alpha@acme.dev, got total=%d users=%v", result.Total, result.Users)
	}

	postRec := directoryRequest(t, h, http.MethodPost, `{"q":"alpha"}`, token, nil)
	if postRec.Code != http.StatusOK || string(postRec.Body.String()) != string(queryRec.Body.String()) {
		t.Errorf("POST twin must answer byte-identically (got %d/%q, want %d/%q)",
			postRec.Code, postRec.Body.String(), queryRec.Code, queryRec.Body.String())
	}

	allRec := directoryRequest(t, h, cqrshtmx.MethodQuery, `{"filter":"unverified"}`, token, nil)
	var all directoryResult
	if err := json.Unmarshal(allRec.Body.Bytes(), &all); err != nil {
		t.Fatalf("decode unfiltered: %v", err)
	}
	if all.Total != 2 {
		t.Errorf("unverified filter must keep both seeded users (none verified), got total=%d", all.Total)
	}
}

// TestDirectorySearch_RequiresSession pins the gate: the API surface answers
// 401 with a problem document for anonymous callers (the session middleware
// only enriches; the route gates itself).
func TestDirectorySearch_RequiresSession(t *testing.T) {
	t.Parallel()
	h, _ := newDirectoryTestServer(t)

	rec := directoryRequest(t, h, cqrshtmx.MethodQuery, `{"q":"alpha"}`, "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous QUERY: status = %d, want 401", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
		t.Errorf("401 Content-Type = %q, want problem+json", ct)
	}
}

// TestOpenAPISpec_ServesAnd304 pins the spec publication: the served document
// is the WithOpenAPI metadata merged at the demo path (OpenAPI 3.2 — the
// query operation auto-bumps the version), ETag-cached with a real 304 path.
func TestOpenAPISpec_ServesAnd304(t *testing.T) {
	t.Parallel()
	h, _ := newDirectoryTestServer(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("spec GET: status = %d", rec.Code)
	}

	var doc struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Title string `json:"title"`
		} `json:"info"`
		Paths map[string]map[string]jsontext.Value `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("spec is not JSON: %v", err)
	}
	if doc.OpenAPI != "3.2.0" {
		t.Errorf("spec version = %q, want 3.2.0 (the query operation auto-bumps)", doc.OpenAPI)
	}
	if doc.Info.Title != "admin-demo API" {
		t.Errorf("spec title = %q", doc.Info.Title)
	}
	pathItem, ok := doc.Paths["/api/users/search"]
	if !ok {
		t.Fatalf("spec missing /api/users/search: paths=%v", doc.Paths)
	}
	if _, hasQuery := pathItem["query"]; !hasQuery {
		t.Error("path item missing the query operation (the WithOpenAPI-collected op)")
	}
	if !strings.Contains(rec.Body.String(), "SearchDirectory") {
		t.Error("spec must carry the SearchDirectory operationId from the per-route metadata")
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("spec must carry an ETag")
	}
	rec304 := httptest.NewRecorder()
	req304 := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	req304.Header.Set("If-None-Match", etag)
	h.ServeHTTP(rec304, req304)
	if rec304.Code != http.StatusNotModified {
		t.Errorf("conditional spec GET: status = %d, want 304", rec304.Code)
	}
}

// TestDirectorySearch_NegativePageRefused pins the refuse-don't-clamp rule:
// a non-positive page is a Rejection (400 problem document), never a silent
// clamp to page 1 — consistent with the panel's refuse-to-none filter rule.
func TestDirectorySearch_NegativePageRefused(t *testing.T) {
	t.Parallel()
	h, token := newDirectoryTestServer(t)

	rec := directoryRequest(t, h, cqrshtmx.MethodQuery, `{"q":"","page":-1}`, token, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("page=-1 QUERY: status = %d, want 400", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "page_invalid") {
		t.Errorf("400 body = %q, want the page_invalid code", body)
	}
}

// TestDirectorySearch_StrictContentTypeLadder exercises the root module's
// opt-in RFC 10008 strict mode end-to-end: the same demo handler rebuilt with
// RequireContentType answers 415 for a non-JSON body and passes JSON through.
// The shipped demo routes stay Content-Type-lenient; this pins the strict
// ladder a consumer opts into.
func TestDirectorySearch_StrictContentTypeLadder(t *testing.T) {
	t.Parallel()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	app, _ := newDirectoryApp(svc)

	strict := app.Query("SearchDirectory",
		cqrshtmx.DecodeJSONQuery(func(body *directorySearchQuery) (query.Query, error) {
			if body.Page == 0 {
				body.Page = 1
			}
			return body, nil
		}),
		cqrshtmx.RenderJSON[directoryResult](),
		cqrshtmx.RequireContentType("application/json"),
	)

	rec := directoryRequest(t, strict, cqrshtmx.MethodQuery, `{"q":""}`, "", map[string]string{"Content-Type": "text/plain"})
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("text/plain QUERY under strict mode: status = %d, want 415", rec.Code)
	}

	rec = directoryRequest(t, strict, cqrshtmx.MethodQuery, `{"q":""}`, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("json QUERY under strict mode: status = %d, want 200", rec.Code)
	}
}

// TestDirectorySearch_TokenLessQueryPasses pins the demo's CSRF posture: the
// API surface is session-gated, NOT CSRF-gated (the panel mux carries the
// CSRF layer; the JSON rungs sit outside it), so a QUERY with a valid session
// cookie and no CSRF token answers 200. Consumers that mount CSRF must
// safe-list QUERY per RFC 10008 — the guide's posture note covers that door.
func TestDirectorySearch_TokenLessQueryPasses(t *testing.T) {
	t.Parallel()
	h, token := newDirectoryTestServer(t)

	rec := directoryRequest(t, h, cqrshtmx.MethodQuery, `{"q":""}`, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("token-less QUERY: status = %d, want 200", rec.Code)
	}
}
