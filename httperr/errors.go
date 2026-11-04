package httperr

import (
	"errors"
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
)

// DefaultLoginRedirect is the login path HTMX auth failures redirect to when the
// caller supplies no override.
const DefaultLoginRedirect = "/login"

// Cross-module auth error codes. Both root and the usermgmt submodule define
// their own sentinel errors but share these codes so MapError can recognise
// auth errors across module boundaries by code (root cannot import usermgmt).
// Exported so callers constructing classified errors can reference the exact
// codes instead of duplicating string literals.
const (
	CodeUnauthorized = "unauthorized"
	CodeForbidden    = "forbidden"
)

// Sentinel errors for HTTP→CQRS integration.
//
// Each sentinel is natively classified via its go-error-family family (re-exported
// through go-cqrs-lite/event/v4), so MapError derives the HTTP status directly —
// no runtime RegisterClassification is needed.
var (
	ErrUnauthorized   error = errorfamily.NewRejection(CodeUnauthorized, "unauthorized: authentication required")
	ErrForbidden      error = errorfamily.NewRejection(CodeForbidden, "forbidden: insufficient permissions")
	ErrDecodeFailed   error = errorfamily.NewRejection("decode_failed", "failed to decode request body")
	ErrDispatchFailed error = errorfamily.NewTransient("dispatch_failed", "command/query dispatch failed")
	ErrEnforcerNil    error = errorfamily.NewInfrastructure(
		"enforcer_nil",
		"casbin enforcer is required for authorization",
	)
	ErrValidationFailed error = errorfamily.NewRejection("validation_failed", "request validation failed")
	ErrRequestTooLarge  error = errorfamily.NewRejection("request_too_large", "request body exceeds maximum size")
	ErrMethodNotAllowed error = errorfamily.NewRejection("method_not_allowed", "HTTP method not allowed")

	// ErrCommandsNil reports a missing command dispatcher — a wiring bug.
	ErrCommandsNil error = errorfamily.NewInfrastructure("commands_nil", "command dispatcher is required")
	// ErrQueriesNil reports a missing query dispatcher — a wiring bug.
	ErrQueriesNil error = errorfamily.NewInfrastructure("queries_nil", "query dispatcher is required")
	// ErrDecoderMissing reports a missing request decoder — a wiring bug.
	ErrDecoderMissing error = errorfamily.NewInfrastructure("decoder_missing", "request decoder is required")
	// ErrDecoderReturnedNil reports a decoder that returned neither a value nor
	// an error — a server-side wiring bug.
	ErrDecoderReturnedNil error = errorfamily.NewCorruption(
		"decoder_returned_nil",
		"request decoder returned a nil result without an error — this is a server-side wiring bug",
	)
)

// MapError translates a CQRS error into an appropriate HTTP status code.
//
// Resolution order (first match wins):
//  1. HTTPStatusCarrier — an error that explicitly declares its status
//     (via WithHTTPStatus or by implementing the interface). Highest authority.
//  2. Explicit overrides for auth/HTTP-semantic sentinels and the panic code.
//  3. The error family via Family.HTTPStatus() from go-error-family (upstream).
//
// Mapping (via upstream, step 3):
//   - Rejection family  → 400 Bad Request
//   - Conflict family   → 409 Conflict
//   - Transient family  → 503 Service Unavailable
//   - Corruption family → 500 Internal Server Error
//   - Infrastructure    → 503 Service Unavailable
//   - nil               → 500 Internal Server Error
//   - unknown (untyped)  → 503 Service Unavailable (Transient fail-open)
//
// See ADR-0017 for the reconciliation rationale and ADR-0034 for the
// HTTPStatusCarrier extension.
func MapError(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	if status, ok := carrierStatus(err); ok {
		return status
	}

	if status := explicitErrorStatus(err); status != 0 {
		return status
	}

	return errorfamily.Classify(err).HTTPStatus()
}

func explicitErrorStatus(err error) int {
	switch {
	case hx.IsPanicError(err):
		return http.StatusInternalServerError
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden) || errors.Is(err, httputil.ErrCSRFInvalid):
		return http.StatusForbidden
	case errors.Is(err, ErrRequestTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, ErrMethodNotAllowed):
		return http.StatusMethodNotAllowed
	default:
		// Cross-module code-based check: usermgmt defines separate sentinels
		// with the same error codes for module independence. This ensures
		// MapError returns the correct HTTP status regardless of source module.
		if status, ok := authStatusFromErrorCode(err); ok {
			return status
		}

		return 0
	}
}

