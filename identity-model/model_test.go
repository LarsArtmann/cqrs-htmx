package identitymodel

import (
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestUserID_ParseRoundTrip(t *testing.T) {
	original := GenerateUserID()
	parsed, err := ParseUserID(original.Get().String())
	if err != nil {
		t.Fatalf("ParseUserID failed: %v", err)
	}
	if original.Get() != parsed.Get() {
		t.Errorf("round-trip mismatch: got %q, want %q", parsed.Get(), original.Get())
	}
}

func TestUserID_ParseInvalid(t *testing.T) {
	_, err := ParseUserID("not-a-ulid")
	if err == nil {
		t.Fatal("expected error for invalid ULID")
	}
}

func TestMustParseUserID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid ULID")
		}
	}()
	MustParseUserID("invalid")
}

func TestSyntheticUserID_Deterministic(t *testing.T) {
	a := SyntheticUserID("alice")
	b := SyntheticUserID("alice")
	if a.Get() != b.Get() {
		t.Errorf("SyntheticUserID is not deterministic")
	}
	c := SyntheticUserID("bob")
	if a.Get() == c.Get() {
		t.Errorf("different inputs produced same UserID")
	}
}

func TestNewUserID_EmptyString(t *testing.T) {
	uid := NewUserID("")
	if uid.Get().String() == "" {
		return // some implementations return empty string for zero value
	}
	// If not empty, it should at least be deterministic
	uid2 := NewUserID("")
	if uid.Get() != uid2.Get() {
		t.Errorf("empty string should produce deterministic UserID")
	}
}

func TestTenantID(t *testing.T) {
	tid := NewTenantID("acme")
	if tid.Get() != "acme" {
		t.Errorf("expected 'acme', got %q", tid.Get())
	}
}

func TestBotID(t *testing.T) {
	bid := NewBotID("deploy-bot")
	if bid.Get() != "deploy-bot" {
		t.Errorf("expected 'deploy-bot', got %q", bid.Get())
	}
}

func TestActorID_RoundTrip(t *testing.T) {
	uid := GenerateUserID()
	actor := ActorIDFromUser(uid)
	if actor.Kind() != ActorUser {
		t.Errorf("expected ActorUser, got %v", actor.Kind())
	}
	prefixed := actor.PrefixedString()
	parsed, err := ParseActorID(prefixed)
	if err != nil {
		t.Fatalf("ParseActorID(%q): %v", prefixed, err)
	}
	if parsed.Kind() != ActorUser {
		t.Errorf("parsed kind mismatch: got %v", parsed.Kind())
	}
	if parsed.String() != actor.String() {
		t.Errorf("parsed raw mismatch: got %q, want %q", parsed.String(), actor.String())
	}
}

func TestActorIDAsUserID(t *testing.T) {
	uid := GenerateUserID()
	got, ok := ActorIDAsUserID(ActorIDFromUser(uid))
	if !ok {
		t.Fatal("expected ok for user actor")
	}
	if got != uid {
		t.Errorf("UserID mismatch: got %q, want %q", got, uid)
	}

	for _, actor := range []ActorID{
		ActorIDFromBot(NewBotID("deploy-bot")),
		NewActorID(ActorSystem, "scheduler"),
		NewActorID(ActorService, "api-gateway"),
		{},
	} {
		got, ok := ActorIDAsUserID(actor)
		if ok {
			t.Errorf("expected !ok for %q", actor.PrefixedString())
		}
		var zero UserID
		if got != zero {
			t.Errorf("expected zero UserID for %q, got %q", actor.PrefixedString(), got)
		}
	}
}

func TestActorID_BotRoundTrip(t *testing.T) {
	bid := NewBotID("ci-bot")
	actor := ActorIDFromBot(bid)
	if actor.Kind() != ActorBot {
		t.Fatalf("expected ActorBot")
	}
	prefixed := actor.PrefixedString()
	parsed, err := ParseActorID(prefixed)
	if err != nil {
		t.Fatalf("ParseActorID(%q): %v", prefixed, err)
	}
	if parsed.Kind() != ActorBot {
		t.Errorf("parsed kind mismatch: got %v", parsed.Kind())
	}
}

