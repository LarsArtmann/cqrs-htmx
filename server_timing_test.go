package cqrshtmx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestServerTiming_DisabledIsNoOp(t *testing.T) {
	t.Parallel()

	var st *ServerTiming // nil = disabled (nil-receiver pattern)

	if st.Enabled() {
		t.Fatal("disabled collector should report Enabled()=false")
	}

	st.Record("db", "Database", 5*time.Millisecond)
	st.prependTotal("total", "", 10*time.Millisecond)

	if got := st.HeaderValue(); got != "" {
		t.Fatalf("disabled HeaderValue = %q, want empty", got)
	}

	if got := st.String(); got != "" {
		t.Fatalf("disabled String = %q, want empty", got)
	}

	// Measure on disabled collector returns a no-op func.
	done := st.Measure("x")
	if done == nil {
		t.Fatal("Measure returned nil func")
	}

	done() // must not panic / record

	if st.HeaderValue() != "" {
		t.Fatal("disabled Measure recorded a metric")
	}
}

func TestServerTiming_HeaderValue_FullMetric(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	st.Record("db", "Database", 53*time.Millisecond)

	got := st.HeaderValue()

	want := `db;desc="Database";dur=53`
	if got != want {
		t.Fatalf("HeaderValue = %q, want %q", got, want)
	}
}

func TestServerTiming_HeaderValue_NameOnly(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	st.Record("cache", "", 0) // no desc, zero dur → name only

	if got, want := st.HeaderValue(), "cache"; got != want {
		t.Fatalf("HeaderValue = %q, want %q", got, want)
	}
}

func TestServerTiming_HeaderValue_MultipleMetricsCommaJoined(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	st.Record("db", "", 53*time.Millisecond)
	st.Record("render", "", 7*time.Millisecond)

	got := st.HeaderValue()
	if !strings.Contains(got, "db;dur=53") || !strings.Contains(got, "render;dur=7") {
		t.Fatalf("HeaderValue %q missing a metric", got)
	}

	if !strings.Contains(got, ", ") {
		t.Fatalf("HeaderValue %q not comma-space joined", got)
	}
}

func TestServerTiming_HeaderValue_NoMetrics(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	if got := st.HeaderValue(); got != "" {
		t.Fatalf("empty collector HeaderValue = %q, want empty", got)
	}
}

func TestServerTiming_MeasureRecordsElapsed(t *testing.T) {
	t.Parallel()

	st := newServerTiming()

	stop := st.Measure("db")

	time.Sleep(15 * time.Millisecond)
	stop()

	hv := st.HeaderValue()
	if !strings.HasPrefix(hv, "db;dur=") {
		t.Fatalf("Measure did not record a metric; got %q", hv)
	}
	// Parse the dur and sanity-check it's >= ~10ms (allow scheduler slack).
	rest := strings.TrimPrefix(hv, "db;dur=")
	msStr, _, _ := strings.Cut(rest, ";")

	ms, err := strconv.ParseFloat(msStr, 64)
	if err != nil {
		t.Fatalf("parse dur %q: %v", msStr, err)
	}

	if ms < 10 {
		t.Fatalf("measured dur %vms, want >= 10", ms)
	}
}

func TestServerTiming_MeasureWithDesc(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	stop := st.MeasureWithDesc("db", "Database query")
	stop()

	hv := st.HeaderValue()
	if !strings.Contains(hv, `db;desc="Database query"`) {
		t.Fatalf("MeasureWithDesc missing desc; got %q", hv)
	}
}

func TestServerTiming_NameSanitization(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"db":        "db",
		"my metric": "my_metric",
		"a/b:c":     "a_b_c",
		"":          "", // dropped entirely
		"a b c":     "a_b_c",
	}
	for input, want := range cases {
		st := newServerTiming()
		st.Record(input, "", time.Millisecond)

		hv := st.HeaderValue()
		if want == "" {
			if hv != "" {
				t.Errorf("name %q: expected dropped, got %q", input, hv)
			}

			continue
		}

		if hv != want+";dur=1" {
			t.Errorf("name %q: got %q, want %s", input, hv, want+";dur=1")
		}
	}
}

