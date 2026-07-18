package cqrshtmx

import (
	"bufio"
	"net"
	"net/http"
)

// delegatingWriter wraps http.ResponseWriter and delegates Flush, Hijack,
// Push, and Unwrap to the underlying writer when available. Embed this in
// custom ResponseWriter wrappers to preserve SSE, WebSocket, and HTTP/2
// capabilities without duplicating the delegation boilerplate.
//
// Type assertion note: methods use the embedded ResponseWriter (not the
// embedding type), so Go's method promotion makes Flush/Hijack/Push/Unwrap
// available on any struct that embeds delegatingWriter.
type delegatingWriter struct {
	http.ResponseWriter
}

// Flush delegates to the underlying Flusher, if available.
func (w delegatingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack delegates to the underlying Hijacker so WebSocket upgrades work
// through wrappers. Returns http.ErrNotSupported when unavailable.
func (w delegatingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}

	return h.Hijack() //nolint:wrapcheck // delegate to underlying Hijacker
}

// Push delegates HTTP/2 server push to the underlying Pusher, if available.
func (w delegatingWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts) //nolint:wrapcheck // delegate to underlying Pusher
	}

	return http.ErrNotSupported
}

// Unwrap exposes the underlying ResponseWriter so http.ResponseController
// (Go 1.20+) can locate Flusher/Hijacker/Pusher through wrapper chains.
func (w delegatingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
