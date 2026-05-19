package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/casbin/casbin/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testUserJSON = `{"email":"alice@example.com","name":"Alice"}`
	emailKey     = "email"
)

type getPageQuery struct{}

func (q *getPageQuery) Type() query.Type { return "GetPage" }

func integrationCSRFConfig() cqrshtmx.CSRFConfig {
	return cqrshtmx.CSRFConfig{
		Secret:         nil,
		CookieName:     "",
		HeaderName:     "",
		FieldName:      "",
		MaxAge:         24 * time.Hour,
		Secure:         false,
		SameSite:       http.SameSiteLaxMode,
		Domain:         "",
		Path:           "/",
		TrustedOrigins: nil,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusForbidden)
		},
	}
}

var _ = Describe("Full Integration", func() {
	Describe("End-to-end CQRS + HTMX + Casbin flow", func() {
		var (
			app    *cqrshtmx.App
			disp   *command.Dispatcher
			enf    *casbin.Enforcer
			userID id.AggregateID
		)

		BeforeEach(func() {
			enf = newTestEnforcer()
			disp = command.NewDispatcher()
			userID = id.NewAggregateID()

			_ = disp.Register("CreateUser", noOpCommandHandler)
			_ = disp.Register("DeleteUser", rejectionHandler(
				"user.not_found", "user does not exist",
			))

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				Enforcer:        enf,
				UserIDExtractor: func(r *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.ParseUserID(r.Header.Get("X-User")) },
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows admin to create user with full HTMX response", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithBodyAndAggID(userID),
				cqrshtmx.Trigger("userCreated"),
				cqrshtmx.PushURL("/users/"+userID.String()),
			)

			body := `{"email":"admin@co.com","name":"Admin"}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User", cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633").String())
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
			Expect(w.Header().Get("HX-Push-Url")).To(ContainSubstring("/users/"))
		})

		It("denies viewer from creating user", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithAggID(userID),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User", cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2").String())
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("redirects unauthenticated HTMX users to login", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithAggID(userID),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})

		It("maps CQRS rejection errors to 400", func() {
			handler := app.Command("DeleteUser",
				cqrshtmx.RequireAuth(),
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &bddDeleteUserCmd{aggID: userID}, nil
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users/delete", strings.NewReader(body))
			r.Header.Set("X-User", cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633").String())
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
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
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Queries: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("renders query results as JSON", func() {
			handler := app.Query("ListUsers",
				cqrshtmx.DecodeJSONQuery(func(_ bddListUsersQuery) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.Render(func(w http.ResponseWriter, _ *http.Request, result any) error {
					w.Header().Set("Content-Type", "application/json")
					return json.NewEncoder(w).Encode(result)
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
		})
	})

	Describe("Command with redirect", func() {
		var app *cqrshtmx.App

		BeforeEach(func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("redirects to URL after command success", func() {
			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Redirect("/users"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("Location")).To(Equal("/users"))
		})
	})

	Describe("TriggerWithDetail handler option", func() {
		var app *cqrshtmx.App

		BeforeEach(func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("sets HX-Trigger with detail data", func() {
			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.TriggerWithDetail("userCreated", map[string]string{"id": "123"}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("userCreated"))
			Expect(trigger).To(ContainSubstring("123"))
		})
	})

	Describe("App.Middleware integration", func() {
		It("propagates user ID from middleware to handler", func() {
			want := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			disp := command.NewDispatcher()
			var receivedUserID cqrshtmx.UserID
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				receivedUserID = cqrshtmx.UserIDFromContext(ctx)
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: func(r *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.ParseUserID(r.Header.Get("X-User-ID")) },
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Middleware()(app.Command("CreateUser",
				decodeCreateUserJSON(),
			))

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User-ID", want.String())
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(receivedUserID).To(Equal(want))
		})
	})

	Describe("End-to-end CQRS + HTMX + CSRF protection", func() {
		var (
			app  *cqrshtmx.App
			disp *command.Dispatcher
		)

		BeforeEach(func() {
			disp = command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows command dispatch with valid CSRF token via HTMX header", func() {
			// Set up the full middleware chain with CSRF
			cmdHandler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
					return &bddCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
				cqrshtmx.NotifySuccess("User created"),
			)

			// Wrap with CSRF middleware
			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(cmdHandler)

			// Step 1: GET request to obtain masked CSRF token
			var csrfToken string
			tokenHandler := csrfMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				csrfToken = cqrshtmx.CSRFTokenFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			tokenHandler.ServeHTTP(w1, r1)

			cookies := w1.Result().Cookies()
			Expect(cookies).To(HaveLen(1))
			Expect(csrfToken).NotTo(BeEmpty())

			// Step 2: POST with CSRF token in HTMX header
			w2 := httptest.NewRecorder()
			body := testUserJSON
			r2 := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r2.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			r2.Header.Set("X-CSRF-Token", csrfToken)
			for _, c := range cookies {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)

			Expect(w2.Code).To(Equal(http.StatusOK))
			Expect(w2.Header().Get("HX-Trigger")).To(ContainSubstring("success"))
		})

		It("rejects command dispatch without CSRF token", func() {
			cmdHandler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
					return &bddCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			)

			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(cmdHandler)

			// POST without CSRF token
			w := httptest.NewRecorder()
			body := testUserJSON
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)

			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("rejects command dispatch with invalid CSRF token", func() {
			cmdHandler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
					return &bddCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			)

			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(cmdHandler)

			// Step 1: GET to obtain token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			// Step 2: POST with wrong token
			w2 := httptest.NewRecorder()
			body := testUserJSON
			r2 := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r2.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			r2.Header.Set("X-CSRF-Token", "wrong-token")
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)

			Expect(w2.Code).To(Equal(http.StatusForbidden))
		})

		It("allows GET queries without CSRF token", func() {
			qryDisp := query.NewDispatcher()
			_ = qryDisp.Register("ListUsers", func(_ context.Context, _ query.Query) (any, error) {
				return []bddUser{{Email: aliceEmail, Name: aliceName}}, nil
			})

			qryApp, err := cqrshtmx.New(cqrshtmx.Config{Queries: qryDisp})
			Expect(err).NotTo(HaveOccurred())

			qryHandler := qryApp.Query("ListUsers",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.Render(func(w http.ResponseWriter, _ *http.Request, result any) error {
					w.Header().Set("Content-Type", "application/json")
					return json.NewEncoder(w).Encode(result)
				}),
			)

			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(qryHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader("{}"))
			handler.ServeHTTP(w, r)

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

			qryHandler := qryApp.Query("GetPage",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &getPageQuery{}, nil
				}),
				cqrshtmx.Render(func(w http.ResponseWriter, r *http.Request, _ any) error {
					token := cqrshtmx.CSRFTokenFromContext(r.Context())
					resp := cqrshtmx.NewResponse(w, r)
					resp.CSRFToken(token).Apply()
					_, _ = w.Write([]byte("<div>page</div>"))
					return nil
				}),
			)

			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			handler := csrfMW(qryHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader("{}"))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("X-CSRF-Token")).NotTo(BeEmpty())
		})
	})

	Describe("End-to-end CQRS + CSRFProtect per-handler option", func() {
		var (
			app  *cqrshtmx.App
			disp *command.Dispatcher
		)

		BeforeEach(func() {
			disp = command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows command dispatch with CSRFProtect and valid token", func() {
			// First, set up CSRF middleware to set cookie + context
			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())

			// Then use CSRFProtect on the specific handler
			cmdHandler := app.Command("CreateUser",
				cqrshtmx.CSRFProtect(integrationCSRFConfig()),
				cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
					return &bddCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			)

			// Wrap with CSRF middleware (sets context)
			handler := csrfMW(cmdHandler)

			// Step 1: GET to obtain masked token
			var csrfToken string
			tokenHandler := csrfMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				csrfToken = cqrshtmx.CSRFTokenFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			tokenHandler.ServeHTTP(w1, r1)

			cookies := w1.Result().Cookies()
			Expect(cookies).To(HaveLen(1))
			Expect(csrfToken).NotTo(BeEmpty())

			// Step 2: POST with token
			w2 := httptest.NewRecorder()
			body := testUserJSON
			r2 := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r2.Header.Set("X-CSRF-Token", csrfToken)
			for _, c := range cookies {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusNoContent))
		})

		It("rejects command dispatch with CSRFProtect but no token", func() {
			cmdHandler := app.Command("CreateUser",
				cqrshtmx.CSRFProtect(integrationCSRFConfig()),
				cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
					return &bddCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			)

			w := httptest.NewRecorder()
			body := testUserJSON
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			cmdHandler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("rejects command dispatch with CSRFProtect and invalid token", func() {
			csrfMW := cqrshtmx.CSRFMiddleware(integrationCSRFConfig())
			cmdHandler := app.Command("CreateUser",
				cqrshtmx.CSRFProtect(integrationCSRFConfig()),
				cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
					return &bddCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			)

			handler := csrfMW(cmdHandler)

			// Step 1: GET to obtain token
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w1, r1)

			// Step 2: POST with wrong token
			w2 := httptest.NewRecorder()
			body := testUserJSON
			r2 := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r2.Header.Set("X-CSRF-Token", "wrong-token")
			for _, c := range w1.Result().Cookies() {
				r2.AddCookie(c)
			}

			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusForbidden))
		})
	})

	Describe("Body size limits", func() {
		It("rejects JSON body larger than MaxBodySize", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			limitedApp, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:    disp,
				MaxBodySize: 10, // 10 bytes
			})
			Expect(err).NotTo(HaveOccurred())

			handler := limitedApp.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
					return &bddCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(testUserJSON))
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("allows JSON body within MaxBodySize", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			limitedApp, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:    disp,
				MaxBodySize: 1024, // 1 KB
			})
			Expect(err).NotTo(HaveOccurred())

			handler := limitedApp.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
					return &bddCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(testUserJSON))
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
		})
	})
})
