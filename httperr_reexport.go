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
// Every symbol in this file is a deprecated alias. Import the subpackage
// directly:
//
//	import "github.com/larsartmann/cqrs-htmx/v4/httperr"
//
// then write httperr.MapError, httperr.ErrForbidden, httperr.StructuredError.
// These aliases are identical types, error values and function values and are
// deleted in cqrs-htmx v5; see ADR-0062 and docs/migrations/v4-to-v5.md.

// DefaultLoginRedirect is the login path used when a Config or call site does
// not supply one.
//
// Deprecated: use httperr.DefaultLoginRedirect from github.com/larsartmann/cqrs-htmx/v4/httperr
// directly. Removed in cqrs-htmx v5.
const DefaultLoginRedirect = httperr.DefaultLoginRedirect

// Cross-module auth error codes.
const (
	// CodeUnauthorized is the machine-readable code for unauthenticated requests.
	//
	// Deprecated: use httperr.CodeUnauthorized from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	CodeUnauthorized = httperr.CodeUnauthorized
	// CodeForbidden is the machine-readable code for denied requests.
	//
	// Deprecated: use httperr.CodeForbidden from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	CodeForbidden = httperr.CodeForbidden
)

// DefaultRetryAfter is the Retry-After delay sent on Transient (503) responses
// when the error carries no hint of its own.
//
// Deprecated: use httperr.DefaultRetryAfter from github.com/larsartmann/cqrs-htmx/v4/httperr
// directly. Removed in cqrs-htmx v5.
const DefaultRetryAfter = httperr.DefaultRetryAfter

// Sentinel errors for HTTP→CQRS integration.
var (
	// ErrUnauthorized maps to 401 Unauthorized.
	//
	// Deprecated: use httperr.ErrUnauthorized from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrUnauthorized = httperr.ErrUnauthorized
	// ErrForbidden maps to 403 Forbidden.
	//
	// Deprecated: use httperr.ErrForbidden from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrForbidden = httperr.ErrForbidden
	// ErrDecodeFailed maps to 400 Bad Request.
	//
	// Deprecated: use httperr.ErrDecodeFailed from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrDecodeFailed = httperr.ErrDecodeFailed
	// ErrDispatchFailed maps to 503 Service Unavailable.
	//
	// Deprecated: use httperr.ErrDispatchFailed from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrDispatchFailed = httperr.ErrDispatchFailed
	// ErrEnforcerNil maps to 503 Service Unavailable.
	//
	// Deprecated: use httperr.ErrEnforcerNil from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrEnforcerNil = httperr.ErrEnforcerNil
	// ErrValidationFailed maps to 400 Bad Request.
	//
	// Deprecated: use httperr.ErrValidationFailed from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrValidationFailed = httperr.ErrValidationFailed
	// ErrRequestTooLarge maps to 413 Request Entity Too Large.
	//
	// Deprecated: use httperr.ErrRequestTooLarge from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrRequestTooLarge = httperr.ErrRequestTooLarge
	// ErrMethodNotAllowed maps to 405 Method Not Allowed.
	//
	// Deprecated: use httperr.ErrMethodNotAllowed from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrMethodNotAllowed = httperr.ErrMethodNotAllowed
)

type (
	// HTTPStatusCarrier is an error that declares its own HTTP status.
	//
	// Deprecated: use httperr.HTTPStatusCarrier from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	HTTPStatusCarrier = httperr.HTTPStatusCarrier

	// RetryAfterCarrier is an error that declares how long a client should wait
	// before retrying.
	//
	// Deprecated: use httperr.RetryAfterCarrier from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	RetryAfterCarrier = httperr.RetryAfterCarrier

	// StructuredError is the RFC 9457 problem-details payload emitted for HTTP
	// error responses and SSE error broadcasts.
	//
	// Deprecated: use httperr.StructuredError from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	StructuredError = httperr.StructuredError

	// Audience classifies who reads an error's detail (public redaction vs
	// operator diagnostics); the zero value is the fail-safe public form.
	//
	// Deprecated: use httperr.Audience from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	Audience = httperr.Audience
)