// DefaultErrorHandler maps CQRS errors to HTTP status codes and writes a plain
// text error response. For HTMX requests with auth errors, it redirects via
// HX-Redirect to the login path instead of returning an error body.
//
// It is intentionally config-unaware — it does NOT read
// Config.IncludeInternalDetails or Config.IncludeRequestIDInErrors, so wiring
// it as Config.ErrorHandler silently bypasses those knobs (it always writes the
// safe, redacted message and never the request id). That is acceptable for the
// default plain-text path, but consumers who want the response body to honor
// those config flags should use [JSONErrorHandler] (which reads Config) or
// [ProblemDetailsErrorHandler] instead. See ADR-0017.
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	DefaultErrorHandlerWithRedirect(w, r, err, DefaultLoginRedirect)
}

// isAuthError returns true if the error is an authentication/authorization error
// that should trigger a login redirect for HTMX requests.
func isAuthError(err error) bool {
	if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) ||
		errors.Is(err, httputil.ErrCSRFInvalid) {
		return true
	}

	_, ok := authStatusFromErrorCode(err)

	return ok
}

// authStatusFromErrorCode checks if the error has an auth-related code,
// enabling cross-module compatibility. Errors from usermgmt (which defines
// separate sentinels with the same codes for module independence) are
// recognized by code rather than sentinel identity.
func authStatusFromErrorCode(err error) (int, bool) {
	coder, ok := errors.AsType[interface {
		error
		Code() string
	}](err)
	if !ok {
		return 0, false
	}

	switch coder.Code() {
	case CodeUnauthorized:
		return http.StatusUnauthorized, true
	case CodeForbidden:
		return http.StatusForbidden, true
	}

	return 0, false
}

// ErrorCode extracts the machine-readable error code from the innermost
// errorfamily error in the cause chain. Returns "" if no error carries a code.
// We walk the full chain to find the deepest (domain-specific) code, not the
// outermost wrapper (which is typically an infrastructure wrapping code like
// "cqrshtmx.dispatch.command_failed").
func ErrorCode(err error) string {
	var deepestCode string

	current := err
	for current != nil {
		coder, ok := errors.AsType[interface {
			error
			Code() string
		}](current)
		if ok {
			deepestCode = coder.Code()
		}

		current = errors.Unwrap(current)
	}

	return deepestCode
}

// writeHTMXAuthRedirect sets HX-Redirect header and writes 303 See Other.
// Returns true if the redirect was written, false if the request is not HTMX or the error is not auth-related.
func writeHTMXAuthRedirect(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
) bool {
	if !htmx.IsRequest(r) || !isAuthError(err) {
		return false
	}

	w.Header().Set(htmx.HeaderRedirect, loginRedirect)
	w.WriteHeader(http.StatusSeeOther)

	return true
}

// HandleErrorCore handles the common logic for error responses:
// default login redirect, HTMX auth redirect, status code mapping, and the
// Retry-After header on 503 responses.
// The writeBody callback handles the response body format.
func HandleErrorCore(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
	writeBody func(http.ResponseWriter, error, int),
) {
	if loginRedirect == "" {
		loginRedirect = DefaultLoginRedirect
	}

	if writeHTMXAuthRedirect(w, r, err, loginRedirect) {
		return
	}

	status := MapError(err)
	setRetryAfterHeader(w, err, status)
	writeBody(w, err, status)
}

// DefaultErrorHandlerWithRedirect maps CQRS errors to HTTP status codes with a custom login redirect.
// If loginRedirect is empty, the default "/login" is used.
func DefaultErrorHandlerWithRedirect(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
) {
	HandleErrorCore(
		w,
		r,
		err,
		loginRedirect,
		PlainBodyWriter(r, PlainBodyOptions{IncludeInternal: false, IncludeRequestID: false}),
	)
}

// PlainBodyOptions names the two plain-body writer flags so call sites
// read unambiguously instead of counting positional booleans.
type PlainBodyOptions struct {
	// IncludeInternal exposes 5xx detail (Config.IncludeInternalDetails opt-in).
	IncludeInternal bool
	// IncludeRequestID prefixes the request ID from context onto the body.
	IncludeRequestID bool
}

// PlainBodyWriter builds a text/plain response body writer that redacts 5xx
// detail unless opts.IncludeInternal is set, optionally prefixing the request ID.
func PlainBodyWriter(r *http.Request, opts PlainBodyOptions) func(http.ResponseWriter, error, int) {
	return func(w http.ResponseWriter, err error, status int) {
		w.Header().Set("Content-Type", hx.ContentTypePlain)
		w.WriteHeader(status)

		detail := SafeDetail(err, status, opts.IncludeInternal)
		if opts.IncludeRequestID {
			detail = prefixRequestID(r, detail)
		}

		hx.WriteAllString(w, detail)
	}
}

