package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/casbin/casbin/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	aliceName  = "Alice"
	aliceEmail = "alice@example.com"
)

type bddUser struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

var _ = Describe("BDD: Consumer Integration Scenarios", func() {
	Describe(
		"As a consumer, I want to create a command handler that validates input, checks authorization, and returns HTMX responses",
		func() {
			var (
				app        *cqrshtmx.App
				enforcer   *casbin.Enforcer
				dispatched bool
			)

			BeforeEach(func() {
				enforcer = newTestEnforcer()
				dispatched = false
				disp := command.NewDispatcher()
				_ = disp.Register("CreateUser", trackingCommandHandler(&dispatched))

				var err error

				app, err = cqrshtmx.New(cqrshtmx.Config{
					Commands:        disp,
					Enforcer:        enforcer,
					UserIDExtractor: headerExtractor("X-User"),
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("successfully creates a user with full authorization and HTMX notification", func() {
				handler := app.Command(
					"CreateUser",
					cqrshtmx.Authorize("users", "create"),
					decodeBDDCreateUserJSONWithBody(),
					cqrshtmx.NotifySuccess("User created successfully"),
					cqrshtmx.PushURL("/users"),
				)

				r := newPostRequest("/users", `{"email":"alice@example.com","name":"Alice"}`,
					withUserHeader("X-User", adminUserID), withHTMX)
				w := serve(handler, r)

				Expect(w.code()).To(Equal(http.StatusOK))
				Expect(dispatched).To(BeTrue())
				Expect(w.Header().Get("Hx-Trigger")).To(ContainSubstring("success"))
				Expect(
					w.Header().Get("Hx-Trigger"),
				).To(ContainSubstring("User created successfully"))
				Expect(w.Header().Get("Hx-Push-Url")).To(Equal("/users"))
			})

			It("rejects an unauthorized viewer", func() {
				handler := app.Command(
					"CreateUser",
					cqrshtmx.Authorize("users", "create"),
					decodeBDDCreateUserJSON(),
				)
				assertStatusCode(handler,
					newPostRequest("/users", `{}`, withUserHeader("X-User", viewerUserID)),
					http.StatusForbidden)
				Expect(dispatched).To(BeFalse())
			})

			It("redirects unauthenticated HTMX users to login", func() {
				handler := app.Command(
					"CreateUser",
					cqrshtmx.Authorize("users", "create"),
					decodeBDDCreateUserJSON(),
				)
				w := serve(handler, newPostRequest("/users", `{}`, withHTMX))
				Expect(w.code()).To(Equal(http.StatusSeeOther))
				Expect(w.Header().Get("Hx-Redirect")).To(Equal("/login"))
				Expect(dispatched).To(BeFalse())
			})

			It("handles invalid JSON input gracefully", func() {
				handler := app.Command(
					"CreateUser",
					cqrshtmx.Authorize("users", "create"),
					decodeBDDCreateUserJSON(),
				)
				w := serve(handler,
					newPostRequest("/users", "{bad json",
						withUserHeader("X-User", adminUserID)))
				Expect(w.code()).To(Equal(http.StatusBadRequest))
				Expect(dispatched).To(BeFalse())
			})
		},
	)

	Describe("As a consumer, I want to query data and render it using templ components", func() {
		It("renders a fixed templ component for a page query", func() {
			app := newQueryAppNamed("GetDashboard", func(_ context.Context, _ query.Query) (any, error) {
				return "dashboard data", nil
			})

			component := &bddTemplComponent{html: "<div class='dashboard'>Welcome</div>"}
			r := httptest.NewRequest(http.MethodGet, "/dashboard", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetDashboard",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &bddDashboardQuery{}, nil
				}),
				cqrshtmx.RenderTempl(component),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("dashboard"))
			Expect(w.Body.String()).To(ContainSubstring("Welcome"))
		})

		It("maps query results to typed templ components", func() {
			disp := query.NewDispatcher()
			users := []bddUser{
				{Email: aliceEmail, Name: aliceName},
				{Email: "bob@example.com", Name: "Bob"},
			}
			_ = disp.Register("ListUsers", func(_ context.Context, _ query.Query) (any, error) {
				return users, nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"ListUsers",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.RenderTemplResult(func(result []bddUser) cqrshtmx.TemplComponent {
					var sb strings.Builder
					sb.WriteString("<ul>")

					for _, u := range result {
						sb.WriteString("<li>")
						sb.WriteString(u.Name)
						sb.WriteString("</li>")
					}

					sb.WriteString("</ul>")

					return &bddTemplComponent{html: sb.String()}
				}),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
			Expect(w.Body.String()).To(ContainSubstring("Bob"))
			Expect(w.Body.String()).To(ContainSubstring("<ul>"))
		})
	})

	Describe("As a consumer, I want CQRS domain errors to map correctly to HTTP responses", func() {
		It("maps a rejection to 400 Bad Request", func() {
			disp := command.NewDispatcher()
			_ = disp.Register(
				"DeleteUser",
				rejectionHandler("user.not_found", "user does not exist"),
			)
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			w := serve(app.Command(
				"DeleteUser",
				cqrshtmx.DecodeJSON(func(_ bddCreateUserReq) (command.Command, error) {
					return &bddDeleteUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
				}),
			), newPostRequest("/users/1", `{}`))
			Expect(w.code()).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("user does not exist"))
		})

		DescribeTable(
			"error family mappings",
			func(err error, expected int) {
				Expect(cqrshtmx.MapError(err)).To(Equal(expected))
			},
			Entry(
				"conflict to 409",
				errorfamily.NewConflict("user.duplicate", "duplicate"),
				http.StatusConflict,
			),
			Entry(
				"transient to 503",
				errorfamily.NewTransient("db.timeout", "timeout"),
				http.StatusServiceUnavailable,
			),
		)
	})

	Describe("As a consumer, I want to use the Response builder for custom HTMX responses", func() {
		It("builds a complex HTMX response with fluent chaining", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/items", nil)
			r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)

			resp := cqrshtmx.NewResponse(w, r)
			resp.Trigger("itemCreated").
				PushURL("/items/42").
				Retarget("#item-list").
				Reswap(cqrshtmx.SwapOuterHTML).
				NotifySuccess("Item created").
				Apply()

			_, _ = w.WriteString("<div>Item 42</div>")

			Expect(w.Header().Get("Hx-Trigger")).To(ContainSubstring("itemCreated"))
			Expect(w.Header().Get("Hx-Push-Url")).To(Equal("/items/42"))
			Expect(w.Header().Get("Hx-Retarget")).To(Equal("#item-list"))
			Expect(w.Header().Get("Hx-Reswap")).To(Equal("outerHTML"))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/html; charset=utf-8"))
			Expect(w.Body.String()).To(ContainSubstring("Item 42"))
		})
	})

	Describe("As a consumer, I want the middleware chain to compose correctly", func() {
		It("chains HTMX parsing and context enrichment in correct order", func() {
			want := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")

			var capturedUserID cqrshtmx.UserID

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				capturedUserID = cqrshtmx.UserIDFromContext(ctx)

				return nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: headerExtractor("X-User-ID"),
			})
			Expect(err).NotTo(HaveOccurred())

			handler := cqrshtmx.Chain(
				cqrshtmx.HTMXMiddleware,
				app.Middleware(),
			)(app.Command("CreateUser", decodeBDDCreateUserJSON()))

			r := newPostRequest("/users", `{}`,
				withUserHeader("X-User-ID", want), withHTMX)
			serve(handler, r)
			Expect(capturedUserID).To(Equal(want))
		})
	})

	Describe("As a consumer, I want to decode form data into commands", func() {
		It("decodes URL-encoded form data correctly", func() {
			var receivedName string

			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, cmd command.Command) error {
				c, ok := cmd.(*bddCreateUserCmd)
				if !ok {
					return fmt.Errorf("unexpected command type: %T", cmd)
				}

				receivedName = c.name

				return nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/users",
				strings.NewReader("Email=alice%40example.com&Name=Alice"))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			serve(app.Command(
				"CreateUser",
				cqrshtmx.DecodeForm(decodeBDDCreateUserFormMapper()),
			), r)
			Expect(receivedName).To(Equal(aliceName))
		})
	})

	Describe("As a consumer, I want to decode form data into queries", func() {
		It("decodes URL-encoded form data into a query", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("ListUsers", func(_ context.Context, _ query.Query) (any, error) {
				return []string{aliceName}, nil
			})
			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			r := httptest.NewRequest(http.MethodGet, "/users",
				strings.NewReader("Email=alice%40example.com&Name=Alice"))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := serve(app.Query(
				"ListUsers",
				cqrshtmx.DecodeFormQuery(func(_ bddCreateUserReq) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
		})
	})

	Describe("As a consumer, I want JSON query results rendered as API responses", func() {
		It("renders query results as JSON for non-HTMX requests", func() {
			disp := query.NewDispatcher()
			registerBDDListUsers(disp)
			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			r := httptest.NewRequest(http.MethodGet, "/api/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"ListUsers",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))

			var users []bddUser
			Expect(json.NewDecoder(w.Body).Decode(&users)).To(Succeed())
			Expect(users[0].Name).To(Equal(aliceName))
		})
	})
})
