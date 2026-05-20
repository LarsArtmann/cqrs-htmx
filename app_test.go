package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/casbin/casbin/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testUserJSONBody = `{"email":"test@example.com","name":"Test"}`
	testEmailKey     = "email"
	testNameKey      = "name"
	testEmailValue   = "test@example.com"
)

// Test user IDs — valid ULIDs used as Casbin policy subjects.
//
//nolint:gochecknoglobals // intentional test fixtures computed once at package init
var (
	adminUserID  = cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
	viewerUserID = cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")
)

var _ = Describe("App", func() {
	Describe("New", func() {
		It("creates an app with command dispatcher", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: command.NewDispatcher(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(app).NotTo(BeNil())
		})

		It("creates an app with query dispatcher", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Queries: query.NewDispatcher(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(app).NotTo(BeNil())
		})

		It("returns error when both dispatchers are nil", func() {
			_, err := cqrshtmx.New(cqrshtmx.Config{})
			Expect(err).To(HaveOccurred())
		})

		It("uses Config.LoginRedirect for HTMX auth errors", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:      command.NewDispatcher(),
				LoginRedirect: "/auth/signin",
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/auth/signin"))
		})
	})

	Describe("Command handler", func() {
		var (
			app        *cqrshtmx.App
			dispatched bool
		)

		BeforeEach(func() {
			disp := command.NewDispatcher()
			dispatched = false
			_ = disp.Register("CreateUser", trackingCommandHandler(&dispatched))

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: staticExtractor(adminUserID),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("dispatches a command from JSON request body", func() {
			handler := app.Command("CreateUser",
				decodeCreateUserJSONWithBody(),
			)

			body := `{"email":"test@example.com","name":"Test User"}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(dispatched).To(BeTrue())
		})

		It("returns error when decoder is missing", func() {
			handler := app.Command("CreateUser")
			r := httptest.NewRequest(http.MethodPost, "/users", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).NotTo(Equal(http.StatusNoContent))
		})

		It("returns error for invalid JSON body", func() {
			handler := app.Command("CreateUser",
				decodeCreateUserJSONWithBody(),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{invalid json"))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Command handler with authorization", func() {
		var (
			app  *cqrshtmx.App
			enf  *casbin.Enforcer
			disp *command.Dispatcher
		)

		BeforeEach(func() {
			enf = newTestEnforcer()
			disp = command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				Enforcer:        enf,
				UserIDExtractor: headerExtractor("X-User-ID"),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows admin to create users", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithBody(),
			)

			body := testUserJSONBody
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User-ID", adminUserID.String())
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
		})

		It("denies viewer from creating users", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithBody(),
			)

			body := testUserJSONBody
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User-ID", viewerUserID.String())
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("rejects unauthenticated users", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
			Expect(w.Body.String()).To(ContainSubstring("users/create"))
		})
	})

	Describe("Command handler with HTMX response options", func() {
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

		It("sets HX-Trigger header with Trigger option", func() {
			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Trigger("userCreated"),
			)

			body := testUserJSONBody
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
		})

		It("sets HX-Push-Url header with PushURL option", func() {
			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.PushURL("/users"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Push-Url")).To(Equal("/users"))
		})

		It("sets HX-Redirect header with Redirect option for HTMX requests", func() {
			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Redirect("/users"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/users"))
		})
	})

	Describe("Query handler", func() {
		var app *cqrshtmx.App

		BeforeEach(func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return map[string]string{testEmailKey: testEmailValue, testNameKey: "Test"}, nil
			})

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Queries: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("dispatches a query and renders the result", func() {
			handler := app.Query("GetUser",
				cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))

			var result map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&result)).To(Succeed())
			Expect(result[testEmailKey]).To(Equal(testEmailValue))
		})
	})

	Describe("Query handler with authorization", func() {
		var (
			app  *cqrshtmx.App
			enf  *casbin.Enforcer
			disp *query.Dispatcher
		)

		BeforeEach(func() {
			enf = newTestEnforcer()
			disp = query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return map[string]string{testEmailKey: testEmailValue}, nil
			})

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Queries:         disp,
				Enforcer:        enf,
				UserIDExtractor: headerExtractor("X-User-ID"),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows authorized queries", func() {
			handler := app.Query("GetUser",
				cqrshtmx.Authorize("users", "read"),
				cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			r.Header.Set("X-User-ID", adminUserID.String())
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("denies unauthorized queries", func() {
			handler := app.Query("GetUser",
				cqrshtmx.Authorize("users", "admin"),
				cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			r.Header.Set("X-User-ID", viewerUserID.String())
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("rejects unauthenticated queries", func() {
			handler := app.Query("GetUser",
				cqrshtmx.Authorize("users", "read"),
				cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("rejects queries without decoder", func() {
			handler := app.Query("GetUser",
				cqrshtmx.Authorize("users", "read"),
			)

			r := httptest.NewRequest(http.MethodGet, "/users", nil)
			r.Header.Set("X-User-ID", adminUserID.String())
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).NotTo(Equal(http.StatusOK))
		})
	})

	Describe("App.Middleware", func() {
		It("returns a context enrichment middleware", func() {
			want := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			disp := command.NewDispatcher()
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: staticExtractor(want),
			})
			Expect(err).NotTo(HaveOccurred())

			var capturedUserID cqrshtmx.UserID
			handler := app.Middleware()(
				http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					capturedUserID = cqrshtmx.UserIDFromContext(r.Context())
				}),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(capturedUserID).To(Equal(want))
		})
	})
})

var _ = Describe("Authorization", func() {
	Describe("Enforce", func() {
		It("allows permitted actions", func() {
			e := newTestEnforcer()
			err := cqrshtmx.Enforce(e, adminUserID.String(), "users", "create")
			Expect(err).NotTo(HaveOccurred())
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
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				staticExtractor(adminUserID))

			called := false
			handler := middleware(middlewareCaptureHandler(&called))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeTrue())
		})

		It("blocks unauthorized requests", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "create",
				staticExtractor(viewerUserID))

			called := false
			handler := middleware(middlewareCaptureHandler(&called))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("blocks unauthenticated requests", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil })

			called := false
			handler := middleware(middlewareCaptureHandler(&called))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("redirects unauthenticated HTMX requests to login", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil })

			called := false
			handler := middleware(middlewareCaptureHandler(&called))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})

		It("uses custom login redirect", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(
				e,
				"users",
				"read",
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil },
				"/auth/signin",
			)

			called := false
			handler := middleware(middlewareCaptureHandler(&called))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/auth/signin"))
		})

		It("prefers branded UserID from context over extractor", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil })

			called := false
			handler := middleware(middlewareCaptureHandler(&called))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = r.WithContext(cqrshtmx.WithUserID(r.Context(), adminUserID))
			handler.ServeHTTP(w, r)
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

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("rejects requests when extractor is nil and no context user ID", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read", nil)

			called := false
			handler := middleware(middlewareCaptureHandler(&called))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})

var _ = Describe("Handler Options", func() {
	Describe("RequireAuth", func() {
		It("rejects requests without user ID", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.RequireAuth(),
				decodeCreateUserJSON(),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})
