package cqrshtmx

import (
	"errors"
	"net/http"
)

// HTTPStatusCarrier is an error that explicitly declares the HTTP status code
// it should produce, overriding the family-based default in [MapError].
//
// Implement this interface (or wrap an error with [WithHTTPStatus]) when a
// specific status is required beyond what the error family implies. The
// motivating case: [event.Rejection]-family errors cover 400, 401, 403, 404,
// and 429 — all "the caller's fault, not retryable" — yet each needs a
// distinct HTTP status. The carrier lets a single error pin the exact code
// while keeping its behavioral family intact for retry/exit-code decisions.
//
// [MapError] honours this interface before any other rule, so it is the most
// authoritative source of an error's HTTP status.
//
// errors.Is and errors.As traverse through the cause via [Unwrap], so wrapping
// a sentinel preserves identity, and errorfamily.Classify still derives the family
// from the wrapped error.
type HTTPStatusCarrier interface {
	error
	HTTPStatus() int
}

// httpStatusError wraps an error with an explicit HTTP status code.
type httpStatusError struct {
	status int
	cause  error
}

func (e *httpStatusError) Error() string   { return e.cause.Error() }
func (e *httpStatusError) Unwrap() error   { return e.cause }
func (e *httpStatusError) HTTPStatus() int { return e.status }

// WithHTTPStatus wraps err so [MapError] returns status instead of the
// family-derived default. Returns nil when err is nil.
//
// The wrapper preserves the cause's error family (errorfamily.Classify traverses
// the chain) and its sentinel identity (errors.Is traverses the chain), so
// existing classification and matching keep working.
//
// Example: surface a "user not found" Rejection as 404 instead of 400:
//
//	return cqrshtmx.WithHTTPStatus(usermgmt.ErrUserNotFound, http.StatusNotFound)
func WithHTTPStatus(err error, status int) error {
	if err == nil {
		return nil
	}
	return &httpStatusError{status: status, cause: err}
}

// carrierStatus returns the explicit HTTP status carried by err, if any.
// Used by [MapError] as the highest-priority status source.
// Errors carrying an out-of-range status fall back to 500 to stay safe.
func carrierStatus(err error) (int, bool) {
	var carrier HTTPStatusCarrier
	if !errors.As(err, &carrier) {
		return 0, false
	}
	status := carrier.HTTPStatus()
	if validHTTPStatus(status) {
		return status, true
	}
	return http.StatusInternalServerError, true
}

// validHTTPStatus reports whether status is a plausible 1xx-5xx HTTP status.
func validHTTPStatus(status int) bool {
	return status >= 100 && status <= 599
}
