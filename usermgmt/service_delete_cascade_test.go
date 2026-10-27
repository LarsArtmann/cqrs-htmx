package usermgmt

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// groupPoliciesInDomain counts Casbin group-policy rows (subject, role,
// domain) whose domain matches — the membership grant + hierarchy-mirror
// surface the TenantDeleted cascade must clear.
func groupPoliciesInDomain(t *testing.T, svc *Service, domain string) int {
	t.Helper()

	rows, err := svc.Authz().GroupPolicies()
	if err != nil {
		t.Fatalf("GroupPolicies: %v", err)
	}

	n := 0
	for _, row := range rows {
		if len(row) == 3 && row[2] == domain {
			n++
		}
	}
	return n
}

// TestService_DeleteUser_CascadeMemberships verifies that deleting a user
// removes all their memberships across all tenants.
func TestService_DeleteUser_CascadeMemberships(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	userID := registerIdentityTestUser(t, svc, "cascade-member@test.com")
	actor := ActorIDFromUser(userID)

	tenantA, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID("01JXTENANT00000000000000A1"),
		Name: "tenant-a",
	})
	if err != nil {
		t.Fatalf("CreateTenant A: %v", err)
	}
	tenantB, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID("01JXTENANT00000000000000B2"),
		Name: "tenant-b",
	})
	if err != nil {
		t.Fatalf("CreateTenant B: %v", err)
	}

	if err := svc.AddMember(ctx, actor, tenantA.ID, []Role{RoleUser}); err != nil {
		t.Fatalf("AddMember A: %v", err)
	}
	if err := svc.AddMember(ctx, actor, tenantB.ID, []Role{RoleAdmin}); err != nil {
		t.Fatalf("AddMember B: %v", err)
	}

	if got := len(svc.membershipReadModel.FindByActor(userID.Get().String())); got != 2 {
		t.Fatalf("expected 2 memberships before delete, got %d", got)
	}

	if err := svc.DeleteUser(ctx, userID, "test cascade"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	if got := len(svc.membershipReadModel.FindByActor(userID.Get().String())); got != 0 {
		t.Fatalf("expected 0 memberships after delete, got %d", got)
	}
}

// TestService_DeleteUser_CascadeBots verifies that deleting a user
// removes all bots owned by that user, preventing orphaned API tokens.
func TestService_DeleteUser_CascadeBots(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	userID := registerIdentityTestUser(t, svc, "cascade-bots@test.com")

	botID1 := NewBotID(id.NewStreamID().String())
	result1, err := svc.RegisterBot(ctx, RegisterBotRequest{
		ID:      botID1,
		Name:    "bot-1",
		OwnerID: userID,
		Scopes:  []string{"read"},
	})
	if err != nil {
		t.Fatalf("RegisterBot 1: %v", err)
	}

	botID2 := NewBotID(id.NewStreamID().String())
	_, err = svc.RegisterBot(ctx, RegisterBotRequest{
		ID:      botID2,
		Name:    "bot-2",
		OwnerID: userID,
		Scopes:  []string{"deploy"},
	})
	if err != nil {
		t.Fatalf("RegisterBot 2: %v", err)
	}

	if got := len(svc.botReadModel.FindByOwner(userID)); got != 2 {
		t.Fatalf("expected 2 bots before delete, got %d", got)
	}

	if err := svc.DeleteUser(ctx, userID, "test cascade"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	if got := len(svc.botReadModel.FindByOwner(userID)); got != 0 {
		t.Fatalf("expected 0 bots after delete, got %d", got)
	}

	if _, ok := svc.botReadModel.FindByTokenHash(result1.Bot.TokenHash); ok {
		t.Fatal("deleted user's bot token should no longer authenticate")
	}
}

// TestService_DeleteTenant_CascadesCasbinPolicies pins the S37a fix: deleting
// a tenant removes every member's role assignment in that tenant's Casbin
// domain. The pre-fix code called RemoveAllRolesForUser(tenantStreamID),
// which removes nothing — the tenant never appears as a grant subject; its
// members do — so deleted tenants kept authorizing their former members.
// Grants in surviving tenants must be untouched.
func TestService_DeleteTenant_CascadesCasbinPolicies(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	userID := registerIdentityTestUser(t, svc, "tenant-cascade@test.com")
	actor := ActorIDFromUser(userID)

	tenantA, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID("01JXTENANT00000000000000C3"),
		Name: "tenant-c",
	})
	if err != nil {
		t.Fatalf("CreateTenant A: %v", err)
	}
	tenantB, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID("01JXTENANT00000000000000D4"),
		Name: "tenant-d",
	})
	if err != nil {
		t.Fatalf("CreateTenant B: %v", err)
	}

	if err := svc.AddMember(ctx, actor, tenantA.ID, []Role{RoleUser}); err != nil {
		t.Fatalf("AddMember A: %v", err)
	}
	if err := svc.AddMember(ctx, actor, tenantB.ID, []Role{RoleAdmin}); err != nil {
		t.Fatalf("AddMember B: %v", err)
	}

	if got := groupPoliciesInDomain(t, svc, tenantA.ID.Get()); got == 0 {
		t.Fatal("expected member grants in tenant A before delete")
	}

	if err := svc.DeleteTenant(ctx, tenantA.ID, "cascade test"); err != nil {
		t.Fatalf("DeleteTenant A: %v", err)
	}

	if got := groupPoliciesInDomain(t, svc, tenantA.ID.Get()); got != 0 {
		t.Fatalf("expected 0 Casbin policies in deleted tenant A, got %d", got)
	}
	if got := groupPoliciesInDomain(t, svc, tenantB.ID.Get()); got == 0 {
		t.Fatal("tenant B grants must survive tenant A's deletion")
	}
}

