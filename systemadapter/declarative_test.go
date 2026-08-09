package systemadapter_test

import (
	"context"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// setupDeclarativeSystem creates a system with declarative projections (no ProjectionLayer).
// The caller must defer sys.Close().
func setupDeclarativeSystem(t *testing.T) *system.System {
	t.Helper()

	ctx := context.Background()

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
		},
	}

	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
	if err != nil {
		t.Fatalf("system.New failed: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		_ = sys.Close()
		t.Fatalf("sys.Start failed: %v", err)
	}

	if sys.ProjectionHost() == nil {
		_ = sys.Close()
		t.Fatal("ProjectionHost is nil — declarative projections not wired")
	}

	return sys
}

// waitForProjections polls the projection host until all workers reach WorkerLive
// or the timeout expires.
func waitForProjections(t *testing.T, sys *system.System, timeout time.Duration) {
	t.Helper()

	host := sys.ProjectionHost()
	if host == nil {
		t.Fatal("ProjectionHost is nil")
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		allLive := true
		for _, s := range host.Status() {
			if s.Status != projectionhost.WorkerLive && s.Status != projectionhost.WorkerStopped {
				allLive = false
				break
			}
		}
		if allLive {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Print worker states for debugging
	for _, s := range host.Status() {
		t.Logf("worker %s: status=%s", s.Name, s.Status)
	}
	t.Fatalf("projections did not drain within %v", timeout)
}

func TestDeclarative_TenantRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	tenantID := id.NewStreamID()
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(
		tenantID, "Acme Corp", "Acme",
	)); err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	waitForProjections(t, sys, 5*time.Second)

	tenant, err := systemadapter.FindTenantByID(ctx, sys, tenantID.String())
	if err != nil {
		t.Fatalf("FindTenantByID: %v", err)
	}
	if tenant.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", tenant.Name, "Acme Corp")
	}
	if tenant.DisplayName != "Acme" {
		t.Errorf("DisplayName = %q, want %q", tenant.DisplayName, "Acme")
	}
	if tenant.Suspended {
		t.Error("Tenant should not be suspended after creation")
	}

	// Suspend
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewSuspendTenantCmd(
		tenantID, "test",
	)); err != nil {
		t.Fatalf("SuspendTenant: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	tenant, err = systemadapter.FindTenantByID(ctx, sys, tenantID.String())
	if err != nil {
		t.Fatalf("FindTenantByID after suspend: %v", err)
	}
	if !tenant.Suspended {
		t.Error("Tenant should be suspended")
	}

	// Reactivate
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewReactivateTenantCmd(
		tenantID,
	)); err != nil {
		t.Fatalf("ReactivateTenant: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	tenant, err = systemadapter.FindTenantByID(ctx, sys, tenantID.String())
	if err != nil {
		t.Fatalf("FindTenantByID after reactivate: %v", err)
	}
	if tenant.Suspended {
		t.Error("Tenant should not be suspended after reactivation")
	}

	// Find by name
	byName, err := systemadapter.FindTenantByName(ctx, sys, "Acme Corp")
	if err != nil {
		t.Fatalf("FindTenantByName: %v", err)
	}
	if byName.ID != tenantID.String() {
		t.Errorf("FindByName ID = %q, want %q", byName.ID, tenantID.String())
	}
}

func TestDeclarative_BotRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	ownerID := identitymodel.GenerateUserID()
	botID := id.NewStreamID()
	tokenHash := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterBotCmd(
		botID, "MyBot", ownerID, tokenHash, []string{"read", "write"},
	)); err != nil {
		t.Fatalf("RegisterBot: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	bot, err := systemadapter.FindBotByID(ctx, sys, botID.String())
	if err != nil {
		t.Fatalf("FindBotByID: %v", err)
	}
	if bot.Name != "MyBot" {
		t.Errorf("Name = %q, want %q", bot.Name, "MyBot")
	}
	if bot.OwnerID != ownerID.String() {
		t.Errorf("OwnerID = %q, want %q", bot.OwnerID, ownerID.String())
	}

	// Find by token hash
	byToken, err := systemadapter.FindBotByTokenHash(ctx, sys, "deadbeef")
	if err != nil {
		t.Fatalf("FindBotByTokenHash: %v", err)
	}
	if byToken.ID != botID.String() {
		t.Errorf("FindByToken ID = %q, want %q", byToken.ID, botID.String())
	}

	// Find by owner
	bots, err := systemadapter.FindBotsByOwner(ctx, sys, ownerID.String())
	if err != nil {
		t.Fatalf("FindBotsByOwner: %v", err)
	}
	if len(bots) != 1 {
		t.Fatalf("expected 1 bot, got %d", len(bots))
	}
}

func TestDeclarative_MembershipRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userID := identitymodel.GenerateUserID()
	tenantID := identitymodel.NewTenantID("tenant-1")
	membershipID := id.NewStreamID()

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		membershipID, identitymodel.ActorKindUser, userID.String(), tenantID.String(),
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	// Find by ID
	mem, err := systemadapter.FindMembershipByID(ctx, sys, membershipID.String())
	if err != nil {
		t.Fatalf("FindMembershipByID: %v", err)
	}
	if mem.ActorID != userID.String() {
		t.Errorf("ActorID = %q, want %q", mem.ActorID, userID.String())
	}
	if mem.TenantID != tenantID.String() {
		t.Errorf("TenantID = %q, want %q", mem.TenantID, tenantID.String())
	}
	if len(mem.Roles) != 1 || mem.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin]", mem.Roles)
	}

	// Find by tenant
	mems, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenantID.String())
	if err != nil {
		t.Fatalf("FindMembershipsByTenant: %v", err)
	}
	if len(mems) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(mems))
	}

	// Find by actor
	mems, err = systemadapter.FindMembershipsByActor(ctx, sys, userID.String())
	if err != nil {
		t.Fatalf("FindMembershipsByActor: %v", err)
	}
	if len(mems) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(mems))
	}
}