// Error mapping, the RFC 9457 payload builders, and the ready-made handlers.
// Each var holds the identical function value declared in the httperr
// subpackage.
var (
	// MapError translates a CQRS error into an appropriate HTTP status code.
	//
	// Deprecated: use httperr.MapError from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	MapError = httperr.MapError

	// SafeDetail returns the error text that is safe to expose for an HTTP
	// response of the given status.
	//
	// Deprecated: use httperr.SafeDetail from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	SafeDetail = httperr.SafeDetail

	// AudiencePublic redacts 5xx detail to the family default message.
	//
	// Deprecated: use httperr.AudiencePublic from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	AudiencePublic = httperr.AudiencePublic

	// AudienceOperator carries full 5xx detail and machine context.
	//
	// Deprecated: use httperr.AudienceOperator from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	AudienceOperator = httperr.AudienceOperator

	// SafeDetailFor is the audience-aware SafeDetail.
	//
	// Deprecated: use httperr.SafeDetailFor from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	SafeDetailFor = httperr.SafeDetailFor

	// ErrorCode extracts the machine-readable error code from the innermost
	// errorfamily error in the cause chain.
	//
	// Deprecated: use httperr.ErrorCode from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ErrorCode = httperr.ErrorCode

	// WithHTTPStatus makes MapError return the given status for err.
	//
	// Deprecated: use httperr.WithHTTPStatus from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	WithHTTPStatus = httperr.WithHTTPStatus

	// WithRetryAfter attaches a Retry-After hint to err for 503 responses.
	//
	// Deprecated: use httperr.WithRetryAfter from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	WithRetryAfter = httperr.WithRetryAfter

	// NewStructuredError builds the RFC 9457 payload for a request.
	//
	// Deprecated: use httperr.NewStructuredError from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	NewStructuredError = httperr.NewStructuredError

	// NewStructuredErrorWithContext builds the RFC 9457 payload from a context.
	//
	// Deprecated: use httperr.NewStructuredErrorWithContext from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	NewStructuredErrorWithContext = httperr.NewStructuredErrorWithContext

	// NewStructuredErrorForAudience builds the RFC 9457 payload for a
	// request with an explicit audience.
	//
	// Deprecated: use httperr.NewStructuredErrorForAudience from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	NewStructuredErrorForAudience = httperr.NewStructuredErrorForAudience

	// NewStructuredErrorForAudienceWithContext builds the RFC 9457 payload
	// from a context with an explicit audience.
	//
	// Deprecated: use httperr.NewStructuredErrorForAudienceWithContext from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	NewStructuredErrorForAudienceWithContext = httperr.NewStructuredErrorForAudienceWithContext

	// DefaultErrorHandler maps CQRS errors to HTTP status codes and writes a
	// plain text error response, redirecting HTMX auth failures to the login
	// path.
	//
	// Deprecated: use httperr.DefaultErrorHandler from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	DefaultErrorHandler = httperr.DefaultErrorHandler

	// DefaultErrorHandlerWithRedirect is DefaultErrorHandler with a custom login
	// redirect path.
	//
	// Deprecated: use httperr.DefaultErrorHandlerWithRedirect from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	DefaultErrorHandlerWithRedirect = httperr.DefaultErrorHandlerWithRedirect

	// DefaultErrorHandlerWithRequestID is DefaultErrorHandler that prefixes the
	// message with the request ID when one is in context.
	//
	// Deprecated: use httperr.DefaultErrorHandlerWithRequestID from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	DefaultErrorHandlerWithRequestID = httperr.DefaultErrorHandlerWithRequestID

	// DefaultErrorHandlerWithRedirectAndRequestID is
	// DefaultErrorHandlerWithRedirect that prefixes the message with the request
	// ID.
	//
	// Deprecated: use httperr.DefaultErrorHandlerWithRedirectAndRequestID from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	DefaultErrorHandlerWithRedirectAndRequestID = httperr.DefaultErrorHandlerWithRedirectAndRequestID

	// JSONErrorHandler writes errors as JSON responses.
	//
	// Deprecated: use httperr.JSONErrorHandler from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	JSONErrorHandler = httperr.JSONErrorHandler

	// JSONErrorHandlerWithRedirect writes errors as JSON responses with a custom
	// login redirect path.
	//
	// Deprecated: use httperr.JSONErrorHandlerWithRedirect from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	JSONErrorHandlerWithRedirect = httperr.JSONErrorHandlerWithRedirect

	// ProblemDetailsErrorHandler writes errors as RFC 9457 problem details,
	// using StructuredError — the same shape emitted for SSE error broadcasts.
	//
	// Deprecated: use httperr.ProblemDetailsErrorHandler from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ProblemDetailsErrorHandler = httperr.ProblemDetailsErrorHandler

	// ProblemDetailsAudienceErrorHandler is ProblemDetailsErrorHandler with
	// per-request audience discrimination (nil resolver fails public).
	//
	// Deprecated: use httperr.ProblemDetailsAudienceErrorHandler from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
	ProblemDetailsAudienceErrorHandler = httperr.ProblemDetailsAudienceErrorHandler

	// ProblemDetailsErrorHandlerWithRedirect is ProblemDetailsErrorHandler with a
	// custom login redirect path.
	//
	// Deprecated: use httperr.ProblemDetailsErrorHandlerWithRedirect from github.com/larsartmann/cqrs-htmx/v4/httperr
	// directly. Removed in cqrs-htmx v5.
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
