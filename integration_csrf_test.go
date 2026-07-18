package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration: CQRS + CSRF Protection", func() {
	Describe("End-to-end CQRS + HTMX + CSRF protection", func() {
		var app *cqrshtmx.App

		BeforeEach(func() { app = newCommandApp() })

		It("allows command dispatch with valid CSRF token via HTMX header", func() {
			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(app.Command(
				"CreateUser",
				decodeBDDCreateUserJSONWithBody(),
				cqrshtmx.NotifySuccess("User created"),
			))

			// GET to obtain token
			var csrfToken string

			tokenHandler := csrfTokenHandler(csrfMW, &csrfToken)
			w1 := httptest.NewRecorder()
			tokenHandler.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/", nil))
			cookies := w1.Result().Cookies()

			Expect(csrfToken).NotTo(BeEmpty())

			r := newPostRequest(
				"/users",
				testUserJSON,
				withHTMX,
				withHeader("X-CSRF-Token", csrfToken),
			)
			for _, c := range cookies {
				r.AddCookie(c)
			}

			w := serve(handler, r)
			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Hx-Trigger")).To(ContainSubstring("success"))
		})

		It("rejects command dispatch without CSRF token", func() {
			handler := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())(
				app.Command("CreateUser", decodeBDDCreateUserJSONWithBody()),
			)
			w := serve(handler, newPostRequest("/users", testUserJSON, withHTMX))
			Expect(w.code()).To(Equal(http.StatusForbidden))
		})

		It("rejects command dispatch with invalid CSRF token", func() {
			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(app.Command("CreateUser", decodeBDDCreateUserJSONWithBody()))

			w1 := httptest.NewRecorder()
			handler.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/", nil))

			r := newPostRequest(
				"/users",
				testUserJSON,
				withHTMX,
				withHeader("X-CSRF-Token", "wrong-token"),
			)
			for _, c := range w1.Result().Cookies() {
				r.AddCookie(c)
			}

			w := serve(handler, r)
			Expect(w.code()).To(Equal(http.StatusForbidden))
		})

		It("allows GET queries without CSRF token", func() {
			qryDisp := query.NewDispatcher()
			registerBDDListUsers(qryDisp)
			qryApp, err := cqrshtmx.New(cqrshtmx.Config{Queries: qryDisp})
			Expect(err).NotTo(HaveOccurred())

			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			// First GET to set the CSRF cookie
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/users", nil)
			csrfOKHandler(csrfMW).ServeHTTP(w1, r1)

			// Second GET with the cookie
			r2 := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader("{}"))
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}

			handler := csrfMW(qryApp.Query(
				"ListUsers",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r2)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
		})

		It("sets CSRF token in response header for frontend consumption", func() {
			qryDisp := query.NewDispatcher()
			_ = qryDisp.Register("GetPage", func(_ context.Context, _ query.Query) (any, error) {
				return "page data", nil
			})
			qryApp, err := cqrshtmx.New(cqrshtmx.Config{Queries: qryDisp})
			Expect(err).NotTo(HaveOccurred())

			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(qryApp.Query(
				"GetPage",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &getPageQuery{}, nil
				}),
				cqrshtmx.Render(func(w http.ResponseWriter, r *http.Request, _ any) error {
					token := cqrshtmx.CSRFTokenFromContext(r.Context())
					cqrshtmx.NewResponse(w, r).CSRFToken(token).Apply()
					_, _ = w.Write([]byte("<div>page</div>"))

					return nil
				}),
			))
			// GET to set the CSRF cookie first
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/page", nil)
			r1.Header.Set("Content-Type", "application/json")
			r1.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
			csrfOKHandler(csrfMW).ServeHTTP(w1, r1)

			r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader("{}"))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)

			for _, c := range w1.Result().Cookies() {
				r.AddCookie(c)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("X-Csrf-Token")).NotTo(BeEmpty())
		})
	})

	Describe("End-to-end CQRS + CSRFProtect per-handler option", func() {
		var app *cqrshtmx.App

		BeforeEach(func() { app = newCommandApp() })

		It("allows command dispatch with CSRFProtect and valid token", func() {
			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(app.Command(
				"CreateUser",
				cqrshtmx.CSRFProtect(integrationCSRFConfig()),
				decodeBDDCreateUserJSONWithBody(),
			))

			// GET to obtain token from the same CSRF middleware instance
			var csrfToken string

			tokenHandler := csrfTokenHandler(csrfMW, &csrfToken)
			w1 := httptest.NewRecorder()
			tokenHandler.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/", nil))
			cookies := w1.Result().Cookies()

			Expect(csrfToken).NotTo(BeEmpty())

			r := newPostRequest("/users", testUserJSON, withHeader("X-CSRF-Token", csrfToken))
			for _, c := range cookies {
				r.AddCookie(c)
			}

			w := serve(handler, r)
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})

		It("rejects command dispatch with CSRFProtect but no token", func() {
			handler := app.Command(
				"CreateUser",
				cqrshtmx.CSRFProtect(integrationCSRFConfig()),
				decodeBDDCreateUserJSONWithBody(),
			)
			w := serve(handler, newPostRequest("/users", testUserJSON))
			Expect(w.code()).To(Equal(http.StatusForbidden))
		})

		It("rejects command dispatch with CSRFProtect and invalid token", func() {
			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(app.Command(
				"CreateUser",
				cqrshtmx.CSRFProtect(integrationCSRFConfig()),
				decodeBDDCreateUserJSONWithBody(),
			))

			w1 := httptest.NewRecorder()
			handler.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/", nil))

			r := newPostRequest(
				"/users",
				testUserJSON,
				withHTMX,
				withHeader("X-CSRF-Token", "wrong-token"),
			)
			for _, c := range w1.Result().Cookies() {
				r.AddCookie(c)
			}

			w := serve(handler, r)
			Expect(w.code()).To(Equal(http.StatusForbidden))
		})
	})
})
