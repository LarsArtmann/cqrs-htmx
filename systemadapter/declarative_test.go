package systemadapter_test

import (
	"context"
	"fmt"
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
			{Role: system.RoleProjections, Engines: []string{"primary"}},
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

// eventually retries fn until it returns nil or the timeout expires.
// This is the standard pattern for testing eventually-consistent projections.
func eventually(t *testing.T, timeout time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := fn(); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := fn(); err != nil {
		t.Fatal(err)
	}
}

// waitForHostLive waits for the projection host workers to reach WorkerLive.
func waitForHostLive(t *testing.T, sys *system.System, timeout time.Duration) {
	t.Helper()
	host := sys.ProjectionHost()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		states := host.Status()
		if len(states) > 0 {
			allLive := true
			for _, s := range states {
				if s.Status != projectionhost.WorkerLive && s.Status != projectionhost.WorkerStopped {
					allLive = false
					break
				}
			}
			if allLive {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("projection host workers did not reach Live state")
}

func TestDeclarative_TenantRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	tenantID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(
		tenantID, "Acme Corp", "Acme",
	)))
	waitForHostLive(t, sys, 5*time.Second)

	eventually(t, 5*time.Second, func() error {
		tenant, err := systemadapter.FindTenantByID(ctx, sys, tenantID.String())
		if err != nil {
			return err
		}
		if tenant.Name != "Acme Corp" {
			return fmt.Errorf("Name = %q", tenant.Name)
		}
		if tenant.DisplayName != "Acme" {
			return fmt.Errorf("DisplayName = %q", tenant.DisplayName)
		}
		if tenant.Suspended {
			return fmt.Errorf("should not be suspended")
		}
		return nil
	})

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewSuspendTenantCmd(tenantID, "test")))
	eventually(t, 5*time.Second, func() error {
		tenant, err := systemadapter.FindTenantByID(ctx, sys, tenantID.String())
		if err != nil {
			return err
		}
		if !tenant.Suspended {
			return fmt.Errorf("should be suspended")
		}
		return nil
	})

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewReactivateTenantCmd(tenantID)))
	eventually(t, 5*time.Second, func() error {
		tenant, err := systemadapter.FindTenantByID(ctx, sys, tenantID.String())
		if err != nil {
			return err
		}
		if tenant.Suspended {
			return fmt.Errorf("should not be suspended")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		byName, err := systemadapter.FindTenantByName(ctx, sys, "Acme Corp")
		if err != nil {
			return err
		}
		if byName.ID != tenantID.String() {
			return fmt.Errorf("ID = %q", byName.ID)
		}
		return nil
	})
}

func TestDeclarative_BotRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	ownerID := identitymodel.GenerateUserID()
	botID := id.NewStreamID()
	tokenHash := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterBotCmd(
		botID, "MyBot", ownerID, tokenHash, []string{"read", "write"},
	)))
	waitForHostLive(t, sys, 5*time.Second)

	eventually(t, 5*time.Second, func() error {
		bot, err := systemadapter.FindBotByID(ctx, sys, botID.String())
		if err != nil {
			return err
		}
		if bot.Name != "MyBot" {
			return fmt.Errorf("Name = %q", bot.Name)
		}
		if bot.OwnerID != ownerID.String() {
			return fmt.Errorf("OwnerID = %q", bot.OwnerID)
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		_, err := systemadapter.FindBotByTokenHash(ctx, sys, "deadbeef")
		return err
	})

	eventually(t, 5*time.Second, func() error {
		bots, err := systemadapter.FindBotsByOwner(ctx, sys, ownerID.String())
		if err != nil {
			return err
		}
		if len(bots) != 1 {
			return fmt.Errorf("got %d bots", len(bots))
		}
		return nil
	})
}

func TestDeclarative_MembershipRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	actorID := identitymodel.NewActorID(identitymodel.ActorUser, userStreamID.String())
	tenantID := identitymodel.NewTenantID("tenant-1")

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		actorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)))
	waitForHostLive(t, sys, 5*time.Second)

	membershipID := identitymodel.DeriveMembershipID(actorID, tenantID)

	eventually(t, 5*time.Second, func() error {
		mem, err := systemadapter.FindMembershipByID(ctx, sys, membershipID.String())
		if err != nil {
			return err
		}
		if mem.ActorID != actorID.String() {
			return fmt.Errorf("ActorID = %q", mem.ActorID)
		}
		if mem.TenantID != tenantID.Get() {
			return fmt.Errorf("TenantID = %q", mem.TenantID)
		}
		if len(mem.Roles) != 1 || mem.Roles[0] != "admin" {
			return fmt.Errorf("Roles = %v", mem.Roles)
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		mems, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenantID.Get())
		if err != nil {
			return err
		}
		if len(mems) != 1 {
			return fmt.Errorf("got %d memberships by tenant", len(mems))
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		mems, err := systemadapter.FindMembershipsByActor(ctx, sys, actorID.String())
		if err != nil {
			return err
		}
		if len(mems) != 1 {
			return fmt.Errorf("got %d memberships by actor", len(mems))
		}
		return nil
	})
}

