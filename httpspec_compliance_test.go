package cqrshtmx_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/httputil/httpspec"
)

// TestHTTPSpecCompliance runs httputil's HTTP-spec compliance suite against the
// middleware stack cqrs-htmx recommends to consumers (security headers + panic
// recovery wrapping a standard mux with a real index route). It locks in that
// the documented defaults produce a spec-compliant server: unknown paths 404,
// HEAD/OPTIONS do not 5xx, TRACE/CONNECT are rejected, error responses carry a
// Content-Type, no version/internal leak, and X-Content-Type-Options is set.
//
// This is the primary place cqrs-htmx leverages httputil as a correctness
// partner rather than just a runtime dependency. See
// docs/research/2026-08-05_httputil-deep-dive.html.
func TestHTTPSpecCompliance(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	// NOTE: a specific path (not "GET /", which is a ServeMux subtree match that
	// would swallow unknown paths and return 200 instead of the expected 404).
	mux.HandleFunc("GET /index", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<!doctype html><html><body><h1>ok</h1></body></html>")
	})

	handler := cqrshtmx.Chain(
		cqrshtmx.SecurityHeadersMiddleware,
		cqrshtmx.RecoveryMiddleware,
	)(mux)

	httpspec.Run(t, handler, httpspec.WithIndexPath("/index"))
}

// TestHTTPSpecCompliance_RealApp runs the same suite against a stack that
// includes an actual cqrs-htmx App, proving the full dispatch pipeline (not
// just the outer middleware) stays HTTP-spec compliant.
func TestHTTPSpecCompliance_RealApp(t *testing.T) {
	t.Parallel()

	disp := query.NewDispatcher()
	_ = disp.Register("Ping", func(_ context.Context, _ query.Query) (any, error) {
		return "ok", nil
	})
	app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
	if err != nil {
		t.Fatalf("cqrshtmx.New: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /index", app.Query("Ping",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) { return pingQuery{}, nil }),
		cqrshtmx.RenderHTML("<!doctype html><html><body><h1>ok</h1></body></html>"),
	))

	handler := cqrshtmx.Chain(
		cqrshtmx.SecurityHeadersMiddleware,
		cqrshtmx.RecoveryMiddleware,
		cqrshtmx.HTMXMiddleware,
		app.Middleware(),
	)(mux)

	httpspec.Run(t, handler, httpspec.WithIndexPath("/index"))
}

// pingQuery is a minimal query.Query for the compliance test.
type pingQuery struct{}

func (pingQuery) Type() query.Type { return "Ping" }
