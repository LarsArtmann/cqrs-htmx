package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// ETagResults: conditional query results with a strong ETag over the rendered
// bytes and 304 Not Modified on If-None-Match revalidation.
// Plan: docs/planning/2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md §W4.2.

func newETagApp() *cqrshtmx.App {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
		return "ok", nil
	})

	return cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})
}

func newMutableETagApp() *cqrshtmx.App {
	counter := &atomic.Int64{}
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
		return counter.Add(1), nil
	})

	return cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})
}

func etagQueryHandler(app *cqrshtmx.App) http.HandlerFunc {
	return app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RenderJSON[string](),
		cqrshtmx.ETagResults(),
	)
}

func mutableETagQueryHandler(app *cqrshtmx.App) http.HandlerFunc {
	return app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RenderJSON[int64](),
		cqrshtmx.ETagResults(),
	)
}

func TestETagResults_304OnRevalidation(t *testing.T) {
	t.Parallel()

	app := newETagApp()
	handler := etagQueryHandler(app)

	first := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, first)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: status = %d, want 200", w1.Code)
	}

	tag := w1.Header().Get("ETag")
	if tag == "" {
		t.Fatalf("first request carries no ETag header")
	}

	second := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	second.Header.Set("If-None-Match", tag)

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, second)

	if w2.Code != http.StatusNotModified {
		t.Fatalf("revalidation: status = %d, want 304 (tag %q)", w2.Code, tag)
	}

	if w2.Body.Len() != 0 {
		t.Fatalf("304 must carry no body, got %q", w2.Body.String())
	}
}

func TestETagResults_ChangedResultChangesTag(t *testing.T) {
	t.Parallel()

	app := newMutableETagApp()
	handler := mutableETagQueryHandler(app)

	tags := make([]string, 2)

	for i := range tags {
		req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		tags[i] = w.Header().Get("ETag")
	}

	if tags[0] == "" || tags[0] == tags[1] {
		t.Fatalf("a changed result set must change the ETag, got %q then %q", tags[0], tags[1])
	}
}

func TestETagResults_WildcardMatches(t *testing.T) {
	t.Parallel()

	app := newETagApp()
	handler := etagQueryHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	req.Header.Set("If-None-Match", "*")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match: * must revalidate to 304, got %d", w.Code)
	}
}

func TestETagResults_StaleTagReplaysBody(t *testing.T) {
	t.Parallel()

	app := newETagApp()
	handler := etagQueryHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	req.Header.Set("If-None-Match", `"stale-tag"`)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("a stale tag must get a full 200 response, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "ok") {
		t.Fatalf("200 response must replay the body, got %q", w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "json") {
		t.Fatalf("buffered Content-Type must be replayed, got %q", ct)
	}
}

func TestETagResults_BeforeRendererIsNoOp(t *testing.T) {
	t.Parallel()

	app := newETagApp()
	handler := app.Query("GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.ETagResults(),
		cqrshtmx.RenderJSON[string](),
	)

	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (the renderer still runs unwrapped)", w.Code)
	}

	if w.Header().Get("ETag") != "" {
		t.Fatalf("ETagResults applied before a renderer must be a no-op, but an ETag appeared")
	}
}
