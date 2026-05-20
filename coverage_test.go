package cqrshtmx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const testQueryResult = "result"

func testNotificationTrigger(opt cqrshtmx.HandlerOption, expectedLevel string) {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error { return nil })
	app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
	Expect(err).NotTo(HaveOccurred())

	r := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/users",
		strings.NewReader(`{}`),
	)
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
	w := serve(app.Command("CreateUser", decodeCreateUserJSON(), opt), r)
	trigger := w.Header().Get("HX-Trigger")
	Expect(trigger).To(ContainSubstring(expectedLevel))
}

func newQueryAppWithResult(handler func(context.Context, query.Query) (any, error)) *cqrshtmx.App {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", handler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
	Expect(err).NotTo(HaveOccurred())
	return app
}

func newCommandAppWithHandler(handler func(context.Context, command.Command) error) *cqrshtmx.App {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", handler)
	app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
	Expect(err).NotTo(HaveOccurred())
	return app
}

var _ = Describe("Coverage Gaps", func() {
	Describe("RenderTempl", func() {
		It("renders a fixed templ component", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return "irrelevant", nil
			})
			r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderTempl(&bddTemplComponent{html: "<h1>Hello</h1>"}),
			), r)
			Expect(w.Body.String()).To(Equal("<h1>Hello</h1>"))
		})
	})

	Describe("RenderTemplResult", func() {
		It("maps result to templ component and renders", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return aliceName, nil
			})
			r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderTemplResult(func(result string) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: "<p>" + result + "</p>"}
				}),
			), r)
			Expect(w.Body.String()).To(Equal("<p>Alice</p>"))
		})

		It("returns error for wrong result type", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return 42, nil
			})
			r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderTemplResult(func(result string) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: result}
				}),
			), r)
			Expect(w.code()).ToNot(Equal(http.StatusOK))
		})
	})

	Describe("DecodeForm", func() {
		It("decodes form data into a command", func() {
			var received string
			app := newCommandAppWithHandler(func(_ context.Context, _ command.Command) error {
				received = "dispatched"
				return nil
			})
			form := url.Values{}
			form.Set("Email", "test@example.com")
			form.Set("Name", "Test User")
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			serve(app.Command(
				"CreateUser",
				cqrshtmx.DecodeForm(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			), r)
			Expect(received).To(Equal("dispatched"))
		})

		It("returns error for invalid form data", func() {
			app, _ := cqrshtmx.New(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(app).NotTo(BeNil())
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("%%%invalid"))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := serve(app.Command(
				"CreateUser",
				cqrshtmx.DecodeForm(func(_ testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			), r)
			Expect(w.code()).ToNot(Equal(http.StatusNoContent))
		})
	})

	Describe("MapError additional families", func() {
		DescribeTable(
			"maps CQRS error families",
			func(err error, expected int) {
				Expect(cqrshtmx.MapError(err)).To(Equal(expected))
			},
			Entry(
				"Conflict to 409",
				event.NewConflict("test.conflict", "conflict occurred"),
				http.StatusConflict,
			),
			Entry(
				"Corruption to 422",
				event.NewCorruption("test.corruption", "corrupted data"),
				http.StatusUnprocessableEntity,
			),
			Entry(
				"Infrastructure to 500",
				event.NewInfrastructure("test.infra", "infrastructure failure"),
				http.StatusInternalServerError,
			),
			Entry(
				"Transient to 503",
				event.NewTransient("test.transient", "transient error"),
				http.StatusServiceUnavailable,
			),
		)
	})

	Describe("Response.Location", func() {
		It("sets HX-Location header", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Location("/other-page").Apply()
			Expect(w.Header().Get("HX-Location")).To(Equal("/other-page"))
		})
	})

	Describe("HTMXMiddleware", func() {
		It("passes through to the next handler", func() {
			called := false
			handler := cqrshtmx.HTMXMiddleware(
				http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
					called = true
				}),
			)
			serve(handler, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(called).To(BeTrue())
		})
	})

	Describe("Enforce error case", func() {
		It("returns error when casbin enforce fails", func() {
			e := newTestEnforcer()
			err := cqrshtmx.Enforce(e, "admin", "users", "create")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, cqrshtmx.ErrForbidden)).To(BeTrue())
		})
	})

	Describe("Query dispatch with no render", func() {
		It("returns 204 when no render and no HTMX options", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery()), r)
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})
	})

	Describe("Render function error", func() {
		It("calls error handler when render fails", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(func(_ http.ResponseWriter, _ *http.Request, _ any) error {
					return errors.New("render failed")
				}),
			), r)
			Expect(w.code()).ToNot(Equal(http.StatusOK))
		})
	})

	Describe("Command dispatch error", func() {
		It("maps handler errors through error handler", func() {
			app := newCommandAppWithHandler(func(_ context.Context, _ command.Command) error {
				return event.NewRejection("user.exists", "user already exists")
			})
			w := serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Query dispatch error", func() {
		It("maps query handler errors through error handler", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return nil, event.NewRejection("user.not_found", "user not found")
			})
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery()), r)
			Expect(w.code()).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Query handler with nil queries", func() {
		It("returns error when query dispatcher is nil", func() {
			app, _ := cqrshtmx.New(cqrshtmx.Config{Commands: command.NewDispatcher()})
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery()), r)
			Expect(w.code()).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("Command handler with nil commands", func() {
		It("returns error when command dispatcher is nil", func() {
			app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: query.NewDispatcher()})
			w := serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("SetTriggerWithDetail merge", func() {
		It("merges with existing JSON trigger data", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			resp := cqrshtmx.NewResponse(w, r)
			resp.TriggerWithDetail("evt1", map[string]string{"a": "1"})
			resp.TriggerWithDetail("evt2", map[string]string{"b": "2"})
			resp.Apply()
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("evt1"))
			Expect(trigger).To(ContainSubstring("evt2"))
		})
	})

	Describe("Query with trigger option", func() {
		It("sets HTMX trigger on query success with render", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return map[string]string{testNameKey: "Test"}, nil
			})
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
				cqrshtmx.Trigger("dataLoaded"),
			), r)
			Expect(w.Header().Get("HX-Trigger")).To(Equal("dataLoaded"))
		})
	})

	Describe("Notification HandlerOptions", func() {
		DescribeTable(
			"notification triggers on command success",
			func(opt cqrshtmx.HandlerOption, expectedLevel string) {
				testNotificationTrigger(opt, expectedLevel)
			},
			Entry("NotifySuccess", cqrshtmx.NotifySuccess("User created"), "success"),
			Entry("NotifyError", cqrshtmx.NotifyError("Something went wrong"), "error"),
			Entry("NotifyWarning", cqrshtmx.NotifyWarning("Check your input"), "warning"),
			Entry("NotifyInfo", cqrshtmx.NotifyInfo("Sync started"), "info"),
		)

		It("NotifyWithEvent uses custom event name", func() {
			app := newCommandAppWithHandler(noOpCommandHandler)
			r := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"/users",
				strings.NewReader(`{}`),
			)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.NotifyWithEvent("showToast").Success("User created"),
			), r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("showToast"))
			Expect(trigger).To(ContainSubstring("success"))
		})
	})

	Describe("Command with redirect and HTMX", func() {
		It("sets HTMX redirect for HTMX requests", func() {
			app := newCommandApp()
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Redirect("/users"),
			), newPostRequest("/users", `{}`, withHTMX))
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/users"))
		})
	})

	Describe("NotifyEventBuilder methods", func() {
		DescribeTable(
			"NotifyEventBuilder triggers notification",
			func(opt cqrshtmx.HandlerOption, level string) {
				testNotificationTrigger(opt, level)
			},
			Entry("Error", cqrshtmx.NotifyWithEvent("showToast").Error("Failed"), "error"),
			Entry("Warning", cqrshtmx.NotifyWithEvent("showToast").Warning("Careful"), "warning"),
			Entry("Info", cqrshtmx.NotifyWithEvent("showToast").Info("FYI"), "info"),
		)
	})

	Describe("DefaultErrorHandlerWithRedirect empty loginRedirect", func() {
		It("uses default /login when empty string is passed", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			cqrshtmx.DefaultErrorHandlerWithRedirect(w, r, cqrshtmx.ErrUnauthorized, "")
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/login"))
		})
	})

	Describe("setTriggerWithDetail fallback merge", func() {
		It("falls back to comma when existing header is not JSON", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			resp := cqrshtmx.NewResponse(w, r)
			w.Header().Set("HX-Trigger", "not-json")
			resp.TriggerWithDetail("newEvent", map[string]string{"x": "1"})
			resp.Apply()
			Expect(w.Header().Get("HX-Trigger")).To(ContainSubstring("newEvent"))
		})
	})

	Describe("NewUserID", func() {
		It("generates a non-zero user ID", func() {
			Expect(cqrshtmx.NewUserID().IsZero()).To(BeFalse())
		})

		It("generates unique IDs", func() {
			Expect(cqrshtmx.NewUserID()).NotTo(Equal(cqrshtmx.NewUserID()))
		})
	})

	Describe("WithTimeout", func() {
		It("overrides app-level timeout for a specific handler", func() {
			app := newCommandAppWithHandler(func(ctx context.Context, _ command.Command) error {
				<-ctx.Done()
				return ctx.Err()
			})
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.WithTimeout(50*time.Millisecond),
			), newPostRequest("/slow", `{}`))
			Expect(w.code()).To(Equal(http.StatusServiceUnavailable))
		})

		It("falls back to app timeout when zero", func() {
			var dispatched bool
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", trackingCommandHandler(&dispatched))
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp, Timeout: 5 * time.Second})
			Expect(err).NotTo(HaveOccurred())
			serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.WithTimeout(0),
			), newPostRequest("/fast", `{}`))
			Expect(dispatched).To(BeTrue())
		})
	})

	DescribeTable(
		"sanitizeRedirectURL edge cases",
		func(url string, expectRedirect bool) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Redirect(url).Apply()
			if expectRedirect {
				Expect(w.Code).To(Equal(http.StatusSeeOther))
			} else {
				Expect(w.Code).ToNot(Equal(http.StatusSeeOther))
			}
		},
		Entry("blocks javascript: URLs", "javascript:alert(1)", false),
		Entry("blocks absolute URLs with host", "https://evil.com", false),
		Entry("allows valid relative path", "/dashboard", true),
		Entry("allows root path", "/", true),
		Entry("blocks empty path", "", false),
		Entry("normalizes path with .. segments", "/a/../b/c", true),
	)

	Describe("decodeFormValues multi-value fields", func() {
		It("decodes form with multi-value fields into slice", func() {
			var receivedEmail string
			app := newCommandAppWithHandler(func(_ context.Context, _ command.Command) error {
				receivedEmail = "dispatched"
				return nil
			})
			form := url.Values{}
			form.Set("Tags", "go")
			form.Add("Tags", "htmx")
			r := httptest.NewRequest(http.MethodPost, "/tags", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			serve(app.Command(
				"CreateUser",
				cqrshtmx.DecodeForm(func(_ struct {
					Tags []string
				},
				) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			), r)
			Expect(receivedEmail).To(Equal("dispatched"))
		})

		It("returns error for form that cannot unmarshal to target type", func() {
			app, _ := cqrshtmx.New(cqrshtmx.Config{Commands: command.NewDispatcher()})
			form := url.Values{}
			form.Set("Count", "not-a-number")
			r := httptest.NewRequest(http.MethodPost, "/bad", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := serve(app.Command(
				"CreateUser",
				cqrshtmx.DecodeForm(func(_ struct{ Count int }) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			), r)
			Expect(w.code()).ToNot(Equal(http.StatusNoContent))
		})
	})

	Describe("Command dispatch with AfterDispatchHook error", func() {
		It("still returns success when AfterDispatchHook fails", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:      disp,
				AfterDispatch: func(_ context.Context, _ *http.Request, _ error) {},
			})
			Expect(err).NotTo(HaveOccurred())
			w := serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})
	})

	Describe("Query dispatch with PushURL", func() {
		It("sets HX-Push-URL on query success", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
				cqrshtmx.PushURL("/users/1"),
			), r)
			Expect(w.Header().Get("HX-Push-URL")).To(Equal("/users/1"))
		})
	})
})

var _ = Describe("CatalogEntries", func() {
	It("returns empty map from command dispatcher", func() {
		app := newCommandApp()
		entries := app.CommandCatalogEntries()
		Expect(entries).To(BeEmpty())
	})

	It("returns nil from query dispatcher when not configured", func() {
		app := newCommandApp()
		entries := app.QueryCatalogEntries()
		Expect(entries).To(BeNil())
	})

	It("returns empty map from query dispatcher", func() {
		qd := query.NewDispatcher()
		app, err := cqrshtmx.New(cqrshtmx.Config{Queries: qd})
		Expect(err).NotTo(HaveOccurred())
		entries := app.QueryCatalogEntries()
		Expect(entries).To(BeEmpty())
	})

	It("returns nil from command dispatcher when not configured", func() {
		qd := query.NewDispatcher()
		app, err := cqrshtmx.New(cqrshtmx.Config{Queries: qd})
		Expect(err).NotTo(HaveOccurred())
		entries := app.CommandCatalogEntries()
		Expect(entries).To(BeNil())
	})
})
