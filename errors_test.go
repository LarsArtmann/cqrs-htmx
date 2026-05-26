package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error Mapping", func() {
	DescribeTable(
		"MapError maps CQRS error families to HTTP status codes",
		func(err error, expectedStatus int) {
			Expect(cqrshtmx.MapError(err)).To(Equal(expectedStatus))
		},
		Entry("nil error returns 500", nil, http.StatusInternalServerError),
		Entry("Rejection errors to 400", cqrshtmx.ErrDecodeFailed, http.StatusBadRequest),
		Entry("nil error to 500", nil, http.StatusInternalServerError),
	)

	Describe("DefaultErrorHandler", func() {
		DescribeTable(
			"error handler responses",
			func(err error, h func(http.ResponseWriter, *http.Request, error), isHTMX bool, check func(*httptest.ResponseRecorder)) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				if isHTMX {
					r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
				}
				h(w, r, err)
				check(w)
			},
			Entry("plain text error response",
				cqrshtmx.ErrDecodeFailed, cqrshtmx.DefaultErrorHandler, false,
				func(w *httptest.ResponseRecorder) {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("failed to decode"))
				}),
			Entry("HX-Redirect for unauthorized HTMX",
				cqrshtmx.ErrUnauthorized, cqrshtmx.DefaultErrorHandler, true,
				func(w *httptest.ResponseRecorder) {
					Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
				}),
			Entry("HX-Redirect for forbidden HTMX",
				cqrshtmx.ErrForbidden, cqrshtmx.DefaultErrorHandler, true,
				func(w *httptest.ResponseRecorder) {
					Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
				}),
			Entry("no redirect for non-auth HTMX errors",
				cqrshtmx.ErrDecodeFailed, cqrshtmx.DefaultErrorHandler, true,
				func(w *httptest.ResponseRecorder) {
					Expect(w.Header().Get("HX-Redirect")).To(BeEmpty())
				}),
			Entry("JSON error response",
				cqrshtmx.ErrDecodeFailed, cqrshtmx.JSONErrorHandler, false,
				func(w *httptest.ResponseRecorder) {
					Expect(
						w.Header().Get("Content-Type"),
					).To(Equal("application/json; charset=utf-8"))
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("decode"))
					Expect(w.Body.String()).To(ContainSubstring("400"))
				}),
			Entry("JSON handler redirects HTMX auth errors",
				cqrshtmx.ErrUnauthorized, cqrshtmx.JSONErrorHandler, true,
				func(w *httptest.ResponseRecorder) {
					Expect(w.Code).To(Equal(http.StatusSeeOther))
					Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
				}),
		)

		It("uses custom LoginRedirect when set", func() {
			assertHTMXErrorRedirect(cqrshtmx.ErrUnauthorized, "/auth/signin", "/auth/signin")
		})
	})

	Describe("JSON error handler with request ID", func() {
		It("includes request_id when present in context", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			rid := cqrshtmx.MustParseRequestID("01HK154ANGZHV2ZW0X3SKSNEN2")
			r = r.WithContext(cqrshtmx.WithRequestID(r.Context(), rid))

			cqrshtmx.JSONErrorHandler(w, r, cqrshtmx.ErrDecodeFailed)
			Expect(w.Body.String()).To(ContainSubstring("request_id"))
			Expect(w.Body.String()).To(ContainSubstring(rid.String()))
		})

		It("omits request_id when not in context", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			cqrshtmx.JSONErrorHandler(w, r, cqrshtmx.ErrDecodeFailed)
			Expect(w.Body.String()).NotTo(ContainSubstring("request_id"))
		})

		It("does not include request_id body for HTMX auth errors", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			rid := cqrshtmx.MustParseRequestID("01HK154ANGZHV2ZW0X3SKSNEN2")
			r = r.WithContext(cqrshtmx.WithRequestID(r.Context(), rid))

			cqrshtmx.JSONErrorHandler(w, r, cqrshtmx.ErrUnauthorized)
			Expect(w.Code).To(Equal(http.StatusSeeOther))
			Expect(w.Body.String()).To(BeEmpty())
		})
	})

	Describe("Sentinel errors", func() {
		It("has distinct sentinel errors", func() {
			Expect(cqrshtmx.ErrUnauthorized).NotTo(Equal(cqrshtmx.ErrForbidden))
			Expect(cqrshtmx.ErrDecodeFailed).NotTo(Equal(cqrshtmx.ErrDispatchFailed))
			Expect(cqrshtmx.ErrEnforcerNil).NotTo(Equal(cqrshtmx.ErrValidationFailed))
		})
	})
})
