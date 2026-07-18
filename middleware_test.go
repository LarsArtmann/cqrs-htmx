package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Middleware", func() {
	Describe("ContextEnrichmentMiddleware", func() {
		var (
			nextCalled bool
			userID     cqrshtmx.UserID
		)

		BeforeEach(func() {
			nextCalled = false
			userID = cqrshtmx.UserID{}
		})

		assertMiddleware := func(
			extractor cqrshtmx.UserIDExtractor,
			assertions func(cqrshtmx.UserID),
		) {
			middleware := cqrshtmx.ContextEnrichmentMiddleware(extractor)
			handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				nextCalled = true
				userID = cqrshtmx.UserIDFromContext(r.Context())
			}))
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(nextCalled).To(BeTrue())
			assertions(userID)
		}

		It("enriches context with user ID from extractor", func() {
			want := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			assertMiddleware(staticExtractor(want), func(got cqrshtmx.UserID) {
				Expect(got).To(Equal(want))
			})
		})

		It("does not set user ID when extractor returns empty or zero value", func() {
			assertMiddleware(
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil },
				func(got cqrshtmx.UserID) { Expect(got).To(BeZero()) },
			)
		})

		It("handles nil extractor gracefully", func() {
			assertMiddleware(nil, func(got cqrshtmx.UserID) { Expect(got).To(BeZero()) })
		})

		DescribeTable(
			"RequestID handling",
			func(setupRequest func(*http.Request), assertRequestID func(cqrshtmx.RequestID)) {
				middleware := cqrshtmx.ContextEnrichmentMiddleware(nil)

				var captured cqrshtmx.RequestID

				handler := middleware(
					http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
						captured = cqrshtmx.RequestIDFromContext(r.Context())
					}),
				)
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				setupRequest(r)
				handler.ServeHTTP(w, r)
				assertRequestID(captured)
			},
			Entry("auto-generates when no header",
				func(_ *http.Request) {},
				func(rid cqrshtmx.RequestID) { Expect(rid.IsZero()).To(BeFalse()) }),
			Entry("extracts from X-Request-ID header",
				func(r *http.Request) {
					r.Header.Set(
						"X-Request-ID",
						cqrshtmx.MustParseRequestID("01HK1549P84T9XF8R94E960633").String(),
					)
				},
				func(rid cqrshtmx.RequestID) {
					Expect(rid).To(Equal(cqrshtmx.MustParseRequestID("01HK1549P84T9XF8R94E960633")))
				}),
			Entry("drops unparseable and generates new",
				func(r *http.Request) { r.Header.Set("X-Request-ID", "invalid-id") },
				func(rid cqrshtmx.RequestID) { Expect(rid.IsZero()).To(BeFalse()) }),
		)
	})

	Describe("Chain", func() {
		It("applies middleware left-to-right", func() {
			var order []string

			tracingMW := func(name string) func(http.Handler) http.Handler {
				return func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						order = append(order, name+"-before")

						next.ServeHTTP(w, r)

						order = append(order, name+"-after")
					})
				}
			}

			final := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				order = append(order, "handler")
			})

			chained := cqrshtmx.Chain(tracingMW("mw1"), tracingMW("mw2"))(final)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			chained.ServeHTTP(w, r)

			Expect(order).To(Equal([]string{
				"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after",
			}))
		})
	})
})
