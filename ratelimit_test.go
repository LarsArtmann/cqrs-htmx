package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rate Limiting", func() {
	Describe("RateLimiterMiddleware", func() {
		It("allows requests under the limit", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        10,
				Window:       time.Minute,
				Burst:        2,
				KeyExtractor: func(_ *http.Request) string { return "test-key" },
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			for range 2 {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
			}
		})

		It("blocks requests over the limit", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Minute,
				Burst:        1,
				KeyExtractor: func(_ *http.Request) string { return "test-key" },
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First request should succeed
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)
			Expect(w1.Code).To(Equal(http.StatusOK))

			// Second request should be rate-limited
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusTooManyRequests))
			Expect(w2.Header().Get("Retry-After")).NotTo(BeEmpty())
		})

		It("allows empty key extractor (global rate limit)", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:  1,
				Window: time.Minute,
				Burst:  1,
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First request should succeed
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)
			Expect(w1.Code).To(Equal(http.StatusOK))

			// Second request should be rate-limited (same key = "")
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusTooManyRequests))
		})

		It("skips rate limiting when key extractor returns empty", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Minute,
				Burst:        1,
				KeyExtractor: func(_ *http.Request) string { return "" },
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			for range 3 {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
			}
		})

		It("uses sensible defaults when config is zero", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				KeyExtractor: func(_ *http.Request) string { return "key" },
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Defaults: Limit=100, Burst=100. Should easily allow 3 requests.
			for range 3 {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
			}
		})

		It("rate-limits per RemoteAddr with KeyExtractorFromRemoteAddr", func() {
			const ip1 = "192.168.1.1:1234"
			const ip2 = "192.168.1.2:5678"

			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Minute,
				Burst:        1,
				KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First request from IP1 should succeed
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			r1.RemoteAddr = ip1
			handler.ServeHTTP(w1, r1)
			Expect(w1.Code).To(Equal(http.StatusOK))

			// Second request from IP1 should be rate-limited
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			r2.RemoteAddr = ip1
			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusTooManyRequests))

			// Request from IP2 should still succeed (different key)
			w3 := httptest.NewRecorder()
			r3 := httptest.NewRequest(http.MethodGet, "/", nil)
			r3.RemoteAddr = ip2
			handler.ServeHTTP(w3, r3)
			Expect(w3.Code).To(Equal(http.StatusOK))
		})
	})
})
