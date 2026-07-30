package cqrshtmx

import (
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

const defaultLoginRedirect = "/login"

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
	ErrUnauthorized     = errorfamily.NewRejection(CodeUnauthorized, "unauthorized: authentication required")
	ErrForbidden        = errorfamily.NewRejection(CodeForbidden, "forbidden: insufficient permissions")
	ErrDecodeFailed     = errorfamily.NewRejection("decode_failed", "failed to decode request body")
	ErrDispatchFailed   = errorfamily.NewTransient("dispatch_failed", "command/query dispatch failed")
	ErrEnforcerNil      = errorfamily.NewInfrastructure("enforcer_nil", "casbin enforcer is required for authorization")
	ErrValidationFailed = errorfamily.NewRejection("validation_failed", "request validation failed")
	ErrRequestTooLarge  = errorfamily.NewRejection("request_too_large", "request body exceeds maximum size")
	ErrMethodNotAllowed = errorfamily.NewRejection("method_not_allowed", "HTTP method not allowed")

	errCommandsNil        = errorfamily.NewInfrastructure("commands_nil", "command dispatcher is required")
	errQueriesNil         = errorfamily.NewInfrastructure("queries_nil", "query dispatcher is required")
	errDecoderMissing     = errorfamily.NewInfrastructure("decoder_missing", "request decoder is required")
	errDecoderReturnedNil = errorfamily.NewCorruption(
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
	case isPanicError(err):
		return http.StatusInternalServerError
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden) || errors.Is(err, ErrCSRFInvalid):
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

// ErrorHandler is now an alias for httputil.CSRFErrorHandler (defined in csrf_reexport.go).
// It is kept here for documentation discoverability.
//
// ErrorHandler writes an HTTP error response with HTMX awareness.
//
// DefaultErrorHandler maps CQRS errors to HTTP status codes and writes
// a plain text error response. For HTMX requests with auth errors,
// it redirects via HX-Redirect to the login path instead of returning an error body.
//
// Config note: this handler is intentionally config-unaware — it does NOT read
// Config.IncludeInternalDetails or Config.IncludeRequestIDInErrors, so wiring
// it as Config.ErrorHandler silently bypasses those knobs (it always writes the
// safe, redacted message and never the request id). That is acceptable for the
// default plain-text path, but consumers who want the response body to honor
// those config flags should use [JSONErrorHandler] (which reads Config) or
// [ProblemDetailsErrorHandler] instead. See ADR-0017.
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	DefaultErrorHandlerWithRedirect(w, r, err, defaultLoginRedirect)
}

// isAuthError returns true if the error is an authentication/authorization error
// that should trigger a login redirect for HTMX requests.
func isAuthError(err error) bool {
	if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrCSRFInvalid) {
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
	if !IsHTMXRequest(r) || !isAuthError(err) {
		return false
	}

	w.Header().Set(headerRedirect, loginRedirect)
	w.WriteHeader(http.StatusSeeOther)

	return true
}

// handleErrorCore handles the common logic for error responses:
// default login redirect, HTMX auth redirect, and status code mapping.
// The writeBody callback handles the response body format.
func handleErrorCore(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
	writeBody func(http.ResponseWriter, error, int),
) {
	if loginRedirect == "" {
		loginRedirect = defaultLoginRedirect
	}

	if writeHTMXAuthRedirect(w, r, err, loginRedirect) {
		return
	}

	status := MapError(err)
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
	handleErrorCore(w, r, err, loginRedirect, plainBodyWriter(r, false, false))
}

// plainBodyWriter builds a text/plain response body writer that redacts 5xx
// detail unless includeInternal is set, optionally prefixing the request ID.
func plainBodyWriter(r *http.Request, includeInternal, includeRequestID bool) func(http.ResponseWriter, error, int) {
	return func(w http.ResponseWriter, err error, status int) {
		w.Header().Set("Content-Type", ContentTypePlain)
		w.WriteHeader(status)

		detail := SafeDetail(err, status, includeInternal)
		if includeRequestID {
			detail = prefixRequestID(r, detail)
		}

		_, _ = io.WriteString(w, detail) //nolint:gosec // text/plain prevents HTML rendering
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
// Unless includeInternal is true (a development or trusted-network opt-in, wired
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
	if rid := RequestIDFromContext(r.Context()); !rid.IsZero() {
		return "[request_id: " + rid.String() + "] " + detail
	}

	return detail
}

// DefaultErrorHandlerWithRequestID is like DefaultErrorHandler but prefixes the
// error message with the request ID when one is present in context.
func DefaultErrorHandlerWithRequestID(w http.ResponseWriter, r *http.Request, err error) {
	DefaultErrorHandlerWithRedirectAndRequestID(w, r, err, defaultLoginRedirect)
}

// DefaultErrorHandlerWithRedirectAndRequestID is like DefaultErrorHandlerWithRedirect
// but prefixes the error message with the request ID when one is present in context.
func DefaultErrorHandlerWithRedirectAndRequestID(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
) {
	handleErrorCore(w, r, err, loginRedirect, plainBodyWriter(r, false, true))
}

// JSONErrorHandler writes errors as JSON responses.
// For HTMX requests with auth errors, redirects via HX-Redirect instead of returning JSON.
// Uses the default login redirect path ("/login").
func JSONErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	JSONErrorHandlerWithRedirect(w, r, err, defaultLoginRedirect)
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
	handleErrorCore(w, r, err, loginRedirect, jsonBodyWriter(r, false))
}

// jsonBodyWriter builds an application/json response body writer that redacts
// 5xx detail unless includeInternal is set.
func jsonBodyWriter(r *http.Request, includeInternal bool) func(http.ResponseWriter, error, int) {
	return func(w http.ResponseWriter, err error, status int) {
		w.Header().Set("Content-Type", ContentTypeJSON)
		w.WriteHeader(status)

		response := map[string]any{
			JSONKeyError:  SafeDetail(err, status, includeInternal),
			JSONKeyStatus: status,
		}
		if code := ErrorCode(err); code != "" {
			response[JSONKeyCode] = code
		}

		if rid := RequestIDFromContext(r.Context()); !rid.IsZero() {
			response[JSONKeyRequestID] = rid.String()
		}

		_ = json.MarshalWrite(w, response)
	}
}

// ProblemDetailsErrorHandler writes errors as RFC 7807 problem details
// (application/problem+json), using StructuredError — the same shape emitted
// for SSE and WebSocket error broadcasts. This unifies the error contract
// across HTTP, SSE, and WS transports: a client sees the same JSON shape
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
	ProblemDetailsErrorHandlerWithRedirect(w, r, err, defaultLoginRedirect)
}

// ProblemDetailsErrorHandlerWithRedirect is like ProblemDetailsErrorHandler
// with a custom login redirect path for HTMX auth errors.
func ProblemDetailsErrorHandlerWithRedirect(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
) {
	handleErrorCore(w, r, err, loginRedirect, func(w http.ResponseWriter, err error, _ int) {
		w.Header().Set("Content-Type", ContentTypeProblem)

		status := MapError(err)
		w.WriteHeader(status)

		payload := NewStructuredError(err, r)
		data, _ := json.Marshal(payload)
		_, _ = w.Write(data) //nolint:gosec // G705: json.Marshal escapes HTML by default
	})
}
