package systemadapter

import (
	"context"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// The systemcatalog drift gates: deriving from DomainConfig()'s declared
// truth must yield exactly the registered command surface plus the declared
// event universe — a new command or event constant without a declaration
// (or a declaration whose constant went away) fails here, and the events
// must agree with the usermgmt published-language table.

// wantCommands is the full 21-command surface, enumerated from the
// identity-model constants (12 User + 3 Membership + 4 Tenant + 2 Bot).
// A new command constant without a DomainConfig registration reddens this
// gate; a registration without a constant fails compilation.
var wantCommands = []string{
	string(identitymodel.CmdRegisterUser),
	string(identitymodel.CmdChangeEmail),
	string(identitymodel.CmdChangeDisplayName),
	string(identitymodel.CmdDeleteUser),
	string(identitymodel.CmdAddCredential),
	string(identitymodel.CmdRemoveCredential),
	string(identitymodel.CmdRecordCredentialUse),
	string(identitymodel.CmdVerifyEmail),
	string(identitymodel.CmdEnableTOTP),
	string(identitymodel.CmdDisableTOTP),
	string(identitymodel.CmdLinkExternalAccount),
	string(identitymodel.CmdUnlinkExternalAccount),
	string(identitymodel.CmdAddMember),
	string(identitymodel.CmdUpdateMemberRoles),
	string(identitymodel.CmdRemoveMember),
	string(identitymodel.CmdCreateTenant),
	string(identitymodel.CmdSuspendTenant),
	string(identitymodel.CmdReactivateTenant),
	string(identitymodel.CmdDeleteTenant),
	string(identitymodel.CmdRegisterBot),
	string(identitymodel.CmdDeleteBot),
}

func TestArchitectureCatalog_CommandInventory(t *testing.T) {
	catalog.AssertCommandInventory(t, mustArchitectureCatalog(t), wantCommands...)
}

func TestArchitectureCatalog_EventInventory(t *testing.T) {
	want := make([]string, 0, 27)
	for _, schema := range EventSchemas() {
		want = append(want, string(schema.Type()))
	}

	catalog.AssertEventInventory(t, mustArchitectureCatalog(t), want...)
}

// TestArchitectureCatalog_EventParityWithUsermgmt pins that the
// systemcatalog-derived event universe equals the usermgmt published-language
// table — the two declarations can never drift apart silently. This test
// lives here (not in usermgmt) because systemadapter is the module that sees
// both sides.
func TestArchitectureCatalog_EventParityWithUsermgmt(t *testing.T) {
	adapter := mustArchitectureCatalog(t)
	derived := usermgmt.ArchitectureCatalog()

	adapterEvents := make(map[string]bool)
	for _, svc := range adapter.Services {
		for _, evt := range svc.Events {
			adapterEvents[string(evt.ID)] = true
		}
	}

	derivedEvents := make(map[string]bool)
	for _, svc := range derived.Services {
		for _, evt := range svc.Events {
			derivedEvents[string(evt.ID)] = true
		}
	}

	for id := range derivedEvents {
		if !adapterEvents[id] {
			t.Errorf("event %q in the usermgmt published language but not in the DomainConfig derivation", id)
		}
	}
	for id := range adapterEvents {
		if !derivedEvents[id] {
			t.Errorf("event %q in the DomainConfig derivation but not in the usermgmt published language", id)
		}
	}
}

func mustArchitectureCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()

	cat, err := ArchitectureCatalog(context.Background())
	if err != nil {
		t.Fatalf("ArchitectureCatalog: %v", err)
	}

	return cat
}
