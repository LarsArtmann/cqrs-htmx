package cqrshtmx

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	logFieldCorrelationID = "correlation_id"
	logFieldUserID        = "user_id"
	logFieldRequestID     = "request_id"
)

// LogFormatter decides what string to write for a request/response pair.
// The returned string is written as a single log line.
type LogFormatter func(r *http.Request, status int, duration time.Duration) string

// LogWriter receives a formatted log line.
type LogWriter func(line string)

// contextFields extracts correlation ID, user ID, and request ID from context
// into a string map. Only non-zero values are included.
func contextFields(r *http.Request) map[string]string {
	fields := make(map[string]string)
	if cid := CorrelationIDFromContext(r.Context()); !cid.IsZero() {
		fields[logFieldCorrelationID] = cid.String()
	}
	if uid := UserIDFromContext(r.Context()); !uid.IsZero() {
		fields[logFieldUserID] = uid.String()
	}
	if rid := RequestIDFromContext(r.Context()); !rid.IsZero() {
		fields[logFieldRequestID] = rid.String()
	}
	return fields
}

// DefaultLogFormatter writes a concise log line with method, path, status,
// duration, and optional correlation ID and user ID when present in context.
//
// Format:
//
//	METHOD PATH → STATUS (DURATION) [correlation=CORR_ID] [user=USER_ID]
func DefaultLogFormatter(r *http.Request, status int, duration time.Duration) string {
	method := r.Method
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}

	var extra string
	fields := contextFields(r)
	if v, ok := fields[logFieldCorrelationID]; ok {
		extra += " [correlation=" + v + "]"
	}
	if v, ok := fields[logFieldUserID]; ok {
		extra += " [user=" + v + "]"
	}

	return method + " " + path + " → " + http.StatusText(status) +
		" (" + duration.String() + ")" + extra
}

// JSONLogFormatter writes a structured JSON log line with method, path, status,
// duration, and optional correlation ID and user ID when present in context.
//
// Output format:
//
//	{"method":"GET","path":"/users","status":"OK","duration":"1.234ms","correlation_id":"...","user_id":"..."}
func JSONLogFormatter(r *http.Request, status int, duration time.Duration) string {
	entry := map[string]any{
		"method":   r.Method,
		"path":     r.URL.Path,
		"status":   http.StatusText(status),
		"duration": duration.String(),
	}

	if r.URL.RawQuery != "" {
		entry["query"] = r.URL.RawQuery
	}

	for key, val := range contextFields(r) {
		entry[key] = val
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return `{"error":"json marshal failed"}`
	}

	return string(b)
}

// RequestLogging returns HTTP middleware that logs each request.
//
// If formatter is nil, DefaultLogFormatter is used.
// If writer is nil, the formatter output is silently discarded (useful for
// tests that only care about side effects like status code capture).
func RequestLogging(
	formatter LogFormatter,
	writer LogWriter,
) func(http.Handler) http.Handler {
	if formatter == nil {
		formatter = DefaultLogFormatter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := newStatusRecorder(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			if writer != nil {
				writer(formatter(r, rw.status, duration))
			}
		})
	}
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, status: 0, wrote: false}
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return fmt.Errorf("push %q: %w", target, pusher.Push(target, opts))
	}
	return http.ErrNotSupported
}

// RequestLoggingSlog returns HTTP middleware that logs each request using
// structured logging via log/slog. It captures method, path, status, duration,
// and any correlation ID, user ID, or request ID present in the context.
//
// Usage:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	middleware := cqrshtmx.RequestLoggingSlog(logger)
func RequestLoggingSlog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := newStatusRecorder(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("duration", duration),
			}

			if r.URL.RawQuery != "" {
				attrs = append(attrs, slog.String("query", r.URL.RawQuery))
			}

			for key, val := range contextFields(r) {
				attrs = append(attrs, slog.String(key, val))
			}

			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
		})
	}
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if !r.wrote {
		r.status = http.StatusOK
		r.wrote = true
	}

	return r.ResponseWriter.Write(p) //nolint:wrapcheck // delegate to underlying ResponseWriter
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	conn, rw, err := h.Hijack()
	return conn, rw, err //nolint:wrapcheck // delegate to underlying ResponseWriter
}
