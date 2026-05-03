package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/casbin/casbin/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type deleteUserCmd struct {
	aggID id.AggregateID
}

func (c *deleteUserCmd) Type() command.Type          { return "DeleteUser" }
func (c *deleteUserCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *deleteUserCmd) IdempotencyKey() string      { return c.aggID.String() }

type listUsersQuery struct{}

func (q *listUsersQuery) Type() query.Type { return "ListUsers" }

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

			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return nil
			})
			_ = disp.Register("DeleteUser", func(ctx context.Context, cmd command.Command) error {
				return event.NewRejection("user.not_found", "user does not exist")
			})

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				Enforcer:        enf,
				UserIDExtractor: func(r *http.Request) string { return r.Header.Get("X-User") },
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows admin to create user with full HTMX response", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: userID, email: req.Email, name: req.Name}, nil
				}),
				cqrshtmx.Trigger("userCreated"),
				cqrshtmx.PushURL("/users/"+userID.String()),
			)

			body := `{"email":"admin@co.com","name":"Admin"}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User", "admin")
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("HX-Trigger")).To(Equal("userCreated"))
			Expect(w.Header().Get("HX-Push-Url")).To(ContainSubstring("/users/"))
		})

		It("denies viewer from creating user", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: userID}, nil
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User", "viewer")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("redirects unauthenticated HTMX users to login", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.Authorize("users", "create"),
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: userID}, nil
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusSeeOther))
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})

		It("maps CQRS rejection errors to 400", func() {
			handler := app.Command("DeleteUser",
				cqrshtmx.RequireAuth(),
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &deleteUserCmd{aggID: userID}, nil
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users/delete", strings.NewReader(body))
			r.Header.Set("X-User", "admin")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Query with render", func() {
		var app *cqrshtmx.App

		BeforeEach(func() {
			disp := query.NewDispatcher()
			_ = disp.Register("ListUsers", func(ctx context.Context, q query.Query) (any, error) {
				return []map[string]string{
					{"email": "a@b.com", "name": "Alice"},
					{"email": "c@d.com", "name": "Carol"},
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
				cqrshtmx.DecodeJSONQuery(func(req listUsersQuery) (query.Query, error) {
					return &listUsersQuery{}, nil
				}),
				cqrshtmx.Render(func(w http.ResponseWriter, r *http.Request, result any) error {
					w.Header().Set("Content-Type", "application/json")
					return json.NewEncoder(w).Encode(result)
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("Alice"))
		})
	})

	Describe("Command with redirect", func() {
		var app *cqrshtmx.App

		BeforeEach(func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return nil
			})

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("redirects to URL after command success", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
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
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return nil
			})

			var err error
			app, err = cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("sets HX-Trigger with detail data", func() {
			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
				cqrshtmx.TriggerWithDetail("userCreated", map[string]string{"id": "123"}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("userCreated"))
			Expect(trigger).To(ContainSubstring("123"))
		})
	})

	Describe("App.Middleware integration", func() {
		It("propagates user ID from middleware to handler", func() {
			disp := command.NewDispatcher()
			var receivedUserID string
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				receivedUserID = cqrshtmx.UserIDFromContext(ctx)
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: func(r *http.Request) string { return r.Header.Get("X-User-ID") },
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Middleware()(app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			))

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("X-User-ID", "user-from-middleware")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(receivedUserID).To(Equal("user-from-middleware"))
		})
	})
})
