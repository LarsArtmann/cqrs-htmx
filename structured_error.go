package cqrshtmx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// StructuredError is a transport-agnostic error payload following the
// RFC 7807 (Problem Details for HTTP APIs) shape. It carries enough
// context for clients to render or handle errors uniformly across HTTP,
// SSE, and WebSocket transports.
//
// StructuredError implements the error interface and supports errors.Is/
// errors.As via Unwrap, so it can participate in Go error chains.
//
// Serialize as JSON for SSE event data or WebSocket message payloads:
//
//	payload := cqrshtmx.NewStructuredError(err, r)
//	jsonBytes, _ := json.Marshal(payload)
//	broadcaster.Broadcast(cqrshtmx.SSEEvent{
//	    Event: "commandError",
//	    Data:  string(jsonBytes),
//	})
type StructuredError struct {
	// Type is a URI or short token identifying the error type.
	// Defaults to "about:blank" per RFC 7807. For domain errors,
	// use a meaningful token like "validation_failed" or "conflict".
	Type string `json:"type"`

	// Title is a short, human-readable summary of the problem type.
	// It should not change between occurrences of the same type.
	Title string `json:"title"`

	// Status is the HTTP status code appropriate for this error.
	// Derived from MapError — carries the same family semantics.
	Status int `json:"status"`

	// Detail is a specific, human-readable explanation of this
	// particular occurrence of the error. For client errors (status < 500)
	// this is the raw error message, which describes the caller's input.
	// For server faults (status >= 500) it is redacted to the error family's
	// generic public-safe message so internal detail is not leaked to
	// SSE/WS/HTTP clients. The original error is preserved in cause for
	// server-side logging.
	Detail string `json:"detail"`

	// Message is the error family's generic, public-safe guidance for this
	// class of problem (RFC 7807 extension). Always safe to show; derived from
	// go-error-family's Family.DefaultMessage.
	Message string `json:"message,omitempty"`

	// Why explains, in public-safe terms, what happened for this family
	// (RFC 7807 extension). Derived from Family.DefaultWhy.
	Why string `json:"why,omitempty"`

	// Fix suggests a public-safe remediation for this family
	// (RFC 7807 extension). Derived from Family.DefaultFix.
	Fix string `json:"fix,omitempty"`

	// Instance is a token identifying this specific occurrence.
	// Usually the request ID or correlation ID for tracing.
	Instance string `json:"instance,omitempty"`

	cause error `json:"-"`
}

// NewStructuredError creates a StructuredError from a Go error and HTTP request.
// It maps the error to an HTTP status via MapError, derives a title from the
// error family, and extracts the request ID for tracing.
//
// If err is nil, returns a zero-value StructuredError (callers should check err first).
func NewStructuredError(err error, r *http.Request) StructuredError {
	if err == nil {
		return StructuredError{} //nolint:exhaustruct // zero-value return
	}
	return newStructuredErrorFromContext(err, requestContextOrBackground(r))
}

// NewStructuredErrorWithContext is like NewStructuredError but accepts a context
// instead of a request. Useful when you only have the dispatch context.
func NewStructuredErrorWithContext(err error, ctx context.Context) StructuredError {
	if err == nil {
		return StructuredError{} //nolint:exhaustruct // zero-value return
	}
	return newStructuredErrorFromContext(err, ctx)
}

// newStructuredErrorFromContext is the shared builder used by
// NewStructuredError and NewStructuredErrorWithContext. Both functions
// delegate here after adapting their input (request vs context) to a
// context.Context.
func newStructuredErrorFromContext(err error, ctx context.Context) StructuredError {
	status := MapError(err)
	family := event.Classify(err)
	instance := ""
	if ctx != nil {
		if rid := RequestIDFromContext(ctx); !rid.IsZero() {
			instance = rid.String()
		}
	}
	return StructuredError{
		Type:     familyType(family),
		Title:    statusTitle(status),
		Status:   status,
		Detail:   SafeDetail(err, status, false),
		Message:  family.DefaultMessage(),
		Why:      family.DefaultWhy(),
		Fix:      family.DefaultFix(),
		Instance: instance,
		cause:    err,
	}
}

// requestContextOrBackground returns r.Context() when r is non-nil, otherwise
// context.Background(). Centralises the nil-request handling so callers can
// pass a possibly-nil *http.Request without triggering the contextcheck
// linter on a fresh context.Background().
func requestContextOrBackground(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

// JSON returns the StructuredError serialized as a JSON string.
// Convenience method for use in SSEEvent.Data or WS message payloads.
// Returns empty string if marshaling fails (should not happen for this type).
func (e StructuredError) JSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return `{"type":"about:blank","title":"Internal Server Error","status":500,"detail":"failed to marshal error"}`
	}
	return string(b)
}

// Error implements the error interface. Returns the Detail field.
func (e StructuredError) Error() string { return e.Detail }

// String implements fmt.Stringer. Returns the full JSON representation,
// making StructuredError suitable for structured logging.
func (e StructuredError) String() string { return e.JSON() }

// Unwrap returns the original error that produced this StructuredError,
// enabling errors.Is and errors.As to traverse the error chain.
// Returns nil if no cause was stored.
func (e StructuredError) Unwrap() error { return e.cause }

func familyType(family event.Family) string {
	if family.IsValid() {
		return family.String()
	}
	return "about:blank"
}

func statusTitle(status int) string {
	// Delegate to http.StatusText — the status code is already derived from
	// family.HTTPStatus() upstream, so this stays in sync automatically.
	if text := http.StatusText(status); text != "" {
		return text
	}
	return "Internal Server Error"
}
