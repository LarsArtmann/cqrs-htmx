package cqrshtmx

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// Server-Timing header (W3C Server Timing API).
// Spec: https://w3c.github.io/server-timing/
//
// Wire format (metrics joined by ", "):
//
//	Server-Timing: db;desc="Database";dur=53, cache;desc="Cache";dur=2, total;dur=120
//
// dur is measured in milliseconds (fractional allowed). The header MUST be set
// before the response body is committed, which is why the middleware wraps the
// http.ResponseWriter and injects it at the first WriteHeader/Write call.

const headerServerTiming = "Server-Timing"

// ServerTiming collects named timing metrics and renders them as a
// Server-Timing response header value (W3C Server Timing API).
//
// It is concurrency-safe. Create one via ServerTimingMiddleware (which stores
// it in the request context) and access it in handlers with
// ServerTimingFromContext.
//
// When disabled (created with newServerTiming(false)), every method is a
// cheap no-op and HeaderValue returns "" — so handlers can call Record/Measure
// unconditionally without per-request branching.
type ServerTiming struct {
	mu      sync.Mutex
	metrics []serverTimingMetric
	enabled bool
}

type serverTimingMetric struct {
	name string
	desc string
	dur  time.Duration // zero duration = omit dur param
}

// newServerTiming creates a collector. When enabled is false the collector is
// a no-op: Record/Measure discard their input and HeaderValue returns "".
func newServerTiming(enabled bool) *ServerTiming {
	return &ServerTiming{
		mu:      sync.Mutex{},
		metrics: nil,
		enabled: enabled,
	}
}

// Enabled reports whether the collector is active.
func (st *ServerTiming) Enabled() bool {
	return st != nil && st.enabled
}

// Record adds a named timing metric.
//
// name must be a valid HTTP token (RFC 7230); invalid characters are replaced
// with '_' so the emitted header is always well-formed. desc is an optional
// human-readable description (pass "" to omit it); it is escaped and quoted
// automatically, so it may contain commas/semicolons/quotes. A zero dur omits
// the dur parameter for that metric.
//
// Safe to call concurrently from any goroutine.
func (st *ServerTiming) Record(name, desc string, dur time.Duration) {
	if !st.Enabled() {
		return
	}
	cleaned := sanitizeMetricName(name)
	if cleaned == "" {
		return
	}
	st.mu.Lock()
	st.metrics = append(st.metrics, serverTimingMetric{name: cleaned, desc: desc, dur: dur})
	st.mu.Unlock()
}

// Measure returns a function that records the elapsed time since it was
// called, under the given metric name. It is designed for the defer idiom:
//
//	defer st.Measure("db")()
//
// When the collector is disabled, the returned function is a no-op.
func (st *ServerTiming) Measure(name string) func() {
	if !st.Enabled() {
		return func() {}
	}
	start := time.Now()
	return func() { st.Record(name, "", time.Since(start)) }
}

// MeasureWithDesc is like Measure but also records a description.
func (st *ServerTiming) MeasureWithDesc(name, desc string) func() {
	if !st.Enabled() {
		return func() {}
	}
	start := time.Now()
	return func() { st.Record(name, desc, time.Since(start)) }
}

// HeaderValue renders the collected metrics as a Server-Timing header value.
// Returns "" when the collector is disabled or has no metrics.
func (st *ServerTiming) HeaderValue() string {
	if !st.Enabled() {
		return ""
	}
	st.mu.Lock()
	metrics := st.metrics
	st.mu.Unlock()
	if len(metrics) == 0 {
		return ""
	}

	var b strings.Builder
	for i, m := range metrics {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(m.name)
		if m.desc != "" {
			b.WriteString(`;desc="`)
			b.WriteString(escapeQuotedString(m.desc))
			b.WriteByte('"')
		}
		if m.dur != 0 {
			b.WriteString(";dur=")
			b.WriteString(formatMillis(m.dur))
		}
	}
	return b.String()
}

