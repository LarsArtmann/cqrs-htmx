package systemadapter_test

import (
	"context"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

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

func waitForProjections(t *testing.T, sys *system.System, timeout time.Duration) {
	t.Helper()

	host := sys.ProjectionHost()
	if host == nil {
		t.Fatal("ProjectionHost is nil")
	}

	deadline := time.Now().Add(timeout)

	// Phase 1: wait for workers to reach Live state.
	for time.Now().Before(deadline) {
		states := host.Status()
		if len(states) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		allLive := true
		for _, s := range states {
			if s.Status != projectionhost.WorkerLive && s.Status != projectionhost.WorkerStopped {
				allLive = false
				break
			}
		}
		if allLive {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Phase 2: wait for processed counters to stabilize.
	// Initialize to -1 so the first snapshot is always "changed".
	states := host.Status()
	prevProcessed := make(map[string]int64, len(states))
	for _, s := range states {
		prevProcessed[s.Name] = -1
	}
	stableCount := 0

	for time.Now().Before(deadline) {
		states := host.Status()
		stable := true
		for _, s := range states {
			if s.Processed != prevProcessed[s.Name] {
				prevProcessed[s.Name] = s.Processed
				stable = false
			}
		}
		if stable {
			stableCount++
			if stableCount >= 10 { // 10 × 20ms = 200ms of stability
				return
			}
		} else {
			stableCount = 0
		}
		time.Sleep(20 * time.Millisecond)
	}

	for _, s := range host.Status() {
		t.Logf("worker %s: status=%s processed=%d", s.Name, s.Status, s.Processed)
	}
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

	byToken, err := systemadapter.FindBotByTokenHash(ctx, sys, "deadbeef")
	if err != nil {
		t.Fatalf("FindBotByTokenHash: %v", err)
	}
	if byToken.ID != botID.String() {
		t.Errorf("FindByToken ID = %q, want %q", byToken.ID, botID.String())
	}

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

	userStreamID := id.NewStreamID()
	actorID := identitymodel.NewActorID(identitymodel.ActorUser, userStreamID.String())
	tenantID := identitymodel.NewTenantID("tenant-1")

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		actorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	membershipID := identitymodel.DeriveMembershipID(actorID, tenantID)

	mem, err := systemadapter.FindMembershipByID(ctx, sys, membershipID.String())
	if err != nil {
		t.Fatalf("FindMembershipByID: %v", err)
	}
	if mem.ActorID != actorID.String() {
		t.Errorf("ActorID = %q, want %q", mem.ActorID, actorID.String())
	}
	if mem.TenantID != tenantID.Get() {
		t.Errorf("TenantID = %q, want %q", mem.TenantID, tenantID.Get())
	}
	if len(mem.Roles) != 1 || mem.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin]", mem.Roles)
	}

	mems, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenantID.Get())
	if err != nil {
		t.Fatalf("FindMembershipsByTenant: %v", err)
	}
	if len(mems) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(mems))
	}

	mems, err = systemadapter.FindMembershipsByActor(ctx, sys, actorID.String())
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

	userStreamID := id.NewStreamID()
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "user@example.com", "Test User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
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

	byEmail, err := systemadapter.FindUserByEmail(ctx, sys, "user@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if byEmail.ID != userStreamID.String() {
		t.Errorf("FindByEmail ID = %q, want %q", byEmail.ID, userStreamID.String())
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeEmailCmd(
		userStreamID, "new@example.com",
	)); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	user, err = systemadapter.FindUserByID(ctx, sys, userStreamID.String())
	if err != nil {
		t.Fatalf("FindUserByID after email change: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "new@example.com")
	}
	if user.EmailVerified {
		t.Error("EmailVerified should be false after email change")
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewVerifyEmailCmd(
		userStreamID,
	)); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	user, err = systemadapter.FindUserByID(ctx, sys, userStreamID.String())
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

	userStreamID := id.NewStreamID()
	tenantID := identitymodel.NewTenantID("tenant-authz")
	actorID := identitymodel.NewActorID(identitymodel.ActorUser, userStreamID.String())

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "admin@example.com", "Admin User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		actorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	allowed, err := systemadapter.Enforce(ctx, sys, userStreamID.String(), tenantID.Get(), "manage")
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if !allowed {
		t.Error("admin should be allowed to manage")
	}

	plainStreamID := id.NewStreamID()
	plainActorID := identitymodel.NewActorID(identitymodel.ActorUser, plainStreamID.String())
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		plainStreamID, "plain@example.com", "Plain User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatalf("RegisterUser plain: %v", err)
	}
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		plainActorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleViewer},
	)); err != nil {
		t.Fatalf("AddMember viewer: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	allowed, err = systemadapter.Enforce(ctx, sys, plainStreamID.String(), tenantID.Get(), "manage")
	if err != nil {
		t.Fatalf("Enforce viewer manage: %v", err)
	}
	if allowed {
		t.Error("viewer should NOT be allowed to manage")
	}

	allowed, err = systemadapter.Enforce(ctx, sys, plainStreamID.String(), tenantID.Get(), "view")
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

	userStreamID := id.NewStreamID()
	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "audit@example.com", "Audit User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeEmailCmd(
		userStreamID, "changed@example.com",
	)); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}
	waitForProjections(t, sys, 5*time.Second)

	entries, err := systemadapter.AuditEntries(ctx, sys)
	if err != nil {
		t.Fatalf("AuditEntries: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 audit entries, got %d", len(entries))
	}

	forUser, err := systemadapter.AuditEntriesFor(ctx, sys, userStreamID.String())
	if err != nil {
		t.Fatalf("AuditEntriesFor: %v", err)
	}
	if len(forUser) < 2 {
		t.Fatalf("expected at least 2 entries for user, got %d", len(forUser))
	}

	foundRegister := false
	foundChangeEmail := false
	for _, e := range forUser {
		if e.AggregateID != userStreamID.String() {
			t.Errorf("AggregateID = %q, want %q", e.AggregateID, userStreamID.String())
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

	recent, err := systemadapter.RecentAuditEntries(ctx, sys, 1)
	if err != nil {
		t.Fatalf("RecentAuditEntries: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent entry, got %d", len(recent))
	}
}
