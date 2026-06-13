package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
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

			handler := csrfTokenCaptureHandler(middleware, &capturedToken)

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
			handler2 := csrfTokenCaptureHandler(middleware, &capturedToken)

			w3 := httptest.NewRecorder()
			r3 := httptest.NewRequest(http.MethodGet, "/", nil)
			r3.AddCookie(w1.Result().Cookies()[0])
			handler2.ServeHTTP(w3, r3)

			Expect(capturedToken).NotTo(BeEmpty())
		})

		It("validates PUT, PATCH, and DELETE methods", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			var token string
			handler := csrfTokenOnceHandler(middleware, &token)

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
})
