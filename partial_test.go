package cqrshtmx_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// errTemplComponent is a TemplComponent that always fails to render.
type errTemplComponent struct{}

func (errTemplComponent) Render(_ context.Context, _ io.Writer) error {
	return errors.New("disk full")
}

var _ = Describe("Partial Rendering", func() {
	Describe("RenderPartialOrFull", func() {
		It("renders the partial component for HTMX requests", func() {
			app := newQueryAppWithResult(constantQueryHandler(aliceName))
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("HX-Request", "true")
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPartialOrFull(
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: "<partial>" + result + "</partial>"}
					},
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: "<full>" + result + "</full>"}
					},
				),
			), r)
			Expect(w.Body.String()).To(Equal("<partial>Alice</partial>"))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/html; charset=utf-8"))
		})

		It("renders the full component for non-HTMX requests", func() {
			app := newQueryAppWithResult(constantQueryHandler(aliceName))
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPartialOrFull(
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: "<partial>" + result + "</partial>"}
					},
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: "<full>" + result + "</full>"}
					},
				),
			), r)
			Expect(w.Body.String()).To(Equal("<full>Alice</full>"))
		})

		It("renders the full component for HTMX history-restore requests", func() {
			app := newQueryAppWithResult(constantQueryHandler(aliceName))
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-History-Restore-Request", "true")
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPartialOrFull(
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: "<partial>" + result + "</partial>"}
					},
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: "<full>" + result + "</full>"}
					},
				),
			), r)
			Expect(w.Body.String()).To(Equal("<full>Alice</full>"))
		})

		It("returns an error for mismatched result type", func() {
			app := newQueryAppWithResult(constantQueryHandler(42))
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPartialOrFull(
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: result}
					},
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: result}
					},
				),
			), r)
			Expect(w.code()).NotTo(Equal(http.StatusOK))
		})

		It("returns an error for nil result", func() {
			app := newQueryAppWithResult(constantQueryHandler(nil))
			r := httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPartialOrFull(
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: result}
					},
					func(result string) cqrshtmx.TemplComponent {
						return &bddTemplComponent{html: result}
					},
				),
			), r)
			Expect(w.code()).NotTo(Equal(http.StatusOK))
		})
	})

	Describe("RenderPartialOrFullFunc", func() {
		It("renders the partial render function for HTMX requests", func() {
			app := newQueryAppWithResult(constantQueryHandler(aliceName))
			r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader(`{}`))
			r.Header.Set("HX-Request", "true")
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPartialOrFullFunc(
					func(w http.ResponseWriter, _ *http.Request, result any) error {
						s, _ := result.(string)
						_, err := w.Write([]byte("<p>" + s + "</p>"))
						return err
					},
					func(w http.ResponseWriter, _ *http.Request, result any) error {
						s, _ := result.(string)
						_, err := w.Write([]byte("<html>" + s + "</html>"))
						return err
					},
				),
			), r)
			Expect(w.Body.String()).To(Equal("<p>Alice</p>"))
		})

		It("renders the full render function for non-HTMX requests", func() {
			app := newQueryAppWithResult(constantQueryHandler(aliceName))
			r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader(`{}`))
			w := serve(app.Query(
				"GetUser",
				decodeGetUserJSONQuery(),
				cqrshtmx.RenderPartialOrFullFunc(
					func(w http.ResponseWriter, _ *http.Request, result any) error {
						s, _ := result.(string)
						_, err := w.Write([]byte("<p>" + s + "</p>"))
						return err
					},
					func(w http.ResponseWriter, _ *http.Request, result any) error {
						s, _ := result.(string)
						_, err := w.Write([]byte("<html>" + s + "</html>"))
						return err
					},
				),
			), r)
			Expect(w.Body.String()).To(Equal("<html>Alice</html>"))
		})
	})

	Describe("RenderIf", func() {
		DescribeTable("selects render function based on predicate",
			func(target string, expectedBody string) {
				app := newQueryAppWithResult(constantQueryHandler(aliceName))
				r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader(`{}`))
				r.Header.Set("HX-Target", target)
				w := serve(app.Query(
					"GetUser",
					decodeGetUserJSONQuery(),
					cqrshtmx.RenderIf(
						func(r *http.Request) bool { return cqrshtmx.HTMXTarget(r) == "#avatar" },
						func(w http.ResponseWriter, _ *http.Request, result any) error {
							s, _ := result.(string)
							_, err := w.Write([]byte("<avatar>" + s + "</avatar>"))
							return err
						},
						func(w http.ResponseWriter, _ *http.Request, result any) error {
							s, _ := result.(string)
							_, err := w.Write([]byte("<full>" + s + "</full>"))
							return err
						},
					),
				), r)
				Expect(w.Body.String()).To(Equal(expectedBody))
			},
			Entry("match when check returns true", "#avatar", "<avatar>Alice</avatar>"),
			Entry("noMatch when check returns false", "#other", "<full>Alice</full>"),
		)
	})

	Describe("RenderTemplComponent (standalone)", func() {
		It("renders the partial component for HTMX requests", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/items", nil)
			r.Header.Set("HX-Request", "true")
			err := cqrshtmx.RenderTemplComponent(w, r,
				&bddTemplComponent{html: "<tr><td>partial</td></tr>"},
				&bddTemplComponent{html: "<table><tr><td>full</td></tr></table>"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Body.String()).To(Equal("<tr><td>partial</td></tr>"))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/html; charset=utf-8"))
		})

		It("renders the full component for non-HTMX requests", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/items", nil)
			err := cqrshtmx.RenderTemplComponent(w, r,
				&bddTemplComponent{html: "<tr><td>partial</td></tr>"},
				&bddTemplComponent{html: "<table><tr><td>full</td></tr></table>"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Body.String()).To(Equal("<table><tr><td>full</td></tr></table>"))
		})

		It("renders the full component for HTMX history-restore requests", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/items", nil)
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-History-Restore-Request", "true")
			err := cqrshtmx.RenderTemplComponent(w, r,
				&bddTemplComponent{html: "<tr><td>partial</td></tr>"},
				&bddTemplComponent{html: "<table><tr><td>full</td></tr></table>"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Body.String()).To(Equal("<table><tr><td>full</td></tr></table>"))
		})

		It("returns the error when Render fails", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/items", nil)
			r.Header.Set("HX-Request", "true")
			err := cqrshtmx.RenderTemplComponent(w, r,
				errTemplComponent{},
				&bddTemplComponent{html: "full"},
			)
			Expect(err).To(MatchError("disk full"))
		})
	})

	Describe("OOBHTML", func() {
		It("wraps HTML with default swap strategy", func() {
			result := cqrshtmx.OOBHTML("counter", "<span>3</span>")
			Expect(result).To(Equal(`<div id="counter" hx-swap-oob="true"><span>3</span></div>`))
		})

		It("wraps HTML with a custom swap strategy", func() {
			result := cqrshtmx.OOBHTML("list", "<li>item</li>", cqrshtmx.SwapBeforeEnd)
			Expect(result).To(Equal(`<div id="list" hx-swap-oob="beforeend"><li>item</li></div>`))
		})

		It("passes through HTML that already contains hx-swap-oob", func() {
			html := `<div id="x" hx-swap-oob="true">already tagged</div>`
			result := cqrshtmx.OOBHTML("x", html)
			Expect(result).To(Equal(html))
		})

		It("wraps HTML with an empty id", func() {
			result := cqrshtmx.OOBHTML("", "<span>x</span>")
			Expect(result).To(Equal(`<div id="" hx-swap-oob="true"><span>x</span></div>`))
		})

		It("wraps empty HTML", func() {
			result := cqrshtmx.OOBHTML("slot", "")
			Expect(result).To(Equal(`<div id="slot" hx-swap-oob="true"></div>`))
		})
	})

	Describe("WSOOBHTML (delegates to OOBHTML)", func() {
		It("produces the same result as OOBHTML", func() {
			html := "<p>hello</p>"
			Expect(cqrshtmx.WSOOBHTML("id", html)).To(Equal(cqrshtmx.OOBHTML("id", html)))
		})

		It("produces the same result with swap strategy", func() {
			html := "<p>hello</p>"
			Expect(cqrshtmx.WSOOBHTML("id", html, cqrshtmx.SwapAfterEnd)).
				To(Equal(cqrshtmx.OOBHTML("id", html, cqrshtmx.SwapAfterEnd)))
		})
	})
})
