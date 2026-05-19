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

		It("evicts stale entries after TTL expires", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Minute,
				Burst:        1,
				KeyExtractor: func(_ *http.Request) string { return "evict-key" },
				TTL:          5 * time.Millisecond,
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First request should succeed
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)
			Expect(w1.Code).To(Equal(http.StatusOK))

			// Second request should be rate-limited (same entry)
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusTooManyRequests))

			// Wait for TTL to expire
			time.Sleep(10 * time.Millisecond)

			// Third request should succeed (entry evicted, new limiter created)
			w3 := httptest.NewRecorder()
			r3 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w3, r3)
			Expect(w3.Code).To(Equal(http.StatusOK))
		})

		It("does not evict entries accessed within TTL", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Minute,
				Burst:        1,
				KeyExtractor: func(_ *http.Request) string { return "fresh-key" },
				TTL:          100 * time.Millisecond,
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First request should succeed
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)
			Expect(w1.Code).To(Equal(http.StatusOK))

			// Access within TTL should still find the same entry (rate-limited)
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusTooManyRequests))
		})

		It("calls OnAllowed hook for allowed requests", func() {
			allowedCalled := false
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        10,
				Window:       time.Minute,
				Burst:        10,
				KeyExtractor: func(_ *http.Request) string { return "hook-key" },
				OnAllowed: func(_ *http.Request) {
					allowedCalled = true
				},
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(allowedCalled).To(BeTrue())
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("calls OnRejected hook for rejected requests", func() {
			rejectedCalled := false
			var capturedRetryAfter string
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Minute,
				Burst:        1,
				KeyExtractor: func(_ *http.Request) string { return "reject-key" },
				OnRejected: func(_ *http.Request, retryAfter string) {
					rejectedCalled = true
					capturedRetryAfter = retryAfter
				},
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First request succeeds
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			// Second request is rejected
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w2, r2)

			Expect(rejectedCalled).To(BeTrue())
			Expect(capturedRetryAfter).NotTo(BeEmpty())
			Expect(w2.Code).To(Equal(http.StatusTooManyRequests))
		})

		It("uses custom RejectionHandler when configured", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Minute,
				Burst:        1,
				KeyExtractor: func(_ *http.Request) string { return "custom-reject" },
				RejectionHandler: func(w http.ResponseWriter, _ *http.Request, _ string) {
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte("custom rejection"))
				},
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First request succeeds
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			// Second request uses custom handler
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w2, r2)

			Expect(w2.Code).To(Equal(http.StatusServiceUnavailable))
			Expect(w2.Body.String()).To(Equal("custom rejection"))
		})
	})
})
