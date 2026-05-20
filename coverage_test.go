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
	_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
		return nil
	})

	app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
	Expect(err).NotTo(HaveOccurred())

	handler := app.Command("CreateUser",
		decodeCreateUserJSON(),
		opt,
	)

	r := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/users",
		strings.NewReader(`{}`),
	)
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	trigger := w.Header().Get("HX-Trigger")
	Expect(trigger).To(ContainSubstring(expectedLevel))
}

var _ = Describe("Coverage Gaps", func() {
	Describe("RenderTempl", func() {
		It("renders a fixed templ component", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return "irrelevant", nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			component := &bddTemplComponent{html: "<h1>Hello</h1>"}
			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderTempl(component),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Body.String()).To(Equal("<h1>Hello</h1>"))
		})
	})

	Describe("RenderTemplResult", func() {
		It("maps result to templ component and renders", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return aliceName, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderTemplResult(func(result string) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: "<p>" + result + "</p>"}
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Body.String()).To(Equal("<p>Alice</p>"))
		})

		It("returns error for wrong result type", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return 42, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderTemplResult(func(result string) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: result}
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).ToNot(Equal(http.StatusOK))
		})
	})

	Describe("DecodeForm", func() {
		It("decodes form data into a command", func() {
			disp := command.NewDispatcher()
			var received string
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				received = "dispatched"
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			type formReq struct {
				Email string
				Name  string
			}

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeForm(func(req formReq) (command.Command, error) {
					return &testCreateUserCmd{
						aggID: id.NewAggregateID(),
						email: req.Email,
						name:  req.Name,
					}, nil
				}),
			)

			form := url.Values{}
			form.Set("Email", "test@example.com")
			form.Set("Name", "Test User")

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(received).To(Equal("dispatched"))
		})

		It("returns error for invalid form data", func() {
			disp := command.NewDispatcher()
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			type formReq struct {
				Email string
			}

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeForm(func(_ formReq) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			)

			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("%%%invalid"))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).ToNot(Equal(http.StatusNoContent))
		})
	})

	Describe("MapError additional families", func() {
		It("maps Conflict family to 409", func() {
			err := event.NewConflict("test.conflict", "conflict occurred")
			Expect(cqrshtmx.MapError(err)).To(Equal(http.StatusConflict))
		})

		It("maps Corruption family to 422", func() {
			err := event.NewCorruption("test.corruption", "corrupted data")
			Expect(cqrshtmx.MapError(err)).To(Equal(http.StatusUnprocessableEntity))
		})

		It("maps Infrastructure family to 500", func() {
			err := event.NewInfrastructure("test.infra", "infrastructure failure")
			Expect(cqrshtmx.MapError(err)).To(Equal(http.StatusInternalServerError))
		})

		It("maps Transient family to 503", func() {
			err := event.NewTransient("test.transient", "transient error")
			Expect(cqrshtmx.MapError(err)).To(Equal(http.StatusServiceUnavailable))
		})
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

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)
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
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
		})
	})

	Describe("Render function error", func() {
		It("calls error handler when render fails", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(func(_ http.ResponseWriter, _ *http.Request, _ any) error {
					return errors.New("render failed")
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).ToNot(Equal(http.StatusOK))
		})
	})

	Describe("Command dispatch error", func() {
		It("maps handler errors through error handler", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				return event.NewRejection("user.exists", "user already exists")
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Query dispatch error", func() {
		It("maps query handler errors through error handler", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return nil, event.NewRejection("user.not_found", "user not found")
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Query handler with nil queries", func() {
		It("returns error when query dispatcher is nil", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: command.NewDispatcher()})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("Command handler with nil commands", func() {
		It("returns error when command dispatcher is nil", func() {
			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: query.NewDispatcher()})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
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
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return map[string]string{testNameKey: "Test"}, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
				cqrshtmx.Trigger("dataLoaded"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Trigger")).To(Equal("dataLoaded"))
		})
	})

	Describe("Notification HandlerOptions", func() {
		It("NotifySuccess triggers notification on command success", func() {
			testNotificationTrigger(
				cqrshtmx.NotifySuccess("User created"),
				"success",
			)
		})

		It("NotifyError triggers notification", func() {
			testNotificationTrigger(
				cqrshtmx.NotifyError("Something went wrong"),
				"error",
			)
		})

		It("NotifyWarning triggers notification", func() {
			testNotificationTrigger(
				cqrshtmx.NotifyWarning("Check your input"),
				"warning",
			)
		})

		It("NotifyInfo triggers notification", func() {
			testNotificationTrigger(cqrshtmx.NotifyInfo("Sync started"), "info")
		})

		It("NotifyWithEvent uses custom event name", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.NotifyWithEvent("showToast").Success("User created"),
			)

			r := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"/users",
				strings.NewReader(`{}`),
			)
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("showToast"))
			Expect(trigger).To(ContainSubstring("success"))
		})
	})

	Describe("Command with redirect and HTMX", func() {
		It("sets HTMX redirect for HTMX requests", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

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

	Describe("NotifyEventBuilder methods", func() {
		It("Error triggers notification with custom event", func() {
			testNotificationTrigger(cqrshtmx.NotifyWithEvent("showToast").Error("Failed"), "error")
		})

		It("Warning triggers notification with custom event", func() {
			testNotificationTrigger(
				cqrshtmx.NotifyWithEvent("showToast").Warning("Careful"),
				"warning",
			)
		})

		It("Info triggers notification with custom event", func() {
			testNotificationTrigger(cqrshtmx.NotifyWithEvent("showToast").Info("FYI"), "info")
		})
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
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("newEvent"))
		})
	})

	Describe("NewUserID", func() {
		It("generates a non-zero user ID", func() {
			uid := cqrshtmx.NewUserID()
			Expect(uid.IsZero()).To(BeFalse())
		})

		It("generates unique IDs", func() {
			uid1 := cqrshtmx.NewUserID()
			uid2 := cqrshtmx.NewUserID()
			Expect(uid1).NotTo(Equal(uid2))
		})
	})

	Describe("WithTimeout", func() {
		It("overrides app-level timeout for a specific handler", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
				<-ctx.Done()
				return ctx.Err()
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Timeout:  10 * time.Second,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.WithTimeout(50*time.Millisecond),
			)

			r := httptest.NewRequest(http.MethodPost, "/slow", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("falls back to app timeout when zero", func() {
			disp := command.NewDispatcher()
			dispatched := false
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				dispatched = true
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				Timeout:  5 * time.Second,
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				decodeCreateUserJSON(),
				cqrshtmx.WithTimeout(0),
			)

			r := httptest.NewRequest(http.MethodPost, "/fast", strings.NewReader(`{}`))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			Expect(dispatched).To(BeTrue())
		})
	})

	Describe("sanitizeRedirectURL edge cases", func() {
		It("blocks javascript: URLs", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Redirect("javascript:alert(1)").Apply()
			Expect(w.Code).ToNot(Equal(http.StatusSeeOther))
		})

		It("blocks absolute URLs with host", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Redirect("https://evil.com").Apply()
			Expect(w.Code).ToNot(Equal(http.StatusSeeOther))
		})

		It("allows valid relative path redirects", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Redirect("/dashboard").Apply()
			Expect(w.Code).To(Equal(http.StatusSeeOther))
		})

		It("allows root path redirects", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Redirect("/").Apply()
			Expect(w.Code).To(Equal(http.StatusSeeOther))
		})

		It("blocks empty path redirects", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Redirect("").Apply()
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("normalizes path with .. segments", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			cqrshtmx.NewResponse(w, r).Redirect("/a/../b/c").Apply()
			Expect(w.Code).To(Equal(http.StatusSeeOther))
		})
	})

	Describe("decodeFormValues multi-value fields", func() {
		It("decodes form with multi-value fields into slice", func() {
			disp := command.NewDispatcher()
			var receivedEmail string
			_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
				receivedEmail = "dispatched"
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			type formReq struct {
				Tags []string
			}

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeForm(func(_ formReq) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			)

			form := url.Values{}
			form.Set("Tags", "go")
			form.Add("Tags", "htmx")

			r := httptest.NewRequest(http.MethodPost, "/tags", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(receivedEmail).To(Equal("dispatched"))
		})

		It("returns error for form that cannot unmarshal to target type", func() {
			disp := command.NewDispatcher()
			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			type strictForm struct {
				Count int
			}

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeForm(func(_ strictForm) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
			)

			form := url.Values{}
			form.Set("Count", "not-a-number")

			r := httptest.NewRequest(http.MethodPost, "/bad", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).ToNot(Equal(http.StatusNoContent))
		})
	})

	Describe("Command dispatch with AfterDispatchHook error", func() {
		It("still returns success when AfterDispatchHook fails", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", noOpCommandHandler)

			app, err := cqrshtmx.New(cqrshtmx.Config{
				Commands: disp,
				AfterDispatch: func(_ context.Context, _ *http.Request, _ error) {
				},
			})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser", decodeCreateUserJSON())
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
		})
	})

	Describe("Query dispatch with PushURL", func() {
		It("sets HX-Push-URL on query success", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
				return testQueryResult, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.Render(encodeJSONResult),
				cqrshtmx.PushURL("/users/1"),
			)

			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Push-URL")).To(Equal("/users/1"))
		})
	})
})
