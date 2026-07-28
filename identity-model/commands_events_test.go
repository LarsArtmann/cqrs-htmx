package identitymodel

import (
	"encoding/json/v2"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func testStreamID() id.StreamID { return id.NewStreamID() }

func TestNewRegisterUserCmd(t *testing.T) {
	aggID := testStreamID()
	roles := []Role{RoleAdmin, RoleUser}
	cmd := NewRegisterUserCmd(aggID, "alice@example.com", "Alice", roles)

	if cmd.Type() != CmdRegisterUser {
		t.Fatalf("Type: got %s, want %s", cmd.Type(), CmdRegisterUser)
	}
	if cmd.AggregateID() != aggID {
		t.Fatalf("AggregateID mismatch")
	}
	if cmd.Email() != "alice@example.com" {
		t.Fatalf("Email: got %q", cmd.Email())
	}
	if cmd.DisplayName() != "Alice" {
		t.Fatalf("DisplayName: got %q", cmd.DisplayName())
	}
	if !slices.Equal(cmd.Roles(), roles) {
		t.Fatalf("Roles: got %v", cmd.Roles())
	}
}

func TestNewChangeEmailCmd(t *testing.T) {
	aggID := testStreamID()
	cmd := NewChangeEmailCmd(aggID, "new@example.com")
	if cmd.Type() != CmdChangeEmail || cmd.Email() != "new@example.com" || cmd.AggregateID() != aggID {
		t.Fatalf("unexpected cmd: %+v", cmd)
	}
}

func TestNewChangeDisplayNameCmd(t *testing.T) {
	cmd := NewChangeDisplayNameCmd(testStreamID(), "Bob")
	if cmd.DisplayName() != "Bob" || cmd.Type() != CmdChangeDisplayName {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewDeleteUserCmd(t *testing.T) {
	cmd := NewDeleteUserCmd(testStreamID(), "gdpr")
	if cmd.Reason() != "gdpr" || cmd.Type() != CmdDeleteUser {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewAddCredentialCmd(t *testing.T) {
	cred := WebAuthnCredential{
		CredentialCore: CredentialCore{ID: []byte{1, 2}, AttestationType: "none"},
	}
	cmd := NewAddCredentialCmd(testStreamID(), cred)
	if !slices.Equal(cmd.Credential().ID, []byte{1, 2}) || cmd.Type() != CmdAddCredential {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewRemoveCredentialCmd(t *testing.T) {
	cmd := NewRemoveCredentialCmd(testStreamID(), []byte{0xAA})
	if !slices.Equal(cmd.CredentialID(), []byte{0xAA}) || cmd.Type() != CmdRemoveCredential {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewVerifyEmailCmd(t *testing.T) {
	if NewVerifyEmailCmd(testStreamID()).Type() != CmdVerifyEmail {
		t.Fatal("type mismatch")
	}
}

func TestNewEnableTOTPCmd(t *testing.T) {
	cmd := NewEnableTOTPCmd(testStreamID(), []byte("secret"))
	if string(cmd.Secret()) != "secret" || cmd.Type() != CmdEnableTOTP {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewDisableTOTPCmd(t *testing.T) {
	if NewDisableTOTPCmd(testStreamID()).Type() != CmdDisableTOTP {
		t.Fatal("type mismatch")
	}
}

func TestNewLinkExternalAccountCmd(t *testing.T) {
	cmd := NewLinkExternalAccountCmd(testStreamID(), "google", "sub123", "g@example.com", "G")
	if cmd.Provider() != "google" || cmd.Subject() != "sub123" || cmd.Email() != "g@example.com" || cmd.DisplayName() != "G" {
		t.Fatalf("unexpected: %+v", cmd)
	}
	if cmd.Type() != CmdLinkExternalAccount {
		t.Fatal("type mismatch")
	}
}

func TestNewUnlinkExternalAccountCmd(t *testing.T) {
	cmd := NewUnlinkExternalAccountCmd(testStreamID(), "github", "sub456")
	if cmd.Provider() != "github" || cmd.Subject() != "sub456" || cmd.Type() != CmdUnlinkExternalAccount {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewAddMemberCmd(t *testing.T) {
	aid := NewActorID(ActorUser, "u1")
	tid := NewTenantID("t1")
	roles := []Role{RoleViewer}
	cmd := NewAddMemberCmd(aid, tid, roles)
	if cmd.ActorID() != aid || cmd.TenantID() != tid || !slices.Equal(cmd.Roles(), roles) {
		t.Fatalf("unexpected: %+v", cmd)
	}
	if cmd.Type() != CmdAddMember {
		t.Fatal("type mismatch")
	}
}

func TestNewUpdateMemberRolesCmd(t *testing.T) {
	roles := []Role{RoleAdmin}
	cmd := NewUpdateMemberRolesCmd(NewActorID(ActorUser, "u1"), NewTenantID("t1"), roles)
	if !slices.Equal(cmd.Roles(), roles) || cmd.Type() != CmdUpdateMemberRoles {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewRemoveMemberCmd(t *testing.T) {
	cmd := NewRemoveMemberCmd(NewActorID(ActorUser, "u1"), NewTenantID("t1"))
	if cmd.Type() != CmdRemoveMember {
		t.Fatal("type mismatch")
	}
}

func TestNewCreateTenantCmd(t *testing.T) {
	cmd := NewCreateTenantCmd(testStreamID(), "acme", "Acme Inc")
	if cmd.Name() != "acme" || cmd.DisplayName() != "Acme Inc" || cmd.Type() != CmdCreateTenant {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewSuspendTenantCmd(t *testing.T) {
	cmd := NewSuspendTenantCmd(testStreamID(), "billing")
	if cmd.Reason() != "billing" || cmd.Type() != CmdSuspendTenant {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewReactivateTenantCmd(t *testing.T) {
	if NewReactivateTenantCmd(testStreamID()).Type() != CmdReactivateTenant {
		t.Fatal("type mismatch")
	}
}

func TestNewDeleteTenantCmd(t *testing.T) {
	cmd := NewDeleteTenantCmd(testStreamID(), "merged")
	if cmd.Reason() != "merged" || cmd.Type() != CmdDeleteTenant {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

func TestNewRegisterBotCmd(t *testing.T) {
	owner := GenerateUserID()
	cmd := NewRegisterBotCmd(testStreamID(), "ci-bot", owner, []byte("hash"), []string{"deploy"})
	if cmd.Name() != "ci-bot" || cmd.OwnerID() != owner || string(cmd.TokenHash()) != "hash" || len(cmd.Scopes()) != 1 {
		t.Fatalf("unexpected: %+v", cmd)
	}
	if cmd.Type() != CmdRegisterBot {
		t.Fatal("type mismatch")
	}
}

func TestNewDeleteBotCmd(t *testing.T) {
	cmd := NewDeleteBotCmd(testStreamID(), "retired")
	if cmd.Reason() != "retired" || cmd.Type() != CmdDeleteBot {
		t.Fatalf("unexpected: %+v", cmd)
	}
}

// --- Event payload round-trip tests ---

func TestUserRegisteredPayload_RoundTrip(t *testing.T) {
	original := UserRegisteredPayload{
		SchemaVersion: 1,
		Email:         "alice@example.com",
		DisplayName:   "Alice",
		Roles:         []Role{RoleAdmin},
	}
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded UserRegisteredPayload
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Email != original.Email || decoded.DisplayName != original.DisplayName {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
	if !slices.Equal(decoded.Roles, original.Roles) {
		t.Fatalf("roles mismatch: %+v", decoded)
	}
}

func TestEmailChangedPayload_RoundTrip(t *testing.T) {
	original := EmailChangedPayload{SchemaVersion: 1, Email: "new@example.com"}
	b, _ := json.Marshal(original)
	var decoded EmailChangedPayload
	_ = json.Unmarshal(b, &decoded)
	if decoded.Email != "new@example.com" {
		t.Fatalf("email mismatch: %q", decoded.Email)
	}
}

func TestMemberAddedPayload_RoundTrip(t *testing.T) {
	original := MemberAddedPayload{
		SchemaVersion: 1,
		ActorKind:     "user",
		ActorID:       "u1",
		TenantID:      "t1",
		Roles:         []Role{RoleOwner},
	}
	b, _ := json.Marshal(original)
	var decoded MemberAddedPayload
	_ = json.Unmarshal(b, &decoded)
	if decoded.ActorID != "u1" || decoded.TenantID != "t1" || len(decoded.Roles) != 1 {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
}

func TestTenantCreatedPayload_RoundTrip(t *testing.T) {
	original := TenantCreatedPayload{SchemaVersion: 1, Name: "acme", DisplayName: "Acme"}
	b, _ := json.Marshal(original)
	var decoded TenantCreatedPayload
	_ = json.Unmarshal(b, &decoded)
	if decoded.Name != "acme" || decoded.DisplayName != "Acme" {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
}

func TestBotRegisteredPayload_RoundTrip(t *testing.T) {
	owner := GenerateUserID()
	original := BotRegisteredPayload{
		SchemaVersion: 1,
		Name:          "ci",
		OwnerID:       owner,
		TokenHash:     []byte{0xAA, 0xBB},
		Scopes:        []string{"deploy", "rollback"},
	}
	b, _ := json.Marshal(original)
	var decoded BotRegisteredPayload
	_ = json.Unmarshal(b, &decoded)
	if decoded.Name != "ci" || decoded.OwnerID != owner || len(decoded.TokenHash) != 2 || len(decoded.Scopes) != 2 {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
}

func TestCredentialAddedPayload_RoundTrip(t *testing.T) {
	original := CredentialAddedPayload{
		SchemaVersion: 1,
		CredentialCore: CredentialCore{
			ID:              []byte{1, 2, 3},
			AttestationType: "none",
			SignCount:       42,
		},
	}
	b, _ := json.Marshal(original)
	var decoded CredentialAddedPayload
	_ = json.Unmarshal(b, &decoded)
	if !slices.Equal(decoded.ID, []byte{1, 2, 3}) || decoded.AttestationType != "none" || decoded.SignCount != 42 {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
}

func TestExternalAccountLinkedPayload_RoundTrip(t *testing.T) {
	original := ExternalAccountLinkedPayload{
		SchemaVersion: 1,
		ExternalAccountCore: ExternalAccountCore{
			Provider:    "google",
			Subject:     "sub123",
			Email:       "g@example.com",
			DisplayName: "G",
		},
	}
	b, _ := json.Marshal(original)
	var decoded ExternalAccountLinkedPayload
	_ = json.Unmarshal(b, &decoded)
	if decoded.Provider != "google" || decoded.Subject != "sub123" {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
}

func TestUserDeletedPayload_RoundTrip(t *testing.T) {
	original := UserDeletedPayload{SchemaVersion: 1, Reason: "gdpr"}
	b, _ := json.Marshal(original)
	var decoded UserDeletedPayload
	_ = json.Unmarshal(b, &decoded)
	if decoded.Reason != "gdpr" {
		t.Fatalf("reason mismatch: %q", decoded.Reason)
	}
}

func TestDeriveMembershipID_Deterministic(t *testing.T) {
	aid := NewActorID(ActorUser, "u1")
	tid := NewTenantID("t1")
	first := DeriveMembershipID(aid, tid)
	second := DeriveMembershipID(aid, tid)
	if first != second {
		t.Fatalf("DeriveMembershipID not deterministic: %s vs %s", first, second)
	}
}

func TestNewCredentialFromPayload(t *testing.T) {
	p := CredentialAddedPayload{
		CredentialCore: CredentialCore{
			ID:         []byte{1},
			Transports: []string{"usb"},
			AAGUID:     []byte{2},
		},
	}
	cred := NewCredentialFromPayload(p, testTimestamp())
	if !slices.Equal(cred.ID, []byte{1}) || cred.Transports[0] != "usb" {
		t.Fatalf("unexpected credential: %+v", cred)
	}
}

func TestWebAuthnCredential_Clone(t *testing.T) {
	cred := WebAuthnCredential{
		CredentialCore: CredentialCore{
			ID:         []byte{1, 2},
			Transports: []string{"usb"},
		},
	}
	clone := cred.Clone()
	clone.ID[0] = 99
	if cred.ID[0] == 99 {
		t.Fatal("Clone did not deep-copy ID slice")
	}
}
