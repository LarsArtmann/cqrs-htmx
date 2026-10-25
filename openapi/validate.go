package openapi

import (
	"errors"
	"fmt"
)

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
		return errors.New("openapi: spec is nil")
	}

	var errs []error

	seenIDs := make(map[string]string) // operationId -> "method path" of first use

	for path, item := range s.Paths {
		if item == nil {
			continue
		}

		for method, op := range item.operations() {
			if op == nil {
				continue
			}

			if op.OperationID == "" {
				errs = append(errs, fmt.Errorf("openapi: %s %s: operation has no operationId", method, path))

				continue
			}

			if first, dup := seenIDs[op.OperationID]; dup {
				errs = append(errs, fmt.Errorf(
					"openapi: duplicate operationId %q on %s %s (first used on %s)",
					op.OperationID, method, path, first))
			} else {
				seenIDs[op.OperationID] = method + " " + path
			}

			if len(op.Responses) == 0 {
				errs = append(errs, fmt.Errorf(
					"openapi: %s %s (%s): no responses — OpenAPI requires at least one (use Response or NoContent)",
					method, path, op.OperationID))
			}
		}
	}

	return errors.Join(errs...)
}
