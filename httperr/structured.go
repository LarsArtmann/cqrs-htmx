package httperr

import (
	"context"
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// StructuredError is a transport-agnostic error payload following the
// RFC 7807 (Problem Details for HTTP APIs) shape. It carries enough
// context for clients to render or handle errors uniformly across HTTP,
// SSE, and HTTP transports.
//
// StructuredError implements the error interface and supports errors.Is/
// errors.As via Unwrap, so it can participate in Go error chains.
//
// Serialize as JSON for SSE event data or HTTP response bodies:
//
//	payload := cqrshtmx.NewStructuredError(err, r)
//	jsonBytes, _ := json.Marshal(payload)
//	broadcaster.Broadcast(sse.Event{
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
	// SSE and HTTP clients. The original error is preserved in cause for
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

	// Code is the error family's machine-readable error code (RFC 7807 extension).
	// Matches the "code" field emitted by JSONErrorHandler, so clients get the
	// same code regardless of whether the response is application/json or
	// application/problem+json. Empty when the error has no Code() method.
	Code string `json:"code,omitempty"`

	// Retryable reports whether the caller may legitimately retry the
	// operation (RFC 7807 extension; derived from the error family —
	// Transient and Infrastructure are retryable, client faults are not).
	// API clients get an explicit machine-readable signal instead of
	// re-deriving it from the status code. Omitted from the wire when
	// false, so non-retryable documents are byte-identical to before (the
	// omitempty tag would NOT do this under encoding/json/v2, which does not
	// treat false as empty — omitzero does, on both v1 1.24+ and v2).
	Retryable bool `json:"retryable,omitzero"`

	// Context carries the error's machine-readable context entries (RFC 7807
	// extension) — go-error-family's ErrorContext() map (form_key,
	// max_body_size, ...). Machine context lets API clients REACT (re-render
	// the offending form field, re-size the body) instead of only displaying
	// prose. Populated ONLY for client-fault statuses (<500) — the same
	// redaction rule as Detail: 5xx context entries are internal diagnostic
	// material and never reach the wire. Empty when none applies.
	Context map[string]string `json:"context,omitempty"`

	cause error `json:"-"`
}

// NewStructuredError creates a StructuredError from a Go error and HTTP request.
// It maps the error to an HTTP status via MapError, derives a title from the
// error family, and extracts the request ID for tracing.
//
// The detail is redacted for a PUBLIC audience (see [AudiencePublic]); use
// [NewStructuredErrorForAudience] when the reader is an authenticated
// operator.
//
// If err is nil, returns a zero-value StructuredError (callers should check err first).
func NewStructuredError(err error, r *http.Request) StructuredError {
	if err == nil {
		return StructuredError{} //nolint:exhaustruct_v5 // zero-value return
	}

	return newStructuredErrorFromContext(err, requestContextOrBackground(r), AudiencePublic)
}

// NewStructuredErrorWithContext is like NewStructuredError but accepts a context
// instead of a request. Useful when you only have the dispatch context.
func NewStructuredErrorWithContext(err error, ctx context.Context) StructuredError {
	if err == nil {
		return StructuredError{} //nolint:exhaustruct_v5 // zero-value return
	}

	return newStructuredErrorFromContext(err, ctx, AudiencePublic)
}

// NewStructuredErrorForAudience is [NewStructuredError] with an explicit
// audience: an operator reading the body gets the raw server-fault detail
// and its machine context (so the problem is actionable from the response
// alone); a public reader gets the redacted form. Client-fault detail is
// identical for both audiences.
func NewStructuredErrorForAudience(err error, r *http.Request, audience Audience) StructuredError {
	if err == nil {
		return StructuredError{} //nolint:exhaustruct_v5 // zero-value return
	}

	return newStructuredErrorFromContext(err, requestContextOrBackground(r), audience)
}

// NewStructuredErrorForAudienceWithContext is
// [NewStructuredErrorForAudience] with a context instead of a request.
func NewStructuredErrorForAudienceWithContext(
	err error, ctx context.Context, audience Audience,
) StructuredError {
	if err == nil {
		return StructuredError{} //nolint:exhaustruct_v5 // zero-value return
	}

	return newStructuredErrorFromContext(err, ctx, audience)
}

// newStructuredErrorFromContext is the shared builder used by the
// NewStructuredError* constructors. Both functions
// delegate here after adapting their input (request vs context) to a
// context.Context.
func newStructuredErrorFromContext(err error, ctx context.Context, audience Audience) StructuredError {
	status := MapError(err)
	family := errorfamily.Classify(err)
	instance := ""

	if ctx != nil {
		if rid := hx.RequestIDFromContext(ctx); !rid.IsZero() {
			instance = rid.String()
		}
	}

	contextual, _ := err.(interface{ ErrorContext() map[string]string })

	var errContext map[string]string
	if contextual != nil && (status < 500 || audience == AudienceOperator) {
		// Client-fault statuses carry the context — it describes the caller's
		// input (the same reason Detail stays raw below 500). For PUBLIC
		// readers 5xx redacts BOTH detail and context: context entries
		// (user_id, paths, sizes) are internal diagnostic material, not
		// public-safe payloads. Operators get both — that is the point of
		// [AudienceOperator].
		errContext = contextual.ErrorContext()
	}

	return StructuredError{
		Type:      familyType(family),
		Title:     statusTitle(status),
		Status:    status,
		Detail:    SafeDetailFor(err, status, audience),
		Message:   family.DefaultMessage(),
		Why:       family.DefaultWhy(),
		Fix:       family.DefaultFix(),
		Instance:  instance,
		Code:      ErrorCode(err),
		Retryable: family.IsRetryable(),
		Context:   errContext,
		cause:     err,
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
// Convenience method for use in SSEEvent.Data or HTTP response bodies.
// Returns a minimum-valid RFC 7807 document if marshalling fails (should not
// happen for this type).
func (e StructuredError) JSON() string {
	return hx.MarshalJSONOrFallback(
		e,
		`{"type":"about:blank","title":"Internal Server Error","status":500,"detail":"failed to marshal error"}`,
	)
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
