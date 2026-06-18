package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Coverage Gaps - Rendering and Decoding", func() {
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
				cqrshtmx.DecodeForm(testCreateUserCommand),
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
})
