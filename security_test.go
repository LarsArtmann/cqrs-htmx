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

	DescribeTable(
		"default security headers",
		func(header, expected string) {
			assertSecurityHeader(cqrshtmx.SecurityHeadersConfig{}, header, expected)
		},
		Entry("X-Content-Type-Options", "X-Content-Type-Options", "nosniff"),
		Entry("X-Frame-Options", "X-Frame-Options", "DENY"),
		Entry("Referrer-Policy", "Referrer-Policy", "strict-origin-when-cross-origin"),
	)

	DescribeTable(
		"configurable security headers",
		func(cfg cqrshtmx.SecurityHeadersConfig, header, expected string) {
			assertSecurityHeader(cfg, header, expected)
		},
		Entry(
			"CSP",
			cqrshtmx.SecurityHeadersConfig{ContentSecurityPolicy: "default-src 'self'"},
			"Content-Security-Policy",
			"default-src 'self'",
		),
		Entry(
			"custom headers",
			cqrshtmx.SecurityHeadersConfig{Custom: map[string]string{"X-Custom-Header": "value"}},
			"X-Custom-Header",
			"value",
		),
		Entry(
			"override FrameOptions",
			cqrshtmx.SecurityHeadersConfig{FrameOptions: "SAMEORIGIN"},
			"X-Frame-Options",
			"SAMEORIGIN",
		),
		Entry(
			"override ContentTypeOptions",
			cqrshtmx.SecurityHeadersConfig{ContentTypeOptions: "none"},
			"X-Content-Type-Options",
			"none",
		),
		Entry(
			"override ReferrerPolicy",
			cqrshtmx.SecurityHeadersConfig{ReferrerPolicy: "no-referrer"},
			"Referrer-Policy",
			"no-referrer",
		),
		Entry(
			"Permissions-Policy",
			cqrshtmx.SecurityHeadersConfig{PermissionsPolicy: "camera=(), microphone=()"},
			"Permissions-Policy",
			"camera=(), microphone=()",
		),
	)

	It("sets HSTS when configured", func() {
		assertSecurityHeader(cqrshtmx.SecurityHeadersConfig{
			StrictTransportSecurity: "max-age=63072000; includeSubDomains",
		}, "Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	})

	It("preserves existing headers set by downstream handlers", func() {
		handler := cqrshtmx.SecurityHeadersMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
			}),
		)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
		Expect(w.Header().Get("X-Content-Type-Options")).To(Equal("nosniff"))
	})
})