func TestActorID_IsZero(t *testing.T) {
	var zero ActorID
	if !zero.IsZero() {
		t.Error("zero ActorID should report IsZero")
	}
	uid := GenerateUserID()
	actor := ActorIDFromUser(uid)
	if actor.IsZero() {
		t.Error("non-zero ActorID should not report IsZero")
	}
}

func TestActorKind_String(t *testing.T) {
	tests := []struct {
		kind ActorKind
		want string
	}{
		{ActorUser, "user"},
		{ActorBot, "bot"},
		{ActorKind(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("ActorKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestActorKindFromString_AllKinds(t *testing.T) {
	cases := []struct {
		str  string
		want ActorKind
	}{
		{ActorKindUserStr, id.ActorUser},
		{ActorKindBotStr, id.ActorBot},
		{ActorKindSystemStr, id.ActorSystem},
		{ActorKindServiceStr, id.ActorService},
	}
	for _, tc := range cases {
		got, err := ActorKindFromString(tc.str)
		if err != nil {
			t.Errorf("ActorKindFromString(%q) unexpected error: %v", tc.str, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ActorKindFromString(%q) = %v, want %v", tc.str, got, tc.want)
		}
		// Round-trip: kind → string → kind must match.
		if got.String() != tc.str {
			t.Errorf("round-trip mismatch: ActorKind(%v).String() = %q, want %q", got, got.String(), tc.str)
		}
	}
}

func TestActorKindFromString_UnknownReturnsError(t *testing.T) {
	_, err := ActorKindFromString("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown actor kind")
	}
}

func TestEmail_Parse(t *testing.T) {
	email, err := ParseEmail("  Alice@Example.COM ")
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}
	if email.String() != "alice@example.com" {
		t.Errorf("expected normalized 'alice@example.com', got %q", email.String())
	}
}

func TestEmail_ParseEmpty(t *testing.T) {
	_, err := ParseEmail("")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestEmail_ParseInvalid(t *testing.T) {
	_, err := ParseEmail("not-an-email")
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestMustParseEmail_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustParseEmail("invalid")
}

func TestSession_Creation(t *testing.T) {
	uid := GenerateUserID()
	session, err := NewSession(uid, time.Hour)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	if session.Token == "" {
		t.Error("session token should not be empty")
	}
	if session.UserID.Get() != uid.Get() {
		t.Error("session UserID mismatch")
	}
	if session.IsExpired() {
		t.Error("freshly created session should not be expired")
	}
	if !session.Valid(session.Token) {
		t.Error("session should be valid with correct token")
	}
	if session.Valid("wrong-token") {
		t.Error("session should be invalid with wrong token")
	}
}

func TestSession_Expiration(t *testing.T) {
	uid := GenerateUserID()
	session, err := NewSession(uid, -time.Hour)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	if !session.IsExpired() {
		t.Error("session with past TTL should be expired")
	}
	if session.Valid(session.Token) {
		t.Error("expired session should not be valid")
	}
}

func TestSession_Impersonation(t *testing.T) {
	target := ActorIDFromUser(GenerateUserID())
	impersonator := ActorIDFromUser(GenerateUserID())
	session, err := NewImpersonationSession(target, impersonator, "audit-investigation", time.Hour)
	if err != nil {
		t.Fatalf("NewImpersonationSession failed: %v", err)
	}
	imp, ok := session.Origin.(Impersonation)
	if !ok {
		t.Fatalf("expected Impersonation origin")
	}
	if imp.By.String() != impersonator.String() {
		t.Error("impersonator mismatch")
	}
	if imp.Reason != "audit-investigation" {
		t.Errorf("reason mismatch: got %q", imp.Reason)
	}
}

func TestSession_BotActorDoesNotPopulateUserID(t *testing.T) {
	botActor := id.NewBotActor("ci-bot")
	session, err := newSession(botActor, DirectLogin{AuthenticatedAs: botActor}, time.Hour)
	if err != nil {
		t.Fatalf("newSession failed: %v", err)
	}
	var zero UserID
	if session.UserID != zero {
		t.Errorf("bot session UserID should be zero, got %s", session.UserID.Get())
	}
	if session.ActorID.Kind() != id.ActorBot {
		t.Errorf("ActorID kind = %v, want ActorBot", session.ActorID.Kind())
	}
}

func TestSession_SystemActorDoesNotPopulateUserID(t *testing.T) {
	sysActor := id.NewSystemActor("scheduler")
	session, err := newSession(sysActor, DirectLogin{AuthenticatedAs: sysActor}, time.Hour)
	if err != nil {
		t.Fatalf("newSession failed: %v", err)
	}
	var zero UserID
	if session.UserID != zero {
		t.Errorf("system session UserID should be zero, got %s", session.UserID.Get())
	}
	if session.ActorID.Kind() != id.ActorSystem {
		t.Errorf("ActorID kind = %v, want ActorSystem", session.ActorID.Kind())
	}
}

func TestRole_Valid(t *testing.T) {
	valid := []Role{RoleSuperAdmin, RoleAdmin, RoleUser, RoleViewer, RoleOwner}
	for _, r := range valid {
		if !r.Valid() {
			t.Errorf("role %q should be valid", r)
		}
	}
	if Role("invalid").Valid() {
		t.Error("invalid role should not pass Valid()")
	}
}

func TestAssignableRoles(t *testing.T) {
	roles := AssignableRoles()
	for _, r := range roles {
		if r == RoleSuperAdmin {
			t.Error("AssignableRoles should not include super_admin")
		}
	}
	if len(roles) != 4 {
		t.Errorf("expected 4 assignable roles, got %d", len(roles))
	}
}

func TestAction_Valid(t *testing.T) {
	if !ActionExecute.Valid() {
		t.Error("ActionExecute should be valid")
	}
	if !ActionRead.Valid() {
		t.Error("ActionRead should be valid")
	}
	if !ActionAll.Valid() {
		t.Error("ActionAll should be valid")
	}
	if Action("invalid").Valid() {
		t.Error("invalid action should not pass Valid()")
	}
}

func TestEffect_Valid(t *testing.T) {
	if !EffectAllow.Valid() {
		t.Error("EffectAllow should be valid")
	}
	if !EffectDeny.Valid() {
		t.Error("EffectDeny should be valid")
	}
	if Effect("maybe").Valid() {
		t.Error("invalid effect should not pass Valid()")
	}
}

func TestDefaultPolicies(t *testing.T) {
	policies := DefaultPolicies()
	if len(policies) != 2 {
		t.Fatalf("expected 2 default policies, got %d", len(policies))
	}
	for _, p := range policies {
		if p.Action != ActionAll || p.Effect != EffectAllow {
			t.Errorf("policy %v should be wildcard allow", p)
		}
	}
}

func TestDefaultRoleHierarchy(t *testing.T) {
	hierarchy := DefaultRoleHierarchy()
	if len(hierarchy) != 3 {
		t.Fatalf("expected 3 hierarchy entries, got %d", len(hierarchy))
	}
	if hierarchy[0].From != RoleSuperAdmin || hierarchy[0].To != RoleAdmin {
		t.Error("first hierarchy entry should be super_admin -> admin")
	}
}

func TestGenerateToken_Uniqueness(t *testing.T) {
	a, _ := GenerateToken()
	b, _ := GenerateToken()
	if a == b {
		t.Error("tokens should be unique")
	}
}

func TestHashToken_VerifyRoundTrip(t *testing.T) {
	pepper := []byte("test-pepper")
	token, _ := GenerateToken()
	hashed := HashToken(token, pepper)
	if !VerifyToken(token, hashed, pepper) {
		t.Error("VerifyToken should return true for correct token")
	}
	if VerifyToken("wrong-token", hashed, pepper) {
		t.Error("VerifyToken should return false for wrong token")
	}
}

func TestUser_Clone(t *testing.T) {
	original := &User{
		ID:        GenerateUserID(),
		Email:     "test@example.com",
		UpdatedAt: time.Now(),
	}
	original.Credentials = []WebAuthnCredential{
		{CredentialCore: CredentialCore{ID: []byte("cred-1")}},
	}
	cloned := original.Clone()
	if cloned.Email != original.Email {
		t.Error("clone email mismatch")
	}
	cloned.Credentials[0].ID[0] = 'X'
	if original.Credentials[0].ID[0] == 'X' {
		t.Error("clone should be deep — modifying clone should not affect original")
	}
}

func TestUser_HasCredential(t *testing.T) {
	u := &User{
		Credentials: []WebAuthnCredential{
			{CredentialCore: CredentialCore{ID: []byte("cred-1")}},
		},
	}
	if !u.HasCredential([]byte("cred-1")) {
		t.Error("should find existing credential")
	}
	if u.HasCredential([]byte("cred-2")) {
		t.Error("should not find non-existing credential")
	}
}

func TestMembership_HasRole(t *testing.T) {
	m := Membership{Roles: []Role{RoleAdmin, RoleUser}}
	if !m.HasRole(RoleAdmin) {
		t.Error("should have admin role")
	}
	if m.HasRole(RoleViewer) {
		t.Error("should not have viewer role")
	}
}

func TestMembership_HasAnyRole(t *testing.T) {
	m := Membership{Roles: []Role{RoleAdmin}}
	if !m.HasAnyRole(RoleAdmin, RoleViewer) {
		t.Error("should match at least one role")
	}
	if m.HasAnyRole(RoleViewer, RoleOwner) {
		t.Error("should not match any role")
	}
}

func TestExternalAccount(t *testing.T) {
	ea := NewExternalAccount("google", "sub-123", "user@example.com", "User", time.Now())
	if ea.Provider != "google" {
		t.Errorf("expected 'google', got %q", ea.Provider)
	}
	if ea.Subject != "sub-123" {
		t.Errorf("expected 'sub-123', got %q", ea.Subject)
	}
}

func TestWebAuthnCredential_Clone(t *testing.T) {
	c := WebAuthnCredential{
		CredentialCore: CredentialCore{
			ID:         []byte("cred-1"),
			PublicKey:  []byte("key"),
			Transports: []string{"usb", "nfc"},
		},
	}
	cloned := c.Clone()
	cloned.Transports[0] = "ble"
	if c.Transports[0] == "ble" {
		t.Error("clone should be deep — modifying clone should not affect original")
	}
}

func TestDeriveMembershipID_Deterministic(t *testing.T) {
	actor := ActorIDFromUser(GenerateUserID())
	tenant := NewTenantID("acme")
	id1 := DeriveMembershipID(actor, tenant)
	id2 := DeriveMembershipID(actor, tenant)
	if id1.String() != id2.String() {
		t.Error("DeriveMembershipID should be deterministic")
	}
}

func TestFoldUser_Registered(t *testing.T) {
	aggID := id.NewStreamID()

	payload, err := MarshalPayload(UserRegisteredPayload{
		SchemaVersion: CurrentSchemaVersion,
		Email:         "test@example.com",
		DisplayName:   "Test",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	evt, evtErr := event.NewEvent(EventUserRegistered, aggID, AggregateTypeUser, 1, payload)
	if evtErr != nil {
		t.Fatalf("event.New: %v", evtErr)
	}

	state, err := FoldUser(UserState{}, evt)
	if err != nil {
		t.Fatalf("FoldUser failed: %v", err)
	}
	if !state.Exists() {
		t.Error("state should exist after UserRegistered")
	}
	if state.Email != "test@example.com" {
		t.Errorf("expected 'test@example.com', got %q", state.Email)
	}
}

func TestFoldTenant_Lifecycle(t *testing.T) {
	aggID := id.NewStreamID()

	createdPayload, _ := MarshalPayload(TenantCreatedPayload{
		SchemaVersion: CurrentSchemaVersion,
		Name:          "acme",
		DisplayName:   "Acme Corp",
	})
	created, err := event.NewEvent(EventTenantCreated, aggID, AggregateTypeTenant, 1, createdPayload)
	if err != nil {
		t.Fatalf("event.New created: %v", err)
	}
	state, foldErr := FoldTenant(TenantState{}, created)
	if foldErr != nil {
		t.Fatalf("FoldTenant created: %v", foldErr)
	}
	if !state.IsActive() {
		t.Error("tenant should be active after creation")
	}

	suspendedPayload, _ := MarshalPayload(TenantSuspendedPayload{
		SchemaVersion: CurrentSchemaVersion,
		Reason:        "non-payment",
	})
	suspended, err := event.NewEvent(EventTenantSuspended, aggID, AggregateTypeTenant, 2, suspendedPayload)
	if err != nil {
		t.Fatalf("event.New suspended: %v", err)
	}
	state, foldErr = FoldTenant(state, suspended)
	if foldErr != nil {
		t.Fatalf("FoldTenant suspended: %v", foldErr)
	}
	if state.IsActive() {
		t.Error("tenant should not be active when suspended")
	}
	if !state.IsValid() {
		t.Error("suspended tenant should be valid")
	}
}

func TestErrors_Exist(t *testing.T) {
	checks := []error{
		ErrUserNotFound, ErrEmailExists, ErrInvalidCredentials,
		ErrForbidden, ErrUnauthorized, ErrValidation,
		ErrAccountLocked, ErrOAuthNotConfigured,
	}
	for _, err := range checks {
		if err == nil {
			t.Error("error sentinel should not be nil")
		}
		if err.Error() == "" {
			t.Error("error sentinel should have non-empty message")
		}
	}
}

func TestFoldMembership_Lifecycle(t *testing.T) {
	userID := GenerateUserID()
	tenantID := NewTenantID("acme")
	actorID := ActorIDFromUser(userID)
	aggID := DeriveMembershipID(actorID, tenantID)

	addedPayload, _ := MarshalPayload(MemberAddedPayload{
		SchemaVersion: CurrentSchemaVersion,
		ActorKind:     ActorKindUserStr,
		ActorID:       actorID.String(),
		TenantID:      tenantID.String(),
		Roles:         []Role{RoleAdmin},
	})
	added, err := event.NewEvent(EventMemberAdded, aggID, AggregateTypeMembership, 1, addedPayload)
	if err != nil {
		t.Fatalf("event.New MemberAdded: %v", err)
	}
	state, foldErr := FoldMembership(MembershipState{}, added)
	if foldErr != nil {
		t.Fatalf("FoldMembership MemberAdded: %v", foldErr)
	}
	if !state.Exists() {
		t.Fatal("membership should exist after MemberAdded")
	}
	if !state.HasRole(RoleAdmin) {
		t.Error("membership should have Admin role")
	}

	changedPayload, _ := MarshalPayload(MemberRolesChangedPayload{
		SchemaVersion: CurrentSchemaVersion,
		ActorKind:     ActorKindUserStr,
		ActorID:       actorID.String(),
		TenantID:      tenantID.String(),
		Roles:         []Role{RoleUser},
	})
	changed, err := event.NewEvent(EventMemberRolesChanged, aggID, AggregateTypeMembership, 2, changedPayload)
	if err != nil {
		t.Fatalf("event.New MemberRolesChanged: %v", err)
	}
	state, foldErr = FoldMembership(state, changed)
	if foldErr != nil {
		t.Fatalf("FoldMembership MemberRolesChanged: %v", foldErr)
	}
	if state.HasRole(RoleAdmin) {
		t.Error("membership should no longer have Admin role")
	}
	if !state.HasRole(RoleUser) {
		t.Error("membership should have User role after change")
	}

	removedPayload, _ := MarshalPayload(MemberRemovedPayload{
		SchemaVersion: CurrentSchemaVersion,
		ActorID:       actorID.String(),
		TenantID:      tenantID.String(),
	})
	removed, err := event.NewEvent(EventMemberRemoved, aggID, AggregateTypeMembership, 3, removedPayload)
	if err != nil {
		t.Fatalf("event.New MemberRemoved: %v", err)
	}
	state, foldErr = FoldMembership(state, removed)
	if foldErr != nil {
		t.Fatalf("FoldMembership MemberRemoved: %v", foldErr)
	}
	if state.Exists() {
		t.Error("membership should not exist after MemberRemoved")
	}
}

func TestFoldBot_Lifecycle(t *testing.T) {
	aggID := id.NewStreamID()
	ownerID := GenerateUserID()

	registeredPayload, _ := MarshalPayload(BotRegisteredPayload{
		SchemaVersion: CurrentSchemaVersion,
		Name:          "deploy-bot",
		OwnerID:       ownerID,
		TokenHash:     []byte("hashed-token"),
		Scopes:        []string{"deploy", "read"},
	})
	registered, err := event.NewEvent(EventBotRegistered, aggID, AggregateTypeBot, 1, registeredPayload)
	if err != nil {
		t.Fatalf("event.New BotRegistered: %v", err)
	}
	state, foldErr := FoldBot(BotState{}, registered)
	if foldErr != nil {
		t.Fatalf("FoldBot BotRegistered: %v", foldErr)
	}
	if !state.Exists() {
		t.Fatal("bot should exist after BotRegistered")
	}
	if state.Name != "deploy-bot" {
		t.Errorf("expected name 'deploy-bot', got %q", state.Name)
	}
	if len(state.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(state.Scopes))
	}

	deletedPayload, _ := MarshalPayload(BotDeletedPayload{
		SchemaVersion: CurrentSchemaVersion,
		Reason:        "decommissioned",
	})
	deleted, err := event.NewEvent(EventBotDeleted, aggID, AggregateTypeBot, 2, deletedPayload)
	if err != nil {
		t.Fatalf("event.New BotDeleted: %v", err)
	}
	state, foldErr = FoldBot(state, deleted)
	if foldErr != nil {
		t.Fatalf("FoldBot BotDeleted: %v", foldErr)
	}
	if state.Exists() {
		t.Error("bot should not exist after BotDeleted")
	}
}

func TestActorID_JSONRoundTrip(t *testing.T) {
	original := ActorIDFromUser(GenerateUserID())
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ActorID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if original.String() != decoded.String() {
		t.Errorf("round-trip mismatch: got %q, want %q", decoded.String(), original.String())
	}
	if original.Kind() != decoded.Kind() {
		t.Errorf("kind mismatch: got %d, want %d", decoded.Kind(), original.Kind())
	}
}

func TestSession_JSONRoundTrip(t *testing.T) {
	userID := GenerateUserID()
	sess, err := NewSession(userID, time.Hour)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Session
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if sess.Token != decoded.Token {
		t.Errorf("token mismatch: got %q, want %q", decoded.Token, sess.Token)
	}
	if sess.UserID.Get() != decoded.UserID.Get() {
		t.Errorf("userID mismatch")
	}
	if sess.ActorID.String() != decoded.ActorID.String() {
		t.Errorf("actorID mismatch")
	}
	if !sess.CreatedAt.Equal(decoded.CreatedAt) {
		t.Errorf("createdAt mismatch: got %v, want %v", decoded.CreatedAt, sess.CreatedAt)
	}
}
