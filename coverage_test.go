package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockTemplComponent struct {
	html string
}

func (m *mockTemplComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.html))
	return err
}

var _ = Describe("Coverage Gaps", func() {
	Describe("RenderTempl", func() {
		It("renders a fixed templ component", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(ctx context.Context, q query.Query) (any, error) {
				return "irrelevant", nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			component := &mockTemplComponent{html: "<h1>Hello</h1>"}
			handler := app.Query("GetUser",
				cqrshtmx.DecodeJSONQuery(func(req testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
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
			_ = disp.Register("GetUser", func(ctx context.Context, q query.Query) (any, error) {
				return "Alice", nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				cqrshtmx.DecodeJSONQuery(func(req testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.RenderTemplResult(func(result string) cqrshtmx.TemplComponent {
					return &mockTemplComponent{html: "<p>" + result + "</p>"}
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
			_ = disp.Register("GetUser", func(ctx context.Context, q query.Query) (any, error) {
				return 42, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				cqrshtmx.DecodeJSONQuery(func(req testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.RenderTemplResult(func(result string) cqrshtmx.TemplComponent {
					return &mockTemplComponent{html: result}
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
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
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
				cqrshtmx.DecodeForm(func(req formReq) (command.Command, error) {
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
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			_, _ = e.AddPolicy("admin", "users", "create")
			err := cqrshtmx.Enforce(e, "admin", "users", "create")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Query dispatch with no render", func() {
		It("returns 204 when no render and no HTMX options", func() {
			disp := query.NewDispatcher()
			_ = disp.Register("GetUser", func(ctx context.Context, q query.Query) (any, error) {
				return "result", nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				cqrshtmx.DecodeJSONQuery(func(req testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
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
			_ = disp.Register("GetUser", func(ctx context.Context, q query.Query) (any, error) {
				return "result", nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				cqrshtmx.DecodeJSONQuery(func(req testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.Render(func(w http.ResponseWriter, r *http.Request, result any) error {
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
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return event.NewRejection("user.exists", "user already exists")
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
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
			_ = disp.Register("GetUser", func(ctx context.Context, q query.Query) (any, error) {
				return nil, event.NewRejection("user.not_found", "user not found")
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				cqrshtmx.DecodeJSONQuery(func(req testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
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
				cqrshtmx.DecodeJSONQuery(func(req testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
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
			_ = disp.Register("GetUser", func(ctx context.Context, q query.Query) (any, error) {
				return map[string]string{"name": "Test"}, nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Query("GetUser",
				cqrshtmx.DecodeJSONQuery(func(req testGetUserQuery) (query.Query, error) {
					return &testGetUserQuery{}, nil
				}),
				cqrshtmx.Render(func(w http.ResponseWriter, r *http.Request, result any) error {
					return json.NewEncoder(w).Encode(result)
				}),
				cqrshtmx.Trigger("dataLoaded"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			Expect(w.Header().Get("HX-Trigger")).To(Equal("dataLoaded"))
		})
	})

	Describe("Notification HandlerOptions", func() {
		It("NotifySuccess triggers notification on command success", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
				cqrshtmx.NotifySuccess("User created"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("success"))
			Expect(trigger).To(ContainSubstring("User created"))
		})

		It("NotifyError triggers notification", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
				cqrshtmx.NotifyError("Something went wrong"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("error"))
		})

		It("NotifyWarning triggers notification", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
				cqrshtmx.NotifyWarning("Check your input"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("warning"))
		})

		It("NotifyInfo triggers notification", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
					return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
				}),
				cqrshtmx.NotifyInfo("Sync started"),
			)

			body := `{}`
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
			r.Header.Set("HX-Request", "true")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			trigger := w.Header().Get("HX-Trigger")
			Expect(trigger).To(ContainSubstring("info"))
		})
	})

	Describe("Command with redirect and HTMX", func() {
		It("sets HTMX redirect for HTMX requests", func() {
			disp := command.NewDispatcher()
			_ = disp.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
				return nil
			})

			app, err := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
			Expect(err).NotTo(HaveOccurred())

			handler := app.Command("CreateUser",
				cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
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
})
