package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/casbin/casbin/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testUserJSON = `{"email":"alice@example.com","name":"Alice"}`
	emailKey     = "email"
)

func integrationCSRFConfig() cqrshtmx.CSRFConfig {
	return cqrshtmx.CSRFConfig{
		MaxAge:   24 * time.Hour,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusForbidden)
		},
	}
}

func newIntegrationApp(
	disp *command.Dispatcher,
	enf *casbin.Enforcer,
) (*cqrshtmx.App, *command.Dispatcher) {
	_ = disp.Register("CreateUser", noOpCommandHandler)
	_ = disp.Register("DeleteUser", rejectionHandler("user.not_found", "user does not exist"))
	cfg := cqrshtmx.Config{
		Commands:        disp,
		Enforcer:        enf,
		UserIDExtractor: headerExtractor("X-User"),
	}
	app, err := cqrshtmx.New(cfg)
	Expect(err).NotTo(HaveOccurred())
	return app, disp
}

var _ = Describe("Full Integration", func() {
	Describe("End-to-end CQRS + HTMX + Casbin flow", func() {
		var (
			app    *cqrshtmx.App
			enf    *casbin.Enforcer
			userID id.AggregateID
		)

		BeforeEach(func() {
			enf = newTestEnforcer()
			userID = id.NewAggregateID()
			app, _ = newIntegrationApp(command.NewDispatcher(), enf)
		})

		It("allows admin to create user with full HTMX response", func() {
			handler := app.Command(
				"CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithBodyAndAggID(userID),
				cqrshtmx.Trigger("userCreated"),
				cqrshtmx.PushURL("/users/"+userID.String()),
			)

			r := newPostRequest(
				"/users", `{"email":"admin@co.com","name":"Admin"}`,
				withUserHeader("X-User", adminUserID),
				withHTMX,
			)
			w := serve(handler, r)
			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
			Expect(w.Header().Get("HX-Push-Url")).To(ContainSubstring("/users/"))
		})

		It("denies viewer from creating user", func() {
			handler := app.Command(
				"CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithAggID(userID),
			)
			assertStatusCode(
				handler,
				newPostRequest("/users", `{}`, withUserHeader("X-User", viewerUserID)),
				http.StatusForbidden,
			)
		})

		It("redirects unauthenticated HTMX users to login", func() {
			handler := app.Command(
				"CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithAggID(userID),
			)
			r := newPostRequest("/users", `{}`, withHTMX)
			w := serve(handler, r)
			Expect(w.code()).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})

		It("maps CQRS rejection errors to 400", func() {
			handler := app.Command(
				"DeleteUser",
				cqrshtmx.RequireAuth(),
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &bddDeleteUserCmd{aggID: userID}, nil
				}),
			)
			assertStatusCode(
				handler,
				newPostRequest("/users/delete", `{}`, withUserHeader("X-User", adminUserID)),
				http.StatusBadRequest,
			)
		})
	})

	Describe("Query with render", func() {
		var app *cqrshtmx.App

		BeforeEach(func() {
			disp := query.NewDispatcher()
			_ = disp.Register("ListUsers", func(_ context.Context, _ query.Query) (any, error) {
				return []map[string]string{
					{emailKey: "a@b.com", testNameKey: aliceName},
					{emailKey: "c@d.com", testNameKey: "Carol"},
				}, nil
			})
			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())
		})

		It("renders query results as JSON", func() {
			handler := app.Query(
				"ListUsers",
				cqrshtmx.DecodeJSONQuery(func(_ bddListUsersQuery) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			)
			w := serve(handler, newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
		})
	})

	Describe("Command with redirect", func() {
		It("redirects to URL after command success", func() {
			app := newCommandApp()
			handler := app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Redirect("/users"),
			)
			w := serve(handler, newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("Location")).To(Equal("/users"))
		})
	})

	Describe("TriggerWithDetail handler option", func() {
		It("sets HX-Trigger with detail data", func() {
			app := newCommandApp()
			handler := app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.TriggerWithDetail("userCreated", map[string]string{"id": "123"}),
			)
			r := newPostRequest("/users", `{}`, withHTMX)
			w := serve(handler, r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("userCreated"))
			Expect(trigger).To(ContainSubstring("123"))
		})
	})

	Describe("App.Middleware integration", func() {
		It("propagates user ID from middleware to handler", func() {
			want := adminUserID
			disp := command.NewDispatcher()
			var receivedUserID cqrshtmx.UserID
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				receivedUserID = cqrshtmx.UserIDFromContext(ctx)
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: headerExtractor("X-User-ID"),
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Middleware()(app.Command("CreateUser", decodeCreateUserJSON()))
			r := newPostRequest("/users", `{}`, withUserHeader("X-User-ID", want))
			serve(handler, r)
			Expect(receivedUserID).To(Equal(want))
		})
	})

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
			tokenHandler := csrfMW(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				csrfToken = cqrshtmx.CSRFTokenFromContext(r.Context())
			}))
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
			Expect(w.Header().Get("HX-Trigger")).To(ContainSubstring("success"))
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
			_ = qryDisp.Register("ListUsers", func(_ context.Context, _ query.Query) (any, error) {
				return []bddUser{{Email: aliceEmail, Name: aliceName}}, nil
			})
			qryApp, err := cqrshtmx.New(cqrshtmx.Config{Queries: qryDisp})
			Expect(err).NotTo(HaveOccurred())

			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			// First GET to set the CSRF cookie
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/users", nil)
			csrfMW(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(w1, r1)

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
			r1.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			csrfMW(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(w1, r1)

			r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader("{}"))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			for _, c := range w1.Result().Cookies() {
				r.AddCookie(c)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("X-CSRF-Token")).NotTo(BeEmpty())
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
			tokenHandler := csrfMW(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				csrfToken = cqrshtmx.CSRFTokenFromContext(r.Context())
			}))
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
			tokenHandler := csrfMW(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				csrfToken = cqrshtmx.CSRFTokenFromContext(r.Context())
			}))
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
