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

var httpAuthStatus = map[error]int{
	ErrUnauthorized: http.StatusUnauthorized,
	ErrForbidden:    http.StatusForbidden,
	ErrCSRFInvalid:  http.StatusForbidden,
}

var familyToStatus = map[event.Family]int{
	event.Rejection:      http.StatusBadRequest,
	event.Conflict:       http.StatusConflict,
	event.Corruption:     http.StatusUnprocessableEntity,
	event.Transient:      http.StatusServiceUnavailable,
	event.Infrastructure: http.StatusInternalServerError,
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
func MapError(err error) int {
	registerErrorClassifications()

	if err == nil {
		return http.StatusInternalServerError
	}

	if status, ok := httpAuthStatus[err]; ok {
		return status
	}

	family := event.Classify(err)
	if status, ok := familyToStatus[family]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// ErrorHandler writes an HTTP error response with HTMX awareness.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// DefaultErrorHandler maps CQRS errors to HTTP status codes and writes
// a plain text error response. For HTMX requests with auth errors,
// it redirects via HX-Redirect to the login path instead of returning an error body.
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	DefaultErrorHandlerWithRedirect(w, r, err, defaultLoginRedirect)
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

	if IsHTMXRequest(r) &&
		(errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrCSRFInvalid)) {
		w.Header().Set(headerRedirect, loginRedirect)
		w.WriteHeader(http.StatusSeeOther)
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

	if IsHTMXRequest(r) &&
		(errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrCSRFInvalid)) {
		w.Header().Set(headerRedirect, loginRedirect)
		w.WriteHeader(http.StatusSeeOther)
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
