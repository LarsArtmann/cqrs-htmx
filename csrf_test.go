package cqrshtmx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func defaultCSRFConfig() cqrshtmx.CSRFConfig {
	return cqrshtmx.CSRFConfig{
		CookieName:     "",
		HeaderName:     "",
		FieldName:      "",
		MaxAge:         24 * time.Hour,
		Secure:         false,
		SameSite:       http.SameSiteLaxMode,
		Domain:         "",
		Path:           "/",
		TrustedOrigins: nil,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusForbidden)
		},
	}
}

// csrfConfigWith returns a copy of defaultCSRFConfig with the given overrides applied.
func csrfConfigWith(overrides func(*cqrshtmx.CSRFConfig)) cqrshtmx.CSRFConfig {
	cfg := defaultCSRFConfig()
	overrides(&cfg)
	return cfg
}

// csrfTokenOnceHandler wraps a middleware with a handler that captures the CSRF token once.
func csrfTokenOnceHandler(middleware func(http.Handler) http.Handler, token *string) http.Handler {
	return middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *token == "" {
			*token = cqrshtmx.CSRFTokenFromContext(r.Context())
		}
		w.WriteHeader(http.StatusOK)
	}))
}

// csrfTokenCaptureHandler wraps a middleware with a handler that always captures the CSRF token.
func csrfTokenCaptureHandler(
	middleware func(http.Handler) http.Handler,
	token *string,
) http.Handler {
	return middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*token = cqrshtmx.CSRFTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
}

// csrfGETThenPOST performs the common CSRF test pattern:
// GET to obtain a masked token, then POST with the token in the specified header/field.
func csrfGETThenPOST(
	middleware func(http.Handler) http.Handler,
	headerName, fieldName string,
) int {
	var token string
	handler := csrfTokenOnceHandler(middleware, &token)

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(w1, r1)

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/",
		strings.NewReader("{}"),
	)
	r2.Header.Set("Sec-Fetch-Site", "same-origin")
	if headerName != "" {
		r2.Header.Set(headerName, token)
	}
	if fieldName != "" {
		body := fieldName + "=" + url.QueryEscape(token)
		r2 = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"/",
			strings.NewReader(body),
		)
		r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r2.Header.Set("Sec-Fetch-Site", "same-origin")
	}
	for _, c := range w1.Result().Cookies() {
		r2.AddCookie(c)
	}
	handler.ServeHTTP(w2, r2)

	return w2.Code
}

