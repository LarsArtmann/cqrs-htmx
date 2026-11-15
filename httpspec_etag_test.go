package cqrshtmx_test

import (
	"context"
	"net/http"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
	"github.com/larsartmann/cqrs-htmx/v4/openapi"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/httputil/httpspec"
)

// etagSpecCase mounts one of the library's five ETag-serving handlers at the
// index path of a fresh mux.
type etagSpecCase struct {
	name    string
	path    string
	handler func(t *testing.T) http.Handler
}

// etagSpecCases builds the five handler fixtures. Every one of them serves a
// content-derived ETag the test author cannot know ahead of time — exactly
// the posture httpspec.ExpectNotModifiedDynamic exists for: harvest the tag
// from the first response, re-request with If-None-Match, expect 304 with an
// empty body and the tag echoed.
func etagSpecCases() []etagSpecCase {
	return []etagSpecCase{
		{
			name: "event catalog",
			path: "/",
			handler: func(t *testing.T) http.Handler {
				t.Helper()

				catalog := eventcatalog.New()
				catalog.Register(eventcatalog.EventMetadata{
					Type:          "UserRegistered",
					Aggregate:     "User",
					SchemaVersion: 2,
					Description:   "A user registered with email",
					PayloadFields: []eventcatalog.PayloadField{
						{Name: "email", Type: "string", Required: true},
					},
				})

				h, err := eventcatalog.Handler(catalog)
				if err != nil {
					t.Fatalf("eventcatalog.Handler: %v", err)
				}

				return muxAt(h)
			},
		},
		{
			name: "openapi spec",
			path: "/",
			handler: func(t *testing.T) http.Handler {
				t.Helper()

				spec := openapi.New("Test API", "1.0.0").
					Path("/items",
						openapi.Get("ListItems").
							Summary("List all items").
							Response(http.StatusOK, "OK"),
					).
					Schema("Error", openapi.ErrorSchema())

				h, err := cqrshtmx.OpenAPISpecHandler(spec)
				if err != nil {
					t.Fatalf("OpenAPISpecHandler: %v", err)
				}

				return muxAt(h)
			},
		},
		{
			name: "projection status",
			path: "/",
			handler: func(t *testing.T) http.Handler {
				t.Helper()

				return muxAt(cqrshtmx.ProjectionStatusHandler(stubProjectionStatuses{}))
			},
		},
		{
			name: "sync pull",
			path: "/",
			handler: func(t *testing.T) http.Handler {
				t.Helper()

				store := memory.NewMemoryStore()

				return muxAt(cqrshtmx.SyncPullHandler(store))
			},
		},
		{
			name: "etag results render",
			path: "/",
			handler: func(t *testing.T) http.Handler {
				t.Helper()

				disp := query.NewDispatcher()
				_ = disp.Register("Ping", func(_ context.Context, _ query.Query) (any, error) {
					return map[string]string{"pong": "ok"}, nil
				})
				app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: disp})

				mux := http.NewServeMux()
				mux.Handle("GET /{$}", app.Query("Ping",
					cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) { return pingQuery{}, nil }),
					cqrshtmx.RenderJSON[map[string]string](),
					cqrshtmx.ETagResults(),
				))

				return recommendedStack(mux)
			},
		},
	}
}

// TestHTTPSpec_ETagHandlers_Dynamic304 runs the httpspec standard suite plus
// the dynamic conditional-request spec against every ETag-serving handler the
// library ships. This is the adoption proof for httputil v1.9.0's
// ExpectNotModifiedDynamic: revalidation (harvest → If-None-Match → 304 empty
// body → ETag echo) is asserted on the real handlers, not just assumed from
// their ETag headers.
func TestHTTPSpec_ETagHandlers_Dynamic304(t *testing.T) {
	t.Parallel()

	for _, tc := range etagSpecCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := tc.handler(t)
			dynamic304 := httpspec.Spec{
				Name:     "etag-dynamic-304",
				Category: httpspec.CategoryHeaders,
				Check:    httpspec.ExpectNotModifiedDynamic(http.MethodGet, tc.path),
			}
			httpspec.Run(t, handler,
				httpspec.WithIndexPath(tc.path),
				httpspec.WithExtraSpecs(dynamic304),
			)
		})
	}
}

// muxAt mounts handler at the index path so the httpspec standard suite's
// index assertions see the handler under test, wrapped in the security
// middleware stack the library recommends to consumers (the nosniff header
// spec is that stack's contract, not a single handler's).
func muxAt(h http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /{$}", h)

	return recommendedStack(mux)
}

// recommendedStack is the middleware chain from TestHTTPSpecCompliance: the
// documented defaults every consumer is advised to ship.
func recommendedStack(next http.Handler) http.Handler {
	return cqrshtmx.Chain(
		cqrshtmx.SecurityHeadersMiddleware,
		cqrshtmx.RecoveryMiddleware,
	)(next)
}

// stubProjectionStatuses is a minimal ProjectionStatusProvider for the spec.
type stubProjectionStatuses struct{}

func (stubProjectionStatuses) ProjectionStatuses() []cqrshtmx.ProjectionStatusEntry {
	return []cqrshtmx.ProjectionStatusEntry{
		{Name: "user-read-model", Status: "running", Processed: 7},
	}
}
