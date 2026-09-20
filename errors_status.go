package cqrshtmx

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"
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
//
// Walks the error chain looking for the first HTTPStatusCarrier with a
// non-zero status. Zero means "no override" (e.g. errorfamily.Error defaults
// to httpStatus=0), so we skip it and continue down the chain to find a
// real override (e.g. from WithHTTPStatus).
func carrierStatus(err error) (int, bool) {
	current := err
	for current != nil {
		carrier, ok := errors.AsType[HTTPStatusCarrier](current)
		if ok {
			status := carrier.HTTPStatus()
			if status == 0 {
				// This carrier has no override; unwrap and keep looking.
				current = errors.Unwrap(current)

				continue
			}

			if validHTTPStatus(status) {
				return status, true
			}

			return http.StatusInternalServerError, true
		}

		current = errors.Unwrap(current)
	}

	return 0, false
}

// validHTTPStatus reports whether status is a plausible 1xx-5xx HTTP status.
func validHTTPStatus(status int) bool {
	return status >= 100 && status <= 599
}

// headerRetryAfter is the canonical HTTP header name (RFC 9110 §10.1.3).
const headerRetryAfter = "Retry-After"

// DefaultRetryAfter is the Retry-After delay sent on Transient (503) error
// responses when the error does not carry its own hint via
// [RetryAfterCarrier]. One second is deliberately conservative: it tells
// standards-compliant clients the failure is retryable without inviting a
// retry storm.
const DefaultRetryAfter = 1 * time.Second

// RetryAfterCarrier is an error that explicitly declares how long a client
// should wait before retrying. On responses whose status maps to 503
// (Service Unavailable), [DefaultErrorHandler], [JSONErrorHandler], and
// [ProblemDetailsErrorHandler] send this duration as the Retry-After
// response header instead of [DefaultRetryAfter].
//
// Wrap any error to pin the hint:
//
//	err = cqrshtmx.WithRetryAfter(err, 30*time.Second)
//
// The interface is honoured through the whole error chain (errors.AsType),
// so the hint survives wrapping, and it does not change the error's family,
// status, or sentinel identity.
type RetryAfterCarrier interface {
	error
	RetryAfter() time.Duration
}

// WithRetryAfter wraps err so Transient (503) responses carry the given
// Retry-After hint instead of [DefaultRetryAfter]. Returns nil when err is
// nil. Non-positive durations are ignored at response time (the default is
// used) so a zero value can never advise clients to retry immediately.
func WithRetryAfter(err error, delay time.Duration) error {
	if err == nil {
		return nil
	}

	return &retryAfterError{delay: delay, cause: err}
}

// retryAfterError wraps an error with an explicit Retry-After hint.
type retryAfterError struct {
	delay time.Duration
	cause error
}

func (e *retryAfterError) Error() string             { return e.cause.Error() }
func (e *retryAfterError) Unwrap() error             { return e.cause }
func (e *retryAfterError) RetryAfter() time.Duration { return e.delay }

// setRetryAfterHeader sets the Retry-After response header (RFC 9110
// §10.1.3: a 503 "SHOULD" carry it) on retryable statuses. A carrier hint
// wins over DefaultRetryAfter; sub-second hints round up to one second so
// the delta-seconds format never advises an immediate retry.
func setRetryAfterHeader(w http.ResponseWriter, err error, status int) {
	if status != http.StatusServiceUnavailable {
		return
	}

	delay := DefaultRetryAfter
	if carrier, ok := errors.AsType[RetryAfterCarrier](err); ok && carrier.RetryAfter() > 0 {
		delay = carrier.RetryAfter()
	}

	seconds := int(math.Ceil(delay.Seconds()))
	if seconds < 1 {
		seconds = 1
	}

	w.Header().Set(headerRetryAfter, strconv.Itoa(seconds))
}
