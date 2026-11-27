package cqrshtmx_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// viewsTestProps is the props struct the selection matrix serves.
type viewsTestProps struct {
	Label string
}

// countingTemplComponent counts its Render calls, for the single-render proof.
type countingTemplComponent struct {
	inner   cqrshtmx.TemplComponent
	renders *atomic.Int32
}

func (c *countingTemplComponent) Render(ctx context.Context, w io.Writer) error {
	c.renders.Add(1)

	return c.inner.Render(ctx, w)
}

// viewsTestComponent renders a marker string so the assertions can tell the
// three shapes apart without parsing HTML.
func viewsTestComponent(marker string) func(viewsTestProps) cqrshtmx.TemplComponent {
	return func(props viewsTestProps) cqrshtmx.TemplComponent {
		return &bddTemplComponent{html: "<" + marker + ">" + props.Label + "</" + marker + ">"}
	}
}

// countingViewsTestComponent is viewsTestComponent plus a Render counter.
func countingViewsTestComponent(marker string, renders *atomic.Int32) func(viewsTestProps) cqrshtmx.TemplComponent {
	return func(props viewsTestProps) cqrshtmx.TemplComponent {
		return &countingTemplComponent{
			inner:   viewsTestComponent(marker)(props),
			renders: renders,
		}
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

		It("renders the primary view exactly once and each OOB view exactly once", func() {
			var page, region, oob atomic.Int32

			app := newViewsTestApp(props)
			handler := app.Query("GetUser", decodeGetUserJSONQuery(), cqrshtmx.Views(
				cqrshtmx.Region("#rows", countingViewsTestComponent("ul", &region)),
				cqrshtmx.Page(countingViewsTestComponent("html", &page)),
				cqrshtmx.OOB("count", countingViewsTestComponent("b", &oob)),
			))

			serve(handler, viewsRequest(map[string]string{
				"Hx-Request": cqrshtmx.HeaderTrue,
				"Hx-Target":  "rows",
			}))
			Expect(region.Load()).To(Equal(int32(1)))
			Expect(oob.Load()).To(Equal(int32(1)))
			Expect(page.Load()).To(Equal(int32(0)))

			serve(handler, viewsRequest(nil))
			Expect(page.Load()).To(Equal(int32(1)))
			Expect(region.Load()).To(Equal(int32(1)))
			Expect(oob.Load()).To(Equal(int32(1)))
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

		It("panics on a zero-value View instead of treating it as a page with no component", func() {
			Expect(func() {
				cqrshtmx.Views(cqrshtmx.View[viewsTestProps]{})
			}).To(PanicWith(ContainSubstring("zero-value View")))
		})

		It("panics on a view without a component", func() {
			Expect(func() {
				cqrshtmx.Views(cqrshtmx.Page[viewsTestProps](nil))
			}).To(PanicWith(ContainSubstring("page requires a component")))
		})

		It("panics when a Page is marked Default", func() {
			Expect(func() {
				cqrshtmx.Views(cqrshtmx.Page(viewsTestComponent("html")).Default())
			}).To(PanicWith(ContainSubstring("Page cannot be Default")))
		})

		It("panics when an OOB view is marked Default", func() {
			Expect(func() {
				cqrshtmx.Views(
					cqrshtmx.Page(viewsTestComponent("html")),
					cqrshtmx.OOB("count", viewsTestComponent("b")).Default(),
				)
			}).To(PanicWith(ContainSubstring("OOB cannot be Default")))
		})

		It("reports every problem at once instead of only the first", func() {
			Expect(func() {
				cqrshtmx.Views(
					cqrshtmx.Region("", viewsTestComponent("ul")).Default(),
					cqrshtmx.Region("#other", viewsTestComponent("ol")).Default(),
					cqrshtmx.OOB("", viewsTestComponent("i")),
				)
			}).To(PanicWith(And(
				ContainSubstring("non-empty selector"),
				ContainSubstring("at most one Region may be Default"),
				ContainSubstring("non-empty element id"),
				ContainSubstring("found none"),
			)))
		})
	})

	Describe("View.String", func() {
		It("names each shape for diagnostics and test output", func() {
			Expect(cqrshtmx.Page(viewsTestComponent("html")).String()).To(Equal("page"))
			Expect(cqrshtmx.Region("#rows", viewsTestComponent("ul")).String()).To(Equal("region #rows"))
			Expect(cqrshtmx.OOB("count", viewsTestComponent("b")).String()).To(Equal("oob count"))
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

		It("names the scope for logs and test output", func() {
			Expect(cqrshtmx.ScopePage.String()).To(Equal("page"))
			Expect(cqrshtmx.ScopeFragment.String()).To(Equal("fragment"))
			Expect(cqrshtmx.LoadScopeFor(viewsRequest(nil)).String()).To(Equal("page"))
		})
	})
})