// String is an alias for HeaderValue.
func (st *ServerTiming) String() string { return st.HeaderValue() }

// formatMillis renders a duration as milliseconds with the shortest
// round-tripping representation (e.g. 53ms → "53", 0.5ms → "0.5").
// The Server-Timing spec expresses dur in milliseconds; fractional values are
// permitted, so sub-millisecond timings are not lost.
func formatMillis(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	return strconv.FormatFloat(ms, 'f', -1, 64)
}

// escapeQuotedString escapes a value for an RFC 7230 quoted-string: backslash
// and double-quote are backslash-escaped. All other bytes (including commas,
// semicolons, and spaces) are safe inside a quoted-string.
func escapeQuotedString(s string) string {
	if !strings.ContainsAny(s, `"\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 4)
	for i := range s {
		c := s[i]
		if c == '\\' || c == '"' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}

// sanitizeMetricName enforces the RFC 7230 "token" rule for a Server-Timing
// metric name. Invalid bytes are replaced with '_'. If the result is empty or
// starts with an invalid char, leading underscores are stripped.
func sanitizeMetricName(name string) string {
	const tchar = "!#$%&'*+-.^_`|~0123456789" +
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if name == "" {
		return ""
	}
	cleaned := strings.Map(func(r rune) rune {
		if r < 128 && strings.ContainsRune(tchar, r) {
			return r
		}
		return '_'
	}, name)
	if cleaned == "" {
		return ""
	}
	// A leading '_' from sanitization is acceptable per the token rule.
	return cleaned
}

// ---------------------------------------------------------------------------
// Context integration
// ---------------------------------------------------------------------------

type serverTimingKey struct{}

// WithServerTiming stores a *ServerTiming in the context.
func WithServerTiming(ctx context.Context, st *ServerTiming) context.Context {
	return context.WithValue(ctx, serverTimingKey{}, st)
}

// ServerTimingFromContext retrieves the *ServerTiming stored by
// ServerTimingMiddleware (or WithServerTiming). Returns nil when no
// Server-Timing collector is present — callers may then skip timing, or use
// the helpers below which are nil-safe.
func ServerTimingFromContext(ctx context.Context) *ServerTiming {
	st, _ := ctx.Value(serverTimingKey{}).(*ServerTiming)
	return st
}

// RecordServerTiming is a nil-safe shortcut for ServerTimingFromContext(ctx).
// Record(name, desc, dur). It is a no-op when no collector is present.
func RecordServerTiming(ctx context.Context, name, desc string, dur time.Duration) {
	if st := ServerTimingFromContext(ctx); st != nil {
		st.Record(name, desc, dur)
	}
}

// MeasureServerTiming is a nil-safe shortcut that mirrors ServerTiming.Measure.
// Use it with defer:
//
//	defer MeasureServerTiming(r.Context(), "render")()
//
// It is a no-op when no collector is present.
func MeasureServerTiming(ctx context.Context, name string) func() {
	if st := ServerTimingFromContext(ctx); st != nil {
		return st.Measure(name)
	}
	return func() {}
}

// ---------------------------------------------------------------------------
// ResponseWriter wrapper
// ---------------------------------------------------------------------------

// serverTimingWriter wraps http.ResponseWriter to inject the Server-Timing
// header at the moment the response is committed (first WriteHeader or Write).
// It delegates Flush/Hijack/Push so SSE, WebSocket, and HTTP/2 push continue
// to work transparently through the wrapper.
type serverTimingWriter struct {
	http.ResponseWriter
	st       *ServerTiming
	start    time.Time
	injected bool
	wrote    bool
}

func (w *serverTimingWriter) WriteHeader(code int) {
	w.flushHeader()
	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *serverTimingWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.flushHeader()
		w.wrote = true
	}
	return w.ResponseWriter.Write(b) //nolint:wrapcheck // delegate to underlying ResponseWriter
}

