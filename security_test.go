package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Security Headers Middleware", func() {
	It("sets X-Content-Type-Options to nosniff", func() {
		middleware := cqrshtmx.SecurityHeadersMiddleware
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		Expect(w.Header().Get("X-Content-Type-Options")).To(Equal("nosniff"))
	})

	It("sets X-Frame-Options to DENY", func() {
		middleware := cqrshtmx.SecurityHeadersMiddleware
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		Expect(w.Header().Get("X-Frame-Options")).To(Equal("DENY"))
	})

	It("sets Referrer-Policy to strict-origin-when-cross-origin", func() {
		middleware := cqrshtmx.SecurityHeadersMiddleware
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		Expect(w.Header().Get("Referrer-Policy")).To(Equal("strict-origin-when-cross-origin"))
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
})
