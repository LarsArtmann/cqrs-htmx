package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// FuzzServerTimingHeaderValue verifies that HeaderValue produces a spec-compliant
// Server-Timing header for any input — no crashes, no malformed output.
// Adversarial metric names (commas, semicolons, control chars) must be sanitized.
func FuzzServerTimingHeaderValue(f *testing.F) {
	f.Add("db", "Database query", int64(50))
	f.Add("", "", int64(0))
	f.Add("name;with;semicolons", "desc, with, commas", int64(1))
	f.Add("name\"with\"quotes", "desc\\with\\backslashes", int64(999))
	f.Add("a", strings.Repeat("x", 1000), int64(-1))
	f.Add("x\ry\nz", "inject\r\nheader", int64(42))

	f.Fuzz(func(t *testing.T, name, desc string, durNanos int64) {
		st := &ServerTiming{} //nolint:exhaustruct // only metrics field is relevant for this test
		st.Record(name, desc, time.Duration(durNanos))

		val := st.HeaderValue()

		// No CRLF injection — the header value must never contain raw newlines
		// regardless of input (security: prevents HTTP header splitting)
		if strings.ContainsAny(val, "\r\n") {
			t.Errorf("HeaderValue contains CRLF: %q", val)
		}

		// Must be a valid RFC 7230 token for the metric name (sanitized)
		// and valid quoted-string for the description (escaped)
		// Basic sanity: no unescaped quotes in description portion
		if strings.Count(val, `"`)%2 != 0 {
			t.Errorf("HeaderValue has unbalanced quotes: %q", val)
		}
	})
}

// FuzzServerTimingMiddleware verifies the middleware doesn't crash on any request
// and produces a valid header when metrics are recorded.
func FuzzServerTimingMiddleware(f *testing.F) {
	f.Add("GET", "/api/users")
	f.Add("", "")
	f.Add(strings.Repeat("A", 500), strings.Repeat("/", 100))

	f.Fuzz(func(t *testing.T, method, path string) {
		handler := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			st := ServerTimingFromContext(r.Context())
			if st == nil {
				t.Error("ServerTiming not in context")

				return
			}

			st.Record("test", "fuzz metric", time.Microsecond)
			w.WriteHeader(http.StatusOK)
		}))

		if method == "" {
			method = http.MethodGet
		}

		if path == "" {
			path = "/"
		}

		r := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		hdr := w.Header().Get(headerServerTiming)
		if strings.ContainsAny(hdr, "\r\n") {
			t.Errorf("Server-Timing header contains CRLF: %q", hdr)
		}
	})
}

// TestServerTimingNilReceiver ensures nil-receiver pattern is a safe no-op.
func TestServerTimingNilReceiver(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// All of these should be no-ops, not panics
	RecordServerTiming(ctx, "test", "desc", time.Millisecond)

	stop := MeasureServerTiming(ctx, "test")
	stop()

	// With an actual nil collector
	var st *ServerTiming
	st.Record("test", "desc", time.Millisecond)

	_ = st.Measure("test")
	if val := st.HeaderValue(); val != "" {
		t.Errorf("nil ServerTiming HeaderValue = %q, want empty", val)
	}
}
