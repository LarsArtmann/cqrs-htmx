package cqrshtmx_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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

	Describe("RenderJSON", func() {
		It("renders query result as JSON with 200", func() {
			app := newQueryAppWithResult(queryNamedResultHandler(aliceName))
			r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderJSON[map[string]string](),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal(cqrshtmx.ContentTypeJSON))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
		})

		It("returns error for mismatched result type", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return 42, nil
			})
			r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderJSON[string](),
			), r)
			Expect(w.code()).ToNot(Equal(http.StatusOK))
		})
	})

	Describe("RenderJSONStatus", func() {
		It("renders query result as JSON with custom status", func() {
			app := newQueryAppWithResult(queryNamedResultHandler(aliceName))
			r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderJSONStatus[map[string]string](http.StatusCreated),
			), r)
			Expect(w.code()).To(Equal(http.StatusCreated))
			Expect(w.Header().Get("Content-Type")).To(Equal(cqrshtmx.ContentTypeJSON))
		})
	})

	Describe("DecodePagination", func() {
		DescribeTable(
			"decodes pagination from query params",
			func(query string, wantPage, wantSize uint) {
				r := httptest.NewRequest(http.MethodGet, "/items"+query, nil)
				p := cqrshtmx.DecodePagination(r)
				Expect(p.Page).To(Equal(wantPage))
				Expect(p.PageSize).To(Equal(wantSize))
			},
			Entry("extracts page and page_size", "?page=2&page_size=50", uint(2), uint(50)),
			Entry("applies defaults for missing params", "", uint(1), uint(20)),
			Entry("defaults on invalid values", "?page=abc&page_size=-1", uint(1), uint(20)),
		)

		It("clamps page_size to max 100", func() {
			r := httptest.NewRequest(http.MethodGet, "/items?page_size=500", nil)
			p := cqrshtmx.DecodePagination(r)
			Expect(p.PageSize).To(Equal(uint(100)))
		})
	})

	Describe("RenderPaginatedJSON", func() {
		It("renders paginated result as JSON", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return query.NewPaginatedResult(
					[]string{aliceName, "Bob"},
					42,
					query.NewPagination(2, 20),
				), nil
			})

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"ListUsers",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPaginatedJSON[string](),
			), r)

			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal(cqrshtmx.ContentTypeJSON))
			Expect(w.Body.String()).To(ContainSubstring(aliceName))
			Expect(w.Body.String()).To(ContainSubstring("42"))
		})

		It("returns error for mismatched result type", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return "not a paginated result", nil
			})

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"ListUsers",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPaginatedJSON[string](),
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
			app := newQueryAppWithResult(queryNamedResultHandler("Test"))
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
			assertHTMXErrorRedirect(cqrshtmx.ErrUnauthorized, "", "/login")
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
		"sanitizeRedirectURL",
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
		Entry("blocks data: URLs", "data:text/html,<script>alert(1)</script>", false),
		Entry("blocks scheme-relative URLs", "//evil.com", false),
		Entry("blocks unparseable URLs", "://\x00bad", false),
		Entry("blocks fragment-only URLs", "#section", false),
		Entry("blocks userinfo URLs", "http://user@host", false),
		Entry("blocks query-only URLs", "?foo=bar", false),
		Entry("blocks escape above root", "/../../../etc/passwd", false),
		Entry("allows legitimate normalization", "/a/../b/c", true),
		Entry("blocks deep traversal", "/../../../../../etc/passwd", false),
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

