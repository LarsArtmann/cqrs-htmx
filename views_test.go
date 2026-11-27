package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// viewsTestProps is the props struct the selection matrix serves.
type viewsTestProps struct {
	Label string
}

// viewsTestComponent renders a marker string so the assertions can tell the
// three shapes apart without parsing HTML.
func viewsTestComponent(marker string) func(viewsTestProps) cqrshtmx.TemplComponent {
	return func(props viewsTestProps) cqrshtmx.TemplComponent {
		return &bddTemplComponent{html: "<" + marker + ">" + props.Label + "</" + marker + ">"}
	}
}

func newViewsTestApp(result any) *cqrshtmx.App {
	return newQueryAppWithResult(constantQueryHandler(result))
}

// viewsRequest builds the GET /page request every case starts from, applying
// the HTMX headers the case needs.
func viewsRequest(headers map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/page", strings.NewReader(`{}`))
	for key, value := range headers {
		r.Header.Set(key, value)
	}

	return r
}

var _ = Describe("Views", func() {
	var props viewsTestProps

	BeforeEach(func() {
		props = viewsTestProps{Label: "rows"}
	})

	Describe("selection", func() {
		It("renders the Page for a plain request", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Region("#rows", viewsTestComponent("ul")),
					cqrshtmx.Page(viewsTestComponent("html")),
				)), viewsRequest(nil))

			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("<html>rows</html>"))
			Expect(w.Header().Get("Content-Type")).To(Equal(cqrshtmx.ContentTypeHTML))
		})

		It("renders the region whose selector matches HX-Target", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Region("#rows", viewsTestComponent("ul")),
					cqrshtmx.Page(viewsTestComponent("html")),
				)), viewsRequest(map[string]string{
				"Hx-Request": cqrshtmx.HeaderTrue,
				"Hx-Target":  "rows",
			}))

			Expect(w.Body.String()).To(Equal("<ul>rows</ul>"))
		})

		It("matches a declared '#' selector against a raw htmx target id", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Region("rows", viewsTestComponent("ul")),
					cqrshtmx.Page(viewsTestComponent("html")),
				)), viewsRequest(map[string]string{
				"Hx-Request": cqrshtmx.HeaderTrue,
				"Hx-Target":  "#rows",
			}))

			Expect(w.Body.String()).To(Equal("<ul>rows</ul>"))
		})

		It("falls back to the Default region for an unmatched partial target", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Region("#rows", viewsTestComponent("ul")),
					cqrshtmx.Region("#other", viewsTestComponent("ol")).Default(),
					cqrshtmx.Page(viewsTestComponent("html")),
				)), viewsRequest(map[string]string{
				"Hx-Request": cqrshtmx.HeaderTrue,
				"Hx-Target":  "#nowhere",
			}))

			Expect(w.Body.String()).To(Equal("<ol>rows</ol>"))
		})

		It("fails loudly when a partial target matches nothing and no Default exists", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Region("#rows", viewsTestComponent("ul")),
					cqrshtmx.Page(viewsTestComponent("html")),
				)), viewsRequest(map[string]string{
				"Hx-Request": cqrshtmx.HeaderTrue,
				"Hx-Target":  "#nowhere",
			}))

			Expect(w.code()).To(Equal(http.StatusBadRequest))
			// Never a silent full-page fallback swapped into a fragment target.
			Expect(w.Body.String()).NotTo(ContainSubstring("<html>"))
			Expect(w.Body.String()).NotTo(ContainSubstring("<ul>"))
		})

		It("renders the Page for a boosted navigation even when a region matches", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Region("#rows", viewsTestComponent("ul")),
					cqrshtmx.Page(viewsTestComponent("html")),
				)), viewsRequest(map[string]string{
				"Hx-Request": cqrshtmx.HeaderTrue,
				"Hx-Boosted": cqrshtmx.HeaderTrue,
				"Hx-Target":  "rows",
			}))

			Expect(w.Body.String()).To(Equal("<html>rows</html>"))
		})

		It("renders the Page for a history restore", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Region("#rows", viewsTestComponent("ul")),
					cqrshtmx.Page(viewsTestComponent("html")),
				)), viewsRequest(map[string]string{
				"Hx-Request":              cqrshtmx.HeaderTrue,
				htmx.HeaderHistoryRestore: cqrshtmx.HeaderTrue,
				"Hx-Target":               "rows",
			}))

			Expect(w.Body.String()).To(Equal("<html>rows</html>"))
		})
	})

	Describe("out-of-band views", func() {
		It("appends OOB fragments in declaration order after the primary view", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Region("#rows", viewsTestComponent("ul")),
					cqrshtmx.Page(viewsTestComponent("html")),
					cqrshtmx.OOB("first", viewsTestComponent("b")),
					cqrshtmx.OOB("second", viewsTestComponent("i")),
				)), viewsRequest(map[string]string{
				"Hx-Request": cqrshtmx.HeaderTrue,
				"Hx-Target":  "rows",
			}))

			body := w.Body.String()
			Expect(body).To(HavePrefix("<ul>rows</ul>"))
			Expect(strings.Index(body, `id="first"`)).To(BeNumerically("<", strings.Index(body, `id="second"`)))
			Expect(body).To(ContainSubstring(`<div id="first" hx-swap-oob="true"><b>rows</b></div>`))
			Expect(body).To(ContainSubstring(`<div id="second" hx-swap-oob="true"><i>rows</i></div>`))
		})

		It("never emits OOB fragments on a full page", func() {
			app := newViewsTestApp(props)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(
					cqrshtmx.Page(viewsTestComponent("html")),
					cqrshtmx.OOB("count", viewsTestComponent("b")),
				)), viewsRequest(nil))

			Expect(w.Body.String()).To(Equal("<html>rows</html>"))
		})
	})

	Describe("misdeclaration", func() {
		It("panics when no Page is declared", func() {
			Expect(func() {
				cqrshtmx.Views(cqrshtmx.Region("#rows", viewsTestComponent("ul")))
			}).To(PanicWith(ContainSubstring("exactly one Page")))
		})

		It("panics when two Pages are declared", func() {
			Expect(func() {
				cqrshtmx.Views(
					cqrshtmx.Page(viewsTestComponent("html")),
					cqrshtmx.Page(viewsTestComponent("body")),
				)
			}).To(PanicWith(ContainSubstring("exactly one Page")))
		})

		It("panics when two Default regions are declared", func() {
			Expect(func() {
				cqrshtmx.Views(
					cqrshtmx.Page(viewsTestComponent("html")),
					cqrshtmx.Region("#a", viewsTestComponent("ul")).Default(),
					cqrshtmx.Region("#b", viewsTestComponent("ol")).Default(),
				)
			}).To(PanicWith(ContainSubstring("at most one Region may be Default")))
		})

		It("panics on a region without a selector", func() {
			Expect(func() {
				cqrshtmx.Views(
					cqrshtmx.Page(viewsTestComponent("html")),
					cqrshtmx.Region("", viewsTestComponent("ul")),
				)
			}).To(PanicWith(ContainSubstring("non-empty selector")))
		})

		It("panics on an OOB view without an id", func() {
			Expect(func() {
				cqrshtmx.Views(
					cqrshtmx.Page(viewsTestComponent("html")),
					cqrshtmx.OOB("", viewsTestComponent("ul")),
				)
			}).To(PanicWith(ContainSubstring("non-empty element id")))
		})
	})

	Describe("result typing", func() {
		It("maps a result-type mismatch to a rejection and writes no view markup", func() {
			app := newViewsTestApp(42)
			w := serve(app.Query("GetUser", decodeGetUserJSONQuery(),
				cqrshtmx.Views(cqrshtmx.Page(viewsTestComponent("html")))), viewsRequest(nil))

			Expect(w.code()).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).NotTo(ContainSubstring("<html>"))
		})
	})

	Describe("LoadScopeFor", func() {
		It("is a page for plain requests", func() {
			Expect(cqrshtmx.LoadScopeFor(viewsRequest(nil))).To(Equal(cqrshtmx.ScopePage))
		})

		It("is a fragment for a targeted HTMX request", func() {
			scope := cqrshtmx.LoadScopeFor(viewsRequest(map[string]string{
				"Hx-Request": cqrshtmx.HeaderTrue,
				"Hx-Target":  "rows",
			}))
			Expect(scope).To(Equal(cqrshtmx.ScopeFragment))
		})

		It("is a page for a history restore", func() {
			scope := cqrshtmx.LoadScopeFor(viewsRequest(map[string]string{
				"Hx-Request":              cqrshtmx.HeaderTrue,
				htmx.HeaderHistoryRestore: cqrshtmx.HeaderTrue,
			}))
			Expect(scope).To(Equal(cqrshtmx.ScopePage))
		})
	})
})
