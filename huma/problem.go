package huma

import (
	"context"
	"errors"

	humav2 "github.com/danielgtaylor/huma/v2"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
)

// ProblemError is the RFC 9457 problem document the adapter answers with.
//
// It embeds httperr.StructuredError — the same document the route-based error
// handler writes — so an OpenAPI-first client gets one error shape whether the
// route behind it is an App.Command endpoint or a Huma operation: status,
// human-readable detail, the error family, the machine-readable code, and (for
// client faults) the machine-readable context entries that let a UI react to
// the specific problem rather than print prose.
//
// The struct implements huma.StatusError and huma.ContentTypeFilter, which is
// what makes Huma answer with this type and label it application/problem+json.
type ProblemError struct {
	httperr.StructuredError
}

// GetStatus implements huma.StatusError: Huma sets the response status from the
// problem's own status, which came from httperr.MapError.
func (p *ProblemError) GetStatus() int { return p.Status }

// ContentType implements huma.ContentTypeFilter so the document is served as
// application/problem+json (RFC 9457) instead of plain application/json.
func (p *ProblemError) ContentType(contentType string) string {
	switch contentType {
	case "application/json":
		return "application/problem+json"
	case "application/cbor":
		return "application/problem+cbor"
	default:
		return contentType
	}
}

// MapError is the error stage: it converts any error into the huma.StatusError a
// handler returns.
//
// The status comes from httperr.MapError (rejection 400, conflict 409, transient
// 503, corruption 500, plus the 401/403/405/413 sentinels), the body from
// httperr.StructuredError with server-fault detail redacted for a public
// audience, and the instance from the request ID in ctx when one is present.
//
// Errors that already implement huma.StatusError pass through untouched: a
// handler that returns huma.Error422UnprocessableEntity has said what the
// response should be, and re-wrapping it would relabel the status.
//
// MapError is pure — it does not log. The registration helpers log the
// classified error once before mapping it.
func MapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	if isStatusError(err) {
		return err
	}

	problem := &ProblemError{
		StructuredError: httperr.NewStructuredErrorWithContext(err, ctx),
	}

	return problem
}

// isStatusError reports whether err is (or wraps) an error Huma can render on
// its own.
func isStatusError(err error) bool {
	_, ok := errors.AsType[humav2.StatusError](err)

	return ok
}
