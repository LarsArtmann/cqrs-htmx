package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const testUserJSONBody = `{"email":"test@example.com","name":"Test"}`

// Test command and query types for integration tests.

type testCreateUserCmd struct {
	aggID id.AggregateID
	email string
	name  string
}

func (c *testCreateUserCmd) Type() command.Type          { return "CreateUser" }
func (c *testCreateUserCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *testCreateUserCmd) IdempotencyKey() string      { return c.aggID.String() }

type testCreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type testGetUserQuery struct{}

func (q *testGetUserQuery) Type() query.Type { return "GetUser" }

func newTestEnforcer() *casbin.Enforcer {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "r.sub == p.sub && r.obj == p.obj && r.act == p.act")

	e, _ := casbin.NewEnforcer(m)
	_, _ = e.AddPolicy("admin", "users", "create")
	_, _ = e.AddPolicy("admin", "users", "read")
	_, _ = e.AddPolicy("viewer", "users", "read")

	return e
}

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
			r.Header.Set("HX-Request", "true")
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
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatched = true
				return nil
			})

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: func(_ *http.Request) string { return "test-user" },
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("dispatches a command from JSON request body", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					aggID := id.NewAggregateID()
					return &testCreateUserCmd{aggID: aggID, email: req.Email, name: req.Name}, nil
				}),
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
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					aggID := id.NewAggregateID()
					return &testCreateUserCmd{aggID: aggID, email: req.Email, name: req.Name}, nil
				}),
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
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return nil
			})

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				Enforcer:        enf,
				UserIDExtractor: func(r *http.Request) string { return r.Header.Get("X-User-ID") },
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows admin to create users", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					aggID := id.NewAggregateID()
					return &testCreateUserCmd{aggID: aggID, email: req.Email, name: req.Name}, nil
				}),
			)

			body := testUserJSONBody
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User-ID", "admin")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
		})

		It("denies viewer from creating users", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					aggID := id.NewAggregateID()
					return &testCreateUserCmd{aggID: aggID, email: req.Email, name: req.Name}, nil
				}),
			)

			body := testUserJSONBody
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User-ID", "viewer")
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
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return nil
			})

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("sets HX-Trigger header with Trigger option", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
				cqrshtmx.Trigger("userCreated"),
			)

			body := testUserJSONBody
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
		})

		It("sets HX-Push-Url header with PushURL option", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
				cqrshtmx.PushURL("/users"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Push-Url")).To(Equal("/users"))
		})

		It("sets HX-Redirect header with Redirect option for HTMX requests", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
				cqrshtmx.Redirect("/users"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
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
				return map[string]string{"email": "test@example.com", "name": "Test"}, nil
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
				cqrshtmx.Render(func(w http.ResponseWriter, _ *http.Request, result any) error {
					return json.NewEncoder(w).Encode(result)
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))

			var result map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&result)).To(Succeed())
			Expect(result["email"]).To(Equal("test@example.com"))
		})
	})

	Describe("App.Middleware", func() {
		It("returns a context enrichment middleware", func() {
			disp := command.NewDispatcher()
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: func(_ *http.Request) string { return "user-1" },
			})
			Expect(err).NotTo(HaveOccurred())

			var capturedUserID string
			handler := app.Middleware()(
				http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					capturedUserID = cqrshtmx.UserIDFromContext(r.Context())
				}),
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			Expect(capturedUserID).To(Equal("user-1"))
		})
	})
})

var _ = Describe("Authorization", func() {
	Describe("Enforce", func() {
		It("allows permitted actions", func() {
			e := newTestEnforcer()
			err := cqrshtmx.Enforce(e, "admin", "users", "create")
			Expect(err).NotTo(HaveOccurred())
		})

		It("denies non-permitted actions", func() {
			e := newTestEnforcer()
			err := cqrshtmx.Enforce(e, "viewer", "users", "create")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, cqrshtmx.ErrForbidden)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("viewer/users/create"))
		})

		It("returns error for nil enforcer", func() {
			err := cqrshtmx.Enforce(nil, "admin", "users", "create")
			Expect(errors.Is(err, cqrshtmx.ErrEnforcerNil)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("users/create"))
		})
	})

	Describe("AuthorizeMiddleware", func() {
		It("allows authorized requests through", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) string { return "admin" })

			called := false
			handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called = true
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeTrue())
		})

		It("blocks unauthorized requests", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "create",
				func(_ *http.Request) string { return "viewer" })

			called := false
			handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called = true
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("blocks unauthenticated requests", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) string { return "" })

			called := false
			handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called = true
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("redirects unauthenticated HTMX requests to login", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) string { return "" })

			called := false
			handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called = true
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})

		It("uses custom login redirect", func() {
			e := newTestEnforcer()
			middleware := cqrshtmx.AuthorizeMiddleware(e, "users", "read",
				func(_ *http.Request) string { return "" }, "/auth/signin")

			called := false
			handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called = true
			}))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			handler.ServeHTTP(w, r)
			Expect(called).To(BeFalse())
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/auth/signin"))
		})
	})
})

var _ = Describe("Handler Options", func() {
	Describe("RequireAuth", func() {
		It("rejects requests without user ID", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.RequireAuth(),
				cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})
