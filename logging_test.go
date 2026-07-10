package cqrshtmx_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newLoggingCapture(wrapped http.Handler) (http.Handler, *string) {
	var logged string
	middleware := cqrshtmx.RequestLogging(nil, func(line string) {
		logged = line
	})
	return middleware(wrapped), &logged
}

func newSlogCapture() (func(http.Handler) http.Handler, *bytes.Buffer) {
	var buf bytes.Buffer
	//nolint:exhaustruct // test config intentionally uses defaults for other fields
	logger := slog.New(slog.NewJSONHandler(
		&buf,
		&slog.HandlerOptions{Level: slog.LevelInfo},
	))
	return cqrshtmx.RequestLoggingSlog(logger), &buf
}

// helloBodyHandler returns a handler that writes "hello" with no explicit status.
func helloBodyHandler() http.Handler {
	return writeStringHandler("hello")
}

func withContextIDs(r *http.Request) *http.Request {
	ctx := cqrshtmx.WithCorrelationID(r.Context(),
		cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633"))
	ctx = cqrshtmx.WithUserID(ctx,
		cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2"))
	return r.WithContext(ctx)
}

var _ = Describe("Request Logging", func() {
	Describe("RequestLoggingMiddleware", func() {
		It("logs requests with default formatter", func() {
			handler, logged := newLoggingCapture(createdHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users", nil)
			handler.ServeHTTP(w, r)

			Expect(*logged).To(ContainSubstring("POST /users"))
			Expect(*logged).To(ContainSubstring("Created"))
		})

		It("captures correlation ID from context", func() {
			handler, logged := newLoggingCapture(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/items", nil)
			r = r.WithContext(cqrshtmx.WithCorrelationID(r.Context(),
				cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633")))
			handler.ServeHTTP(w, r)

			Expect(*logged).To(ContainSubstring("correlation=01HK1549P84T9XF8R94E960633"))
		})

		It("captures user ID from context", func() {
			handler, logged := newLoggingCapture(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			r = r.WithContext(cqrshtmx.WithUserID(r.Context(),
				cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")))
			handler.ServeHTTP(w, r)

			Expect(*logged).To(ContainSubstring("user=01HK154ANGZHV2ZW0X3SKSNEN2"))
		})

		It("captures both correlation ID and user ID", func() {
			handler, logged := newLoggingCapture(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			r = withContextIDs(r)
			handler.ServeHTTP(w, r)

			Expect(*logged).To(ContainSubstring("correlation=01HK1549P84T9XF8R94E960633"))
			Expect(*logged).To(ContainSubstring("user=01HK154ANGZHV2ZW0X3SKSNEN2"))
		})

		It("defaults to 200 status when handler writes body without explicit status", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(nil, func(line string) {
				logged = line
			})

			handler := middleware(helloBodyHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring("OK"))
		})

		It("does not panic when writer is nil", func() {
			middleware := cqrshtmx.RequestLogging(nil, nil)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
			Expect(func() { handler.ServeHTTP(w, r) }).NotTo(Panic())
		})

		It("formats logs as JSON with JSONLogFormatter", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(line string) {
				logged = line
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/items/42", nil)
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring(`"method":"PUT"`))
			Expect(logged).To(ContainSubstring(`"path":"/items/42"`))
			Expect(logged).NotTo(ContainSubstring(`"query"`))
			Expect(logged).To(ContainSubstring(`"status":"Accepted"`))
			Expect(logged).To(ContainSubstring(`"duration"`))
		})

		It("includes correlation_id and user_id in JSON logs", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(line string) {
				logged = line
			})

			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			r = withContextIDs(r)
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring(`"correlation_id":"01HK1549P84T9XF8R94E960633"`))
			Expect(logged).To(ContainSubstring(`"user_id":"01HK154ANGZHV2ZW0X3SKSNEN2"`))
		})
	})

	Describe("RequestLoggingSlog", func() {
		It("logs requests with structured slog output", func() {
			middleware, buf := newSlogCapture()

			handler := middleware(createdHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users", nil)
			handler.ServeHTTP(w, r)

			logged := buf.String()
			Expect(logged).To(ContainSubstring(`"method":"POST"`))
			Expect(logged).To(ContainSubstring(`"path":"/users"`))
			Expect(logged).To(ContainSubstring(`"status":201`))
			Expect(logged).To(ContainSubstring(`"duration"`))
		})

		It("includes request_id from context in slog output", func() {
			middleware, buf := newSlogCapture()

			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api", nil)
			rid := cqrshtmx.MustParseRequestID("01HK1549P84T9XF8R94E960633")
			r = r.WithContext(cqrshtmx.WithRequestID(r.Context(), rid))
			handler.ServeHTTP(w, r)

			logged := buf.String()
			Expect(logged).To(ContainSubstring(`"request_id":"01HK1549P84T9XF8R94E960633"`))
		})
		It("omits query string from slog output", func() {
			middleware, buf := newSlogCapture()

			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api?expand=true", nil)
			handler.ServeHTTP(w, r)

			logged := buf.String()
			Expect(logged).NotTo(ContainSubstring(`"query"`))
		})

		It("includes correlation_id and user_id in slog output", func() {
			middleware, buf := newSlogCapture()

			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			r = withContextIDs(r)
			handler.ServeHTTP(w, r)

			logged := buf.String()
			Expect(logged).To(ContainSubstring(`"correlation_id":"01HK1549P84T9XF8R94E960633"`))
			Expect(logged).To(ContainSubstring(`"user_id":"01HK154ANGZHV2ZW0X3SKSNEN2"`))
		})

		It("defaults to 200 status when handler writes body without explicit status", func() {
			middleware, buf := newSlogCapture()

			handler := middleware(helloBodyHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			logged := buf.String()
			Expect(logged).To(ContainSubstring(`"status":200`))
		})

		It("includes error code, family, and context on dispatch failure", func() {
			middleware, buf := newSlogCapture()

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return errorfamily.NewRejection("test.bad_input", "bad input").
					WithContext("user_id", "01JX123")
			})
			app := mustNewApp(cqrshtmx.Config{Commands: disp})

			handler := middleware(app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ struct{}) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
			))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, newPostJSONRequest(`{}`))

			logged := buf.String()
			Expect(logged).To(ContainSubstring(`"error_code"`))
			Expect(logged).To(ContainSubstring(`"error_family":"rejection"`))
			Expect(logged).To(ContainSubstring(`"error_ctx_user_id":"01JX123"`))
		})

		It("logs transient family on transient dispatch failure", func() {
			middleware, buf := newSlogCapture()

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return errorfamily.NewTransient("test.db_down", "database temporarily unavailable")
			})
			app := mustNewApp(cqrshtmx.Config{Commands: disp})

			handler := middleware(app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ struct{}) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
			))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, newPostJSONRequest(`{}`))

			logged := buf.String()
			Expect(logged).To(ContainSubstring(`"error_family":"transient"`))
			Expect(logged).To(ContainSubstring(`"error_code"`))
		})

		It("logs conflict family on conflict dispatch failure", func() {
			middleware, buf := newSlogCapture()

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return errorfamily.NewConflict("test.email_taken", "email already registered")
			})
			app := mustNewApp(cqrshtmx.Config{Commands: disp})

			handler := middleware(app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ struct{}) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
			))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, newPostJSONRequest(`{}`))

			logged := buf.String()
			Expect(logged).To(ContainSubstring(`"error_family":"conflict"`))
			Expect(logged).To(ContainSubstring(`"error_code"`))
		})
	})

	Describe("statusRecorder", func() {
		It("delegates Push to underlying http.Pusher", func() {
			var pushedTarget string
			pusher := &mockPusher{
				ResponseWriter: httptest.NewRecorder(),
				pushFunc: func(target string, _ *http.PushOptions) error {
					pushedTarget = target
					return nil
				},
			}
			handler := cqrshtmx.RequestLogging(nil, func(_ string) {})
			pushHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if p, ok := w.(http.Pusher); ok {
					_ = p.Push("/style.css", nil)
				}
			})

			w := newPusherRecorder(pusher)
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler(pushHandler).ServeHTTP(w, r)

			Expect(pushedTarget).To(Equal("/style.css"))
		})

		It("returns ErrNotSupported when underlying ResponseWriter has no Pusher", func() {
			handler := cqrshtmx.RequestLogging(nil, func(_ string) {})
			pushHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if p, ok := w.(http.Pusher); ok {
					err := p.Push("/style.css", nil)
					Expect(err).To(Equal(http.ErrNotSupported))
				}
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler(pushHandler).ServeHTTP(w, r)
		})

		It("delegates Flush to underlying http.Flusher", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(nil, func(line string) {
				logged = line
			})

			flushHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				w.WriteHeader(http.StatusOK)
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/stream", nil)
			middleware(flushHandler).ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring("OK"))
		})

		It("delegates Hijack to underlying http.Hijacker", func() {
			handler := cqrshtmx.RequestLogging(nil, func(_ string) {})
			hijackHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if h, ok := w.(http.Hijacker); ok {
					_, _, _ = h.Hijack()
				}
			})

			w := newHijackRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler(hijackHandler).ServeHTTP(w, r)
		})
	})
})
