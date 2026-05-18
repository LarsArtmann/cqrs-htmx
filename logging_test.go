package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Request Logging", func() {
	Describe("RequestLoggingMiddleware", func() {
		It("logs requests with default formatter", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(nil, func(line string) {
				logged = line
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users", nil)
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring("POST /users"))
			Expect(logged).To(ContainSubstring("Created"))
		})

		It("captures correlation ID from context", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(nil, func(line string) {
				logged = line
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/items", nil)
			r = r.WithContext(cqrshtmx.WithCorrelationID(r.Context(),
				cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633")))
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring("correlation=01HK1549P84T9XF8R94E960633"))
		})

		It("captures user ID from context", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(nil, func(line string) {
				logged = line
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			r = r.WithContext(cqrshtmx.WithUserID(r.Context(),
				cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")))
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring("user=01HK154ANGZHV2ZW0X3SKSNEN2"))
		})

		It("captures both correlation ID and user ID", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(nil, func(line string) {
				logged = line
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			ctx := cqrshtmx.WithCorrelationID(r.Context(),
				cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633"))
			ctx = cqrshtmx.WithUserID(ctx,
				cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2"))
			r = r.WithContext(ctx)
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring("correlation=01HK1549P84T9XF8R94E960633"))
			Expect(logged).To(ContainSubstring("user=01HK154ANGZHV2ZW0X3SKSNEN2"))
		})

		It("defaults to 200 status when handler writes body without explicit status", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(nil, func(line string) {
				logged = line
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("hello"))
			}))

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
			r := httptest.NewRequest(http.MethodPut, "/items/42?expand=true", nil)
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring(`"method":"PUT"`))
			Expect(logged).To(ContainSubstring(`"path":"/items/42"`))
			Expect(logged).To(ContainSubstring(`"query":"expand=true"`))
			Expect(logged).To(ContainSubstring(`"status":"Accepted"`))
			Expect(logged).To(ContainSubstring(`"duration"`))
		})

		It("includes correlation_id and user_id in JSON logs", func() {
			var logged string
			middleware := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(line string) {
				logged = line
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			ctx := cqrshtmx.WithCorrelationID(r.Context(),
				cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633"))
			ctx = cqrshtmx.WithUserID(ctx,
				cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2"))
			r = r.WithContext(ctx)
			handler.ServeHTTP(w, r)

			Expect(logged).To(ContainSubstring(`"correlation_id":"01HK1549P84T9XF8R94E960633"`))
			Expect(logged).To(ContainSubstring(`"user_id":"01HK154ANGZHV2ZW0X3SKSNEN2"`))
		})
	})
})
