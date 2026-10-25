package openapi

import (
	"errors"
	"fmt"
)

// Validation sentinels. [Validate] wraps them with operation context, so a
// caller can classify outcomes with errors.Is. They are package-level
// static errors by design: this package is deliberately dependency-free
// (see [doc]) and uses only the standard library for its errors.
var (
	errNilSpec              = errors.New("openapi: spec is nil")
	errNoOperationID        = errors.New("openapi: operation has no operationId")
	errDuplicateOperationID = errors.New("openapi: duplicate operationId")
)

// errNoResponses names the fix in its message: per the OpenAPI Specification
// an operation without a response is invalid, and [Response]/[NoContent] are
// the constructors that provide one.
var errNoResponses = errors.New("openapi: no responses — OpenAPI requires at least one (use Response or NoContent)")

// Validate checks the invariants OpenAPI 3.1 requires that the builder cannot
// enforce structurally, so an invalid spec fails loudly at serialization (or
// explicitly) instead of silently producing a spec consumers' tooling rejects:
//
//   - every operation declares at least one response (responses is REQUIRED
//     per the OpenAPI Specification — an operation without one is invalid),
//   - operationIds are unique across the whole spec (best practice that
//     codegen tooling relies on; duplicates silently shadow each other),
//   - every registered operation carries a non-empty operationId (the
//     Get/Post/... constructors take one — an empty one is a copy-paste bug).
//
// [Spec.JSON] calls this before marshaling; call it directly when assembling a
// spec programmatically and serializing later.
func (s *Spec) Validate() error {
	if s == nil {
		return errNilSpec
	}

	var errs []error

	seenIDs := make(map[string]string) // operationId -> "method path" of first use

	for path, item := range s.Paths {
		if item == nil {
			continue
		}

		for method, operation := range item.operations() {
			if operation == nil {
				continue
			}

			if operation.OperationID == "" {
				errs = append(errs, fmt.Errorf("%w: %s %s", errNoOperationID, method, path))

				continue
			}

			if first, dup := seenIDs[operation.OperationID]; dup {
				errs = append(errs, fmt.Errorf(
					"%w %q on %s %s (first used on %s)",
					errDuplicateOperationID, operation.OperationID, method, path, first))
			} else {
				seenIDs[operation.OperationID] = method + " " + path
			}

			if len(operation.Responses) == 0 {
				errs = append(errs, fmt.Errorf(
					"%w: %s %s (%s)",
					errNoResponses, method, path, operation.OperationID))
			}
		}
	}

	return errors.Join(errs...)
}
