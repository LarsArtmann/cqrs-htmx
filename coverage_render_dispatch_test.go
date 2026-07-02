package cqrshtmx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Coverage Gaps - Render Dispatch and HTMX", func() {
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
				"Corruption to 500",
				event.NewCorruption("test.corruption", "corrupted data"),
				http.StatusInternalServerError,
			),
			Entry(
				"Infrastructure to 503",
				event.NewInfrastructure("test.infra", "infrastructure failure"),
				http.StatusServiceUnavailable,
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
			app := newQueryAppWithResult(testResultQueryHandler())
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery()), r)
			Expect(w.code()).To(Equal(http.StatusNoContent))
		})
	})

	Describe("Render function error", func() {
		It("calls error handler when render fails", func() {
			app := newQueryAppWithResult(testResultQueryHandler())
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
			Expect(w.code()).To(Equal(http.StatusServiceUnavailable))
		})
	})

	Describe("Command handler with nil commands", func() {
		It("returns error when command dispatcher is nil", func() {
			app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: query.NewDispatcher()})
			w := serve(app.Command("CreateUser", decodeCreateUserJSON()),
				newPostRequest("/users", `{}`))
			Expect(w.code()).To(Equal(http.StatusServiceUnavailable))
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
})