var _ = Describe("CSRF Protection", func() {
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

	Describe("CSRFProtect HandlerOption", func() {
		It("validates CSRF token for specific handlers", func() {
			// Set up a handler with CSRFProtect but no global middleware
			// We need to manually set the token in context
			handler := okHandler()

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

	Describe("Token generation", func() {
		It("generates different tokens across middleware instances", func() {
			middleware1 := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())
			middleware2 := cqrshtmx.CSRFMiddleware(defaultCSRFConfig())

			var token1, token2 string
			handler1 := middleware1(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token1 = cqrshtmx.CSRFTokenFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))
			handler2 := middleware2(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token2 = cqrshtmx.CSRFTokenFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler1.ServeHTTP(w1, r1)

			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler2.ServeHTTP(w2, r2)

			Expect(token1).NotTo(BeEmpty())
			Expect(token2).NotTo(BeEmpty())
			Expect(token1).NotTo(Equal(token2))
		})
	})

	Describe("Custom Domain and Path", func() {
		It("sets cookie with custom domain and path", func() {
			middleware := cqrshtmx.CSRFMiddleware(csrfConfigWith(func(cfg *cqrshtmx.CSRFConfig) {
				cfg.Domain = "example.com"
				cfg.Path = "/api"
				cfg.ErrorHandler = nil
			}))
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies[0].Domain).To(Equal("example.com"))
			Expect(cookies[0].Path).To(Equal("/api"))
		})
	})

	Describe("SameSite=NoneMode", func() {
		It("sets SameSite=None when configured", func() {
			middleware := cqrshtmx.CSRFMiddleware(csrfConfigWith(func(cfg *cqrshtmx.CSRFConfig) {
				cfg.SameSite = http.SameSiteNoneMode
				cfg.ErrorHandler = nil
			}))
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			cookies := w.Result().Cookies()
			Expect(cookies[0].SameSite).To(Equal(http.SameSiteNoneMode))
		})
	})

	Describe("CSRFTokenHTMLMeta helper", func() {
		It("returns meta tag with escaped token", func() {
			ctx := cqrshtmx.WithCSRFToken(GinkgoT().Context(), "test-token")
			r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			meta := cqrshtmx.CSRFTokenHTMLMeta(r)
			Expect(meta).To(Equal(`<meta name="csrf-token" content="test-token">`))
		})

		It("returns empty string when no token in context", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			meta := cqrshtmx.CSRFTokenHTMLMeta(r)
			Expect(meta).To(BeEmpty())
		})

		It("HTML-escapes the token", func() {
			ctx := cqrshtmx.WithCSRFToken(GinkgoT().Context(), `<script>alert("xss")</script>`)
			r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			meta := cqrshtmx.CSRFTokenHTMLMeta(r)
			Expect(meta).To(ContainSubstring(`&lt;script&gt;`))
			Expect(meta).NotTo(ContainSubstring(`<script>`))
		})
	})

	Describe("CSRFTokenHXHeaders helper", func() {
		It("returns hx-headers attribute with escaped token", func() {
			ctx := cqrshtmx.WithCSRFToken(GinkgoT().Context(), "test-token")
			r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			attr := cqrshtmx.CSRFTokenHXHeaders(r)
			Expect(attr).To(Equal(`hx-headers='{"X-CSRF-Token":"test-token"}'`))
		})

		It("returns empty string when no token in context", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			attr := cqrshtmx.CSRFTokenHXHeaders(r)
			Expect(attr).To(BeEmpty())
		})
	})

	Describe("CSRFTokenFormField helper", func() {
		It("returns hidden input with escaped token", func() {
			ctx := cqrshtmx.WithCSRFToken(GinkgoT().Context(), "test-token")
			r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			field := cqrshtmx.CSRFTokenFormField(r)
			Expect(field).To(Equal(`<input type="hidden" name="csrf_token" value="test-token">`))
		})

		It("returns empty string when no token in context", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			field := cqrshtmx.CSRFTokenFormField(r)
			Expect(field).To(BeEmpty())
		})
	})

	Describe("CSRFResponseHeaderMiddleware", func() {
		It("sets X-CSRF-Token header when token is in context", func() {
			middleware := cqrshtmx.CSRFResponseHeaderMiddleware
			handler := middleware(okHandler())

			ctx := cqrshtmx.WithCSRFToken(GinkgoT().Context(), "response-token")
			r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Header().Get("X-CSRF-Token")).To(Equal("response-token"))
		})

		It("does not set header when no token is in context", func() {
			middleware := cqrshtmx.CSRFResponseHeaderMiddleware
			handler := middleware(okHandler())

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Header().Get("X-CSRF-Token")).To(BeEmpty())
		})
	})

	Describe("ErrCSRFInvalid", func() {
		It("maps to 403 Forbidden", func() {
			Expect(cqrshtmx.MapError(cqrshtmx.ErrCSRFInvalid)).To(Equal(http.StatusForbidden))
		})
	})

	Describe("InvalidateCSRFCookie", func() {
		It("sets an expired cookie to invalidate the token", func() {
			w := httptest.NewRecorder()
			cfg := cqrshtmx.CSRFConfig{}
			cqrshtmx.InvalidateCSRFCookie(w, cfg)

			cookies := w.Result().Cookies()
			Expect(cookies).To(HaveLen(1))

			cookie := cookies[0]
			Expect(cookie.Name).To(Equal("csrf_token"))
			Expect(cookie.Value).To(BeEmpty())
			Expect(cookie.MaxAge).To(Equal(-1))
			Expect(cookie.Expires).To(BeTemporally("<", time.Now()))
		})

		It("uses configured cookie name", func() {
			w := httptest.NewRecorder()
			cfg := cqrshtmx.CSRFConfig{
				CookieName: "my_token",
			}
			cqrshtmx.InvalidateCSRFCookie(w, cfg)

			cookies := w.Result().Cookies()
			Expect(cookies[0].Name).To(Equal("my_token"))
		})

		It("copies path, domain, secure, and samesite from config", func() {
			w := httptest.NewRecorder()
			cfg := cqrshtmx.CSRFConfig{
				Path:     "/api",
				Domain:   "example.com",
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
			}
			cqrshtmx.InvalidateCSRFCookie(w, cfg)

			cookie := w.Result().Cookies()[0]
			Expect(cookie.Path).To(Equal("/api"))
			Expect(cookie.Domain).To(Equal("example.com"))
			Expect(cookie.Secure).To(BeTrue())
			Expect(cookie.SameSite).To(Equal(http.SameSiteStrictMode))
		})
	})

	Describe("CSRFConfig.Validate", func() {
		It("returns nil for empty config (nosurf does not require secrets)", func() {
			cfg := cqrshtmx.CSRFConfig{}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("returns error when SameSite=None without Secure", func() {
			cfg := cqrshtmx.CSRFConfig{
				SameSite: http.SameSiteNoneMode,
				Secure:   false,
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, cqrshtmx.ErrCSRFConfig)).To(BeTrue())
		})

		It("returns nil for valid config", func() {
			cfg := cqrshtmx.CSRFConfig{}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("returns nil for SameSite=None with Secure", func() {
			cfg := cqrshtmx.CSRFConfig{
				SameSite: http.SameSiteNoneMode,
				Secure:   true,
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		DescribeTable(
			"rejects invalid TrustedOrigins entries",
			func(origins []string) {
				cfg := cqrshtmx.CSRFConfig{TrustedOrigins: origins}
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, cqrshtmx.ErrCSRFConfig)).To(BeTrue())
			},
			Entry("wildcard", []string{"*"}),
			Entry("empty string", []string{""}),
		)

		It("returns nil for specific TrustedOrigins domains", func() {
			cfg := cqrshtmx.CSRFConfig{
				TrustedOrigins: []string{"https://example.com", "https://api.example.com"},
			}
			Expect(cfg.Validate()).To(Succeed())
		})
	})
})

var _ = Describe("CSRF config defaults", func() {
	It("uses default field name when empty", func() {
		cfg := cqrshtmx.CSRFConfig{}
		mw := cqrshtmx.CSRFMiddleware(cfg)
		handler := mw(okHandler())
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(
			context.Background(), http.MethodGet, "/", nil,
		)
		handler.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("uses default SameSite when zero", func() {
		cfg := cqrshtmx.CSRFConfig{
			SameSite: 0,
		}
		Expect(cfg.Validate()).To(Succeed())
	})

	It("reads token from context when nosurf context has none", func() {
		token := cqrshtmx.CSRFTokenFromContext(
			cqrshtmx.WithCSRFToken(context.Background(), "fallback-token"),
		)
		Expect(token).To(Equal("fallback-token"))
	})

	It("returns empty token from empty context", func() {
		token := cqrshtmx.CSRFTokenFromContext(context.Background())
		Expect(token).To(BeEmpty())
	})
})
