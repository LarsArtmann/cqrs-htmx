package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CSRF Middleware", func() {
	Describe("CSRFMiddleware", func() {
		It("sets a CSRF token cookie on GET requests", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))

			cookies := w.Result().Cookies()
			Expect(cookies).To(HaveLen(1))
			Expect(cookies[0].Name).To(Equal("csrf_token"))
			Expect(cookies[0].Value).NotTo(BeEmpty())
		})

		It("stores the token in context for downstream use", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			var capturedToken string

			handler := csrfTokenCaptureHandler(middleware, &capturedToken, false)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(capturedToken).NotTo(BeEmpty())
		})

		DescribeTable(
			"allows POST with valid CSRF token",
			func(headerName, fieldName string) {
				mw := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
				code := csrfGETThenPOST(mw, headerName, fieldName)
				Expect(code).To(Equal(http.StatusOK))
			},
			Entry("in header", "X-CSRF-Token", ""),
			Entry("in form field", "", "csrf_token"),
			Entry("validates generated token (header)", "X-CSRF-Token", ""),
		)

		It("rejects POST without CSRF token", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			r.Header.Set("Sec-Fetch-Site", "same-origin")
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("rejects POST with invalid CSRF token", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			handler := middleware(okHandler())

			// First GET to obtain token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			// POST with wrong token
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			r2.Header.Set("X-CSRF-Token", "invalid-token")
			r2.Header.Set("Sec-Fetch-Site", "same-origin")

			// Copy cookie to POST request
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusForbidden))
		})

		It("reuses existing cookie token on subsequent requests", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			handler := middleware(okHandler())

			// First GET
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			// Second GET with cookie
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			r2.AddCookie(w1.Result().Cookies()[0])
			handler.ServeHTTP(w2, r2)

			// Should not set a new cookie
			cookies2 := w2.Result().Cookies()
			Expect(cookies2).To(BeEmpty())

			// Context should have a valid masked token
			var capturedToken string
			handler2 := csrfTokenCaptureHandler(middleware, &capturedToken, false)

			w3 := httptest.NewRecorder()
			r3 := httptest.NewRequest(http.MethodGet, "/", nil)
			r3.AddCookie(w1.Result().Cookies()[0])
			handler2.ServeHTTP(w3, r3)

			Expect(capturedToken).NotTo(BeEmpty())
		})

		It("validates PUT, PATCH, and DELETE methods", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			var token string
			handler := csrfTokenCaptureHandler(middleware, &token, true)

			// First GET to obtain masked token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			Expect(token).NotTo(BeEmpty())

			for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
				w2 := httptest.NewRecorder()
				r2 := httptest.NewRequest(method, "/", strings.NewReader("{}"))
				r2.Header.Set("X-CSRF-Token", token)
				r2.Header.Set("Sec-Fetch-Site", "same-origin")
				for _, c := range w1.Result().Cookies() {
					r2.AddCookie(c)
				}

				handler.ServeHTTP(w2, r2)
				Expect(
					w2.Code,
				).To(Equal(http.StatusOK), "method %s should pass with valid token", method)
			}
		})

		It("skips validation for GET, HEAD, OPTIONS, TRACE", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			handler := middleware(okHandler())

			for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace} {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(method, "/", nil)
				handler.ServeHTTP(w, r)
				Expect(w.Code).To(Equal(http.StatusOK), "method %s should not require CSRF", method)
			}
		})

		It("uses custom cookie name when configured", func() {
			middleware := cqrshtmx.CSRFMiddleware(csrfConfigWith(func(cfg *cqrshtmx.CSRFConfig) {
				cfg.CookieName = "my_csrf"
			}))
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies).To(HaveLen(1))
			Expect(cookies[0].Name).To(Equal("my_csrf"))
		})

		It("uses custom header name when configured", func() {
			mw := cqrshtmx.CSRFMiddleware(csrfConfigWith(func(cfg *cqrshtmx.CSRFConfig) {
				cfg.HeaderName = "X-Custom-CSRF"
			}))
			code := csrfGETThenPOST(mw, "X-Custom-CSRF", "")
			Expect(code).To(Equal(http.StatusOK))
		})

		It("uses custom error handler on failure", func() {
			customCalled := false
			middleware := cqrshtmx.CSRFMiddleware(csrfConfigWith(func(cfg *cqrshtmx.CSRFConfig) {
				cfg.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
					customCalled = true
					w.WriteHeader(http.StatusTeapot)
					_, _ = w.Write([]byte("custom csrf error"))
				}
			}))
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			r.Header.Set("Sec-Fetch-Site", "same-origin")
			handler.ServeHTTP(w, r)

			Expect(customCalled).To(BeTrue())
			Expect(w.Code).To(Equal(http.StatusTeapot))
			Expect(w.Body.String()).To(Equal("custom csrf error"))
		})

		It("sets cookie with secure flag when configured", func() {
			cfg := defaultCSRFConfig()
			cfg.Secure = true
			middleware := cqrshtmx.CSRFMiddleware(cfg)
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies[0].Secure).To(BeTrue())
		})

		It("sets cookie with lax SameSite by default", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies[0].SameSite).To(Equal(http.SameSiteLaxMode))
		})

		It("sets HttpOnly to false for double-submit pattern", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies[0].HttpOnly).To(BeFalse())
		})
	})

	Describe("CSRF TrustedProxies (plaintext HTTP origin bypass)", func() {
		// commonProxyCIDR is reused across proxy-bypass tests to avoid
		// goconst repetition and keep fixtures readable.
		const (
			commonProxyCIDR  = "10.0.0.0/8"
			commonProxyIP    = "192.168.1.5"
			attackerRemoteIP = "203.0.113.42:54321"
		)

		// csrfGETThenPOSTWithRemoteAddr is the GET-then-POST CSRF happy path,
		// but with a configurable RemoteAddr on both requests. Used to test
		// the plaintext-HTTP origin bypass logic in setPlaintextHTTPOrigin.
		csrfGETThenPOSTWithRemoteAddr := func(
			middleware func(http.Handler) http.Handler, remoteAddr string,
		) int {
			var token string
			handler := csrfTokenCaptureHandler(middleware, &token, true)

			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequestWithContext(
				context.Background(), http.MethodGet, "/", nil,
			)
			r1.RemoteAddr = remoteAddr
			handler.ServeHTTP(w1, r1)

			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequestWithContext(
				context.Background(), http.MethodPost, "/",
				strings.NewReader(url.Values{"x": {"1"}}.Encode()),
			)
			r2.RemoteAddr = remoteAddr
			r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r2.Header.Set("X-CSRF-Token", token)
			// Critical: do NOT set Origin, Referer, or Sec-Fetch-Site.
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}
			handler.ServeHTTP(w2, r2)

			return w2.Code
		}

		It("bypasses origin check for loopback (localhost) without TrustedProxies", func() {
			mw := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			code := csrfGETThenPOSTWithRemoteAddr(mw, "127.0.0.1:54321")
			Expect(code).To(Equal(http.StatusOK))
		})

		It("rejects plain HTTP POST from non-loopback by default (secure zero-value config)", func() {
			// defaultCSRFConfig leaves TrustedProxies empty and AllowPlaintextBypass
			// false: a non-loopback attacker with no Origin/Referer/Sec-Fetch-Site
			// must NOT get the bypass injected.
			mw := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			code := csrfGETThenPOSTWithRemoteAddr(mw, attackerRemoteIP)
			Expect(code).To(Equal(http.StatusForbidden))
		})

		It("allows plain HTTP POST from non-loopback when AllowPlaintextBypass is set (opt-in)", func() {
			cfg := defaultCSRFConfig()
			cfg.AllowPlaintextBypass = true
			mw := cqrshtmx.CSRFMiddleware(cfg)
			code := csrfGETThenPOSTWithRemoteAddr(mw, attackerRemoteIP)
			Expect(code).To(Equal(http.StatusOK))
		})

		It("bypasses origin check for configured TrustedProxies (CIDR)", func() {
			cfg := defaultCSRFConfig()
			cfg.TrustedProxies = []string{commonProxyCIDR}
			mw := cqrshtmx.CSRFMiddleware(cfg)
			code := csrfGETThenPOSTWithRemoteAddr(mw, "10.5.6.7:54321")
			Expect(code).To(Equal(http.StatusOK))
		})

		It("bypasses origin check for configured TrustedProxies (single IP)", func() {
			cfg := defaultCSRFConfig()
			cfg.TrustedProxies = []string{commonProxyIP}
			mw := cqrshtmx.CSRFMiddleware(cfg)
			code := csrfGETThenPOSTWithRemoteAddr(mw, commonProxyIP+":54321")
			Expect(code).To(Equal(http.StatusOK))
		})

		It("rejects plain HTTP POST from untrusted remote when TrustedProxies configured", func() {
			cfg := defaultCSRFConfig()
			cfg.TrustedProxies = []string{commonProxyCIDR}
			mw := cqrshtmx.CSRFMiddleware(cfg)
			code := csrfGETThenPOSTWithRemoteAddr(mw, attackerRemoteIP)
			// Attacker IP outside trusted range -> nosurf rejects (403).
			Expect(code).To(Equal(http.StatusForbidden))
		})

		It("allows plain HTTP POST from untrusted remote when Sec-Fetch-Site is set explicitly", func() {
			cfg := defaultCSRFConfig()
			cfg.TrustedProxies = []string{commonProxyCIDR}
			mw := cqrshtmx.CSRFMiddleware(cfg)
			handler := mw(okHandler())

			// Simulate a browser that DID send Sec-Fetch-Site: same-origin.
			// nosurf accepts via the explicit Sec-Fetch-Site path; TrustedProxies
			// is irrelevant when the header is already present.
			var token string
			h := csrfTokenCaptureHandler(mw, &token, true)
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequestWithContext(
				context.Background(), http.MethodGet, "/", nil,
			)
			r1.RemoteAddr = attackerRemoteIP
			h.ServeHTTP(w1, r1)

			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequestWithContext(
				context.Background(), http.MethodPost, "/",
				strings.NewReader(""),
			)
			r2.RemoteAddr = attackerRemoteIP
			r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r2.Header.Set("X-CSRF-Token", token)
			r2.Header.Set("Sec-Fetch-Site", "same-origin")
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}
			h.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusOK))
			_ = handler // silence unused
		})

		DescribeTable(
			"Validate rejects invalid TrustedProxies",
			func(proxy string) {
				cfg := defaultCSRFConfig()
				cfg.TrustedProxies = []string{proxy}
				Expect(cfg.Validate()).To(HaveOccurred())
			},
			Entry("empty entry", ""),
			Entry("malformed CIDR", "not-a-cidr/xyz"),
		)
	})
})
