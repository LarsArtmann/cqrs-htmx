package cqrshtmx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// querySetProps is the props struct the QuerySet specs assemble.
type querySetProps struct {
	Name  string
	Count int
	Badge string
}

// newQuerySetTestApp builds an App with a query dispatcher and an optional
// error sink, so specs can assert the mapped error identity, not just a status.
func newQuerySetTestApp(errorSink *error) *cqrshtmx.App {
	dispatch := query.NewDispatcher()

	config := cqrshtmx.Config{Queries: dispatch}
	if errorSink != nil {
		config.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			*errorSink = err

			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	app, err := cqrshtmx.New(config)
	Expect(err).NotTo(HaveOccurred())

	return app
}

// querySetPageRequest is a plain browser visit.
func querySetPageRequest() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/page", nil)
}

// querySetFragmentRequest is an HTMX swap aimed at one region.
func querySetFragmentRequest(target string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/page", nil)
	r.Header.Set("Hx-Request", cqrshtmx.HeaderTrue)
	r.Header.Set("Hx-Target", target)

	return r
}

var _ = Describe("QuerySet", func() {
	Describe("loaders", func() {
		It("applies every loader's value to the props struct", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("name", func(_ context.Context) (string, error) {
					return "widgets", nil
				}, func(props *querySetProps, value string) { props.Name = value }),
				cqrshtmx.Load("count", func(_ context.Context) (int, error) {
					return 7, nil
				}, func(props *querySetProps, value int) { props.Count = value }),
			)

			var rendered cqrshtmx.TemplComponent

			app := newQuerySetTestApp(nil)
			w := serve(app.QuerySet(set, cqrshtmx.RenderTemplResult(func(props querySetProps) cqrshtmx.TemplComponent {
				rendered = &bddTemplComponent{html: props.Name}

				return rendered
			})), querySetPageRequest())

			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("widgets"))
			Expect(rendered).NotTo(BeNil())
		})

		It("runs loaders in parallel", func() {
			firstReleased := make(chan struct{})
			secondReleased := make(chan struct{})

			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("first", func(_ context.Context) (string, error) {
					close(firstReleased)

					select {
					case <-secondReleased:
						return "first", nil
					case <-time.After(2 * time.Second):
						return "", errors.New("loaders did not run in parallel")
					}
				}, func(props *querySetProps, value string) { props.Name = value }),
				cqrshtmx.Load("second", func(_ context.Context) (int, error) {
					close(secondReleased)

					select {
					case <-firstReleased:
						return 1, nil
					case <-time.After(2 * time.Second):
						return 0, errors.New("loaders did not run in parallel")
					}
				}, func(props *querySetProps, value int) { props.Count = value }),
			)

			var sink error

			app := newQuerySetTestApp(&sink)
			w := serve(app.QuerySet(set, cqrshtmx.Views(
				cqrshtmx.Page(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				}))), querySetPageRequest())

			Expect(sink).NotTo(HaveOccurred())
			Expect(w.Body.String()).To(Equal("first"))
		})

		It("aborts the request and cancels siblings when a fatal loader fails", func() {
			cancelled := make(chan struct{})

			failing := errorfamily.NewRejection("read_failed", "tenant read failed")

			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("slow", func(ctx context.Context) (string, error) {
					<-ctx.Done()
					close(cancelled)

					return "", ctx.Err()
				}, func(_ *querySetProps, _ string) {}),
				cqrshtmx.Load("broken", func(_ context.Context) (int, error) {
					return 0, failing
				}, func(_ *querySetProps, _ int) {}),
			)

			var sink error

			app := newQuerySetTestApp(&sink)

			rendered := false
			w := serve(app.QuerySet(set, cqrshtmx.Render(func(
				http.ResponseWriter, *http.Request, any,
			) error {
				rendered = true

				return nil
			})), querySetPageRequest())

			Expect(sink).To(MatchError(failing))
			Expect(rendered).To(BeFalse())
			Expect(w.code()).To(Equal(http.StatusInternalServerError))
			Eventually(cancelled, time.Second).Should(BeClosed())
		})

		It("reports the first fatal failure in declaration order, not completion order", func() {
			declaredFirst := errorfamily.NewRejection("first_failed", "declared first, fails last")
			declaredSecond := errorfamily.NewRejection("second_failed", "declared second, fails first")

			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("first", func(_ context.Context) (string, error) {
					time.Sleep(20 * time.Millisecond)

					return "", declaredFirst
				}, func(_ *querySetProps, _ string) {}),
				cqrshtmx.Load("second", func(_ context.Context) (int, error) {
					return 0, declaredSecond
				}, func(_ *querySetProps, _ int) {}),
			)

			var sink error

			app := newQuerySetTestApp(&sink)
			serve(app.QuerySet(set, cqrshtmx.RenderTempl(&bddTemplComponent{html: "x"})),
				querySetPageRequest())

			Expect(sink).To(MatchError(declaredFirst))
		})

		It("degrades an Optional loader: the page renders and the report names the failure", func() {
			optionalErr := errorfamily.NewTransient("count_unavailable", "counts are down")

			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("name", func(_ context.Context) (string, error) {
					return "widgets", nil
				}, func(props *querySetProps, value string) { props.Name = value }),
				cqrshtmx.Load("count", func(_ context.Context) (int, error) {
					return 0, optionalErr
				}, func(props *querySetProps, value int) { props.Count = value }).Optional(),
			)

			var report cqrshtmx.LoadReport

			app := newQuerySetTestApp(nil)
			w := serve(app.QuerySet(set,
				cqrshtmx.RenderTemplResult(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				}),
				cqrshtmx.OnLoadError(func(_ *http.Request, got cqrshtmx.LoadReport) {
					report = got
				}),
			), querySetPageRequest())

			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("widgets"))
			Expect(report.HasFailures()).To(BeTrue())
			Expect(report.Failures).To(HaveLen(1))
			Expect(report.Failures[0].Name).To(Equal("count"))
			Expect(report.Failures[0].Err).To(MatchError(optionalErr))
		})

		It("still renders when an optional loader fails with no hook installed", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("name", func(_ context.Context) (string, error) {
					return "widgets", nil
				}, func(props *querySetProps, value string) { props.Name = value }),
				cqrshtmx.Load("count", func(_ context.Context) (int, error) {
					return 0, errorfamily.NewTransient("count_unavailable", "counts are down")
				}, func(props *querySetProps, value int) { props.Count = value }).Optional(),
			)

			app := newQuerySetTestApp(nil)
			w := serve(app.QuerySet(set,
				cqrshtmx.RenderTemplResult(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				})), querySetPageRequest())

			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("widgets"))
		})
	})

	Describe("PageOnly scoping", func() {
		var (
			pageRuns     atomic.Int32
			fragmentRuns atomic.Int32
		)

		BeforeEach(func() {
			pageRuns.Store(0)
			fragmentRuns.Store(0)
		})

		newSet := func() cqrshtmx.QuerySet[querySetProps] {
			return cqrshtmx.NewQuerySet(
				cqrshtmx.Load("rows", func(_ context.Context) (string, error) {
					fragmentRuns.Add(1)

					return "rows", nil
				}, func(props *querySetProps, value string) { props.Name = value }),
				cqrshtmx.Load("page-nav", func(_ context.Context) (string, error) {
					pageRuns.Add(1)

					return "nav", nil
				}, func(props *querySetProps, value string) { props.Badge = value }).PageOnly(),
			)
		}

		It("skips page-only loaders for a targeted HTMX request", func() {
			app := newQuerySetTestApp(nil)
			serve(app.QuerySet(newSet(), cqrshtmx.Views(
				cqrshtmx.Region("#item-list", func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				}),
				cqrshtmx.Page(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Badge}
				}),
			)), querySetFragmentRequest("item-list"))

			Expect(fragmentRuns.Load()).To(Equal(int32(1)))
			Expect(pageRuns.Load()).To(Equal(int32(0)))
		})

		It("runs every loader for a full page request", func() {
			app := newQuerySetTestApp(nil)
			serve(app.QuerySet(newSet(), cqrshtmx.Views(
				cqrshtmx.Page(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Badge}
				}),
			)), querySetPageRequest())

			Expect(fragmentRuns.Load()).To(Equal(int32(1)))
			Expect(pageRuns.Load()).To(Equal(int32(1)))
		})
	})

	Describe("validation", func() {
		It("panics on an empty set", func() {
			Expect(func() { cqrshtmx.NewQuerySet[querySetProps]() }).NotTo(Panic())

			app := newQuerySetTestApp(nil)

			Expect(func() {
				app.QuerySet(cqrshtmx.NewQuerySet[querySetProps]())
			}).To(PanicWith(ContainSubstring("at least one Loader")))
		})

		It("panics on an unnamed loader", func() {
			app := newQuerySetTestApp(nil)

			Expect(func() {
				app.QuerySet(cqrshtmx.NewQuerySet(
					cqrshtmx.Load("", func(_ context.Context) (int, error) { return 0, nil },
						func(_ *querySetProps, _ int) {}),
				))
			}).To(PanicWith(ContainSubstring("non-empty name")))
		})

		It("panics on duplicate loader names", func() {
			app := newQuerySetTestApp(nil)

			Expect(func() {
				app.QuerySet(cqrshtmx.NewQuerySet(
					cqrshtmx.Load("dup", func(_ context.Context) (int, error) { return 0, nil },
						func(_ *querySetProps, _ int) {}),
					cqrshtmx.Load("dup", func(_ context.Context) (int, error) { return 0, nil },
						func(_ *querySetProps, _ int) {}),
				))
			}).To(PanicWith(ContainSubstring("declared twice")))
		})

		It("panics on a loader with no function", func() {
			app := newQuerySetTestApp(nil)

			Expect(func() {
				app.QuerySet(cqrshtmx.NewQuerySet(
					cqrshtmx.Load[querySetProps, int]("broken", nil, nil),
				))
			}).To(PanicWith(ContainSubstring("no load function")))
		})
	})

	Describe("QuerySetHandler", func() {
		It("serves the page without an App", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("name", func(_ context.Context) (string, error) {
					return "standalone", nil
				}, func(props *querySetProps, value string) { props.Name = value }),
			)

			handler := cqrshtmx.QuerySetHandler(set,
				cqrshtmx.RenderTemplResult(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				}))

			w := serve(handler, querySetPageRequest())
			Expect(w.code()).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("standalone"))
		})

		It("maps a fatal loader error to its HTTP status without an App", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("broken", func(_ context.Context) (string, error) {
					return "", errorfamily.NewRejection("read_failed", "no")
				}, func(_ *querySetProps, _ string) {}),
			)

			handler := cqrshtmx.QuerySetHandler(set,
				cqrshtmx.RenderTempl(&bddTemplComponent{html: "x"}))

			w := serve(handler, querySetPageRequest())
			Expect(w.code()).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("dispatched loaders", func() {
		It("dispatches through the given dispatcher and applies the typed result", func() {
			dispatched := make(chan string, 1)

			dispatch := query.NewDispatcher()
			dispatch.Use(func(next query.Handler) query.Handler {
				return func(ctx context.Context, qry query.Query) (any, error) {
					dispatched <- string(qry.Type())

					return next(ctx, qry)
				}
			})
			_ = query.RegisterTyped(dispatch, "CountWidgets",
				func(_ context.Context, _ *countWidgetsQuery) (int, error) { return 3, nil })

			set := cqrshtmx.NewQuerySet(
				cqrshtmx.LoadQuery("CountWidgets", cqrshtmx.FromDispatcher(dispatch),
					func(_ context.Context) (query.Query, error) { return &countWidgetsQuery{}, nil },
					func(props *querySetProps, value int) { props.Count = value }),
			)

			app := newQuerySetTestApp(nil)
			w := serve(app.QuerySet(set,
				cqrshtmx.RenderTemplResult(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: strings.Repeat("x", props.Count)}
				})), querySetPageRequest())

			Expect(w.Body.String()).To(Equal("xxx"))
			Eventually(dispatched).Should(Receive(Equal("CountWidgets")))
		})

		It("maps a query-build failure", func() {
			buildErr := errorfamily.NewRejection("bad_filter", "filter is invalid")

			set := cqrshtmx.NewQuerySet(
				cqrshtmx.LoadQuery("CountWidgets", cqrshtmx.FromDispatcher(query.NewDispatcher()),
					func(_ context.Context) (query.Query, error) { return nil, buildErr },
					func(_ *querySetProps, _ int) {}),
			)

			var sink error

			app := newQuerySetTestApp(&sink)
			serve(app.QuerySet(set, cqrshtmx.RenderTempl(&bddTemplComponent{html: "x"})),
				querySetPageRequest())

			Expect(sink).To(MatchError(buildErr))
		})

		It("accepts a loader with a nil apply (the value is simply unused)", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.LoadQuery[querySetProps, int]("CountWidgets",
					func(_ context.Context, _ query.Query) (any, error) { return 3, nil },
					func(_ context.Context) (query.Query, error) { return &countWidgetsQuery{}, nil },
					nil),
			)

			app := newQuerySetTestApp(nil)
			w := serve(app.QuerySet(set, cqrshtmx.RenderTempl(&bddTemplComponent{html: "ok"})),
				querySetPageRequest())

			Expect(w.code()).To(Equal(http.StatusOK))
		})

		It("rejects a nil dispatcher at registration time", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.LoadQuery("CountWidgets", cqrshtmx.FromDispatcher(nil),
					func(_ context.Context) (query.Query, error) { return &countWidgetsQuery{}, nil },
					func(_ *querySetProps, _ int) {}),
			)

			app := newQuerySetTestApp(nil)

			Expect(func() {
				app.QuerySet(set, cqrshtmx.RenderTempl(&bddTemplComponent{html: "x"}))
			}).To(PanicWith(ContainSubstring("no load function")))
		})
	})

	Describe("ETagResults composition", func() {
		It("answers 304 on a repeat request with the same props", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("name", func(_ context.Context) (string, error) {
					return "stable", nil
				}, func(props *querySetProps, value string) { props.Name = value }),
			)

			app := newQuerySetTestApp(nil)
			handler := app.QuerySet(set,
				cqrshtmx.Views(cqrshtmx.Page(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				})),
				cqrshtmx.ETagResults(),
			)

			first := serve(handler, querySetPageRequest())
			Expect(first.code()).To(Equal(http.StatusOK))
			tag := first.Header().Get("ETag")
			Expect(tag).NotTo(BeEmpty())

			repeat := querySetPageRequest()
			repeat.Header.Set("If-None-Match", tag)
			second := serve(handler, repeat)

			Expect(second.code()).To(Equal(http.StatusNotModified))
			Expect(second.Body.String()).To(BeEmpty())
		})

		It("changes the ETag when the props change", func() {
			var name atomic.Value
			name.Store("one")

			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("name", func(_ context.Context) (string, error) {
					return name.Load().(string), nil
				}, func(props *querySetProps, value string) { props.Name = value }),
			)

			app := newQuerySetTestApp(nil)
			handler := app.QuerySet(set,
				cqrshtmx.RenderTemplResult(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				}),
				cqrshtmx.ETagResults(),
			)

			first := serve(handler, querySetPageRequest())

			name.Store("two")

			second := serve(handler, querySetPageRequest())

			Expect(first.Header().Get("ETag")).NotTo(Equal(second.Header().Get("ETag")))
		})
	})

	Describe("region selection inside a QuerySet handler", func() {
		It("renders the region for a targeted request and the page otherwise", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("name", func(_ context.Context) (string, error) {
					return "items", nil
				}, func(props *querySetProps, value string) { props.Name = value }),
			)

			app := newQuerySetTestApp(nil)
			handler := app.QuerySet(set, cqrshtmx.Views(
				cqrshtmx.Region("#item-list", func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: "region:" + props.Name}
				}),
				cqrshtmx.Page(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: "page:" + props.Name}
				}),
			))

			region := serve(handler, querySetFragmentRequest("item-list"))
			Expect(region.Body.String()).To(Equal("region:items"))

			page := serve(handler, querySetPageRequest())
			Expect(page.Body.String()).To(Equal("page:items"))
		})

		It("answers an unroutable region request with a mapped rejection", func() {
			set := cqrshtmx.NewQuerySet(
				cqrshtmx.Load("name", func(_ context.Context) (string, error) {
					return "items", nil
				}, func(props *querySetProps, value string) { props.Name = value }),
			)

			app := newQuerySetTestApp(nil)
			w := serve(app.QuerySet(set, cqrshtmx.Views(
				cqrshtmx.Region("#item-list", func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				}),
				cqrshtmx.Page(func(props querySetProps) cqrshtmx.TemplComponent {
					return &bddTemplComponent{html: props.Name}
				}),
			)), querySetFragmentRequest("#nowhere"))

			Expect(w.code()).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("LoadScopeFor", func() {
		It("treats a boosted navigation as a page", func() {
			r := httptest.NewRequest(http.MethodGet, "/page", nil)
			r.Header.Set("Hx-Request", htmx.HeaderTrue)
			r.Header.Set(htmx.HeaderBoosted, htmx.HeaderTrue)
			r.Header.Set("Hx-Target", "item-list")

			Expect(cqrshtmx.LoadScopeFor(r)).To(Equal(cqrshtmx.ScopePage))
		})
	})
})

// countWidgetsQuery is the dispatched query the LoadQuery specs register.
type countWidgetsQuery struct{}

func (q *countWidgetsQuery) Type() query.Type { return "CountWidgets" }
