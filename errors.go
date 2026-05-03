package cqrshtmx

import (
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

func init() {
	event.RegisterClassification(ErrUnauthorized, event.Rejection)
	event.RegisterClassification(ErrForbidden, event.Rejection)
	event.RegisterClassification(ErrDecodeFailed, event.Rejection)
	event.RegisterClassification(ErrDispatchFailed, event.Transient)
	event.RegisterClassification(ErrNoUserID, event.Rejection)
	event.RegisterClassification(ErrEnforcerNil, event.Infrastructure)
	event.RegisterClassification(ErrCommandsNil, event.Infrastructure)
	event.RegisterClassification(ErrQueriesNil, event.Infrastructure)
	event.RegisterClassification(ErrDecoderMissing, event.Infrastructure)
	event.RegisterClassification(ErrRendererMissing, event.Infrastructure)
}

// Sentinel errors for HTTP→CQRS integration.
var (
	ErrUnauthorized    = errors.New("unauthorized: authentication required")
	ErrForbidden       = errors.New("forbidden: insufficient permissions")
	ErrDecodeFailed    = errors.New("failed to decode request body")
	ErrDispatchFailed  = errors.New("command/query dispatch failed")
	ErrNoUserID        = errors.New("no user ID in context")
	ErrEnforcerNil     = errors.New("casbin enforcer is required for authorization")
	ErrCommandsNil     = errors.New("command dispatcher is required")
	ErrQueriesNil      = errors.New("query dispatcher is required")
	ErrDecoderMissing  = errors.New("request decoder is required")
	ErrRendererMissing = errors.New("response renderer is required for query handlers")
)

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
	if err == nil {
		return http.StatusInternalServerError
	}

	if errors.Is(err, ErrUnauthorized) {
		return http.StatusUnauthorized
	}

	if errors.Is(err, ErrForbidden) {
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

// LoginRedirect is the URL used by DefaultErrorHandler for HTMX auth redirects.
// Override before creating an App to customize the login path.
// Defaults to "/login".
var LoginRedirect = "/login"

// DefaultErrorHandler maps CQRS errors to HTTP status codes and writes
// a plain text error response. For HTMX requests with auth errors,
// it redirects via HX-Redirect to LoginRedirect instead of returning an error body.
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if IsHTMXRequest(r) && (errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden)) {
		w.Header().Set("HX-Redirect", LoginRedirect)
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	status := MapError(err)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(err.Error()))
}
