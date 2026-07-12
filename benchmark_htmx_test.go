package cqrshtmx_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

type benchComponent struct{}

func (benchComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte("ok"))
	return err
}

// sink prevents the compiler from optimizing away pure function calls.
var sink string //nolint:gochecknoglobals // benchmark sink

func BenchmarkParseHTMXRequest(b *testing.B) {
	handler := cqrshtmx.HTMXMiddleware(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}),
	)

	b.Run("AllHeaders", func(b *testing.B) {
		for range b.N {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-Boosted", "true")
			r.Header.Set("HX-Target", "main")
			r.Header.Set("HX-Trigger", "btn")
			r.Header.Set("HX-Trigger-Name", "action")
			r.Header.Set("HX-Prompt", "yes")
			r.Header.Set("HX-Current-URL", "https://example.com/page")
			r.Header.Set("HX-History-Restore-Request", "true")
			handler.ServeHTTP(httptest.NewRecorder(), r)
		}
	})
	b.Run("NoHeaders", func(b *testing.B) {
		for range b.N {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(httptest.NewRecorder(), r)
		}
	})
}

func BenchmarkRenderPartial(b *testing.B) {
	mw := cqrshtmx.HTMXMiddleware(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if cqrshtmx.RenderPartial(r) {
				sink = "partial"
			} else {
				sink = "full"
			}
		}),
	)
	b.Run("HTMXRequest", func(b *testing.B) {
		for range b.N {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			mw.ServeHTTP(httptest.NewRecorder(), r)
		}
	})
	b.Run("HTMXHistoryRestore", func(b *testing.B) {
		for range b.N {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-History-Restore-Request", "true")
			mw.ServeHTTP(httptest.NewRecorder(), r)
		}
	})
	b.Run("NonHTMX", func(b *testing.B) {
		for range b.N {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			mw.ServeHTTP(httptest.NewRecorder(), r)
		}
	})
}

func BenchmarkRenderTemplComponent(b *testing.B) {
	c := benchComponent{}
	b.Run("Partial", func(b *testing.B) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("HX-Request", "true")
		for range b.N {
			_ = cqrshtmx.RenderTemplComponent(httptest.NewRecorder(), r, c, c)
		}
	})
	b.Run("Full", func(b *testing.B) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		for range b.N {
			_ = cqrshtmx.RenderTemplComponent(httptest.NewRecorder(), r, c, c)
		}
	})
}

func BenchmarkOOBHTML(b *testing.B) {
	html := "<span>3</span>"
	b.Run("DefaultSwap", func(b *testing.B) {
		for range b.N {
			sink = cqrshtmx.OOBHTML("counter", html)
		}
	})
	b.Run("CustomSwap", func(b *testing.B) {
		for range b.N {
			sink = cqrshtmx.OOBHTML("counter", html, cqrshtmx.SwapBeforeEnd)
		}
	})
	b.Run("Passthrough", func(b *testing.B) {
		preTagged := `<div id="x" hx-swap-oob="true">tagged</div>`
		for range b.N {
			sink = cqrshtmx.OOBHTML("x", preTagged)
		}
	})
}
