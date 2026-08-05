package cqrshtmx

import (
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/openapi"
)

// TestWithOpenAPI_StoresMetadataOnConfig verifies that the WithOpenAPI option
// actually writes the operation onto the handler config. The external test can
// only assert the returned func is non-nil; this internal test reaches the
// unexported openapiMeta field to prove the metadata round-trips.
func TestWithOpenAPI_StoresMetadataOnConfig(t *testing.T) {
	config := &handlerConfig{}

	operation := openapi.Post("CreateItem").
		Summary("Create an item").
		Op()

	WithOpenAPI(operation)(config)

	if config.openapiMeta == nil {
		t.Fatal("openapiMeta is nil after applying WithOpenAPI")
	}

	if config.openapiMeta.OperationID != operation.OperationID {
		t.Errorf("openapiMeta.OperationID = %q, want %q", config.openapiMeta.OperationID, operation.OperationID)
	}

	if config.openapiMeta.Summary != operation.Summary {
		t.Errorf("openapiMeta.Summary = %q, want %q", config.openapiMeta.Summary, operation.Summary)
	}
}

// TestWithOpenAPI_AliasesCallerValue documents the current attach semantics:
// the option stores a pointer to its own copy of the passed-by-value
// Operation, so reassigning a caller-held scalar field afterwards cannot
// corrupt the attached metadata.
func TestWithOpenAPI_AliasesCallerValue(t *testing.T) {
	config := &handlerConfig{}

	operation := openapi.Post("CreateItem").Summary("first").Op()

	WithOpenAPI(operation)(config)

	operation.Summary = "mutated-after-attach"

	if config.openapiMeta.Summary != "first" {
		t.Fatalf(
			"attached Summary = %q, want %q; attach must snapshot value fields",
			config.openapiMeta.Summary,
			"first",
		)
	}
}

func TestOpenAPISpecHandler_NilSpecReturnsError(t *testing.T) {
	handler, err := OpenAPISpecHandler(nil)
	if err == nil {
		t.Fatal("expected an error for nil spec, got nil")
	}

	if handler != nil {
		t.Errorf("expected nil handler for nil spec, got non-nil")
	}
}
