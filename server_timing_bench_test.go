package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkServerTiming_DisabledOverhead measures the cost of calling
// MeasureServerTiming when no collector is in context (middleware not active).
// This should be near-zero — just a context.Value lookup + nil check.
func BenchmarkServerTiming_DisabledOverhead(b *testing.B) {
	ctx := context.Background()
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	b.ResetTimer()

	for range b.N {
		stop := MeasureServerTiming(r.Context(), "db")
		stop()
	}
}

// BenchmarkServerTiming_EnabledMeasure measures the cost of Measure+stop when
// a collector IS active (middleware enabled).
func BenchmarkServerTiming_EnabledMeasure(b *testing.B) {
	st := newServerTiming()

	b.ResetTimer()

	for range b.N {
		stop := st.Measure("db")
		stop()
	}
}

// BenchmarkServerTiming_EnabledMeasureViaContext measures the same but going
// through the context-aware helper (includes ctx.Value lookup per call).
func BenchmarkServerTiming_EnabledMeasureViaContext(b *testing.B) {
	st := newServerTiming()
	ctx := WithServerTiming(context.Background(), st)

	b.ResetTimer()

	for range b.N {
		stop := MeasureServerTiming(ctx, "db")
		stop()
	}
}

// BenchmarkServerTiming_Record measures direct Record calls.
func BenchmarkServerTiming_Record(b *testing.B) {
	st := newServerTiming()

	b.ResetTimer()

	for range b.N {
		st.Record("metric", "description", 1234567*time.Nanosecond)
	}
}

// BenchmarkServerTiming_HeaderValue measures rendering 5 metrics to the wire
// format (the typical flush-time cost per request).
func BenchmarkServerTiming_HeaderValue(b *testing.B) {
	st := newServerTiming()
	st.Record("total", "Total request", 12000000*time.Nanosecond)
	st.Record("decode", "", 1000000*time.Nanosecond)
	st.Record("auth", "", 500000*time.Nanosecond)
	st.Record("dispatch", "Command dispatch", 8000000*time.Nanosecond)
	st.Record("render", "", 2000000*time.Nanosecond)

	b.ResetTimer()

	for range b.N {
		_ = st.HeaderValue()
	}
}

// BenchmarkServerTiming_MiddlewareDisabledPassthrough measures the overhead
// of ServerTimingMiddlewareWhen(false) on a no-op handler — should be nearly
// identical to calling the handler directly.
func BenchmarkServerTiming_MiddlewareDisabledPassthrough(b *testing.B) {
	handler := ServerTimingMiddlewareWhen(func(*http.Request) bool { return false })(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()

	for range b.N {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
	}
}