// SafeDetail returns the error text that is safe to expose for an HTTP
// response of the given status.
//
// Client errors (status < 500 — Rejection, Conflict, NotFound, Unauthorized,
// Forbidden, TooManyRequests, …) describe the caller's input, so the raw error
// message is returned: it helps the user correct their request and leaks no
// sensitive internals.
//
// Server faults (status >= 500 — Corruption, Infrastructure, Transient) may
// leak internal wiring, infrastructure addresses, or data-integrity detail.
// Unless IncludeInternal is true (a development or trusted-network opt-in, wired
// via [Config.IncludeInternalDetails]), they are replaced by the error family's
// generic, public-safe default message so attackers cannot probe internals
// through error text. The full detail is still logged server-side by the
// dispatch path (see [App.handleErr]).
func SafeDetail(err error, status int, includeInternal bool) string {
	if err == nil {
		return ""
	}

	if status < 500 || includeInternal {
		return err.Error()
	}

	return errorfamily.Classify(err).DefaultMessage()
}

// prefixRequestID prefixes detail with the request ID when one is in context.
func prefixRequestID(r *http.Request, detail string) string {
	if rid := hx.RequestIDFromContext(r.Context()); !rid.IsZero() {
		return "[request_id: " + rid.String() + "] " + detail
	}

	return detail
}

// DefaultErrorHandlerWithRequestID is like DefaultErrorHandler but prefixes the
// error message with the request ID when one is present in context.
func DefaultErrorHandlerWithRequestID(w http.ResponseWriter, r *http.Request, err error) {
	DefaultErrorHandlerWithRedirectAndRequestID(w, r, err, DefaultLoginRedirect)
}

// DefaultErrorHandlerWithRedirectAndRequestID is like DefaultErrorHandlerWithRedirect
// but prefixes the error message with the request ID when one is present in context.
func DefaultErrorHandlerWithRedirectAndRequestID(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
) {
	HandleErrorCore(w, r, err, loginRedirect, PlainBodyWriter(r,
		PlainBodyOptions{IncludeInternal: false, IncludeRequestID: true}))
}

// JSONErrorHandler writes errors as JSON responses.
// For HTMX requests with auth errors, redirects via HX-Redirect instead of returning JSON.
// Uses the default login redirect path ("/login").
func JSONErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	JSONErrorHandlerWithRedirect(w, r, err, DefaultLoginRedirect)
}

// JSONErrorHandlerWithRedirect writes errors as JSON responses with a custom login redirect.
// For HTMX requests with auth errors, redirects via HX-Redirect instead of returning JSON.
// If loginRedirect is empty, the default "/login" is used.
func JSONErrorHandlerWithRedirect(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
) {
	HandleErrorCore(w, r, err, loginRedirect, jsonBodyWriter(r, false))
}

// jsonBodyWriter builds an application/json response body writer that redacts
// 5xx detail unless IncludeInternal is set.
func jsonBodyWriter(r *http.Request, includeInternal bool) func(http.ResponseWriter, error, int) {
	return func(w http.ResponseWriter, err error, status int) {
		w.Header().Set("Content-Type", hx.ContentTypeJSON)
		w.WriteHeader(status)

		response := map[string]any{
			hx.JSONKeyError:  SafeDetail(err, status, includeInternal),
			hx.JSONKeyStatus: status,
		}
		if code := ErrorCode(err); code != "" {
			response[hx.JSONKeyCode] = code
		}

		if rid := hx.RequestIDFromContext(r.Context()); !rid.IsZero() {
			response[hx.JSONKeyRequestID] = rid.String()
		}

		hx.MarshalJSONForResponse(w, response)
	}
}

// ProblemDetailsErrorHandler writes errors as RFC 7807 problem details
// (application/problem+json), using StructuredError — the same shape emitted
// for SSE error broadcasts. This unifies the error contract
// across HTTP and SSE transports: a client sees the same JSON shape
// regardless of how the error arrives.
//
// 5xx detail is redacted to the family's public-safe message by default.
// Use Config.IncludeInternalDetails to override for dev/trusted environments.
//
// Set via Config.ErrorHandler:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    ErrorHandler: cqrshtmx.ProblemDetailsErrorHandler,
//	    ...
//	})
func ProblemDetailsErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	ProblemDetailsErrorHandlerWithRedirect(w, r, err, DefaultLoginRedirect)
}

// ProblemDetailsErrorHandlerWithRedirect is like ProblemDetailsErrorHandler
// with a custom login redirect path for HTMX auth errors.
func ProblemDetailsErrorHandlerWithRedirect(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
) {
	HandleErrorCore(w, r, err, loginRedirect, func(w http.ResponseWriter, err error, _ int) {
		w.Header().Set("Content-Type", hx.ContentTypeProblem)

		status := MapError(err)
		w.WriteHeader(status)

		payload := NewStructuredError(err, r)
		hx.MarshalJSONForResponse(w, payload)
	})
}