// flushHeader finalizes the total metric and writes the Server-Timing header
// exactly once. Idempotent.
func (w *serverTimingWriter) flushHeader() {
	if w.injected {
		return
	}
	w.injected = true
	w.st.prependTotal("total", "Total request", time.Since(w.start))
	if h := w.st.HeaderValue(); h != "" {
		w.ResponseWriter.Header().Set(headerServerTiming, h)
	}
}

// Flush delegates to the underlying Flusher so streaming responses (SSE)
// remain flushable through the wrapper.
func (w *serverTimingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack delegates to the underlying Hijacker so WebSocket upgrades work
// through the wrapper. It returns the same values as http.Hijacker.Hijack.
func (w *serverTimingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack() //nolint:wrapcheck // delegate to underlying Hijacker
	}
	return nil, nil, errHijackUnavailable
}

var errHijackUnavailable = event.NewInfrastructure(
	"cqrshtmx.server_timing.hijack_unavailable",
	"[cqrs-htmx] underlying ResponseWriter does not implement http.Hijacker",
)

// Push delegates HTTP/2 server push to the underlying Pusher, if available.
func (w *serverTimingWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts) //nolint:wrapcheck // delegate to underlying Pusher
	}
	return http.ErrNotSupported
}

// Unwrap exposes the underlying ResponseWriter so http.ResponseController (Go
// 1.20+) can locate Flusher/Hijacker/Pusher through this wrapper.
func (w *serverTimingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// prependTotal inserts the total metric at the front of the collector. It is
// only called once, at flushHeader time, so the total reflects time-to-first
// byte (TTFB) — the standard semantics for a Server-Timing total.
func (st *ServerTiming) prependTotal(name, desc string, dur time.Duration) {
	if !st.Enabled() {
		return
	}
	st.mu.Lock()
	st.metrics = append(
		[]serverTimingMetric{{name: name, desc: desc, dur: dur}},
		st.metrics...,
	)
	st.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// ServerTimingMiddleware enables the Server-Timing response header for every
// request.
//
// It measures the total request time-to-first-byte and exposes a *ServerTiming
// collector in the request context so downstream handlers can record
// sub-metrics:
//
//	mux.Use(cqrshtmx.ServerTimingMiddleware())
//	// …in a handler:
//	defer cqrshtmx.MeasureServerTiming(r.Context(), "db")()
//	db.Query(...)
//
// Server-Timing can leak internal performance details; gate it for
// debug/admin requests with ServerTimingMiddlewareWhen.
//
// This middleware is OPT-IN and never auto-applied (library principle).
func ServerTimingMiddleware() func(http.Handler) http.Handler {
	return ServerTimingMiddlewareWhen(func(*http.Request) bool { return true })
}

// ServerTimingMiddlewareWhen enables Server-Timing only for requests where pred
// returns true. When pred returns false the request is passed through with no
// ResponseWriter wrapping and a disabled collector in context — handlers calling
// Record/Measure incur only a cheap no-op, so no per-handler branching is needed.
//
// Use this to gate Server-Timing behind a debug flag, an admin role, or a
// request query/header check:
//
//	cqrshtmx.ServerTimingMiddlewareWhen(func(r *http.Request) bool {
//	    return r.URL.Query().Has("debug")
//	})
func ServerTimingMiddlewareWhen(pred func(*http.Request) bool) func(http.Handler) http.Handler {
	if pred == nil {
		pred = func(*http.Request) bool { return false }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enabled := pred(r)
			st := newServerTiming(enabled)
			ctx := WithServerTiming(r.Context(), st)

			if !enabled {
				// Zero-overhead passthrough: no writer wrapping.
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			wrapped := &serverTimingWriter{
				ResponseWriter: w,
				st:             st,
				start:          time.Now(),
				injected:       false,
				wrote:          false,
			}
			next.ServeHTTP(wrapped, r.WithContext(ctx))
		})
	}
}
