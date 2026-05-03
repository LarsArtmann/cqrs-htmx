package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/larsartmann/cqrs-htmx"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error Mapping", func() {
	DescribeTable("MapError maps CQRS error families to HTTP status codes",
		func(err error, expectedStatus int) {
			Expect(cqrshtmx.MapError(err)).To(Equal(expectedStatus))
		},
		Entry("nil error returns 500", nil, http.StatusInternalServerError),
	)

	Describe("MapError with CQRS event errors", func() {
		It("maps Rejection errors to 400", func() {
			err := cqrshtmx.ErrDecodeFailed
			Expect(cqrshtmx.MapError(err)).To(Equal(http.StatusBadRequest))
		})

		It("maps nil error to 500", func() {
			Expect(cqrshtmx.MapError(nil)).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("DefaultErrorHandler", func() {
		It("writes a plain text error response", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.DefaultErrorHandler(w, r, cqrshtmx.ErrDecodeFailed)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("failed to decode"))
		})

		It("sets HX-Redirect for unauthorized HTMX requests", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			cqrshtmx.DefaultErrorHandler(w, r, cqrshtmx.ErrUnauthorized)
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})

		It("sets HX-Redirect for forbidden HTMX requests", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			cqrshtmx.DefaultErrorHandler(w, r, cqrshtmx.ErrForbidden)
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})

		It("does not redirect for non-auth HTMX errors", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			cqrshtmx.DefaultErrorHandler(w, r, cqrshtmx.ErrDecodeFailed)
			Expect(w.Header().Get("HX-Redirect")).To(BeEmpty())
		})
	})

	Describe("Sentinel errors", func() {
		It("has distinct sentinel errors", func() {
			Expect(cqrshtmx.ErrUnauthorized).NotTo(Equal(cqrshtmx.ErrForbidden))
			Expect(cqrshtmx.ErrDecodeFailed).NotTo(Equal(cqrshtmx.ErrDispatchFailed))
			Expect(cqrshtmx.ErrEnforcerNil).NotTo(Equal(cqrshtmx.ErrCommandsNil))
		})
	})
})
