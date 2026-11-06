package usermgmt

import (
	"slices"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// The architecture catalog's drift gates: the derived inventory must equal
// the canonical event-constant lists exactly (catalog.AssertEventInventory
// reports both directions), the two wire formats must agree (same events,
// same descriptions), and the shared catalog reflection must see every field
// the served Go-vocabulary catalog sees.

func TestArchitectureCatalog_InventoryMatchesEventConstants(t *testing.T) {
	want := make([]string, 0, 27)
	for _, eventType := range slices.Concat(
		allUserEventTypes,
		allMembershipEventTypes,
		allTenantEventTypes,
		allBotEventTypes,
		allAuthEventTypes,
	) {
		want = append(want, string(eventType))
	}

	catalog.AssertEventInventory(t, ArchitectureCatalog(), want...)
}

// TestArchitectureCatalog_ParityWithServedCatalog pins that the architecture
// format and the served cqrshtmx catalog expose the SAME event language —
// same inventory, same summaries (descriptions ride both formats from one
// table row). A divergence here means someone bypassed eventCatalogSpecs.
func TestArchitectureCatalog_ParityWithServedCatalog(t *testing.T) {
	served := DefaultEventCatalog()
	byType := make(map[string]string, len(served.Events()))
	for _, e := range served.Events() {
		byType[e.Type] = e.Description
	}

	arch := ArchitectureCatalog()
	if len(arch.Services) != 1 {
		t.Fatalf("architecture catalog has %d services, want exactly 1 (usermgmt)", len(arch.Services))
	}

	seen := 0
	for _, evt := range arch.Services[0].Events {
		desc, ok := byType[string(evt.ID)]
		if !ok {
			t.Errorf("architecture catalog event %q missing from the served catalog", evt.ID)
			continue
		}
		if string(evt.Summary) != desc {
			t.Errorf("event %q summary drifted: architecture %q vs served %q", evt.ID, evt.Summary, desc)
		}
		seen++
	}
	if seen != len(byType) {
		t.Errorf("architecture catalog has %d events, served catalog has %d", seen, len(byType))
	}
}

// TestArchitectureCatalog_FieldParity cross-checks the two derivations on
// pilot payloads: the shared go-cqrs-lite schema reflection must surface
// every property the Go-vocabulary reflection surfaces (by json field name),
// including fields promoted from embedded structs (CredentialCore).
func TestArchitectureCatalog_FieldParity(t *testing.T) {
	arch := ArchitectureCatalog()
	schemas := make(map[string]*catalog.Schema, len(arch.Services[0].Events))
	for _, evt := range arch.Services[0].Events {
		schemas[string(evt.ID)] = evt.Schema
	}

	served := DefaultEventCatalog()
	servedFields := make(map[string][]eventcatalog.PayloadField, len(served.Events()))
	for _, e := range served.Events() {
		servedFields[e.Type] = e.PayloadFields
	}

	pilots := []string{"CredentialAdded", "LoginFailed", "SessionStarted", "TOTPEnabled"}
	for _, eventType := range pilots {
		sch := schemas[eventType]
		if sch == nil {
			t.Fatalf("%s: architecture catalog has no payload schema", eventType)
		}
		if sch.Type != catalog.TypeObject || len(sch.Properties) == 0 {
			t.Fatalf("%s: derived schema is not a populated object: %+v", eventType, sch)
		}

		for _, field := range servedFields[eventType] {
			if _, ok := sch.Properties[field.Name]; !ok {
				t.Errorf("%s: property %q in served catalog but missing from the shared schema reflection",
					eventType, field.Name)
			}
		}
	}
}

// TestArchitectureCatalog_Validate runs the catalog's own structural
// validation (required names/summaries present, ids well-formed) so a broken
// derivation fails here rather than in a consumer's export pipeline.
func TestArchitectureCatalog_Validate(t *testing.T) {
	if violations := ArchitectureCatalog().Validate(); len(violations) > 0 {
		t.Errorf("architecture catalog validation failed: %v", violations)
	}
}
