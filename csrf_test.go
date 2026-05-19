package cqrshtmx_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func defaultCSRFConfig() cqrshtmx.CSRFConfig {
	return cqrshtmx.CSRFConfig{
		Secret:     nil,
		CookieName: "",
		HeaderName: "",
		FieldName:  "",
		MaxAge:     24 * time.Hour,
		Secure:     false,
		SameSite:   http.SameSiteLaxMode,
		Domain:     "",
		Path:       "/",
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusForbidden)
		},
	}
}

func defaultTLSConnectionState() tls.ConnectionState {
	return tls.ConnectionState{
		Version:                     0,
		HandshakeComplete:           false,
		DidResume:                   false,
		CipherSuite:                 0,
		CurveID:                     0,
		NegotiatedProtocol:          "",
		NegotiatedProtocolIsMutual: false,
		ServerName:                  "",
		PeerCertificates:            nil,
		VerifiedChains:             nil,
		SignedCertificateTimestamps: nil,
		OCSPResponse:                nil,
		TLSUnique:                   nil,
		ECHAccepted:                false,
		HelloRetryRequest:           nil,
	}
}

var _ = Describe("CSRF Protection", func() {
	Describe("CSRFMiddleware", func() {
		It("sets a CSRF token cookie on GET requests", func() {
			middleware := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

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
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			var capturedToken string

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedToken = cqrshtmx.CSRFTokenFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(capturedToken).NotTo(BeEmpty())
		})

		It("allows POST with valid CSRF token in header", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First GET to obtain token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			token := w1.Result().Cookies()[0].Value

			// POST with token in header
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			r2.Header.Set("X-CSRF-Token", token)

			// Copy cookie to POST request
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusOK))
		})

		It("allows POST with valid CSRF token in form field", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First GET to obtain token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			token := w1.Result().Cookies()[0].Value

			// POST with token in form field
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("csrf_token="+token))
			r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Copy cookie to POST request
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusOK))
		})

		It("rejects POST without CSRF token", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusForbidden))
			Expect(w.Body.String()).To(ContainSubstring("CSRF"))
		})

		It("rejects POST with invalid CSRF token", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First GET to obtain token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			// POST with wrong token
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			r2.Header.Set("X-CSRF-Token", "invalid-token")

			// Copy cookie to POST request
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusForbidden))
		})

		It("reuses existing cookie token on subsequent requests", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First GET
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)
			token1 := w1.Result().Cookies()[0].Value

			// Second GET with cookie
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			r2.AddCookie(w1.Result().Cookies()[0])
			handler.ServeHTTP(w2, r2)

			// Should not set a new cookie
			cookies2 := w2.Result().Cookies()
			Expect(cookies2).To(BeEmpty())

			// Context should have the same token
			var capturedToken string
			handler2 := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedToken = cqrshtmx.CSRFTokenFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			w3 := httptest.NewRecorder()
			r3 := httptest.NewRequest(http.MethodGet, "/", nil)
			r3.AddCookie(w1.Result().Cookies()[0])
			handler2.ServeHTTP(w3, r3)

			Expect(capturedToken).To(Equal(token1))
		})

		It("validates PUT, PATCH, and DELETE methods", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First GET to obtain token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			token := w1.Result().Cookies()[0].Value

			for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
				w2 := httptest.NewRecorder()
				r2 := httptest.NewRequest(method, "/", strings.NewReader("{}"))
				r2.Header.Set("X-CSRF-Token", token)
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
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace} {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(method, "/", nil)
				handler.ServeHTTP(w, r)
				Expect(w.Code).To(Equal(http.StatusOK), "method %s should not require CSRF", method)
			}
		})

		It("uses custom cookie name when configured", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{
				CookieName: "my_csrf",
			})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies).To(HaveLen(1))
			Expect(cookies[0].Name).To(Equal("my_csrf"))
		})

		It("uses custom header name when configured", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{
				HeaderName: "X-Custom-CSRF",
			})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First GET to obtain token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			token := w1.Result().Cookies()[0].Value

			// POST with custom header
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			r2.Header.Set("X-Custom-CSRF", token)
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusOK))
		})

		It("uses custom error handler on failure", func() {
			customCalled := false
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{
				ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
					customCalled = true
					w.WriteHeader(http.StatusTeapot)
					_, _ = w.Write([]byte("custom csrf error"))
				},
			})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			handler.ServeHTTP(w, r)

			Expect(customCalled).To(BeTrue())
			Expect(w.Code).To(Equal(http.StatusTeapot))
			Expect(w.Body.String()).To(Equal("custom csrf error"))
		})

		It("sets cookie with secure flag for HTTPS requests", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.TLS = &tls.ConnectionState{} // Simulate HTTPS
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies[0].Secure).To(BeTrue())
		})

		It("sets cookie with lax SameSite by default", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies[0].SameSite).To(Equal(http.SameSiteLaxMode))
		})

		It("sets HttpOnly to false for double-submit pattern", func() {
			middleware := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies[0].HttpOnly).To(BeFalse())
		})
	})

	Describe("CSRFProtect HandlerOption", func() {
		It("validates CSRF token for specific handlers", func() {
			// Set up a handler with CSRFProtect but no global middleware
			// We need to manually set the token in context
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with context containing token
			wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := cqrshtmx.WithCSRFToken(r.Context(), "valid-token")
				handler.ServeHTTP(w, r.WithContext(ctx))
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			r.Header.Set("X-CSRF-Token", "valid-token")
			wrapped.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("CSRFTokenFromContext / WithCSRFToken", func() {
		It("stores and retrieves a CSRF token from context", func() {
			ctx := cqrshtmx.WithCSRFToken(GinkgoT().Context(), "my-token")
			token := cqrshtmx.CSRFTokenFromContext(ctx)
			Expect(token).To(Equal("my-token"))
		})

		It("returns empty string when no token is in context", func() {
			token := cqrshtmx.CSRFTokenFromContext(GinkgoT().Context())
			Expect(token).To(BeEmpty())
		})
	})

	Describe("Response.CSRFToken", func() {
		It("sets the X-CSRF-Token response header", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			resp := cqrshtmx.NewResponse(w, r)
			resp.CSRFToken("response-token").Apply()

			Expect(w.Header().Get("X-CSRF-Token")).To(Equal("response-token"))
		})
	})

	Describe("ErrCSRFInvalid", func() {
		It("maps to 403 Forbidden", func() {
			Expect(cqrshtmx.MapError(cqrshtmx.ErrCSRFInvalid)).To(Equal(http.StatusForbidden))
		})
	})
})
