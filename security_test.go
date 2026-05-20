package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Security Headers Middleware", func() {
	assertSecurityHeader := func(cfg cqrshtmx.SecurityHeadersConfig, header, expected string) {
		middleware := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cfg)
		handler := middleware(okHandler())

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		Expect(w.Header().Get(header)).To(Equal(expected))
	}

	It("sets X-Content-Type-Options to nosniff", func() {
		assertSecurityHeader(cqrshtmx.SecurityHeadersConfig{}, "X-Content-Type-Options", "nosniff")
	})

	It("sets X-Frame-Options to DENY", func() {
		assertSecurityHeader(cqrshtmx.SecurityHeadersConfig{}, "X-Frame-Options", "DENY")
	})

	It("sets Referrer-Policy to strict-origin-when-cross-origin", func() {
		assertSecurityHeader(
			cqrshtmx.SecurityHeadersConfig{},
			"Referrer-Policy",
			"strict-origin-when-cross-origin",
		)
	})

	It("preserves existing headers set by downstream handlers", func() {
		middleware := cqrshtmx.SecurityHeadersMiddleware
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
		Expect(w.Header().Get("X-Content-Type-Options")).To(Equal("nosniff"))
	})

	Describe("SecurityHeadersMiddlewareWithConfig", func() {
		It("sets CSP when configured", func() {
			assertSecurityHeader(cqrshtmx.SecurityHeadersConfig{
				ContentSecurityPolicy: "default-src 'self'",
			}, "Content-Security-Policy", "default-src 'self'")
		})

		It("sets HSTS when configured", func() {
			cfg := cqrshtmx.SecurityHeadersConfig{
				StrictTransportSecurity: "max-age=63072000; includeSubDomains",
			}
			middleware := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cfg)
			handler := middleware(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(w.Header().Get("Strict-Transport-Security")).To(
				Equal("max-age=63072000; includeSubDomains"),
			)
		})

		It("sets custom headers when configured", func() {
			assertSecurityHeader(cqrshtmx.SecurityHeadersConfig{
				Custom: map[string]string{"X-Custom-Header": "value"},
			}, "X-Custom-Header", "value")
		})

		It("allows overriding defaults", func() {
			assertSecurityHeader(
				cqrshtmx.SecurityHeadersConfig{FrameOptions: "SAMEORIGIN"},
				"X-Frame-Options",
				"SAMEORIGIN",
			)
		})

		It("allows overriding ContentTypeOptions default", func() {
			assertSecurityHeader(
				cqrshtmx.SecurityHeadersConfig{ContentTypeOptions: "none"},
				"X-Content-Type-Options",
				"none",
			)
		})

		It("allows overriding ReferrerPolicy default", func() {
			assertSecurityHeader(
				cqrshtmx.SecurityHeadersConfig{ReferrerPolicy: "no-referrer"},
				"Referrer-Policy",
				"no-referrer",
			)
		})

		It("sets Permissions-Policy when configured", func() {
			assertSecurityHeader(cqrshtmx.SecurityHeadersConfig{
				PermissionsPolicy: "camera=(), microphone=()",
			}, "Permissions-Policy", "camera=(), microphone=()")
		})
	})
})
