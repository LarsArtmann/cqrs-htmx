package gflusermgmt

import (
	"encoding/json/v2"
	"os"
	"slices"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// TestEventsFileMatchesIdentityModel pins usermgmt-events.json (the event
// universe the GFL drift gate validates against) to the identity-model
// constants — the same contract usermgmt's own
// TestEventCatalog_InventoryMatchesEventConstants polices for its catalog,
// extended one level: spec-side, not just catalog-side.
func TestEventsFileMatchesIdentityModel(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("usermgmt-events.json")
	if err != nil {
		t.Fatalf("read usermgmt-events.json: %v", err)
	}

	var universe struct {
		System string   `json:"system"`
		Events []string `json:"events"`
	}
	if err := json.Unmarshal(raw, &universe); err != nil {
		t.Fatalf("decode usermgmt-events.json: %v", err)
	}
	if universe.System != "usermgmt" {
		t.Errorf("system = %q, want %q", universe.System, "usermgmt")
	}

	want := []string{
		string(identitymodel.EventUserRegistered),
		string(identitymodel.EventRolesUpdated),
		string(identitymodel.EventEmailChanged),
		string(identitymodel.EventDisplayNameChanged),
		string(identitymodel.EventUserDeleted),
		string(identitymodel.EventCredentialAdded),
		string(identitymodel.EventCredentialRemoved),
		string(identitymodel.EventCredentialUsed),
		string(identitymodel.EventEmailVerified),
		string(identitymodel.EventTOTPEnabled),
		string(identitymodel.EventTOTPDisabled),
		string(identitymodel.EventExternalAccountLinked),
		string(identitymodel.EventExternalAccountUnlinked),
		string(identitymodel.EventLoginSucceeded),
		string(identitymodel.EventLoginFailed),
		string(identitymodel.EventRegistrationClosed),
		string(identitymodel.EventSessionStarted),
		string(identitymodel.EventSessionEnded),
		string(identitymodel.EventMemberAdded),
		string(identitymodel.EventMemberRolesChanged),
		string(identitymodel.EventMemberRemoved),
		string(identitymodel.EventTenantCreated),
		string(identitymodel.EventTenantSuspended),
		string(identitymodel.EventTenantReactivated),
		string(identitymodel.EventTenantDeleted),
		string(identitymodel.EventBotRegistered),
		string(identitymodel.EventBotDeleted),
	}

	got := slices.Clone(universe.Events)
	slices.Sort(got)
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Errorf("events file drifted from identity-model constants:\n got: %v\nwant: %v", got, want)
	}
}