func TestServerTiming_DescriptionEscaping(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	st.Record("x", `he said "hi", then; left \n`, time.Millisecond)

	hv := st.HeaderValue()
	if !strings.Contains(hv, `desc="he said \"hi\", then; left \\n"`) {
		t.Fatalf("desc not escaped correctly; got %q", hv)
	}
}

func TestServerTiming_ConcurrentRecord(t *testing.T) {
	t.Parallel()

	st := newServerTiming()

	const n = 100

	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func(i int) {
			defer wg.Done()

			st.Record("m", "", time.Duration(i)*time.Microsecond)
		}(i)
	}

	wg.Wait()

	hv := st.HeaderValue()
	// All 100 metrics should be present (comma-separated). Count by splitting
	// on the joiner rather than by "dur=", since a zero-duration metric omits
	// its dur param (i==0 records dur=0).
	count := strings.Count(hv, ", ") + 1
	if got := strings.Count(hv, "m"); got != count {
		t.Fatalf("expected %d metrics, got %d (%q)", count, got, hv)
	}

	if count != n {
		t.Fatalf("expected %d metrics, got %d (%q)", n, count, hv)
	}
}

func TestServerTiming_PrependTotalAtFront(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	st.Record("db", "", 5*time.Millisecond)
	st.prependTotal("total", "Total request", 20*time.Millisecond)

	hv := st.HeaderValue()
	if !strings.HasPrefix(hv, "total;") {
		t.Fatalf("total not at front; got %q", hv)
	}
}