// TestService_DeleteTenant_PartialCleanupIsVisibleAndStillCascades: when a
// membership's RemoveMember dispatch fails during tenant deletion (here: a
// phantom read-model row with no event stream), DeleteTenant still succeeds,
// the partial cleanup is summarized in one warning line (not silently
// swallowed), and the Casbin cascade removes the phantom's grant anyway —
// authorization cleanup never depends on the best-effort dispatches.
func TestService_DeleteTenant_PartialCleanupIsVisibleAndStillCascades(t *testing.T) {
	buf := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	svc, err := NewService(ServiceConfig{
		TokenPepper: TokenPepper("test-pepper-32-bytes-long-xxxxxxx"),
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	userID := registerIdentityTestUser(t, svc, "partial-cleanup@test.com")
	realActor := ActorIDFromUser(userID)

	tenant, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID("01JXTENANT00000000000000E5"),
		Name: "tenant-e",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	if err := svc.AddMember(ctx, realActor, tenant.ID, []Role{RoleUser}); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	phantom := ActorIDFromUser(GenerateUserID())
	aggID := deriveMembershipID(phantom, tenant.ID)
	svc.membershipReadModel.mu.Lock()
	svc.membershipReadModel.memberships[aggID] = &Membership{
		ActorID:  phantom,
		TenantID: tenant.ID,
		Roles:    []Role{RoleUser},
	}
	svc.membershipReadModel.byActor[phantom.String()] = append(
		svc.membershipReadModel.byActor[phantom.String()], aggID,
	)
	svc.membershipReadModel.mu.Unlock()
	if err := svc.authz.AddGroupPolicy(GroupPolicy{
		Subject: phantom.String(),
		Role:    RoleUser,
		Domain:  tenant.ID.Get(),
	}); err != nil {
		t.Fatalf("grant phantom policy: %v", err)
	}

	if err := svc.DeleteTenant(ctx, tenant.ID, "partial cleanup test"); err != nil {
		t.Fatalf("DeleteTenant must succeed despite failed member removals: %v", err)
	}

	if !strings.Contains(buf.String(), "partial membership cleanup") {
		t.Fatalf("expected a partial-cleanup summary warning, got log:\n%s", buf.String())
	}
	if got := groupPoliciesInDomain(t, svc, tenant.ID.Get()); got != 0 {
		t.Fatalf("expected 0 Casbin policies after cascade (incl. phantom), got %d", got)
	}
}

// syncBuffer is a mutex-guarded slog sink: projection goroutines and the
// service share the logger, so the writer must be safe for concurrent use.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}
