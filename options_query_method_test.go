package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	. "github.com/onsi/gomega"
)

// The RFC 10008 query-method option surface: RequireAnyMethod (the fallback
// ladder's multi-method guard), RequireContentType (the media-type ladder,
// 400/415), and the decoder-driven Accept-Query advertisement.
// Plan: docs/planning/2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md §4.2.

func newMethodOptionApp() *cqrshtmx.App {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
		return testQueryResult, nil
	})

	return cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})
}

func TestRequireAnyMethod_AllowsConfiguredMethods(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	app := newMethodOptionApp()
	handler := app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RenderJSON[string](),
		cqrshtmx.RequireAnyMethod(cqrshtmx.MethodQuery, http.MethodPost),
	)

	for _, method := range []string{cqrshtmx.MethodQuery, http.MethodPost} {
		req := httptest.NewRequest(method, "/contacts", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		g.Expect(w.Code).To(Equal(http.StatusOK), "method %s should be allowed", method)
	}
}

func TestRequireAnyMethod_RejectsOthersWithAllowHeader(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	app := newMethodOptionApp()
	handler := app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RequireAnyMethod(cqrshtmx.MethodQuery, http.MethodPost),
	)

	req := httptest.NewRequest(http.MethodPut, "/contacts", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	g.Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
	g.Expect(w.Header().Get("Allow")).To(Equal("QUERY, POST"),
		"405 must advertise the configured methods (RFC 9110/10008)")
	g.Expect(w.Body.String()).To(ContainSubstring("want QUERY, POST"))
}

func TestRequireMethod_StillEnforcesSingleMethod(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	app := newMethodOptionApp()
	handler := app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RenderJSON[string](),
		cqrshtmx.RequireMethod(cqrshtmx.MethodQuery),
	)

	ok := httptest.NewRequest(cqrshtmx.MethodQuery, "/contacts", strings.NewReader(`{}`))
	ok.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ok)
	g.Expect(w.Code).To(Equal(http.StatusOK))

	bad := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, bad)
	g.Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
	g.Expect(w.Header().Get("Allow")).To(Equal(cqrshtmx.MethodQuery))
}

func TestRequireContentType_Ladder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Wrong declared type → 415 (RFC 10008: known type, not accepted here).
	w := doQueryWithContentType(t, "text/plain")
	g.Expect(w.Code).To(Equal(http.StatusUnsupportedMediaType))
	g.Expect(w.Body.String()).To(ContainSubstring("unsupported_media_type"))

	// Missing type on a body-carrying request → 400 (server must not guess).
	w = doQueryWithContentType(t, "")
	g.Expect(w.Code).To(Equal(http.StatusBadRequest))
	g.Expect(w.Body.String()).To(ContainSubstring("content_type_missing"))

	// Parameters are ignored: "; charset=utf-8" still matches.
	w = doQueryWithContentType(t, "application/json; charset=utf-8")
	g.Expect(w.Code).To(Equal(http.StatusOK))
}

func TestRequireContentType_NoBodyAlwaysAllowed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	app := newMethodOptionApp()
	handler := app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RenderJSON[string](),
		cqrshtmx.RequireContentType("application/json"),
	)

	// Empty body decodes to the zero-value query: nothing to mis-declare.
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	g.Expect(w.Code).To(Equal(http.StatusOK))
}

// doQueryWithContentType mirrors doQuery with a controllable Content-Type.
func doQueryWithContentType(t *testing.T, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	app := newMethodOptionApp()
	handler := app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RenderJSON[string](),
		cqrshtmx.RequireContentType("application/json"),
	)

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	return w
}

func TestAcceptQuery_DecoderDrivenAdvertisement(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	app := newMethodOptionApp()

	// The JSON body-query decoder advertises what it decodes — on success...
	handler := app.Query("GetUser", decodeGetUserJSONQuery(), cqrshtmx.RenderJSON[string]())
	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	g.Expect(w.Code).To(Equal(http.StatusOK))
	g.Expect(w.Header().Get("Accept-Query")).To(Equal("application/json"))

	// ...and on the 405 too (discovery must work before the first success).
	rejected := app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RequireAnyMethod(cqrshtmx.MethodQuery),
	)
	bad := httptest.NewRequest(http.MethodPut, "/contacts", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	rejected.ServeHTTP(w, bad)
	g.Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
	g.Expect(w.Header().Get("Accept-Query")).To(Equal("application/json"))
}

func TestAcceptQuery_NoBodyDecoderNoAdvertisement(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	app := newMethodOptionApp()

	// A URL-form decoder reads no request body, so it must not advertise a
	// body media type — no aspirational Accept-Query values.
	handler := app.Query("GetUser",
		cqrshtmx.DecodeFormQuery(func(_ struct{}) (query.Query, error) {
			return &testGetUserQuery{}, nil
		}),
		cqrshtmx.RenderJSON[string](),
	)

	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	g.Expect(w.Code).To(Equal(http.StatusOK))
	g.Expect(w.Header().Get("Accept-Query")).To(BeEmpty())
}

func TestAcceptQuery_OptionAppendsWithoutDuplicating(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	app := newMethodOptionApp()

	handler := app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RenderJSON[string](),
		cqrshtmx.AcceptQuery("application/sql", "application/json"),
	)

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	g.Expect(w.Code).To(Equal(http.StatusOK))
	g.Expect(w.Header().Get("Accept-Query")).To(Equal("application/json, application/sql"),
		"the decoder's type first, the consumer's addition once")
}
