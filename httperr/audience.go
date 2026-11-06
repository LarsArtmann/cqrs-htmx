package httperr

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Audience classifies WHO will read an error's detail. Server-fault (5xx)
// detail redaction is a function of the audience, not of the deployment:
// the same running server serves anonymous attackers (public) and
// authenticated operators (who need the real diagnostic text to act).
//
// The zero value is AudiencePublic — fail-safe: a caller that never thinks
// about audiences leaks nothing.
type Audience int

const (
	// AudiencePublic is the anonymous or untrusted reader. Server-fault
	// (status >= 500) detail is replaced by the error family's generic,
	// public-safe default message; client-fault detail (< 500) is passed
	// through unchanged (it describes the caller's own input).
	AudiencePublic Audience = iota

	// AudienceOperator is an authenticated operator (admin, on-call) reading
	// the same response. Full detail on every status — including the raw
	// server-fault message and its machine context — so the problem is
	// actionable from the response alone.
	AudienceOperator
)

// SafeDetailFor is the audience-aware [SafeDetail]: it picks the detail text
// for err given the mapped HTTP status and the audience reading it.
//
//	< 500  → err.Error() always (client-fault text belongs to the caller)
//	>= 500 → AudienceOperator: err.Error(); AudiencePublic: the family's
//	         DefaultMessage (generic, safe to show anyone)
//
// Wire a per-request resolver via [ProblemDetailsAudienceErrorHandler] so
// operators see real diagnostics in the RFC 9457 body while everyone else
// sees the redacted form.
func SafeDetailFor(err error, status int, audience Audience) string {
	if err == nil {
		return ""
	}

	if status < 500 || audience == AudienceOperator {
		return err.Error()
	}

	return errorfamily.Classify(err).DefaultMessage()
}
