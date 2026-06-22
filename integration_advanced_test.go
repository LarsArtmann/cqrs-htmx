package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration: Body Size and Middleware", func() {
	Describe("Body size limits", func() {
		DescribeTable(
			"MaxBodySize",
			func(maxBodySize int64, body string, expectedCode int) {
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", noOpCommandHandler)
				app, err := cqrshtmx.New(cqrshtmx.Config{
					Commands:    disp,
					MaxBodySize: maxBodySize,
				})
				Expect(err).NotTo(HaveOccurred())

				handler := app.Command("CreateUser", decodeBDDCreateUserJSONWithBody())
				assertStatusCode(handler, newPostRequest("/users", body), expectedCode)
			},
			Entry("rejects body larger than limit", int64(10), testUserJSON, http.StatusRequestEntityTooLarge),
			Entry("allows body within limit", int64(1024), testUserJSON, http.StatusNoContent),
		)
	})

	Describe("Full middleware chain", func() {
		It("dispatches command through complete middleware stack", func() {
			disp := command.NewDispatcher()
			var receivedUserID cqrshtmx.UserID
			var htmxFromContext *cqrshtmx.HTMXRequest
			var requestID cqrshtmx.RequestID
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				receivedUserID = cqrshtmx.UserIDFromContext(ctx)
				htmxFromContext = cqrshtmx.HTMXFromContext(ctx)
				requestID = cqrshtmx.RequestIDFromContext(ctx)
				return nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: headerExtractor("X-User"),
			})
			Expect(err).NotTo(HaveOccurred())

			csrfCfg := integrationCSRFConfig()
			csrfMW := cqrshtmx.CSRFMiddleware(csrfCfg)
			handler := app.Command("CreateUser", decodeCreateUserJSON())

			chain := cqrshtmx.Chain(
				cqrshtmx.SecurityHeadersMiddleware,
				cqrshtmx.RecoveryMiddleware,
				csrfMW,
				cqrshtmx.HTMXMiddleware,
				app.Middleware(),
			)(handler)

			// Acquire CSRF token via the same CSRF middleware instance.
			var csrfToken string
			tokenHandler := csrfTokenHandler(csrfMW, &csrfToken)
			w1 := httptest.NewRecorder()
			tokenHandler.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/", nil))
			cookies := w1.Result().Cookies()
			Expect(csrfToken).NotTo(BeEmpty())

			// POST through the full chain with HTMX and user ID.
			r := newPostRequest(
				"/users",
				`{}`,
				withHTMX,
				withUserHeader("X-User", adminUserID),
				withHeader("X-CSRF-Token", csrfToken),
			)
			for _, c := range cookies {
				r.AddCookie(c)
			}

			w := httptest.NewRecorder()
			chain.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(receivedUserID).To(Equal(adminUserID))
			Expect(htmxFromContext).NotTo(BeNil())
			Expect(htmxFromContext.IsHTMX).To(BeTrue())
			Expect(requestID.IsZero()).To(BeFalse())
			Expect(w.Header().Get("X-Frame-Options")).To(Equal("DENY"))
		})

		It("recovers from panic in handler through full chain", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				panic("intentional panic")
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())
			chain := cqrshtmx.Chain(
				cqrshtmx.SecurityHeadersMiddleware,
				cqrshtmx.RecoveryMiddleware,
				cqrshtmx.HTMXMiddleware,
				app.Middleware(),
			)(handler)

			r := newPostRequest("/users", `{}`)
			w := httptest.NewRecorder()
			chain.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Header().Get("X-Frame-Options")).To(Equal("DENY"))
		})
	})
})