var _ = Describe("Root Coverage Gaps", func() {
	Describe("WriteJSON error path", func() {
		It("returns error when encoder fails", func() {
			w := httptest.NewRecorder()
			err := cqrshtmx.WriteJSON(w, http.StatusOK, make(chan int))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("MapError nil input", func() {
		It("returns 500 for nil error", func() {
			Expect(cqrshtmx.MapError(nil)).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("MapError unknown family", func() {
		It("returns 500 for errors with no classification", func() {
			Expect(cqrshtmx.MapError(
				event.NewInfrastructure("test.unknown", "unknown"),
			)).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("Enforce nil enforcer", func() {
		It("returns ErrEnforcerNil when enforcer is nil", func() {
			err := cqrshtmx.Enforce(nil, "user1", "resource", "read")
			Expect(errors.Is(err, cqrshtmx.ErrEnforcerNil)).To(BeTrue())
		})
	})

	Describe("handleCommandDispatch auth denied", func() {
		It("rejects when authorization fails", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			enforcer := newTestEnforcer()
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Enforcer: enforcer,
			})
			Expect(err).NotTo(HaveOccurred())

			r := newPostRequest("/users", `{}`)
			r = r.WithContext(context.Background())
			w := serve(app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.Authorize("users", "create"),
			), r)
			Expect(w.code()).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("applyQueryResponse with HTMX response option", func() {
		It("applies HTMX headers on query success", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Trigger("dataLoaded"),
			), r)
			Expect(w.Header().Get("HX-Trigger")).To(Equal("dataLoaded"))
		})
	})

	Describe("RateLimiter eviction", func() {
		It("evicts oldest entry when MaxKeys exceeded", func() {
			handler := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        100,
				Window:       time.Second,
				MaxKeys:      2,
				KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
			})

			called := 0
			next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called++
			})

			for i := range 5 {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = fmt.Sprintf("192.168.1.%d:1234", i%3)
				w := httptest.NewRecorder()
				handler(next).ServeHTTP(w, r)
			}
			Expect(called).To(Equal(5))
		})

		It("exempts requests with empty key", func() {
			handler := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
				Limit:        1,
				Window:       time.Second,
				KeyExtractor: func(_ *http.Request) string { return "" },
			})

			called := 0
			next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				called++
			})

			for range 5 {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				w := httptest.NewRecorder()
				handler(next).ServeHTTP(w, r)
			}
			Expect(called).To(Equal(5))
		})
	})

	Describe("StatusRecorder Hijack non-Hijacker fallback", func() {
		It("returns ErrNotSupported when underlying writer has no Hijacker", func() {
			rec := httptest.NewRecorder()
			sr := cqrshtmx.NewStatusRecorder(rec)
			_, _, err := sr.Hijack()
			Expect(err).To(Equal(http.ErrNotSupported))
		})
	})

	Describe("CSRF sameSite all branches", func() {
		It("maps SameSiteDefaultMode", func() {
			cfg := cqrshtmx.CSRFConfig{
				SameSite: http.SameSiteDefaultMode,
			}
			mw := cqrshtmx.CSRFMiddleware(cfg)
			handler := mw(okHandler())
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("csrfTokenFromRequest context fallback", func() {
		It("falls back to context token when gorilla has none", func() {
			ctx := cqrshtmx.WithCSRFToken(context.Background(), "ctx-token")
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			token := cqrshtmx.CSRFTokenFromContext(r.Context())
			Expect(token).To(Equal("ctx-token"))
		})
	})

	Describe("Enforce with enforcer error", func() {
		It("wraps error from enforcer", func() {
			enforcer := newFailingEnforcer(errors.New("internal error"))
			err := cqrshtmx.Enforce(enforcer, "user1", "resource", "read")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("casbin enforce failed"))
			Expect(err.Error()).To(ContainSubstring("internal error"))
		})
	})

	Describe("applyQueryResponse render error", func() {
		It("calls error handler when render fails", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})
			renderErr := errors.New("render failed")
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(func(_ http.ResponseWriter, _ *http.Request, _ any) error {
					return renderErr
				}),
			), r)
			Expect(w.code()).To(Equal(http.StatusServiceUnavailable))
		})
	})

	Describe("StatusRecorder Push with actual pusher error", func() {
		It("wraps error from underlying Pusher", func() {
			w := newPusherRecorder(&mockPusher{
				ResponseWriter: httptest.NewRecorder(),
				pushFunc: func(_ string, _ *http.PushOptions) error {
					return errors.New("push failed")
				},
			})
			sr := cqrshtmx.NewStatusRecorder(w)
			err := sr.Push("/target", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("push failed"))
		})
	})

	Describe("MapError conflict family", func() {
		It("returns 409 for conflict family errors", func() {
			err := fmt.Errorf("wrap: %w", event.NewConflict("test.conflict", "conflict occurred"))
			status := cqrshtmx.MapError(err)
			Expect(status).To(Equal(http.StatusConflict))
		})
	})

	Describe("MapError transient family", func() {
		It("returns 503 for transient family errors", func() {
			err := fmt.Errorf("wrap: %w", event.NewTransient("test.transient", "temporary failure"))
			status := cqrshtmx.MapError(err)
			Expect(status).To(Equal(http.StatusServiceUnavailable))
		})
	})

	Describe("MapError ErrRequestTooLarge", func() {
		It("returns 413 for body size exceeded", func() {
			status := cqrshtmx.MapError(cqrshtmx.ErrRequestTooLarge)
			Expect(status).To(Equal(http.StatusRequestEntityTooLarge))
		})
	})

	Describe("IsAuthenticated helper", func() {
		It("returns false when no user ID in context", func() {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			Expect(cqrshtmx.IsAuthenticated(r)).To(BeFalse())
		})

		It("returns true when user ID is in context", func() {
			uid := cqrshtmx.NewUserID()
			ctx := cqrshtmx.WithUserID(context.Background(), uid)
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			Expect(cqrshtmx.IsAuthenticated(r)).To(BeTrue())
		})
	})

	Describe("MustNew convenience function", func() {
		It("panics on invalid config", func() {
			Expect(func() { cqrshtmx.MustNew(cqrshtmx.Config{}) }).To(Panic())
		})

		It("returns app on valid config", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(app).NotTo(BeNil())
		})
	})

	Describe("HasCommands and HasQueries", func() {
		It("reports correctly for command-only app", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(app.HasCommands()).To(BeTrue())
			Expect(app.HasQueries()).To(BeFalse())
		})

		It("reports correctly for query-only app", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: query.NewDispatcher()})
			Expect(app.HasCommands()).To(BeFalse())
			Expect(app.HasQueries()).To(BeTrue())
		})
	})

	Describe("Empty type validation", func() {
		It("panics on empty command type", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(func() { app.Command("") }).To(PanicWith(MatchRegexp("command type must not be empty")))
		})

		It("panics on empty query type", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: query.NewDispatcher()})
			Expect(func() { app.Query("") }).To(PanicWith(MatchRegexp("query type must not be empty")))
		})

		It("panics on empty command type even with query-only app", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: query.NewDispatcher()})
			Expect(func() { app.Command("") }).To(PanicWith(MatchRegexp("command type must not be empty")))
		})

		It("panics on empty query type even with command-only app", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(func() { app.Query("") }).To(PanicWith(MatchRegexp("query type must not be empty")))
		})
	})

	Describe("Response builder enhancements", func() {
		Describe("Status", func() {
			It("defers the status code to Apply()", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				resp := cqrshtmx.NewResponse(w, r)
				result := resp.Status(http.StatusCreated)
				Expect(result).To(Equal(resp))
				Expect(w.Code).To(Equal(http.StatusOK)) // not written yet
				resp.Apply()
				Expect(w.Code).To(Equal(http.StatusCreated))
			})

			It("allows Status then Redirect without breaking the chain", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				resp := cqrshtmx.NewResponse(w, r)
				resp.Status(http.StatusCreated).Redirect("/users").Apply()
				Expect(w.Code).To(Equal(http.StatusSeeOther))
				Expect(w.Header().Get("Location")).To(Equal("/users"))
			})
		})

		Describe("Header", func() {
			It("sets a custom header", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				resp := cqrshtmx.NewResponse(w, r)
				result := resp.Header("X-Custom", "value")
				Expect(result).To(Equal(resp))
				Expect(w.Header().Get("X-Custom")).To(Equal("value"))
			})
		})

		Describe("ContentType", func() {
			It("sets Content-Type header", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				resp := cqrshtmx.NewResponse(w, r)
				result := resp.ContentType("text/xml")
				Expect(result).To(Equal(resp))
				Expect(w.Header().Get("Content-Type")).To(Equal("text/xml"))
			})
		})

		Describe("JSON", func() {
			It("encodes and writes JSON body", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				cqrshtmx.NewResponse(w, r).JSON(map[string]string{"s": "ok"})
				Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
				Expect(w.Body.String()).To(ContainSubstring(`"s":"ok"`))
			})
		})

		Describe("WriteString", func() {
			It("writes string body", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				cqrshtmx.NewResponse(w, r).WriteString("hello world")
				Expect(w.Body.String()).To(Equal("hello world"))
			})
		})

		Describe("Body", func() {
			It("writes byte body", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				result := cqrshtmx.NewResponse(w, r).Body([]byte("raw bytes"))
				Expect(result).NotTo(BeNil())
				Expect(w.Body.String()).To(Equal("raw bytes"))
			})
		})

		Describe("WriteString", func() {
			It("writes via WriteString when StringWriter is available", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				cqrshtmx.NewResponse(w, r).WriteString("hello world")
				Expect(w.Body.String()).To(Equal("hello world"))
			})

			It("writes via Write when StringWriter is not available", func() {
				rec := httptest.NewRecorder()
				w := &nonStringWriter{ResponseWriter: rec, recorder: rec}
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				cqrshtmx.NewResponse(w, r).WriteString("fallback")
				Expect(w.recorder.Body.String()).To(Equal("fallback"))
			})
		})

		Describe("JSON", func() {
			It("encodes and writes JSON body", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				cqrshtmx.NewResponse(w, r).JSON(map[string]string{"s": "ok"})
				Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
				Expect(w.Body.String()).To(ContainSubstring(`"s":"ok"`))
			})

			It("returns 500 on marshal error", func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				cqrshtmx.NewResponse(w, r).JSON(make(chan int))
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("WithMaxBodySize HandlerOption", func() {
		It("allows per-handler override of max body size", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:    disp,
				MaxBodySize: 1,
			})
			Expect(err).NotTo(HaveOccurred())

			smallBody := `{"email":"test@test.com"}`
			handler := app.Command("CreateUser", decodeBDDCreateUserJSONWithBody(), cqrshtmx.WithMaxBodySize(1024))
			w := serve(handler, newPostRequest("/users", smallBody))
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})
	})

	Describe("WithSuccessStatus HandlerOption", func() {
		It("returns custom success status code", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON(), cqrshtmx.WithSuccessStatus(http.StatusCreated))
			w := serve(handler, newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusCreated))
		})
	})

	Describe("OnError HandlerOption", func() {
		It("calls per-handler error callback on failure", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			var capturedErr error
			handler := app.Command(
				"CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.OnError(func(_ *http.Request, err error) { capturedErr = err }),
			)
			w := serve(handler, newPostRequest("/users", `{invalid json`))
			Expect(w.code()).To(Equal(http.StatusBadRequest))
			Expect(capturedErr).To(HaveOccurred())
		})
	})

	Describe("DefaultMaxBodySize", func() {
		It("has a reasonable default", func() {
			Expect(cqrshtmx.DefaultMaxBodySize).To(BeNumerically(">", 0))
		})
	})

	Describe("NewRateLimiter monitoring", func() {
		It("reports active keys", func() {
			rl := cqrshtmx.NewRateLimiter(cqrshtmx.RateLimiterConfig{
				Limit:        100,
				Window:       time.Second,
				KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
			})
			Expect(rl.ActiveKeys()).To(Equal(0))

			mw := rl.Middleware()
			next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = "1.2.3.4:1234"
			mw(next).ServeHTTP(httptest.NewRecorder(), r)
			Expect(rl.ActiveKeys()).To(Equal(1))
		})
	})

	Describe("Response JSON marshal error", func() {
		It("handles non-encodable values gracefully", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			resp := cqrshtmx.NewResponse(w, r)
			result := resp.JSON(make(chan int))
			Expect(result).To(Equal(resp))
		})
	})

	Describe("sanitizeRedirectURL traversal blocks", func() {
		DescribeTable(
			"blocks traversal",
			func(input string, expected bool) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				cqrshtmx.NewResponse(w, r).Redirect(input).Apply()
				if expected {
					Expect(w.Code).To(Equal(http.StatusSeeOther))
				} else {
					Expect(w.Code).ToNot(Equal(http.StatusSeeOther))
				}
			},
			Entry("blocks escape above root", "/../../../etc/passwd", false),
			Entry("allows legitimate normalization", "/a/../b/c", true),
			Entry("blocks deep traversal", "/../../../../../etc/passwd", false),
		)
	})

	Describe("WithSuccessStatus for query", func() {
		It("returns custom status for query success without body", func() {
			app := newQueryAppWithResult(func(_ context.Context, _ query.Query) (any, error) {
				return "ok", nil
			})
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.WithSuccessStatus(http.StatusOK),
			), r)
			Expect(w.code()).To(Equal(http.StatusOK))
		})
	})

	Describe("RequireMethod HandlerOption", func() {
		It("rejects wrong HTTP method with 405", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON(), cqrshtmx.RequireMethod(http.MethodPost))
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(handler, r)
			Expect(w.code()).To(Equal(http.StatusMethodNotAllowed))
		})

		It("allows correct HTTP method", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON(), cqrshtmx.RequireMethod(http.MethodPost))
			w := serve(handler, newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})
	})

	Describe("HX-Redirect sanitization", func() {
		It("sanitizes redirect for HTMX requests", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			cqrshtmx.NewResponse(w, r).Redirect("/../../etc/passwd")
			Expect(w.Header().Get("HX-Redirect")).To(BeEmpty())
		})

		It("allows safe redirect for HTMX requests", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			cqrshtmx.NewResponse(w, r).Redirect("/users")
			Expect(w.Header().Get("HX-Redirect")).To(Equal("/users"))
		})
	})

	Describe("Recommended security constants", func() {
		It("provides recommended HSTS value", func() {
			Expect(cqrshtmx.RecommendedHSTS).To(ContainSubstring("max-age="))
		})

		It("provides recommended CSP value", func() {
			Expect(cqrshtmx.RecommendedCSP).To(ContainSubstring("default-src"))
		})

		It("applies recommended HSTS in security middleware", func() {
			handler := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
				StrictTransportSecurity: cqrshtmx.RecommendedHSTS,
			})(okHandler())

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("Strict-Transport-Security")).To(Equal(cqrshtmx.RecommendedHSTS))
		})
	})

	Describe("X-Request-ID response header propagation", func() {
		It("sets X-Request-ID in response when generated", func() {
			middleware := cqrshtmx.ContextEnrichmentMiddleware(nil)
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			middleware(next).ServeHTTP(w, r)
			Expect(w.Header().Get("X-Request-ID")).NotTo(BeEmpty())
		})

		It("propagates provided X-Request-ID in response", func() {
			middleware := cqrshtmx.ContextEnrichmentMiddleware(nil)
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("X-Request-ID", "01HK1549P84T9XF8R94E960633")
			middleware(next).ServeHTTP(w, r)
			Expect(w.Header().Get("X-Request-ID")).To(Equal("01HK1549P84T9XF8R94E960633"))
		})
	})

	Describe("ErrMethodNotAllowed mapping", func() {
		It("maps to 405", func() {
			Expect(cqrshtmx.MapError(cqrshtmx.ErrMethodNotAllowed)).To(Equal(http.StatusMethodNotAllowed))
		})
	})

	Describe("enrichUserID logs extractor errors", func() {
		It("continues when extractor returns error", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)
			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands:        disp,
				UserIDExtractor: func(_ *http.Request) (id.UserID, error) { return id.UserID{}, errors.New("broken") },
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())
			w := serve(handler, newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})
	})

	Describe("HealthHandler", func() {
		It("returns 200 when dispatchers are configured", func() {
			app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/health", nil)
			app.HealthHandler().ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("ok"))
		})
	})
})

// nonStringWriter wraps an http.ResponseWriter without implementing
// io.StringWriter, forcing WriteString to fall back to Write.
type nonStringWriter struct {
	http.ResponseWriter
	recorder *httptest.ResponseRecorder
}

func (n *nonStringWriter) Write(p []byte) (int, error) {
	return n.recorder.Write(p)
}

func (n *nonStringWriter) Header() http.Header {
	return n.recorder.Header()
}

func (n *nonStringWriter) WriteHeader(code int) {
	n.recorder.WriteHeader(code)
}
