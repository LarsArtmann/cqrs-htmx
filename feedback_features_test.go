package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- Feedback-driven feature tests ---

var _ = Describe("Feedback-driven features", func() {
	Describe("DefaultRateLimiterConfig", func() {
		It("returns usable defaults that work as middleware", func() {
			cfg := cqrshtmx.DefaultRateLimiterConfig()
			Expect(cfg.Limit).To(BeNumerically(">", 0))
			mw := cqrshtmx.RateLimiterMiddleware(cfg)
			handler := mw(okHandler())
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("SSE event name constants", func() {
		It("provides standard event names", func() {
			Expect(cqrshtmx.SSEEventConnected).To(Equal("connected"))
			Expect(cqrshtmx.SSEEventHeartbeat).To(Equal("heartbeat"))
		})
	})

	Describe("SecurityHeaderSkip sentinel", func() {
		It("suppresses ContentTypeOptions when set to skip", func() {
			mw := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
				ContentTypeOptions: cqrshtmx.SecurityHeaderSkip,
			})
			w := httptest.NewRecorder()
			mw(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(w.Header().Get("X-Content-Type-Options")).To(BeEmpty())
		})

		It("still sets other headers when one is skipped", func() {
			mw := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
				ContentTypeOptions: cqrshtmx.SecurityHeaderSkip,
			})
			w := httptest.NewRecorder()
			mw(okHandler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(w.Header().Get("X-Frame-Options")).To(Equal("DENY"))
			Expect(w.Header().Get("Referrer-Policy")).To(Equal("strict-origin-when-cross-origin"))
		})
	})

	Describe("RenderHTML", func() {
		It("renders a static HTML string with correct content type", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetPage", func(_ context.Context, _ query.Query) (any, error) {
				return nil, nil
			})
			app := mustNewApp(cqrshtmx.Config{Queries: disp})

			handler := app.Query("GetPage",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &getPageQuery{}, nil
				}),
				cqrshtmx.RenderHTML("<div>Hello, HTMX!</div>"),
			)

			w := serve(handler, newPostJSONRequest(`{}`))
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(ContainSubstring("text/html"))
			Expect(w.Body.String()).To(Equal("<div>Hello, HTMX!</div>"))
		})
	})

	Describe("DecodeJSONWithRequest", func() {
		It("gives the mapper access to the *http.Request", func() {
			disp := command.NewDispatcher()
			var capturedHeader string
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return nil
			})
			app := mustNewApp(cqrshtmx.Config{Commands: disp})

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSONWithRequest(func(r *http.Request, _ testCreateUserRequest) (command.Command, error) {
					capturedHeader = r.Header.Get("X-Custom-Auth")
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
			)

			r := newPostJSONRequest(`{}`)
			r.Header.Set("X-Custom-Auth", "player-123")
			w := serve(handler, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(capturedHeader).To(Equal("player-123"))
		})
	})

	Describe("RequestGuard", func() {
		It("runs the guard after decode and blocks dispatch on error", func() {
			disp := command.NewDispatcher()
			dispatchCalled := false
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatchCalled = true
				return nil
			})
			app := mustNewApp(cqrshtmx.Config{
				Commands:     disp,
				ErrorHandler: cqrshtmx.JSONErrorHandler,
			})

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
				cqrshtmx.RequestGuard(func(_ *http.Request, _ any) error {
					return cqrshtmx.ErrForbidden
				}),
			)

			w := serve(handler, newPostJSONRequest(`{}`))
			Expect(w.Code).To(Equal(http.StatusForbidden))
			Expect(dispatchCalled).To(BeFalse(), "dispatch should not be called when guard fails")
		})

		It("allows dispatch when guard returns nil", func() {
			disp := command.NewDispatcher()
			dispatchCalled := false
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatchCalled = true
				return nil
			})
			app := mustNewApp(cqrshtmx.Config{Commands: disp})

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
				cqrshtmx.RequestGuard(func(_ *http.Request, _ any) error {
					return nil
				}),
			)

			w := serve(handler, newPostJSONRequest(`{}`))
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(dispatchCalled).To(BeTrue())
		})
	})

	Describe("Broadcaster.Close", func() {
		It("closes all subscriber channels", func() {
			b := cqrshtmx.NewBroadcaster()
			ch1 := b.Subscribe()
			ch2 := b.Subscribe()
			Expect(b.SubscriberCount()).To(Equal(2))

			b.Close()

			// Both channels should be closed (receive returns zero value, !ok)
			_, ok1 := <-ch1
			Expect(ok1).To(BeFalse())
			_, ok2 := <-ch2
			Expect(ok2).To(BeFalse())
			Expect(b.SubscriberCount()).To(Equal(0))
		})

		It("returns closed channel from Subscribe after Close", func() {
			b := cqrshtmx.NewBroadcaster()
			b.Close()

			ch := b.Subscribe()
			_, ok := <-ch
			Expect(ok).To(BeFalse(), "Subscribe after Close should return a closed channel")
		})

		It("is idempotent", func() {
			b := cqrshtmx.NewBroadcaster()
			b.Close()
			Expect(func() { b.Close() }).NotTo(Panic())
		})
	})

	Describe("Error code field in JSON responses", func() {
		It("includes the code field for classified errors", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return event.NewConflict("battle.exists", "battle already exists")
			})
			app := mustNewApp(cqrshtmx.Config{
				Commands:     disp,
				ErrorHandler: cqrshtmx.JSONErrorHandler,
			})

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
			)

			w := serve(handler, newPostJSONRequest(`{}`))
			Expect(w.Code).To(Equal(http.StatusConflict))

			var body map[string]any
			Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
			Expect(body).To(HaveKeyWithValue("code", "battle.exists"))
			Expect(body).To(HaveKeyWithValue("status", float64(http.StatusConflict)))
		})
	})

	Describe("CSRFTestToken", func() {
		It("extracts a valid CSRF token from the middleware chain", func() {
			mw := cqrshtmx.Chain(
				cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{
					Secure:   false,
					SameSite: http.SameSiteLaxMode,
				}),
				cqrshtmx.CSRFResponseHeaderMiddleware,
			)

			token := cqrshtmx.CSRFTestToken(mw)
			Expect(token).NotTo(BeEmpty())
		})
	})
})

// mustNewApp is a test helper that panics on error.
func mustNewApp(cfg cqrshtmx.Config) *cqrshtmx.App {
	app, err := cqrshtmx.New(cfg)
	if err != nil {
		panic(err)
	}
	return app
}

var _ = Describe("Empty-body GET decode (the SKILL.md GET query bug)", func() {
	It("DecodeJSONQuery succeeds on GET with no body (zero-value T)", func() {
		disp := query.NewDispatcher()
		_ = disp.Register("GetPage", func(_ context.Context, _ query.Query) (any, error) {
			return map[string]any{"ok": true}, nil
		})
		app := mustNewApp(cqrshtmx.Config{Queries: disp})

		handler := app.Query("GetPage",
			cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
				return &getPageQuery{}, nil
			}),
			cqrshtmx.RenderJSON[map[string]any](),
		)

		// GET request with no body — must NOT return 400
		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		w := serve(handler, r)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(ContainSubstring("ok"))
	})
})
