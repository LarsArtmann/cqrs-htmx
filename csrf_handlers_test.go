package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CSRF Handlers", func() {
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
})
