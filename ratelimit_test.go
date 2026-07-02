package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rate Limiting", func() {
	Describe("RateLimiterMiddleware", func() {
		// Rate limiter configs are highly repetitive in the test suite.
		// The helpers below produce a base config with sensible defaults
		// (Limit=1, Window=1m, Burst=1) and let callers override the
		// extractor and TTL — the two knobs that actually change behavior.
		baseLimitConfig := func() cqrshtmx.RateLimiterConfig {
			return cqrshtmx.RateLimiterConfig{
				Limit:  1,
				Window: time.Minute,
				Burst:  1,
			}
		}
		keyedLimitConfig := func(key string) cqrshtmx.RateLimiterConfig {
			cfg := baseLimitConfig()
			cfg.KeyExtractor = func(_ *http.Request) string { return key }
			return cfg
		}

		assertRateLimit := func(cfg cqrshtmx.RateLimiterConfig, requests int) []int {
			middleware := cqrshtmx.RateLimiterMiddleware(cfg)
			handler := middleware(okHandler())
			codes := make([]int, 0, requests)
			for range requests {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(w, r)
				codes = append(codes, w.Code)
			}
			return codes
		}

		It("allows requests under the limit", func() {
			cfg := baseLimitConfig()
			cfg.Limit = 10
			cfg.Burst = 2
			cfg.KeyExtractor = func(_ *http.Request) string { return "test-key" }
			codes := assertRateLimit(cfg, 2)
			Expect(codes).To(Equal([]int{http.StatusOK, http.StatusOK}))
		})

		It("blocks requests over the limit", func() {
			codes := assertRateLimit(keyedLimitConfig("test-key"), 2)
			Expect(codes[0]).To(Equal(http.StatusOK))
			Expect(codes[1]).To(Equal(http.StatusTooManyRequests))
		})

		It("includes Retry-After header on rejection", func() {
			middleware := cqrshtmx.RateLimiterMiddleware(keyedLimitConfig("retry-key"))
			handler := middleware(okHandler())
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(w.Header().Get("Retry-After")).NotTo(BeEmpty())
		})

		It("allows empty key extractor (global rate limit)", func() {
			codes := assertRateLimit(baseLimitConfig(), 2)
			Expect(codes[0]).To(Equal(http.StatusOK))
			Expect(codes[1]).To(Equal(http.StatusTooManyRequests))
		})

		It("skips rate limiting when key extractor returns empty", func() {
			codes := assertRateLimit(keyedLimitConfig(""), 3)
			Expect(codes).To(Equal([]int{http.StatusOK, http.StatusOK, http.StatusOK}))
		})

		It("uses sensible defaults when config is zero", func() {
			codes := assertRateLimit(cqrshtmx.RateLimiterConfig{
				KeyExtractor: func(_ *http.Request) string { return "key" },
			}, 3)
			Expect(codes).To(Equal([]int{http.StatusOK, http.StatusOK, http.StatusOK}))
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

			handler := middleware(okHandler())

			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			r1.RemoteAddr = ip1
			handler.ServeHTTP(w1, r1)
			Expect(w1.Code).To(Equal(http.StatusOK))

			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			r2.RemoteAddr = ip1
			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusTooManyRequests))

			w3 := httptest.NewRecorder()
			r3 := httptest.NewRequest(http.MethodGet, "/", nil)
			r3.RemoteAddr = ip2
			handler.ServeHTTP(w3, r3)
			Expect(w3.Code).To(Equal(http.StatusOK))
		})

		It("evicts stale entries after TTL expires", func() {
			cfg := keyedLimitConfig("evict-key")
			cfg.TTL = 5 * time.Millisecond
			middleware := cqrshtmx.RateLimiterMiddleware(cfg)
			handler := middleware(okHandler())

			codes := make([]int, 2)
			for i := range 2 {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
				codes[i] = w.Code
			}
			Expect(codes[0]).To(Equal(http.StatusOK))
			Expect(codes[1]).To(Equal(http.StatusTooManyRequests))

			time.Sleep(10 * time.Millisecond)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("does not evict entries accessed within TTL", func() {
			cfg := keyedLimitConfig("fresh-key")
			cfg.TTL = 100 * time.Millisecond
			codes := assertRateLimit(cfg, 2)
			Expect(codes[0]).To(Equal(http.StatusOK))
			Expect(codes[1]).To(Equal(http.StatusTooManyRequests))
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

			handler := middleware(okHandler())

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

			handler := middleware(okHandler())

			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

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

			handler := middleware(okHandler())

			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w2, r2)

			Expect(w2.Code).To(Equal(http.StatusServiceUnavailable))
			Expect(w2.Body.String()).To(Equal("custom rejection"))
		})
		It("evicts oldest key when MaxKeys is exceeded", func() {
			var allowed int
			middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        100,
				Window:       time.Minute,
				Burst:        100,
				MaxKeys:      2,
				TTL:          time.Hour,
				KeyExtractor: func(r *http.Request) string { return r.Header.Get("X-Key") },
				OnAllowed:    func(_ *http.Request) { allowed++ },
			})

			handler := middleware(okHandler())

			for _, key := range []string{"a", "b", "c"} {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("X-Key", key)
				handler.ServeHTTP(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
			}

			Expect(allowed).To(Equal(3))
		})
	})
})
