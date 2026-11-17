package gflusermgmt

import (
	"bytes"
	"flag"
	"os"
	"slices"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// update regenerates usermgmt-events.json:
// go test ./gfl-usermgmt/ -run TestEventsFileMatchesIdentityModel -update
var update = flag.Bool("update", false, "rewrite generated golden artifacts")

// identityModelEvents is the event universe of systemadapter.DomainConfig():
// every identity-model event constant, sorted for a stable byte-stable file.
// The list is the single hand-maintained source; the test fails on any
// identitymodel event constant missing here (and on any entry that stops
// existing).
func identityModelEvents() []string {
	events := []string{
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

	slices.Sort(events)

	return events
}

// renderedEventsUniverse is the canonical --events-file JSON for the
// usermgmt universe: the same two-space-indented, newline-terminated bytes
// GFL's export.EventsUniverse renderer produces (kept local so this module
// needs no dependency on the gfl-lang module path).
func renderedEventsUniverse() []byte {
	events := identityModelEvents()

	var buf bytes.Buffer

	buf.WriteString("{\n  \"system\": \"usermgmt\",\n  \"events\": [\n")
	for i, name := range events {
		buf.WriteString("    \"" + name + "\"")
		if i < len(events)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("  ]\n}\n")

	return buf.Bytes()
}

// TestEventsFileMatchesIdentityModel pins usermgmt-events.json (the event
// universe the GFL drift gate validates against) byte-for-byte to the
// identity-model constants — the same contract usermgmt's own
// TestEventCatalog_InventoryMatchesEventConstants polices for its catalog,
// extended one level: spec-side, not just catalog-side. The file is
// generated, never hand-edited; regenerate after any constants change with
// the -update flag.
func TestEventsFileMatchesIdentityModel(t *testing.T) {
	want := renderedEventsUniverse()

	if *update {
		if err := os.WriteFile("usermgmt-events.json", want, 0o600); err != nil {
			t.Fatalf("rewrite usermgmt-events.json: %v", err)
		}
	}

	raw, err := os.ReadFile("usermgmt-events.json")
	if err != nil {
		t.Fatalf("read usermgmt-events.json: %v", err)
	}

	if !bytes.Equal(raw, want) {
		t.Errorf(
			"usermgmt-events.json drifted from the identity-model constants; regenerate with: go test ./gfl-usermgmt/ -run TestEventsFileMatchesIdentityModel -update",
		)
	}
}