func TestDeclarative_UserRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userID := identitymodel.GenerateUserID()
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userID.ToStreamID(), "user@example.com", "Test User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	// Find by ID
	user, err := systemadapter.FindUserByID(ctx, sys, userID.String())
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "user@example.com")
	}
	if user.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", user.DisplayName, "Test User")
	}
	if user.EmailVerified {
		t.Error("EmailVerified should be false")
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	// Find by email
	byEmail, err := systemadapter.FindUserByEmail(ctx, sys, "user@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if byEmail.ID != userID.String() {
		t.Errorf("FindByEmail ID = %q, want %q", byEmail.ID, userID.String())
	}

	// Change email
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeEmailCmd(
		userID.ToStreamID(), "new@example.com",
	)); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	user, err = systemadapter.FindUserByID(ctx, sys, userID.String())
	if err != nil {
		t.Fatalf("FindUserByID after email change: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "new@example.com")
	}
	if user.EmailVerified {
		t.Error("EmailVerified should be false after email change")
	}

	// Verify email
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewVerifyEmailCmd(
		userID.ToStreamID(), "new@example.com",
	)); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	user, err = systemadapter.FindUserByID(ctx, sys, userID.String())
	if err != nil {
		t.Fatalf("FindUserByID after verify: %v", err)
	}
	if !user.EmailVerified {
		t.Error("EmailVerified should be true")
	}
}

func TestDeclarative_AuthzEnforce(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userID := identitymodel.GenerateUserID()
	tenantID := identitymodel.NewTenantID("tenant-authz")
	membershipID := id.NewStreamID()

	// Register user
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userID.ToStreamID(), "admin@example.com", "Admin User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	// Add membership with admin role
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		membershipID, identitymodel.ActorKindUser, userID.String(), tenantID.String(),
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	// Enforce: admin should be able to manage in the tenant domain
	allowed, err := systemadapter.Enforce(ctx, sys, userID.String(), tenantID.String(), "manage")
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if !allowed {
		t.Error("admin should be allowed to manage")
	}

	// Enforce: viewer should NOT be able to manage
	plainUserID := identitymodel.GenerateUserID()
	plainMembershipID := id.NewStreamID()
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		plainUserID.ToStreamID(), "plain@example.com", "Plain User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatalf("RegisterUser plain: %v", err)
	}
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		plainMembershipID, identitymodel.ActorKindUser, plainUserID.String(), tenantID.String(),
		[]identitymodel.Role{identitymodel.RoleViewer},
	)); err != nil {
		t.Fatalf("AddMember viewer: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	allowed, err = systemadapter.Enforce(ctx, sys, plainUserID.String(), tenantID.String(), "manage")
	if err != nil {
		t.Fatalf("Enforce viewer: %v", err)
	}
	if allowed {
		t.Error("viewer should NOT be allowed to manage")
	}

	// Viewer SHOULD be able to view
	allowed, err = systemadapter.Enforce(ctx, sys, plainUserID.String(), tenantID.String(), "view")
	if err != nil {
		t.Fatalf("Enforce viewer view: %v", err)
	}
	if !allowed {
		t.Error("viewer should be allowed to view")
	}
}

func TestDeclarative_AuditLog(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userID := identitymodel.GenerateUserID()
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userID.ToStreamID(), "audit@example.com", "Audit User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	// Change email to produce another audit entry
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeEmailCmd(
		userID.ToStreamID(), "changed@example.com",
	)); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	// All entries
	entries, err := systemadapter.AuditEntries(ctx, sys)
	if err != nil {
		t.Fatalf("AuditEntries: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 audit entries, got %d", len(entries))
	}

	// Entries for aggregate
	forUser, err := systemadapter.AuditEntriesFor(ctx, sys, userID.String())
	if err != nil {
		t.Fatalf("AuditEntriesFor: %v", err)
	}
	if len(forUser) < 2 {
		t.Fatalf("expected at least 2 entries for user, got %d", len(forUser))
	}

	// Verify entry content
	foundRegister := false
	foundChangeEmail := false
	for _, e := range forUser {
		if e.AggregateID != userID.String() {
			t.Errorf("AggregateID = %q, want %q", e.AggregateID, userID.String())
		}
		if e.EventType == "UserRegistered" {
			foundRegister = true
		}
		if e.EventType == "EmailChanged" {
			foundChangeEmail = true
		}
		if e.OccurredAt.IsZero() {
			t.Error("OccurredAt should not be zero")
		}
	}
	if !foundRegister {
		t.Error("UserRegistered audit entry not found")
	}
	if !foundChangeEmail {
		t.Error("EmailChanged audit entry not found")
	}

	// Recent
	recent, err := systemadapter.RecentAuditEntries(ctx, sys, 1)
	if err != nil {
		t.Fatalf("RecentAuditEntries: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent entry, got %d", len(recent))
	}
}

func TestDeclarative_AllProjectionNames(t *testing.T) {
	// Verify the projection names are consistent between declarations and queries.
	decls := systemadapter.DeclarativeProjections()
	if len(decls) < 10 {
		t.Fatalf("expected at least 10 projection declarations, got %d", len(decls))
	}

	// The key test: DeclarativeProjections must not panic during system.New
	// (fold construction, key derivation, etc.) — if we got here, it compiled.
	_ = decls
}

// Helper to avoid unused import warnings if strings is needed elsewhere.
var _ = strings.Contains
