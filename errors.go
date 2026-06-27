package cqrshtmx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

const defaultLoginRedirect = "/login"

// Sentinel errors for HTTP→CQRS integration.
//
// Each sentinel is natively classified via its go-error-family family (re-exported
// through go-cqrs-lite/event/v3), so MapError derives the HTTP status directly —
// no runtime RegisterClassification is needed.
var (
	ErrUnauthorized     = event.NewRejection("unauthorized", "unauthorized: authentication required")
	ErrForbidden        = event.NewRejection("forbidden", "forbidden: insufficient permissions")
	ErrDecodeFailed     = event.NewRejection("decode_failed", "failed to decode request body")
	ErrDispatchFailed   = event.NewTransient("dispatch_failed", "command/query dispatch failed")
	ErrEnforcerNil      = event.NewInfrastructure("enforcer_nil", "casbin enforcer is required for authorization")
	ErrValidationFailed = event.NewRejection("validation_failed", "request validation failed")
	ErrCSRFConfig       = event.NewInfrastructure("csrf_config", "invalid CSRF configuration")
	ErrRequestTooLarge  = event.NewRejection("request_too_large", "request body exceeds maximum size")
	ErrMethodNotAllowed = event.NewRejection("method_not_allowed", "HTTP method not allowed")

	errCommandsNil    = event.NewInfrastructure("commands_nil", "command dispatcher is required")
	errQueriesNil     = event.NewInfrastructure("queries_nil", "query dispatcher is required")
	errDecoderMissing = event.NewInfrastructure("decoder_missing", "request decoder is required")
)

// MapError translates a CQRS error into an appropriate HTTP status code.
//
// The family→status mapping delegates to Family.HTTPStatus() from
// go-error-family (the upstream canonical source, adopted in v0.5.0).
// Explicit overrides for auth and HTTP-semantic errors take precedence
// over the family default.
//
// Mapping (via upstream):
//   - Rejection family  → 400 Bad Request
//   - Conflict family   → 409 Conflict
//   - Transient family  → 503 Service Unavailable
//   - Corruption family → 500 Internal Server Error
//   - Infrastructure    → 503 Service Unavailable
//   - nil or unknown    → 500 Internal Server Error
//
// See ADR-0017 for the reconciliation rationale.
func MapError(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	if status := explicitErrorStatus(err); status != 0 {
		return status
	}

	return event.Classify(err).HTTPStatus()
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

// ErrorHandler writes an HTTP error response with HTMX awareness.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// DefaultErrorHandler maps CQRS errors to HTTP status codes and writes
// a plain text error response. For HTMX requests with auth errors,
// it redirects via HX-Redirect to the login path instead of returning an error body.
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
	var coder interface{ Code() string }
	if errors.As(err, &coder) {
		switch coder.Code() {
		case "unauthorized":
			return http.StatusUnauthorized, true
		case "forbidden":
			return http.StatusForbidden, true
		}
	}
	return 0, false
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
	handleErrorCore(w, r, err, loginRedirect, func(w http.ResponseWriter, err error, status int) {
		w.Header().Set("Content-Type", ContentTypePlain)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, err.Error()) //nolint:gosec // text/plain prevents HTML rendering
	})
}

// formatErrorWithRequestID returns the error message, prefixed with the request ID
// when one is present in the request context.
func formatErrorWithRequestID(r *http.Request, err error) string {
	if rid := RequestIDFromContext(r.Context()); !rid.IsZero() {
		return "[request_id: " + rid.String() + "] " + err.Error()
	}
	return err.Error()
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
	handleErrorCore(w, r, err, loginRedirect, func(w http.ResponseWriter, err error, status int) {
		w.Header().Set("Content-Type", ContentTypePlain)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, formatErrorWithRequestID(r, err)) //nolint:gosec // text/plain prevents HTML rendering
	})
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
	handleErrorCore(w, r, err, loginRedirect, func(w http.ResponseWriter, err error, status int) {
		w.Header().Set("Content-Type", ContentTypeJSON)
		w.WriteHeader(status)

		response := map[string]any{
			JSONKeyError:  err.Error(),
			JSONKeyStatus: status,
		}
		if rid := RequestIDFromContext(r.Context()); !rid.IsZero() {
			response["request_id"] = rid.String()
		}

		encoder := json.NewEncoder(w)
		_ = encoder.Encode(response)
	})
}
