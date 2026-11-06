package usermgmt

import (
	"slices"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// The catalog's drift gates (R2): the served inventory must equal the
// canonical event-constant lists exactly (both directions — a new event
// constant without a catalog registration fails, and so does a registration
// without a constant), and the served field lists must equal the payload
// structs' json contracts exactly.

func catalogEventTypes() map[event.Type]bool {
	cat := DefaultEventCatalog()
	seen := make(map[event.Type]bool, len(cat.Events()))
	for _, e := range cat.Events() {
		seen[event.Type(e.Type)] = true
	}
	return seen
}

func TestEventCatalog_InventoryMatchesEventConstants(t *testing.T) {
	want := slices.Concat(
		allUserEventTypes,
		allMembershipEventTypes,
		allTenantEventTypes,
		allBotEventTypes,
		allAuthEventTypes,
	)

	seen := catalogEventTypes()
	if len(seen) != len(want) {
		t.Errorf("catalog has %d events, want exactly %d", len(seen), len(want))
	}
	for _, eventType := range want {
		if !seen[eventType] {
			t.Errorf("event %q missing from DefaultEventCatalog — register it in es_event_catalog.go", eventType)
		}
	}

	registered := make([]event.Type, 0, len(seen))
	for eventType := range seen {
		registered = append(registered, eventType)
	}
	for _, eventType := range registered {
		if !slices.Contains(want, eventType) {
			t.Errorf(
				"catalog event %q is not in any all*EventTypes list — add the constant or remove the registration",
				eventType,
			)
		}
	}
}

func TestEventCatalog_FieldsDerivedFromPayloads(t *testing.T) {
	cat := DefaultEventCatalog()
	byType := make(map[string][]eventcatalog.PayloadField, len(cat.Events()))
	for _, e := range cat.Events() {
		byType[e.Type] = e.PayloadFields
	}

	assertFields := func(t *testing.T, eventType string, want []eventcatalog.PayloadField) {
		t.Helper()
		got := byType[eventType]
		if !slices.Equal(got, want) {
			t.Errorf("%s fields drifted from the payload struct:\n got  %v\n want %v", eventType, got, want)
		}
	}

	t.Run("TOTPEnabled", func(t *testing.T) {
		assertFields(t, "TOTPEnabled", []eventcatalog.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "secret", Type: "[]byte", Required: true},
		})
	})

	// The hand-maintained list documented only id/public_key; the struct (via
	// the embedded CredentialCore) carries eleven wire fields. This pilot
	// proves embedded walking and pins the corrected contract.
	t.Run("CredentialAdded", func(t *testing.T) {
		assertFields(t, "CredentialAdded", []eventcatalog.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "id", Type: "[]byte", Required: true},
			{Name: "public_key", Type: "[]byte", Required: true},
			{Name: "attestation_type", Type: "string", Required: true},
			{Name: "transports", Type: "[]string"},
			{Name: "aaguid", Type: "[]byte"},
			{Name: "sign_count", Type: "uint32", Required: true},
			{Name: "user_present", Type: "bool"},
			{Name: "user_verified", Type: "bool"},
			{Name: "backup_eligible", Type: "bool", Required: true},
			{Name: "backup_state", Type: "bool", Required: true},
			{Name: "name", Type: "string"},
		})
	})

	t.Run("LoginFailed", func(t *testing.T) {
		assertFields(t, "LoginFailed", []eventcatalog.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "method", Type: "AuthMethod", Required: true},
			{Name: "reason", Type: "AuthFailureReason", Required: true},
			{Name: "provider", Type: "string"},
		})
	})
}
