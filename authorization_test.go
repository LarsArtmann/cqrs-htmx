package cqrshtmx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authorization", func() {
	Describe("Enforce", func() {
		It("allows permitted actions", func() {
			e := newTestEnforcer()
			Expect(
				cqrshtmx.Enforce(e, adminUserID.String(), "users", "create"),
			).NotTo(HaveOccurred())
		})

		It("denies non-permitted actions", func() {
			e := newTestEnforcer()
			err := cqrshtmx.Enforce(e, viewerUserID.String(), "users", "create")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, cqrshtmx.ErrForbidden)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring(
				"subject=" + viewerUserID.String() + " resource=users action=create",
			))
		})

		It("returns error for nil enforcer", func() {
			err := cqrshtmx.Enforce(nil, "admin", "users", "create")
			Expect(errors.Is(err, cqrshtmx.ErrEnforcerNil)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("subject=admin resource=users action=create"))
		})
	})

	Describe("AuthorizeMiddleware", func() {
		It("allows authorized requests through", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(
				e,
				"users",
				"read",
				staticExtractor(adminUserID),
			)
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			w := serve(handler, httptest.NewRequest(http.MethodGet, "/", nil))

			Expect(called).To(BeTrue())
			Expect(w.code()).To(Equal(http.StatusOK))
		})

		It("blocks unauthorized requests", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(
				e,
				"users",
				"create",
				staticExtractor(viewerUserID),
			)
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			w := serve(handler, httptest.NewRequest(http.MethodGet, "/", nil))

			Expect(called).To(BeFalse())
			Expect(w.code()).To(Equal(http.StatusForbidden))
		})

		It("blocks unauthenticated requests", func() {
			middleware := unauthenticatedReadMiddleware()
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			w := serve(handler, httptest.NewRequest(http.MethodGet, "/", nil))

			Expect(called).To(BeFalse())
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})

		It("redirects unauthenticated HTMX requests to login", func() {
			middleware := unauthenticatedReadMiddleware()
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			w := serve(handler, r)

			Expect(called).To(BeFalse())
			Expect(w.code()).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("Hx-Redirect")).To(Equal("/login"))
		})

		It("uses custom login redirect", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(
				e, "users", "read",
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil },
				"/auth/signin",
			)
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			w := serve(handler, r)

			Expect(called).To(BeFalse())
			Expect(w.Header().Get("Hx-Redirect")).To(Equal("/auth/signin"))
		})

		It("prefers branded UserID from context over extractor", func() {
			middleware := unauthenticatedReadMiddleware()
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = r.WithContext(cqrshtmx.WithUserID(r.Context(), adminUserID))
			serve(handler, r)
			Expect(called).To(BeTrue())
		})

		It("rejects unparseable user IDs with 401", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) (cqrshtmx.UserID, error) {
					return cqrshtmx.UserID{}, errors.New("invalid user id")
				})
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			w := serve(handler, httptest.NewRequest(http.MethodGet, "/", nil))

			Expect(called).To(BeFalse())
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})

		It("rejects requests when extractor is nil and no context user ID", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read", nil)
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			w := serve(handler, httptest.NewRequest(http.MethodGet, "/", nil))

			Expect(called).To(BeFalse())
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})
	})
})
