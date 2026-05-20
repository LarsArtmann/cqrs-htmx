package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/casbin/casbin/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
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
				disp := command.NewDispatcher()
				dispatched = false
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
				handler := app.Command("CreateUser",
					cqrshtmx.Authorize("users", "create"),
					decodeBDDCreateUserJSONWithBody(),
					cqrshtmx.NotifySuccess("User created successfully"),
					cqrshtmx.PushURL("/users"),
				)

				body := `{"email":"alice@example.com","name":"Alice"}`
				r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
				adminID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
				r.Header.Set("X-User", adminID.String())
				r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, r)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(dispatched).To(BeTrue())
				Expect(w.Header().Get("HX-Trigger")).To(ContainSubstring("success"))
				Expect(
					w.Header().Get("HX-Trigger"),
				).To(ContainSubstring("User created successfully"))
				Expect(w.Header().Get("HX-Push-Url")).To(Equal("/users"))
			})

			It("rejects an unauthorized viewer", func() {
				handler := app.Command("CreateUser",
					cqrshtmx.Authorize("users", "create"),
					decodeBDDCreateUserJSON(),
				)

				body := `{}`
				r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
				viewerID := cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")
				r.Header.Set("X-User", viewerID.String())
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, r)

				Expect(w.Code).To(Equal(http.StatusForbidden))
				Expect(dispatched).To(BeFalse())
			})

			It("redirects unauthenticated HTMX users to login", func() {
				handler := app.Command("CreateUser",
					cqrshtmx.Authorize("users", "create"),
					decodeBDDCreateUserJSON(),
				)

				body := `{}`
				r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
				r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, r)

				Expect(w.Code).To(Equal(http.StatusSeeOther))
				Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
				Expect(dispatched).To(BeFalse())
			})

			It("handles invalid JSON input gracefully", func() {
				handler := app.Command("CreateUser",
					cqrshtmx.Authorize("users", "create"),
					decodeBDDCreateUserJSON(),
				)

				r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{bad json"))
				adminID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
				r.Header.Set("X-User", adminID.String())
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, r)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(dispatched).To(BeFalse())
			})
		},
	)

	Describe("As a consumer, I want to query data and render it using templ components", func() {
		It("renders a fixed templ component for a page query", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetDashboard", func(_ context.Context, _ query.Query) (any, error) {
				return "dashboard data", nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			component := &bddTemplComponent{html: "<div class='dashboard'>Welcome</div>"}
			handler := app.Query("GetDashboard",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &bddDashboardQuery{}, nil
				}),
				cqrshtmx.RenderTempl(component),
			)

			r := httptest.NewRequest(http.MethodGet, "/dashboard", strings.NewReader(`{}`))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
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

			handler := app.Query("ListUsers",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.RenderTemplResult(func(result []bddUser) cqrshtmx.TemplComponent {
					html := "<ul>"
					var sb strings.Builder
					for _, u := range result {
						sb.WriteString("<li>" + u.Name + "</li>")
					}
					html += sb.String()
					html += "</ul>"
					return &bddTemplComponent{html: html}
				}),
			)

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
			Expect(w.Body.String()).To(ContainSubstring("Bob"))
			Expect(w.Body.String()).To(ContainSubstring("<ul>"))
		})
	})

	Describe("As a consumer, I want CQRS domain errors to map correctly to HTTP responses", func() {
		It("maps a rejection to 400 Bad Request", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("DeleteUser", rejectionHandler(
				"user.not_found", "user does not exist",
			))

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("DeleteUser",
				cqrshtmx.DecodeJSON(func(_ bddCreateUserReq) (command.Command, error) {
					return &bddDeleteUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			)

			r := httptest.NewRequest(http.MethodPost, "/users/1", strings.NewReader(`{}`))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("user does not exist"))
		})

		It("maps a conflict to 409 Conflict", func() {
			Expect(
				cqrshtmx.MapError(event.NewConflict("user.duplicate", "duplicate")),
			).To(Equal(http.StatusConflict))
		})

		It("maps a transient to 503 Service Unavailable", func() {
			Expect(
				cqrshtmx.MapError(event.NewTransient("db.timeout", "timeout")),
			).To(Equal(http.StatusServiceUnavailable))
		})
	})

	Describe("As a consumer, I want to use the Response builder for custom HTMX responses", func() {
		It("builds a complex HTMX response with fluent chaining", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/items", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)

			resp := cqrshtmx.NewResponse(w, r)
			resp.Trigger("itemCreated").
				PushURL("/items/42").
				Retarget("#item-list").
				Reswap(cqrshtmx.SwapOuterHTML).
				NotifySuccess("Item created").
				Apply()

			_, _ = w.WriteString("<div>Item 42</div>")

			Expect(w.Header().Get("HX-Trigger")).To(ContainSubstring("itemCreated"))
			Expect(w.Header().Get("HX-Push-Url")).To(Equal("/items/42"))
			Expect(w.Header().Get("HX-Retarget")).To(Equal("#item-list"))
			Expect(w.Header().Get("HX-Reswap")).To(Equal("outerHTML"))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/html; charset=utf-8"))
			Expect(w.Body.String()).To(ContainSubstring("Item 42"))
		})
	})

	Describe("As a consumer, I want the middleware chain to compose correctly", func() {
		It("chains HTMX parsing and context enrichment in correct order", func() {
			want := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			disp := command.NewDispatcher()
			var capturedUserID cqrshtmx.UserID
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
			)(app.Command("CreateUser",
				decodeBDDCreateUserJSON(),
			))

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			r.Header.Set("X-User-ID", want.String())
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			Expect(capturedUserID).To(Equal(want))
		})
	})

	Describe("As a consumer, I want user identity to propagate into event metadata", func() {
		It("builds event options from context with valid user ID", func() {
			ctx := context.Background()
			userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
			ctx = cqrshtmx.WithUserID(ctx, userID)
			opts := cqrshtmx.EventOptionsFromContext(ctx)

			Expect(opts).NotTo(BeNil())
			Expect(opts).To(HaveLen(1))
		})

		It("returns nil when no user ID is in context", func() {
			opts := cqrshtmx.EventOptionsFromContext(context.Background())
			Expect(opts).To(BeNil())
		})
	})

	Describe("As a consumer, I want to decode form data into commands", func() {
		It("decodes URL-encoded form data correctly", func() {
			disp := command.NewDispatcher()
			var receivedName string
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

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeForm(decodeBDDCreateUserFormMapper()),
			)

			form := strings.NewReader("Email=alice%40example.com&Name=Alice")
			r := httptest.NewRequest(http.MethodPost, "/users", form)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
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

			handler := app.Query("ListUsers",
				cqrshtmx.DecodeFormQuery(func(_ bddCreateUserReq) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.Render(encodeJSONResult),
			)

			form := strings.NewReader("Email=alice%40example.com&Name=Alice")
			r := httptest.NewRequest(http.MethodGet, "/users", form)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
		})
	})

	Describe("As a consumer, I want JSON query results rendered as API responses", func() {
		It("renders query results as JSON for non-HTMX requests", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("ListUsers", func(_ context.Context, _ query.Query) (any, error) {
				return []bddUser{
					{Email: aliceEmail, Name: aliceName},
				}, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("ListUsers",
				cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
					return &bddListUsersQuery{}, nil
				}),
				cqrshtmx.Render(func(w http.ResponseWriter, _ *http.Request, result any) error {
					w.Header().Set("Content-Type", "application/json")
					return json.NewEncoder(w).Encode(result)
				}),
			)

			r := httptest.NewRequest(http.MethodGet, "/api/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			var users []bddUser
			Expect(json.NewDecoder(w.Body).Decode(&users)).To(Succeed())
			Expect(users[0].Name).To(Equal(aliceName))
		})
	})
})
