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

// The RFC 10008 fallback seam: MethodOverrideMiddleware rewrites a POST with
// an X-HTTP-Method-Override: QUERY header into a QUERY request BEFORE
// routing, so one "QUERY /path" pattern serves both transports.
// Plan: docs/planning/2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md §4.2.

// newOverrideMux is self-contained (no global-Expect helpers) so these plain
// tests run green under any -run filter, independent of the Ginkgo suite.
func newOverrideMux() *http.ServeMux {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
		return testQueryResult, nil
	})
	app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})

	mux := http.NewServeMux()
	mux.Handle(cqrshtmx.MethodQuery+" /contacts", app.Query("GetUser",
		decodeGetUserJSONQuery(), cqrshtmx.RenderJSON[string]()))

	return mux
}

func TestMethodOverride_POSTWithOverrideHeaderServesQueryRoute(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var seenMethod string
	capture := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod = r.Method
		w.WriteHeader(http.StatusOK)
	})
	handler := cqrshtmx.MethodOverrideMiddleware(capture)

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	req.Header.Set(cqrshtmx.MethodOverrideHeader, cqrshtmx.MethodQuery)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	g.Expect(w.Code).To(Equal(http.StatusOK))
	g.Expect(seenMethod).To(Equal(cqrshtmx.MethodQuery), "the rewrite must reach the inner handler as QUERY")
}

func TestMethodOverride_RewritesBeforeRouting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// ONE "QUERY /contacts" pattern; the fallback POST reaches it through
	// the middleware — no POST twin registration needed.
	handler := cqrshtmx.MethodOverrideMiddleware(newOverrideMux())

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	req.Header.Set(cqrshtmx.MethodOverrideHeader, cqrshtmx.MethodQuery)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	g.Expect(w.Code).To(Equal(http.StatusOK), "override POST must be served by the QUERY pattern")
	g.Expect(w.Body.String()).To(ContainSubstring(testQueryResult))
}

func TestMethodOverride_LegacyHeaderSpellingHonored(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	handler := cqrshtmx.MethodOverrideMiddleware(newOverrideMux())

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	req.Header.Set("X-HTTP-Method", cqrshtmx.MethodQuery)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	g.Expect(w.Code).To(Equal(http.StatusOK))
}

func TestMethodOverride_OriginalRequestNotMutated(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	capture := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := cqrshtmx.MethodOverrideMiddleware(capture)

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	req.Header.Set(cqrshtmx.MethodOverrideHeader, cqrshtmx.MethodQuery)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	g.Expect(req.Method).To(Equal(http.MethodPost), "the caller's request stays the wire POST")
	g.Expect(w.Code).To(Equal(http.StatusOK))
}

func TestMethodOverride_UnsafeOverrideTargetIgnored(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var seenMethod string
	capture := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod = r.Method
		w.WriteHeader(http.StatusOK)
	})
	handler := cqrshtmx.MethodOverrideMiddleware(capture)

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	req.Header.Set(cqrshtmx.MethodOverrideHeader, http.MethodDelete)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	g.Expect(seenMethod).To(Equal(http.MethodPost), "unsafe override targets must never be honored")
	g.Expect(w.Code).To(Equal(http.StatusOK))
}

func TestMethodOverride_PlainPostAndNativeQueryPassThrough(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mux := newOverrideMux()
	// A POST without the header is NOT rewritten: against a QUERY-only
	// pattern it 405s (routing stays honest).
	plainPOST := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	cqrshtmx.MethodOverrideMiddleware(mux).ServeHTTP(w, plainPOST)
	g.Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))

	native := httptest.NewRequest(cqrshtmx.MethodQuery, "/contacts", strings.NewReader(`{}`))
	native.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	cqrshtmx.MethodOverrideMiddleware(mux).ServeHTTP(w, native)
	g.Expect(w.Code).To(Equal(http.StatusOK))
}

func TestMethodOverride_WireMethodSeenByOuterLayers(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// The placement proof: mounted INSIDE the security layers, an outer
	// (CSRF-shaped) middleware still sees the wire method POST — the
	// fallback stays token-checked no matter how QUERY exemptions evolve.
	var outerSeen string
	outer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outerSeen = r.Method
		cqrshtmx.MethodOverrideMiddleware(newOverrideMux()).ServeHTTP(w, r)
	})

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{}`))
	req.Header.Set(cqrshtmx.MethodOverrideHeader, cqrshtmx.MethodQuery)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	outer.ServeHTTP(w, req)

	g.Expect(outerSeen).To(Equal(http.MethodPost), "outer security layers validate the WIRE method")
	g.Expect(w.Code).To(Equal(http.StatusOK))
}
