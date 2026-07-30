package cqrshtmx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestApp_ServerTiming_WrapsWhenEnabled verifies that applyServerTiming
// wraps the ResponseWriter and injects a collector when the predicate is true.
func TestApp_ServerTiming_WrapsWhenEnabled(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("Test", func(context.Context, command.Command) error { return nil })
	app := MustNew(Config{
		Commands:     disp,
		ServerTiming: func(*http.Request) bool { return true },
	})
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	w, r2 := app.applyServerTiming(rec, r)

	// Writer must be wrapped (not the raw recorder).
	if w == http.ResponseWriter(rec) {
		t.Fatal("ResponseWriter should be wrapped when predicate is true")
	}

	// Collector must be in context.
	if st := ServerTimingFromContext(r2.Context()); st == nil {
		t.Fatal("collector should be in context when enabled")
	}
}

func TestApp_ServerTiming_NoWrapWhenDisabled(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("Test", func(context.Context, command.Command) error { return nil })
	app := MustNew(Config{Commands: disp}) // no ServerTiming predicate

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	w, r2 := app.applyServerTiming(rec, r)

	if w != http.ResponseWriter(rec) {
		t.Fatal("ResponseWriter should not be wrapped when Config.ServerTiming is nil")
	}

	if r2 != r {
		t.Fatal("request should not be modified when disabled")
	}
}

func TestApp_ServerTiming_NoWrapWhenPredicateReturnsFalse(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("Test", func(context.Context, command.Command) error { return nil })
	app := MustNew(Config{
		Commands:     disp,
		ServerTiming: func(*http.Request) bool { return false },
	})
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	w, _ := app.applyServerTiming(rec, r)

	if w != http.ResponseWriter(rec) {
		t.Fatal("ResponseWriter should not be wrapped when predicate returns false")
	}
}

// TestApp_ServerTiming_EndToEndDispatch verifies that Config.ServerTiming
// produces the header in a REAL App.Command() dispatch flow.
func TestApp_ServerTiming_EndToEndDispatch(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("Ping", func(ctx context.Context, _ command.Command) error {
		stop := MeasureServerTiming(ctx, "handler")
		stop()

		return nil
	})

	app := MustNew(Config{
		Commands:     disp,
		ServerTiming: func(r *http.Request) bool { return true },
	})

	handler := app.Command("Ping", DecodeJSON(func(_ struct{}) (command.Command, error) {
		return command.New("Ping", id.NewStreamID())
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)))

	hv := rec.Header().Get(headerServerTiming)
	if hv == "" {
		t.Fatal("Server-Timing header missing on end-to-end dispatch")
	}

	if !strings.Contains(hv, "total;") {
		t.Errorf("missing total metric in %q", hv)
	}

	if !strings.Contains(hv, "handler;") {
		t.Errorf("missing handler metric in %q", hv)
	}
}

// TestApp_ServerTiming_EndToEndDisabled verifies that Config.ServerTiming=nil
// produces NO header in a real dispatch flow — zero overhead.
func TestApp_ServerTiming_EndToEndDisabled(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("Ping", func(ctx context.Context, _ command.Command) error {
		stop := MeasureServerTiming(ctx, "handler") // nil-safe no-op
		stop()

		return nil
	})

	app := MustNew(Config{Commands: disp}) // no ServerTiming

	handler := app.Command("Ping")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if hv := rec.Header().Get(headerServerTiming); hv != "" {
		t.Fatalf("Server-Timing should be absent when Config.ServerTiming=nil, got %q", hv)
	}
}

// TestServerTimingMiddleware_ComposesWithChain verifies Server-Timing works
// when composed with cqrs-htmx's Chain + HTMXMiddleware.
func TestServerTimingMiddleware_ComposesWithChain(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stopWork := MeasureServerTiming(r.Context(), "work")

		time.Sleep(5 * time.Millisecond)
		stopWork()

		_, _ = io.WriteString(w, "ok")
	})

	stacked := Chain(
		ServerTimingMiddleware(),
		HTMXMiddleware,
	)(inner)

	rec := httptest.NewRecorder()
	stacked.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	hv := rec.Header().Get(headerServerTiming)
	if hv == "" {
		t.Fatal("Server-Timing header missing through Chain")
	}

	if !strings.Contains(hv, "work;") {
		t.Errorf("work metric missing in %q", hv)
	}

	if !strings.Contains(hv, "total;") {
		t.Errorf("total metric missing in %q", hv)
	}
}
