package cqrshtmx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// These benchmarks isolate the props seam's own cost — view selection, the
// out-of-band envelope, and the loader scheduler — with trivial components and
// loaders, so the numbers are the seam and not a template engine or a database.
// Canonical invocation (idle machine only; ns/op under load is noise):
//
//	GOEXPERIMENT=jsonv2 go test ./ -run '^$' -bench 'ViewRender|QuerySet' -benchmem -benchtime=2s -count=5

// seamBenchProps is the props struct the seam benchmarks assemble.
type seamBenchProps struct {
	Label string
	Count int
}

// seamComponent writes a fixed fragment in place of a templ component.
type seamComponent struct{ html string }

func (c seamComponent) Render(_ context.Context, w io.Writer) error {
	_, err := io.WriteString(w, c.html)

	return err
}

// discardResponse is an http.ResponseWriter that throws the body away, so a
// benchmark loop does not grow a buffer.
type discardResponse struct{ header http.Header }

func (d *discardResponse) Header() http.Header { return d.header }

func (d *discardResponse) Write(body []byte) (int, error) { return len(body), nil }

func (d *discardResponse) WriteHeader(int) {}

func seamRegion(props seamBenchProps) TemplComponent {
	return seamComponent{`<ul id="rows"><li>` + props.Label + `</li></ul>`}
}

func seamPage(props seamBenchProps) TemplComponent {
	return seamComponent{`<main>` + props.Label + `</main>`}
}

// seamViews declares the triad; the out-of-band view is optional so the
// benchmarks can price it separately.
func seamViews(withOOB bool) []View[seamBenchProps] {
	views := []View[seamBenchProps]{
		Region("#rows", seamRegion),
		Page(seamPage),
	}

	if withOOB {
		views = append(views, OOB("count", func(seamBenchProps) TemplComponent {
			return seamComponent{"<b>7</b>"}
		}))
	}

	return views
}

func seamRequest(headers map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/page", nil)
	for key, value := range headers {
		r.Header.Set(key, value)
	}

	return r
}

// BenchmarkViewRender measures the render half alone: one selection plus one
// component write per iteration, for each shape, against the two-way
// RenderPartialOrFull branch it replaces.
func BenchmarkViewRender(b *testing.B) {
	props := seamBenchProps{Label: "rows"}

	plain := newViewSet(seamViews(false))
	withOOB := newViewSet(seamViews(true))

	legacy := &handlerConfig{}
	RenderPartialOrFull(seamRegion, seamPage)(legacy)

	page := seamRequest(nil)
	region := seamRequest(map[string]string{htmx.HeaderRequest: HeaderTrue, htmx.HeaderTarget: "rows"})

	cases := []struct {
		name   string
		render RenderFunc
		r      *http.Request
	}{
		{"RenderPartialOrFull/page", legacy.render, page},
		{"RenderPartialOrFull/partial", legacy.render, region},
		{"Views/page", plain.render, page},
		{"Views/region", plain.render, region},
		{"Views/region+oob", withOOB.render, region},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			w := &discardResponse{header: http.Header{}}

			b.ReportAllocs()

			for b.Loop() {
				if err := bc.render(w, bc.r, props); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// seamQuerySet builds a set of trivial loaders: fragment of them run for every
// request and pageOnly of them only for a full page.
func seamQuerySet(fragment, pageOnly int) QuerySet[seamBenchProps] {
	loaders := make([]Loader[seamBenchProps], 0, fragment+pageOnly)

	apply := func(props *seamBenchProps, value int) { props.Count += value }

	trivial := func(context.Context) (int, error) { return 1, nil }

	for range fragment {
		loaders = append(loaders, Load(fmt.Sprintf("fragment-%d", len(loaders)), trivial, apply))
	}

	for range pageOnly {
		loaders = append(loaders, Load(fmt.Sprintf("page-only-%d", len(loaders)), trivial, apply).PageOnly())
	}

	return NewQuerySet(loaders...)
}

// BenchmarkQuerySetLoad measures the load half alone: the scheduler's own cost
// per request (a goroutine, a WaitGroup, a cancel cause, and the single apply
// pass per loader) for a fragment render versus a page render of one set.
func BenchmarkQuerySetLoad(b *testing.B) {
	ctx := context.Background()

	cases := []struct {
		name            string
		set             QuerySet[seamBenchProps]
		includePageOnly bool
	}{
		{"1-loader", seamQuerySet(1, 0), true},
		{"page/5+2", seamQuerySet(5, 2), true},
		{"region/5+2", seamQuerySet(5, 2), false},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				if _, _, err := bc.set.load(ctx, bc.includePageOnly); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkQuerySetHandler measures one whole request through the App-less
// handler (pre-load check, loaders, selection, render) for both scopes.
func BenchmarkQuerySetHandler(b *testing.B) {
	handler := QuerySetHandler(seamQuerySet(5, 2), Views(seamViews(true)...))

	cases := []struct {
		name string
		r    *http.Request
	}{
		{"page", seamRequest(nil)},
		{"region", seamRequest(map[string]string{htmx.HeaderRequest: HeaderTrue, htmx.HeaderTarget: "rows"})},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			w := &discardResponse{header: http.Header{}}

			b.ReportAllocs()

			for b.Loop() {
				handler.ServeHTTP(w, bc.r)
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "req/s")
		})
	}
}
