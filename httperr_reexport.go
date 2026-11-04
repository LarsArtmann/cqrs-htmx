package cqrshtmx

import (
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
)

// The error surface — CQRS error families mapped to HTTP status codes, the
// RFC 9457 problem-details payload, and the ready-made error handlers — lives
// in the httperr subpackage. It depends only on the HTMX request vocabulary and
// the shared response plumbing, so a consumer can adopt the error contract
// without the handler builder.
//
// Everything below is a re-export: cqrshtmx.ErrForbidden and
// httperr.ErrForbidden are the same error value, cqrshtmx.MapError and
// httperr.MapError are the same function value, and cqrshtmx.StructuredError is
// the identical type.

// DefaultLoginRedirect is the login path used when a Config or call site does
// not supply one.
const DefaultLoginRedirect = httperr.DefaultLoginRedirect

// Cross-module auth error codes.
const (
	// CodeUnauthorized is the machine-readable code for unauthenticated requests.
	CodeUnauthorized = httperr.CodeUnauthorized
	// CodeForbidden is the machine-readable code for denied requests.
	CodeForbidden = httperr.CodeForbidden
)

// DefaultRetryAfter is the Retry-After delay sent on Transient (503) responses
// when the error carries no hint of its own.
const DefaultRetryAfter = httperr.DefaultRetryAfter

// Sentinel errors for HTTP→CQRS integration.
var (
	// ErrUnauthorized maps to 401 Unauthorized.
	ErrUnauthorized = httperr.ErrUnauthorized
	// ErrForbidden maps to 403 Forbidden.
	ErrForbidden = httperr.ErrForbidden
	// ErrDecodeFailed maps to 400 Bad Request.
	ErrDecodeFailed = httperr.ErrDecodeFailed
	// ErrDispatchFailed maps to 503 Service Unavailable.
	ErrDispatchFailed = httperr.ErrDispatchFailed
	// ErrEnforcerNil maps to 503 Service Unavailable.
	ErrEnforcerNil = httperr.ErrEnforcerNil
	// ErrValidationFailed maps to 400 Bad Request.
	ErrValidationFailed = httperr.ErrValidationFailed
	// ErrRequestTooLarge maps to 413 Request Entity Too Large.
	ErrRequestTooLarge = httperr.ErrRequestTooLarge
	// ErrMethodNotAllowed maps to 405 Method Not Allowed.
	ErrMethodNotAllowed = httperr.ErrMethodNotAllowed
)

type (
	// HTTPStatusCarrier is an error that declares its own HTTP status.
	HTTPStatusCarrier = httperr.HTTPStatusCarrier

	// RetryAfterCarrier is an error that declares how long a client should wait
	// before retrying.
	RetryAfterCarrier = httperr.RetryAfterCarrier

	// StructuredError is the RFC 9457 problem-details payload emitted for HTTP
	// error responses and SSE error broadcasts.
	StructuredError = httperr.StructuredError
)

// Error mapping, the RFC 9457 payload builders, and the ready-made handlers.
// Each var holds the identical function value declared in the httperr
// subpackage.
var (
	// MapError translates a CQRS error into an appropriate HTTP status code.
	MapError = httperr.MapError

	// SafeDetail returns the error text that is safe to expose for an HTTP
	// response of the given status.
	SafeDetail = httperr.SafeDetail

	// ErrorCode extracts the machine-readable error code from the innermost
	// errorfamily error in the cause chain.
	ErrorCode = httperr.ErrorCode

	// WithHTTPStatus makes MapError return the given status for err.
	WithHTTPStatus = httperr.WithHTTPStatus

	// WithRetryAfter attaches a Retry-After hint to err for 503 responses.
	WithRetryAfter = httperr.WithRetryAfter

	// NewStructuredError builds the RFC 9457 payload for a request.
	NewStructuredError = httperr.NewStructuredError

	// NewStructuredErrorWithContext builds the RFC 9457 payload from a context.
	NewStructuredErrorWithContext = httperr.NewStructuredErrorWithContext

	// DefaultErrorHandler maps CQRS errors to HTTP status codes and writes a
	// plain text error response, redirecting HTMX auth failures to the login
	// path.
	DefaultErrorHandler = httperr.DefaultErrorHandler

	// DefaultErrorHandlerWithRedirect is DefaultErrorHandler with a custom login
	// redirect path.
	DefaultErrorHandlerWithRedirect = httperr.DefaultErrorHandlerWithRedirect

	// DefaultErrorHandlerWithRequestID is DefaultErrorHandler that prefixes the
	// message with the request ID when one is in context.
	DefaultErrorHandlerWithRequestID = httperr.DefaultErrorHandlerWithRequestID

	// DefaultErrorHandlerWithRedirectAndRequestID is
	// DefaultErrorHandlerWithRedirect that prefixes the message with the request
	// ID.
	DefaultErrorHandlerWithRedirectAndRequestID = httperr.DefaultErrorHandlerWithRedirectAndRequestID

	// JSONErrorHandler writes errors as JSON responses.
	JSONErrorHandler = httperr.JSONErrorHandler

	// JSONErrorHandlerWithRedirect writes errors as JSON responses with a custom
	// login redirect path.
	JSONErrorHandlerWithRedirect = httperr.JSONErrorHandlerWithRedirect

	// ProblemDetailsErrorHandler writes errors as RFC 9457 problem details,
	// using StructuredError — the same shape emitted for SSE error broadcasts.
	ProblemDetailsErrorHandler = httperr.ProblemDetailsErrorHandler

	// ProblemDetailsErrorHandlerWithRedirect is ProblemDetailsErrorHandler with a
	// custom login redirect path.
	ProblemDetailsErrorHandlerWithRedirect = httperr.ProblemDetailsErrorHandlerWithRedirect
)

// Unexported aliases keep the root package's own wiring-error call sites
// reading the way they always have. They point at the single httperr values —
// no second sentinel with a duplicate code.
var (
	errCommandsNil        = httperr.ErrCommandsNil
	errQueriesNil         = httperr.ErrQueriesNil
	errDecoderMissing     = httperr.ErrDecoderMissing
	errDecoderReturnedNil = httperr.ErrDecoderReturnedNil
)

const defaultLoginRedirect = httperr.DefaultLoginRedirect