func TestDeclarative_UserRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "user@example.com", "Test User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	waitForHostLive(t, sys, 5*time.Second)

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if user.Email != "user@example.com" {
			return fmt.Errorf("Email = %q", user.Email)
		}
		if user.DisplayName != "Test User" {
			return fmt.Errorf("DisplayName = %q", user.DisplayName)
		}
		if user.EmailVerified {
			return fmt.Errorf("EmailVerified should be false")
		}
		if user.CreatedAt.IsZero() {
			return fmt.Errorf("CreatedAt is zero")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		_, err := systemadapter.FindUserByEmail(ctx, sys, "user@example.com")
		return err
	})

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeEmailCmd(
		userStreamID, "new@example.com",
	)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if user.Email != "new@example.com" {
			return fmt.Errorf("Email = %q", user.Email)
		}
		if user.EmailVerified {
			return fmt.Errorf("EmailVerified should be false after change")
		}
		return nil
	})

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewVerifyEmailCmd(userStreamID)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if !user.EmailVerified {
			return fmt.Errorf("EmailVerified should be true")
		}
		return nil
	})
}

func TestDeclarative_AuthzEnforce(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	tenantID := identitymodel.NewTenantID("tenant-authz")
	actorID := identitymodel.NewActorID(identitymodel.ActorUser, userStreamID.String())

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "admin@example.com", "Admin User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		actorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)))
	waitForHostLive(t, sys, 5*time.Second)

	eventually(t, 5*time.Second, func() error {
		allowed, err := systemadapter.Enforce(ctx, sys, userStreamID.String(), tenantID.Get(), "manage")
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("admin should be allowed to manage")
		}
		return nil
	})

	plainStreamID := id.NewStreamID()
	plainActorID := identitymodel.NewActorID(identitymodel.ActorUser, plainStreamID.String())
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		plainStreamID, "plain@example.com", "Plain User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		plainActorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleViewer},
	)))

	eventually(t, 5*time.Second, func() error {
		allowed, err := systemadapter.Enforce(ctx, sys, plainStreamID.String(), tenantID.Get(), "manage")
		if err != nil {
			return err
		}
		if allowed {
			return fmt.Errorf("viewer should NOT be allowed to manage")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		allowed, err := systemadapter.Enforce(ctx, sys, plainStreamID.String(), tenantID.Get(), "view")
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("viewer should be allowed to view")
		}
		return nil
	})
}

func TestDeclarative_AuditLog(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "audit@example.com", "Audit User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeEmailCmd(
		userStreamID, "changed@example.com",
	)))
	waitForHostLive(t, sys, 5*time.Second)

	eventually(t, 5*time.Second, func() error {
		entries, err := systemadapter.AuditEntries(ctx, sys)
		if err != nil {
			return err
		}
		if len(entries) < 2 {
			return fmt.Errorf("got %d entries, want >= 2", len(entries))
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		forUser, err := systemadapter.AuditEntriesFor(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if len(forUser) < 2 {
			return fmt.Errorf("got %d entries for user", len(forUser))
		}
		foundRegister, foundChangeEmail := false, false
		for _, e := range forUser {
			if e.EventType == "UserRegistered" {
				foundRegister = true
			}
			if e.EventType == "EmailChanged" {
				foundChangeEmail = true
			}
			if e.OccurredAt.IsZero() {
				return fmt.Errorf("OccurredAt is zero")
			}
		}
		if !foundRegister {
			return fmt.Errorf("UserRegistered entry missing")
		}
		if !foundChangeEmail {
			return fmt.Errorf("EmailChanged entry missing")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		recent, err := systemadapter.RecentAuditEntries(ctx, sys, 1)
		if err != nil {
			return err
		}
		if len(recent) != 1 {
			return fmt.Errorf("got %d recent entries", len(recent))
		}
		return nil
	})
}

func TestDeclarative_AllProjectionNames(t *testing.T) {
	decls := systemadapter.DeclarativeProjections()
	if len(decls) < 10 {
		t.Fatalf("expected at least 10 projection declarations, got %d", len(decls))
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
