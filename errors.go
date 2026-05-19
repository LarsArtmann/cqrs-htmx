package cqrshtmx

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

var (
	defaultLoginRedirect = "/login"
	registerErrors       sync.Once
)

// Sentinel errors for HTTP→CQRS integration.
var (
	ErrUnauthorized     = errors.New("unauthorized: authentication required")
	ErrForbidden        = errors.New("forbidden: insufficient permissions")
	ErrDecodeFailed     = errors.New("failed to decode request body")
	ErrDispatchFailed   = errors.New("command/query dispatch failed")
	ErrEnforcerNil      = errors.New("casbin enforcer is required for authorization")
	ErrValidationFailed = errors.New("request validation failed")

	errCommandsNil    = errors.New("command dispatcher is required")
	errQueriesNil     = errors.New("query dispatcher is required")
	errDecoderMissing = errors.New("request decoder is required")
)

func registerErrorClassifications() {
	registerErrors.Do(func() {
		event.RegisterClassification(ErrUnauthorized, event.Rejection)
		event.RegisterClassification(ErrForbidden, event.Rejection)
		event.RegisterClassification(ErrDecodeFailed, event.Rejection)
		event.RegisterClassification(ErrDispatchFailed, event.Transient)
		event.RegisterClassification(ErrEnforcerNil, event.Infrastructure)
		event.RegisterClassification(errCommandsNil, event.Infrastructure)
		event.RegisterClassification(errQueriesNil, event.Infrastructure)
		event.RegisterClassification(errDecoderMissing, event.Infrastructure)
		event.RegisterClassification(ErrValidationFailed, event.Rejection)
	})
}

// MapError translates a CQRS error into an appropriate HTTP status code.
//
// Mapping:
//   - Rejection family  → 400 Bad Request
//   - Conflict family   → 409 Conflict
//   - Corruption family → 422 Unprocessable Entity
//   - Transient family  → 503 Service Unavailable
//   - Infrastructure    → 500 Internal Server Error
//   - nil or unknown    → 500 Internal Server Error
//
//nolint:cyclop // auth checks + family switch are inherently branching
func MapError(err error) int {
	registerErrorClassifications()

	if err == nil {
		return http.StatusInternalServerError
	}

	if errors.Is(err, ErrUnauthorized) {
		return http.StatusUnauthorized
	}

	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrCSRFInvalid) {
		return http.StatusForbidden
	}

	family := event.Classify(err)
	switch family {
	case event.Rejection:
		return http.StatusBadRequest
	case event.Conflict:
		return http.StatusConflict
	case event.Corruption:
		return http.StatusUnprocessableEntity
	case event.Transient:
		return http.StatusServiceUnavailable
	case event.Infrastructure:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
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
	return errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrCSRFInvalid)
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

// DefaultErrorHandlerWithRedirect maps CQRS errors to HTTP status codes with a custom login redirect.
// If loginRedirect is empty, the default "/login" is used.
func DefaultErrorHandlerWithRedirect(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	loginRedirect string,
) {
	if loginRedirect == "" {
		loginRedirect = defaultLoginRedirect
	}

	if writeHTMXAuthRedirect(w, r, err, loginRedirect) {
		return
	}

	status := MapError(err)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(err.Error())) //nolint:gosec // text/plain prevents HTML rendering
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
	if loginRedirect == "" {
		loginRedirect = defaultLoginRedirect
	}

	if writeHTMXAuthRedirect(w, r, err, loginRedirect) {
		return
	}

	status := MapError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	response := struct {
		Error  string `json:"error"`
		Status int    `json:"status"`
	}{
		Error:  err.Error(),
		Status: status,
	}
	_ = json.NewEncoder(w).Encode(response) //nolint:errchkjson
}