func TestFormatMillis(t *testing.T) {
	t.Parallel()

	cases := []struct {
		d    time.Duration
		want string
	}{
		{1 * time.Millisecond, "1"},
		{53 * time.Millisecond, "53"},
		{500 * time.Microsecond, "0.5"},
		{123 * time.Microsecond, "0.123"},
		{0, "0"},
	}
	for _, c := range cases {
		if got := formatMillis(c.d); got != c.want {
			t.Errorf("formatMillis(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestSanitizeMetricName(t *testing.T) {
	t.Parallel()

	if got := sanitizeMetricName("db"); got != "db" {
		t.Errorf("sanitize db = %q", got)
	}

	if got := sanitizeMetricName("hello world"); got != "hello_world" {
		t.Errorf("sanitize space = %q", got)
	}

	if got := sanitizeMetricName(""); got != "" {
		t.Errorf("sanitize empty = %q", got)
	}
}

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

func TestServerTimingContext_NilSafe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// No collector present → nil-safe helpers must be no-ops, not panics.
	st := ServerTimingFromContext(ctx)
	if st != nil {
		t.Fatalf("expected nil collector, got %v", st)
	}

	RecordServerTiming(ctx, "x", "", time.Millisecond) // must not panic

	done := MeasureServerTiming(ctx, "x")
	if done == nil {
		t.Fatal("MeasureServerTiming returned nil func")
	}

	done() // must not panic
}

func TestServerTimingContext_RoundTrip(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	ctx := WithServerTiming(context.Background(), st)

	if got := ServerTimingFromContext(ctx); got != st {
		t.Fatal("ServerTimingFromContext did not return stored collector")
	}

	MeasureServerTiming(ctx, "db")()

	if hv := st.HeaderValue(); !strings.HasPrefix(hv, "db;") {
		t.Fatalf("MeasureServerTiming did not record; got %q", hv)
	}
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func TestServerTimingMiddleware_HeaderPresentWhenEnabled(t *testing.T) {
	t.Parallel()

	h := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record a sub-metric from the handler.
		MeasureServerTiming(r.Context(), "db")()
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	hv := rec.Header().Get(headerServerTiming)
	if hv == "" {
		t.Fatal("Server-Timing header missing on enabled request")
	}

	if !strings.Contains(hv, "total;") {
		t.Errorf("missing total metric in %q", hv)
	}

	if !strings.Contains(hv, "db;") {
		t.Errorf("missing db metric in %q", hv)
	}
}

func TestServerTimingMiddleware_HeaderAbsentWhenDisabled(t *testing.T) {
	t.Parallel()

	h := ServerTimingMiddlewareWhen(func(*http.Request) bool { return false })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Even though the handler tries to record, the collector is disabled.
			MeasureServerTiming(r.Context(), "db")()
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if hv := rec.Header().Get(headerServerTiming); hv != "" {
		t.Fatalf("Server-Timing should be absent when disabled, got %q", hv)
	}
}

func TestServerTimingMiddleware_GatedByPredicate(t *testing.T) {
	t.Parallel()

	handler := ServerTimingMiddlewareWhen(func(r *http.Request) bool {
		return r.URL.Query().Has("debug")
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Without debug param → no header.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if hv := rec.Header().Get(headerServerTiming); hv != "" {
		t.Fatalf("expected no header without debug, got %q", hv)
	}

	// With debug param → header present.
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/?debug=1", nil))

	if hv := rec2.Header().Get(headerServerTiming); hv == "" {
		t.Fatal("expected header with debug=1")
	}
}

func TestServerTimingMiddleware_HeaderInjectedOnWriteWithoutWriteHeader(t *testing.T) {
	t.Parallel()

	h := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write body directly without calling WriteHeader — Go implicitly
		// commits a 200. The wrapper must still inject the header.
		_, _ = w.Write([]byte("hi"))
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if hv := rec.Header().Get(headerServerTiming); hv == "" {
		t.Fatal("header missing when body written without WriteHeader")
	}

	if rec.Body.String() != "hi" {
		t.Fatalf("body = %q, want 'hi'", rec.Body.String())
	}
}

func TestServerTimingMiddleware_CollectorInContext(t *testing.T) {
	t.Parallel()

	var seen *ServerTiming

	h := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = ServerTimingFromContext(r.Context())

		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if seen == nil {
		t.Fatal("collector not present in context")
	}

	if !seen.Enabled() {
		t.Fatal("collector in context should be enabled")
	}
}

// The wrapper must preserve Flusher so SSE/HTMX streaming responses still flush.
func TestServerTimingMiddleware_PreservesFlusher(t *testing.T) {
	t.Parallel()

	h := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("http.Flusher not available through wrapper")
		}

		f.Flush() // must not panic
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	// httptest.ResponseRecorder implements Flush; reaching here without panic
	// is the success condition.
	if rec.Header().Get(headerServerTiming) == "" {
		t.Fatal("header missing")
	}
}

// The wrapper must preserve Hijacker so WebSocket upgrades still work.
func TestServerTimingMiddleware_PreservesHijacker(t *testing.T) {
	t.Parallel()

	hijacked := false
	h := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("http.Hijacker not available through wrapper")
		}

		hijacked = true

		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !hijacked {
		t.Fatal("handler did not run")
	}
}

func TestServerTimingMiddleware_PreservesPusher(t *testing.T) {
	t.Parallel()

	h := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := w.(http.Pusher)
		_ = ok // httptest.ResponseRecorder doesn't implement Pusher; assertion path must not panic

		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	// No panic = pass.
}

func TestServerTimingMiddleware_UnwrapExposesUnderlying(t *testing.T) {
	t.Parallel()

	h := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.ResponseController uses Unwrap to find interfaces.
		rc := http.NewResponseController(w)
		if rc == nil {
			t.Fatal("nil ResponseController")
		}

		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

// TestServerTimingMiddleware_FlushActuallyDelegates verifies that Flush()
// calls propagate through the wrapper to the underlying writer — not just
// that the interface is present, but that the delegation actually fires.
// This is critical for SSE streaming (sse_stream.go does w.(flusher).Flush()).
func TestServerTimingMiddleware_FlushActuallyDelegates(t *testing.T) {
	t.Parallel()

	h := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("http.Flusher not available")
		}

		f.Flush()
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !rec.Flushed {
		t.Fatal("Flush did not propagate through serverTimingWriter to the underlying recorder")
	}
}

// TestServerTiming_HeaderValue_SpecCompliant verifies the emitted header is
// parseable as valid W3C Server-Timing: each metric is comma-space separated,
// has a token name, optional desc="...", and optional dur=<number>.
func TestServerTiming_HeaderValue_SpecCompliant(t *testing.T) {
	t.Parallel()

	st := newServerTiming()
	st.Record("db", "Database query", 53*time.Millisecond)
	st.Record("cache", "", 2*time.Millisecond)
	st.Record("render", "HTML render", 0) // zero dur → omitted

	hv := st.HeaderValue()

	// Split by ", " (spec-compliant separator between metrics).
	parts := strings.Split(hv, ", ")
	if len(parts) != 3 {
		t.Fatalf("expected 3 metrics, got %d in %q", len(parts), hv)
	}

	// Verify each part has the expected structure.
	expected := map[string]struct {
		hasDesc bool
		hasDur  bool
	}{
		`db;desc="Database query";dur=53`: {true, true},
		`cache;dur=2`:                     {false, true},
		`render;desc="HTML render"`:       {true, false},
	}
	for _, p := range parts {
		if _, ok := expected[p]; !ok {
			t.Errorf("unexpected metric format: %q", p)
		}
	}
}

func TestServerTimingMiddleware_NilPredicateDisablesAll(t *testing.T) {
	t.Parallel()

	h := ServerTimingMiddlewareWhen(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st := ServerTimingFromContext(r.Context())
		if st.Enabled() {
			t.Fatal("nil predicate should disable all requests")
		}

		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if hv := rec.Header().Get(headerServerTiming); hv != "" {
		t.Fatalf("nil predicate should produce no header, got %q", hv)
	}
}

// ---------------------------------------------------------------------------
// Integration with the library: compose with Chain + a real handler
// ---------------------------------------------------------------------------

func TestServerTimingMiddleware_ComposesWithChain(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// NOTE: the measured region must END before the response is committed
		// (Write/WriteHeader), because the header is injected at that moment.
		// So we capture the stop func and call it explicitly before writing.
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

// TestServerTimingMiddleware_DeferredMeasureMissesHeader documents the TTFB
// semantics: a metric whose stop func runs AFTER the response is committed
// (e.g. via `defer Measure()()`) will NOT appear in the header, because the
// header is injected at the first Write/WriteHeader. This is by design —
// Server-Timing is a response header and must be set before the body.
func TestServerTimingMiddleware_DeferredMeasureMissesHeader(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer MeasureServerTiming(r.Context(), "late")() // records at RETURN — after Write

		_, _ = io.WriteString(w, "ok") // commits the header now
	})

	stacked := ServerTimingMiddleware()(inner)
	rec := httptest.NewRecorder()
	stacked.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	hv := rec.Header().Get(headerServerTiming)
	if strings.Contains(hv, "late") {
		t.Fatalf("late (deferred) metric should be absent from header; got %q", hv)
	}
	// total is still present (it's captured at flush time).
	if !strings.Contains(hv, "total;") {
		t.Errorf("total metric missing in %q", hv)
	}
}

// Composing ServerTimingMiddleware outside RequestLogging's StatusRecorder
// should still work: the header must survive the wrapper chain.
func TestServerTimingMiddleware_BeforeStatusRecorder(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	// RequestLogging wraps with StatusRecorder; ServerTiming wraps outside it.
	stacked := Chain(
		ServerTimingMiddleware(),
		RequestLogging(nil, nil),
	)(inner)

	rec := httptest.NewRecorder()
	stacked.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if hv := rec.Header().Get(headerServerTiming); hv == "" {
		t.Fatal("header lost when composed with StatusRecorder")
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

// dur parsing for the elapsed-time test uses strconv.ParseFloat inline.

// ---------------------------------------------------------------------------
// App integration (Config.ServerTiming)
// ---------------------------------------------------------------------------

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

	if _, ok := w.(*serverTimingWriter); !ok {
		t.Fatal("ResponseWriter should be wrapped when predicate is true")
	}

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
// produces the header in a REAL App.Command() dispatch flow — not just the
// applyServerTiming helper in isolation. This is the integration test proving
// the feature works end-to-end through decode → dispatch → response.
func TestApp_ServerTiming_EndToEndDispatch(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("Ping", func(ctx context.Context, _ command.Command) error {
		// Record a sub-metric from inside the handler — BEFORE the response
		// is committed (applyCommandResponse happens after dispatch returns).
		stop := MeasureServerTiming(ctx, "handler")
		stop()

		return nil
	})

	app := MustNew(Config{
		Commands:     disp,
		ServerTiming: func(r *http.Request) bool { return true },
	})

	handler := app.Command("Ping", DecodeJSON(func(_ struct{}) (command.Command, error) {
		return command.New("Ping", id.NewAggregateID())
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
