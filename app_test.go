package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testUserJSONBody = `{"email":"test@example.com","name":"Test"}`
	testEmailKey     = "email"
	testNameKey      = "name"
	testEmailValue   = "test@example.com"
)

var (
	adminUserID  = cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
	viewerUserID = cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")
)

var _ = Describe("App", func() {
	Describe("New", func() {
		It("creates an app with command dispatcher", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(err).NotTo(HaveOccurred())
			Expect(app).NotTo(BeNil())
		})

		It("creates an app with query dispatcher", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: query.NewDispatcher()})
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
			handler := app.Command("CreateUser", cqrshtmx.Authorize("users", "create"))
			r := newPostRequest("/users", "", withHTMX)
			w := serve(handler, r)
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/auth/signin"))
		})

		It("includes request_id in errors when IncludeRequestIDInErrors is true", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:                 command.NewDispatcher(),
				IncludeRequestIDInErrors: true,
			})
			Expect(err).NotTo(HaveOccurred())
			rid := cqrshtmx.MustParseRequestID("01HK154ANGZHV2ZW0X3SKSNEN2")
			r := newPostRequest("/users", "")
			r = r.WithContext(cqrshtmx.WithRequestID(r.Context(), rid))
			w := serve(app.Command("CreateUser"), r)
			Expect(w.Body.String()).To(ContainSubstring("[request_id: " + rid.String() + "]"))
		})

		It("does not affect custom ErrorHandler", func() {
			called := false
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:                 command.NewDispatcher(),
				IncludeRequestIDInErrors: true,
				ErrorHandler: func(_ http.ResponseWriter, _ *http.Request, _ error) {
					called = true
				},
			})
			Expect(err).NotTo(HaveOccurred())
			r := newPostRequest("/users", "")
			w := serve(app.Command("CreateUser"), r)
			Expect(called).To(BeTrue())
			Expect(w.Body.String()).To(BeEmpty())
		})
	})

	Describe("Command handler", func() {
		var dispatched bool

		BeforeEach(func() { dispatched = false })

		It("dispatches a command from JSON request body", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", trackingCommandHandler(&dispatched))
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			w := serve(app.Command("CreateUser", decodeCreateUserJSONWithBody()),
				newPostJSONRequest(`{"email":"test@example.com","name":"Test User"}`))
			Expect(w.code()).To(Equal(http.StatusNoContent))
			Expect(dispatched).To(BeTrue())
		})

		It("returns error when decoder is missing", func() {
			app := newCommandApp()
			w := serve(app.Command("CreateUser"), newPostRequest("/users", ""))
			Expect(w.code()).NotTo(Equal(http.StatusNoContent))
		})

		It("returns error for invalid JSON body", func() {
			app := newCommandApp()
			w := serve(app.Command("CreateUser", decodeCreateUserJSONWithBody()),
				newPostJSONRequest("{invalid json"))
			Expect(w.code()).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Command handler with authorization", func() {
		var app *cqrshtmx.App

		BeforeEach(func() {
			enf := newTestEnforcer()
			disp := command.NewDispatcher()
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
			w := serve(app.Command(
				"CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithBody(),
			), newPostJSONRequest(testUserJSONBody,
				withUserHeader("X-User-ID", adminUserID)))
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})

		It("denies viewer from creating users", func() {
			w := serve(app.Command(
				"CreateUser",
				cqrshtmx.Authorize("users", "create"),
				decodeCreateUserJSONWithBody(),
			), newPostJSONRequest(testUserJSONBody,
				withUserHeader("X-User-ID", viewerUserID)))
			Expect(w.code()).To(Equal(http.StatusForbidden))
		})

		It("rejects unauthenticated users", func() {
			w := serve(app.Command(
				"CreateUser",
				cqrshtmx.Authorize("users", "create"),
			), newPostRequest("/users", ""))
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
			Expect(w.Body.String()).To(ContainSubstring("users/create"))
		})
	})

	Describe("Command handler with HTMX response options", func() {
		var app *cqrshtmx.App

		BeforeEach(func() { app = newCommandApp() })

		It("sets HX-Trigger header with Trigger option", func() {
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Trigger("userCreated"),
			), newPostRequest("/users", testUserJSONBody, withHTMX))
			Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
		})

		It("sets HX-Push-Url header with PushURL option", func() {
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.PushURL("/users"),
			), newPostRequest("/users", `{}`, withHTMX))
			Expect(w.Header().Get("HX-Push-Url")).To(Equal("/users"))
		})

		It("sets HX-Redirect header with Redirect option for HTMX requests", func() {
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Redirect("/users"),
			), newPostRequest("/users", `{}`, withHTMX))
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/users"))
		})
	})

	Describe("Query handler", func() {
		It("dispatches a query and renders the result", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return map[string]string{testEmailKey: testEmailValue, testNameKey: "Test"}, nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))

			var result map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&result)).To(Succeed())
			Expect(result[testEmailKey]).To(Equal(testEmailValue))
		})
	})

	Describe("Query handler with authorization", func() {
		var app *cqrshtmx.App

		BeforeEach(func() {
			enf := newTestEnforcer()
			disp := query.NewDispatcher()
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
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("X-User-ID", adminUserID.String())
			w := serve(app.Query(
				"GetUser",
				cqrshtmx.Authorize("users", "read"),
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))
		})

		It("denies unauthorized queries", func() {
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("X-User-ID", viewerUserID.String())
			w := serve(app.Query(
				"GetUser",
				cqrshtmx.Authorize("users", "admin"),
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
			), r)
			Expect(w.code()).To(Equal(http.StatusForbidden))
		})

		It("rejects unauthenticated queries", func() {
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				cqrshtmx.Authorize("users", "read"),
				cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
			), r)
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})

		It("rejects queries without decoder", func() {
			r := httptest.NewRequest(http.MethodGet, "/users", nil)
			r.Header.Set("X-User-ID", adminUserID.String())
			w := serve(app.Query("GetUser", cqrshtmx.Authorize("users", "read")), r)
			Expect(w.code()).NotTo(Equal(http.StatusOK))
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
			serve(handler, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(capturedUserID).To(Equal(want))
		})
	})
})

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
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil })
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			w := serve(handler, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(called).To(BeFalse())
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})

		It("redirects unauthenticated HTMX requests to login", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil })
			called := false
			handler := middleware(middlewareCaptureHandler(&called))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := serve(handler, r)
			Expect(called).To(BeFalse())
			Expect(w.code()).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
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
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := serve(handler, r)
			Expect(called).To(BeFalse())
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/auth/signin"))
		})

		It("prefers branded UserID from context over extractor", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) (cqrshtmx.UserID, error) { return cqrshtmx.UserID{}, nil })
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

var _ = Describe("Handler Options", func() {
	Describe("RequireAuth", func() {
		It("rejects requests without user ID", func() {
			app := newCommandApp()
			w := serve(app.Command(
				"CreateUser",
				cqrshtmx.RequireAuth(),
				decodeCreateUserJSON(),
			), newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})
	})
})
