package httperr_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error Mapping", func() {
	DescribeTable(
		"MapError maps CQRS error families to HTTP status codes",
		func(err error, expectedStatus int) {
			Expect(httperr.MapError(err)).To(Equal(expectedStatus))
		},
		Entry("nil error returns 500", nil, http.StatusInternalServerError),
		Entry("Rejection errors to 400", httperr.ErrDecodeFailed, http.StatusBadRequest),
		Entry("nil error to 500", nil, http.StatusInternalServerError),
	)

	expectHXRedirectToLogin := func(w *httptest.ResponseRecorder) {
		Expect(w.Header().Get("Hx-Redirect")).To(Equal("/login"))
	}

	Describe("DefaultErrorHandler", func() {
		DescribeTable(
			"error handler responses",
			func(err error, h func(http.ResponseWriter, *http.Request, error), isHTMX bool, check func(*httptest.ResponseRecorder)) {
				w := httptest.NewRecorder()

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				if isHTMX {
					r.Header.Set("Hx-Request", htmx.HeaderTrue)
				}

				h(w, r, err)
				check(w)
			},
			Entry("plain text error response",
				httperr.ErrDecodeFailed, httperr.DefaultErrorHandler, false,
				func(w *httptest.ResponseRecorder) {
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("failed to decode"))
				}),
			Entry("HX-Redirect for unauthorized HTMX",
				httperr.ErrUnauthorized, httperr.DefaultErrorHandler, true,
				expectHXRedirectToLogin),
			Entry("HX-Redirect for forbidden HTMX",
				httperr.ErrForbidden, httperr.DefaultErrorHandler, true,
				expectHXRedirectToLogin),
			Entry("no redirect for non-auth HTMX errors",
				httperr.ErrDecodeFailed, httperr.DefaultErrorHandler, true,
				func(w *httptest.ResponseRecorder) {
					Expect(w.Header().Get("Hx-Redirect")).To(BeEmpty())
				}),
			Entry("JSON error response",
				httperr.ErrDecodeFailed, httperr.JSONErrorHandler, false,
				func(w *httptest.ResponseRecorder) {
					Expect(
						w.Header().Get("Content-Type"),
					).To(Equal("application/json; charset=utf-8"))
					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("decode"))
					Expect(w.Body.String()).To(ContainSubstring("400"))
				}),
			Entry("JSON handler redirects HTMX auth errors",
				httperr.ErrUnauthorized, httperr.JSONErrorHandler, true,
				func(w *httptest.ResponseRecorder) {
					Expect(w.Code).To(Equal(http.StatusSeeOther))
					Expect(w.Header().Get("Hx-Redirect")).To(Equal("/login"))
				}),
		)

		It("uses custom LoginRedirect when set", func() {
			assertHTMXErrorRedirect(httperr.ErrUnauthorized, "/auth/signin", "/auth/signin")
		})
	})

	Describe("JSON error handler with request ID", func() {
		It("includes request_id when present in context", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			rid := hx.MustParseRequestID("01HK154ANGZHV2ZW0X3SKSNEN2")
			r = r.WithContext(hx.WithRequestID(r.Context(), rid))

			httperr.JSONErrorHandler(w, r, httperr.ErrDecodeFailed)
			Expect(w.Body.String()).To(ContainSubstring("request_id"))
			Expect(w.Body.String()).To(ContainSubstring(rid.String()))
		})

		It("omits request_id when not in context", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			httperr.JSONErrorHandler(w, r, httperr.ErrDecodeFailed)
			Expect(w.Body.String()).NotTo(ContainSubstring("request_id"))
		})

		It("does not include request_id body for HTMX auth errors", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", htmx.HeaderTrue)

			rid := hx.MustParseRequestID("01HK154ANGZHV2ZW0X3SKSNEN2")
			r = r.WithContext(hx.WithRequestID(r.Context(), rid))

			httperr.JSONErrorHandler(w, r, httperr.ErrUnauthorized)
			Expect(w.Code).To(Equal(http.StatusSeeOther))
			Expect(w.Body.String()).To(BeEmpty())
		})
	})

	Describe("Plain text error handler with request ID", func() {
		It("prefixes error with request_id when present", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			rid := hx.MustParseRequestID("01HK154ANGZHV2ZW0X3SKSNEN2")
			r = r.WithContext(hx.WithRequestID(r.Context(), rid))

			httperr.DefaultErrorHandlerWithRequestID(w, r, httperr.ErrDecodeFailed)
			Expect(w.Body.String()).To(ContainSubstring("[request_id: " + rid.String() + "]"))
			Expect(w.Body.String()).To(ContainSubstring("failed to decode"))
		})

		It("falls back to plain error when request_id absent", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			httperr.DefaultErrorHandlerWithRequestID(w, r, httperr.ErrDecodeFailed)
			Expect(w.Body.String()).To(ContainSubstring("failed to decode request body"))
			Expect(w.Body.String()).NotTo(ContainSubstring("request_id"))
		})

		It("uses custom redirect for HTMX auth errors", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", htmx.HeaderTrue)
			httperr.DefaultErrorHandlerWithRedirectAndRequestID(w, r, httperr.ErrUnauthorized, "/auth/signin")
			Expect(w.Header().Get("Hx-Redirect")).To(Equal("/auth/signin"))
		})
	})

	Describe("Sentinel errors", func() {
		It("has distinct sentinel errors", func() {
			Expect(httperr.ErrUnauthorized).NotTo(Equal(httperr.ErrForbidden))
			Expect(httperr.ErrDecodeFailed).NotTo(Equal(httperr.ErrDispatchFailed))
			Expect(httperr.ErrEnforcerNil).NotTo(Equal(httperr.ErrValidationFailed))
		})
	})

	Describe("Retry-After header", func() {
		DescribeTable(
			"error handlers send Retry-After on 503 responses",
			func(h func(http.ResponseWriter, *http.Request, error)) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				h(w, r, httperr.ErrDispatchFailed)

				Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
				Expect(w.Header().Get("Retry-After")).To(Equal("1"))
			},
			Entry("plain text handler", httperr.DefaultErrorHandler),
			Entry("JSON handler", httperr.JSONErrorHandler),
			Entry("problem details handler", httperr.ProblemDetailsErrorHandler),
		)

		It("honours a RetryAfterCarrier hint over the default", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			httperr.JSONErrorHandler(w, r, httperr.WithRetryAfter(httperr.ErrDispatchFailed, 30*time.Second))

			Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
			Expect(w.Header().Get("Retry-After")).To(Equal("30"))
		})

		It("rounds sub-second hints up to one second", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			httperr.JSONErrorHandler(w, r, httperr.WithRetryAfter(httperr.ErrDispatchFailed, 300*time.Millisecond))

			Expect(w.Header().Get("Retry-After")).To(Equal("1"))
		})

		It("falls back to the default for non-positive carrier hints", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			httperr.JSONErrorHandler(w, r, httperr.WithRetryAfter(httperr.ErrDispatchFailed, 0))

			Expect(w.Header().Get("Retry-After")).To(Equal("1"))
		})

		It("does not send Retry-After on client errors", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			httperr.JSONErrorHandler(w, r, httperr.ErrDecodeFailed)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Header().Get("Retry-After")).To(BeEmpty())
		})

		It("preserves status and sentinel identity when wrapping", func() {
			wrapped := httperr.WithRetryAfter(httperr.ErrDispatchFailed, 5*time.Second)

			Expect(errors.Is(wrapped, httperr.ErrDispatchFailed)).To(BeTrue())
			Expect(httperr.MapError(wrapped)).To(Equal(http.StatusServiceUnavailable))
		})

		It("returns nil when wrapping nil", func() {
			Expect(httperr.WithRetryAfter(nil, time.Second)).To(Succeed())
		})
	})
})

// assertHTMXErrorRedirect drives the HTMX auth-redirect path and asserts the
// HX-Redirect target. It is the httperr-local copy of the helper root's
// middleware suite uses: the two suites live in different packages, and a test
// helper is not importable across packages.
func assertHTMXErrorRedirect(err error, loginRedirect, expectedRedirect string) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(htmx.HeaderRequest, htmx.HeaderTrue)

	httperr.DefaultErrorHandlerWithRedirect(w, r, err, loginRedirect)

	ExpectWithOffset(1, w.Header().Get(htmx.HeaderRedirect)).To(Equal(expectedRedirect))
}
