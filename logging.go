package cqrshtmx

import (
	"net/http"
	"time"
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
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}

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

			rw := &statusRecorder{ResponseWriter: w, status: 0, wrote: false}

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

func (r *statusRecorder) Write(p []byte) (int, error) {
	if !r.wrote {
		r.status = http.StatusOK
		r.wrote = true
	}

	return r.ResponseWriter.Write(p) //nolint:wrapcheck // delegate to underlying ResponseWriter
}
