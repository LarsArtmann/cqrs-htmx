package cqrshtmx

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	errorfamily "github.com/larsartmann/go-error-family"
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

// DefaultLogFormatter writes a concise log line with method, path, status,
// duration, and optional correlation ID and user ID when present in context.
//
// Format:
//
//	METHOD PATH → STATUS (DURATION) [correlation=CORR_ID] [user=USER_ID]
func DefaultLogFormatter(r *http.Request, status int, duration time.Duration) string {
	method := r.Method
	path := r.URL.Path

	var extra string
	if cid := CorrelationIDFromContext(r.Context()); !cid.IsZero() {
		extra += " [correlation=" + cid.String() + "]"
	}
	if uid := UserIDFromContext(r.Context()); !uid.IsZero() {
		extra += " [user=" + uid.String() + "]"
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
var jsonLogBufferPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// JSONLogFormatter formats an HTTP request log entry as a JSON string.
// Uses a sync.Pool'd buffer for allocation efficiency across requests.
func JSONLogFormatter(r *http.Request, status int, duration time.Duration) string {
	entry := make(map[string]any, 7)
	entry["method"] = r.Method
	entry["path"] = r.URL.Path
	entry["status"] = http.StatusText(status)
	entry["duration"] = duration.String()

	if cid := CorrelationIDFromContext(r.Context()); !cid.IsZero() {
		entry[logFieldCorrelationID] = cid.String()
	}
	if uid := UserIDFromContext(r.Context()); !uid.IsZero() {
		entry[logFieldUserID] = uid.String()
	}
	if rid := RequestIDFromContext(r.Context()); !rid.IsZero() {
		entry[logFieldRequestID] = rid.String()
	}

	buf, ok := jsonLogBufferPool.Get().(*bytes.Buffer)
	if !ok {
		buf = new(bytes.Buffer)
	}
	buf.Reset()
	defer jsonLogBufferPool.Put(buf)

	data, err := json.Marshal(entry)
	if err != nil {
		return `{"error":"json marshal failed"}`
	}
	buf.Write(data)

	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
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

			rw := NewStatusRecorder(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			if writer != nil {
				writer(formatter(r, rw.Status(), duration))
			}
		})
	}
}

// ErrorRecorder captures the dispatch error for logging middleware.
// It is a separate concern from status recording — embedded by StatusRecorder
// but usable standalone by middleware that only needs error context.
type ErrorRecorder struct {
	dispatchErr error
}

// NewErrorRecorder returns an ErrorRecorder ready for use by middleware that
// only needs dispatch-error context without HTTP status recording.
func NewErrorRecorder() *ErrorRecorder {
	return &ErrorRecorder{} //nolint:exhaustruct // zero values are correct
}

// SetDispatchError captures the dispatch error so that logging middleware
// (RequestLoggingSlog) can include error context in the request log.
func (r *ErrorRecorder) SetDispatchError(err error) { r.dispatchErr = err }

// DispatchError returns the captured dispatch error, or nil if none was set.
func (r *ErrorRecorder) DispatchError() error { return r.dispatchErr }

// StatusRecorder wraps http.ResponseWriter to capture the HTTP status code.
// It embeds delegatingWriter so Flush/Hijack/Push/Unwrap are promoted
// automatically — preserving SSE, WebSocket, and HTTP/2 capabilities.
// ErrorRecorder is embedded to capture dispatch errors for logging.
type StatusRecorder struct {
	delegatingWriter
	ErrorRecorder
	status int
	wrote  bool
}

// NewStatusRecorder wraps w to capture the status code. The initial status is
// 0 (unset) — callers should check WroteHeader() before relying on Status().
func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{
		delegatingWriter: delegatingWriter{ResponseWriter: w},
		ErrorRecorder:    ErrorRecorder{}, //nolint:exhaustruct // zero values are correct
		status:           0,
		wrote:            false,
	}
}

// Status returns the captured HTTP status code, or 0 if WriteHeader has not
// been called.
func (r *StatusRecorder) Status() int { return r.status }

// WroteHeader reports whether WriteHeader has been called.
func (r *StatusRecorder) WroteHeader() bool { return r.wrote }

// WriteHeader records the status code and delegates to the underlying ResponseWriter.
func (r *StatusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}
	r.ResponseWriter.WriteHeader(code)
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

			rw := NewStatusRecorder(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			attrs := make([]slog.Attr, 0, 7)
			attrs = append(
				attrs,
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.Status()),
				slog.Duration("duration", duration),
			)

			if cid := CorrelationIDFromContext(r.Context()); !cid.IsZero() {
				attrs = append(attrs, slog.String(logFieldCorrelationID, cid.String()))
			}
			if uid := UserIDFromContext(r.Context()); !uid.IsZero() {
				attrs = append(attrs, slog.String(logFieldUserID, uid.String()))
			}
			if rid := RequestIDFromContext(r.Context()); !rid.IsZero() {
				attrs = append(attrs, slog.String(logFieldRequestID, rid.String()))
			}

			if err := rw.DispatchError(); err != nil {
				attrs = appendDispatchErrorAttrs(attrs, err)
			}

			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
		})
	}
}

// Write implements io.Writer. It sets the status to 200 on first write if no
// explicit status was recorded via WriteHeader.
func (r *StatusRecorder) Write(p []byte) (int, error) {
	if !r.wrote {
		r.status = http.StatusOK
		r.wrote = true
	}

	return r.ResponseWriter.Write(p) //nolint:wrapcheck // delegate to underlying ResponseWriter
}

// dispatchErrorRecorder is implemented by ResponseWriters that capture
// the dispatch error for logging middleware. ErrorRecorder (embedded by
// StatusRecorder) implements it.
type dispatchErrorRecorder interface {
	SetDispatchError(err error)
}

// appendDispatchErrorAttrs extracts the error code, family, and context
// key-values from a classified error and appends them as slog attributes.
func appendDispatchErrorAttrs(attrs []slog.Attr, err error) []slog.Attr {
	if code := ErrorCode(err); code != "" {
		attrs = append(attrs, slog.String("error_code", code))
	}
	if family := errorfamily.Classify(err); family.IsValid() {
		attrs = append(attrs, slog.String("error_family", family.String()))
	}
	// Traverse the full error chain — context may be on inner errors
	// that were wrapped by dispatch error handling.
	current := err
	for current != nil {
		var ee *event.Error
		if errors.As(current, &ee) && ee != nil {
			for k, v := range ee.ErrorContext() {
				attrs = append(attrs, slog.String("error_ctx_"+k, v))
			}
			// Move past this event.Error to find deeper ones.
			current = errors.Unwrap(ee)
			continue
		}
		current = errors.Unwrap(current)
	}
	return attrs
}
