package cqrshtmx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CSRF Advanced", func() {
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
