package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/larsartmann/cqrs-htmx"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Middleware", func() {
	Describe("ContextEnrichmentMiddleware", func() {
		var (
			nextCalled bool
			userID     string
		)

		BeforeEach(func() {
			nextCalled = false
			userID = ""
		})

		It("enriches context with user ID from extractor", func() {
			extractor := func(r *http.Request) string { return "user-42" }
			middleware := cqrshtmx.ContextEnrichmentMiddleware(extractor)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				userID = cqrshtmx.UserIDFromContext(r.Context())
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(nextCalled).To(BeTrue())
			Expect(userID).To(Equal("user-42"))
		})

		It("does not set user ID when extractor returns empty", func() {
			extractor := func(r *http.Request) string { return "" }
			middleware := cqrshtmx.ContextEnrichmentMiddleware(extractor)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				userID = cqrshtmx.UserIDFromContext(r.Context())
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(nextCalled).To(BeTrue())
			Expect(userID).To(BeEmpty())
		})

		It("handles nil extractor gracefully", func() {
			middleware := cqrshtmx.ContextEnrichmentMiddleware(nil)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				userID = cqrshtmx.UserIDFromContext(r.Context())
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(nextCalled).To(BeTrue())
			Expect(userID).To(BeEmpty())
		})
	})

	Describe("Chain", func() {
		It("applies middleware left-to-right", func() {
			var order []string

			mw1 := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					order = append(order, "mw1-before")
					next.ServeHTTP(w, r)
					order = append(order, "mw1-after")
				})
			}

			mw2 := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					order = append(order, "mw2-before")
					next.ServeHTTP(w, r)
					order = append(order, "mw2-after")
				})
			}

			final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "handler")
			})

			chained := cqrshtmx.Chain(mw1, mw2)(final)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			chained.ServeHTTP(w, r)

			Expect(order).To(Equal([]string{
				"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after",
			}))
		})
	})
})
