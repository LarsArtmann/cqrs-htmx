package systemadapter

import (
	"slices"
	"testing"

	cqrsschema "github.com/larsartmann/go-cqrs-lite/schema/v4"
)

// TestEventSchemasCoverDecoderRegistrations pins the lockstep between the two
// production lists that enumerate the identity-model event universe from
// opposite sides: EventTypeDecoder (decode shape) and EventSchemas (declared
// schema). A new event type added to one list but not the other either fails
// projection decoding or silently drops out of the coeffect gate's event
// universe — this test fires in both directions.
func TestEventSchemasCoverDecoderRegistrations(t *testing.T) {
	t.Parallel()

	decoderTypes := EventTypeDecoder().EventTypes()

	schemaTypes := make([]string, 0, len(decoderTypes))
	for _, declared := range EventSchemas() {
		schemaTypes = append(schemaTypes, string(declared.Type()))
	}

	slices.Sort(decoderTypes)
	slices.Sort(schemaTypes)

	if len(decoderTypes) != len(schemaTypes) {
		t.Fatalf("decoder registers %d event types, schema declares %d — lists drifted",
			len(decoderTypes), len(schemaTypes))
	}

	for i := range decoderTypes {
		if decoderTypes[i] != schemaTypes[i] {
			t.Fatalf("decoder has %q where schema declares %q — lists drifted", decoderTypes[i], schemaTypes[i])
		}
	}
}

// TestEventSchemasCompile proves the declarations system.New compiles at boot
// are valid: schema.Declare runs the same validation applySchemaDeclaration
// applies, so a bad declaration fails here instead of at first composition.
func TestEventSchemasCompile(t *testing.T) {
	t.Parallel()

	if _, err := cqrsschema.Declare(EventSchemas()...); err != nil {
		t.Fatalf("schema.Declare(EventSchemas()...): %v", err)
	}
}

// TestEventSchemasCurrentVersion pins the declared envelope version: every
// identity-model event is emitted at the same current version, and the first
// envelope-level schema evolution must consciously bump this (and add ops),
// not discover it from a boot failure.
func TestEventSchemasCurrentVersion(t *testing.T) {
	t.Parallel()

	for _, declared := range EventSchemas() {
		if declared.CurrentVersion() != envelopeSchemaVersion {
			t.Errorf("%s declared at v%d, want v%d", declared.Type(), declared.CurrentVersion(), envelopeSchemaVersion)
		}
	}
}
