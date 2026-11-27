package cqrshtmx_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// viewsGoldenDir holds the pinned wire bytes of every response shape ONE Views
// declaration produces. The selection matrix spec proves WHICH view renders;
// this golden proves the exact BYTES on the wire — the out-of-band envelope,
// its swap strategy, and the fact that out-of-band fragments follow the
// primary fragment with nothing in between. Regenerate deliberately (review
// the diff: this is what htmx parses) with:
//
//	go test ./ -run TestViews_Golden -update
const viewsGoldenDir = "testdata/views"

type viewsGoldenCase struct {
	name    string
	headers map[string]string
}

var viewsGoldenCases = []viewsGoldenCase{
	{name: "page"},
	{
		name: "boosted-navigation",
		headers: map[string]string{
			htmx.HeaderRequest: cqrshtmx.HeaderTrue,
			htmx.HeaderBoosted: cqrshtmx.HeaderTrue,
			htmx.HeaderTarget:  "rows",
		},
	},
	{
		name: "history-restore",
		headers: map[string]string{
			htmx.HeaderRequest:        cqrshtmx.HeaderTrue,
			htmx.HeaderHistoryRestore: cqrshtmx.HeaderTrue,
			htmx.HeaderTarget:         "rows",
		},
	},
	{
		name: "region-with-oob",
		headers: map[string]string{
			htmx.HeaderRequest: cqrshtmx.HeaderTrue,
			htmx.HeaderTarget:  "rows",
		},
	},
}

// TestViews_Golden pins the complete response for each shape a request can
// select, through the real QuerySet pipeline.
func TestViews_Golden(t *testing.T) {
	t.Parallel()

	set := cqrshtmx.NewQuerySet(
		cqrshtmx.Load("rows", func(_ context.Context) (string, error) { return "rows", nil },
			func(props *viewsTestProps, value string) { props.Label = value }),
	)

	handler := cqrshtmx.QuerySetHandler(set, cqrshtmx.Views(
		cqrshtmx.Region("#rows", func(props viewsTestProps) cqrshtmx.TemplComponent {
			return &bddTemplComponent{html: `<ul id="rows"><li>` + props.Label + `</li></ul>`}
		}),
		cqrshtmx.Page(func(props viewsTestProps) cqrshtmx.TemplComponent {
			return &bddTemplComponent{html: `<!doctype html><html><body><main>` + props.Label + `</main></body></html>`}
		}),
		cqrshtmx.OOB("count", func(props viewsTestProps) cqrshtmx.TemplComponent {
			return &bddTemplComponent{html: `<b>` + props.Label + `</b>`}
		}, cqrshtmx.SwapBeforeEnd),
	))

	for _, tc := range viewsGoldenCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/tenants", nil)
			for key, value := range tc.headers {
				r.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if got := w.Header().Get("Content-Type"); got != cqrshtmx.ContentTypeHTML {
				t.Fatalf("Content-Type = %q, want %q", got, cqrshtmx.ContentTypeHTML)
			}

			golden := filepath.Join(viewsGoldenDir, tc.name+".golden.html")

			if *updateGoldens {
				if err := os.MkdirAll(viewsGoldenDir, 0o755); err != nil {
					t.Fatalf("mkdir golden dir: %v", err)
				}

				if err := os.WriteFile(golden, w.Body.Bytes(), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}

				t.Logf("golden rewritten: %s (%d bytes)", golden, w.Body.Len())

				return
			}

			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden %s (run with -update to create): %v", golden, err)
			}

			if !bytes.Equal(w.Body.Bytes(), want) {
				t.Fatalf(
					"Views wire bytes drifted for %s; diff old/new and re-pin deliberately with -update:\n--- golden ---\n%s\n--- got ---\n%s",
					tc.name,
					want,
					w.Body.Bytes(),
				)
			}
		})
	}
}
