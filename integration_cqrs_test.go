package cqrshtmx_test

import (
	"context"
	"net/http"

	"github.com/casbin/casbin/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration: CQRS + HTMX + Casbin", func() {
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
})
