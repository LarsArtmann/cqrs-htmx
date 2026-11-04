package hx

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// PanicCode is the error code the recovery path stamps on a recovered panic.
// A panic is a server bug, not a "service unavailable" condition, so it is
// mapped to 500 rather than 503.
const PanicCode = "panic"

// IsPanicError reports whether err carries the panic recovery code.
func IsPanicError(err error) bool {
	e, ok := errors.AsType[*event.Error](err)

	return ok && e.Code() == PanicCode
}
