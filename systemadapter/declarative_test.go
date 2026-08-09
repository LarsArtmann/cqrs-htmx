package systemadapter_test

import (
	"context"
	"errors"
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

	waitForProjectionsLive(t, sys)

	return sys
}

// waitForProjectionsLive blocks until all projection host workers reach
// WorkerLive status. This is critical for race-free testing: the GoChannel bus
// uses Persistent=false, meaning events published before the subscriber
// registers are dropped. Once workers are live,
// BlockPublishUntilSubscriberAck makes Dispatch synchronous — the event is
// fully processed by all projections before Dispatch returns.
func waitForProjectionsLive(t *testing.T, sys *system.System) {
	t.Helper()
	host := sys.ProjectionHost()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		states := host.Status()
		if len(states) > 0 {
			allLive := true
			for _, s := range states {
				if s.Status != projectionhost.WorkerLive {
					allLive = false
					break
				}
			}
			if allLive {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("projection workers did not reach WorkerLive within 10s; states: %+v", host.Status())
}

// eventually retries fn until it returns nil or the timeout expires.
// Once projection workers are live, Dispatch is synchronous so assertions
// typically pass on the first try. This is a safety net for edge cases.
func eventually(t *testing.T, timeout time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := fn(); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := fn(); err != nil {
		t.Fatal(err)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// Tenant
// ---------------------------------------------------------------------------

func TestDeclarative_TenantRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	tenantID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(
		tenantID, "Acme Corp", "Acme",
	)))

	eventually(t, 5*time.Second, func() error {
		tenant, err := systemadapter.FindTenantByID(ctx, sys, tenantID.String())
		if err != nil {
			return err
		}
		if tenant.Name != "Acme Corp" {
			return errors.New("Name mismatch")
		}
		if tenant.DisplayName != "Acme" {
			return errors.New("DisplayName mismatch")
		}
		if tenant.Suspended {
			return errors.New("should not be suspended")
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
			return errors.New("should be suspended")
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
			return errors.New("should not be suspended")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		byName, err := systemadapter.FindTenantByName(ctx, sys, "Acme Corp")
		if err != nil {
			return err
		}
		if byName.ID != tenantID.String() {
			return errors.New("ID mismatch")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		all, err := systemadapter.AllTenants(ctx, sys)
		if err != nil {
			return err
		}
		if len(all) != 1 {
			return errors.New("expected 1 tenant")
		}
		return nil
	})
}

func TestDeclarative_TenantDelete(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	tenantID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(
		tenantID, "ToDelete", "TD",
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewDeleteTenantCmd(tenantID, "cleanup")))

	eventually(t, 5*time.Second, func() error {
		_, err := systemadapter.FindTenantByID(ctx, sys, tenantID.String())
		if err == nil {
			return errors.New("tenant should be deleted")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		all, err := systemadapter.AllTenants(ctx, sys)
		if err != nil {
			return err
		}
		if len(all) != 0 {
			return errors.New("expected 0 tenants after delete")
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Bot
// ---------------------------------------------------------------------------

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

	eventually(t, 5*time.Second, func() error {
		bot, err := systemadapter.FindBotByID(ctx, sys, botID.String())
		if err != nil {
			return err
		}
		if bot.Name != "MyBot" {
			return errors.New("Name mismatch")
		}
		if bot.OwnerID != ownerID.String() {
			return errors.New("OwnerID mismatch")
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
			return errors.New("expected 1 bot")
		}
		return nil
	})
}

func TestDeclarative_BotDelete(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	ownerID := identitymodel.GenerateUserID()
	botID := id.NewStreamID()
	tokenHash := []byte{0x01, 0x02}

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterBotCmd(
		botID, "DeleteBot", ownerID, tokenHash, []string{"read"},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewDeleteBotCmd(botID, "retired")))

	eventually(t, 5*time.Second, func() error {
		_, err := systemadapter.FindBotByID(ctx, sys, botID.String())
		if err == nil {
			return errors.New("bot should be deleted")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		bots, err := systemadapter.FindBotsByOwner(ctx, sys, ownerID.String())
		if err != nil {
			return err
		}
		if len(bots) != 0 {
			return errors.New("expected 0 bots after delete")
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Membership
// ---------------------------------------------------------------------------

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

	membershipID := identitymodel.DeriveMembershipID(actorID, tenantID)

	eventually(t, 5*time.Second, func() error {
		mem, err := systemadapter.FindMembershipByID(ctx, sys, membershipID.String())
		if err != nil {
			return err
		}
		if mem.ActorID != actorID.String() {
			return errors.New("ActorID mismatch")
		}
		if mem.TenantID != tenantID.Get() {
			return errors.New("TenantID mismatch")
		}
		if len(mem.Roles) != 1 || mem.Roles[0] != "admin" {
			return errors.New("Roles mismatch")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		mems, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenantID.Get())
		if err != nil {
			return err
		}
		if len(mems) != 1 {
			return errors.New("expected 1 membership by tenant")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		mems, err := systemadapter.FindMembershipsByActor(ctx, sys, actorID.String())
		if err != nil {
			return err
		}
		if len(mems) != 1 {
			return errors.New("expected 1 membership by actor")
		}
		return nil
	})
}

func TestDeclarative_MembershipRolesUpdate(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	actorID := identitymodel.NewActorID(identitymodel.ActorUser, userStreamID.String())
	tenantID := identitymodel.NewTenantID("tenant-roles")

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		actorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleViewer},
	)))

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewUpdateMemberRolesCmd(
		actorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleAdmin, identitymodel.RoleUser},
	)))

	membershipID := identitymodel.DeriveMembershipID(actorID, tenantID)

	eventually(t, 5*time.Second, func() error {
		mem, err := systemadapter.FindMembershipByID(ctx, sys, membershipID.String())
		if err != nil {
			return err
		}
		if len(mem.Roles) != 2 {
			return errors.New("expected 2 roles")
		}
		foundAdmin, foundUser := false, false
		for _, r := range mem.Roles {
			if r == "admin" {
				foundAdmin = true
			}
			if r == "user" {
				foundUser = true
			}
		}
		if !foundAdmin || !foundUser {
			return errors.New("missing expected roles")
		}
		return nil
	})
}

func TestDeclarative_MembershipRemove(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	actorID := identitymodel.NewActorID(identitymodel.ActorUser, userStreamID.String())
	tenantID := identitymodel.NewTenantID("tenant-remove")

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
		actorID, tenantID,
		[]identitymodel.Role{identitymodel.RoleUser},
	)))

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRemoveMemberCmd(actorID, tenantID)))

	membershipID := identitymodel.DeriveMembershipID(actorID, tenantID)

	eventually(t, 5*time.Second, func() error {
		_, err := systemadapter.FindMembershipByID(ctx, sys, membershipID.String())
		if err == nil {
			return errors.New("membership should be removed")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		mems, err := systemadapter.FindMembershipsByTenant(ctx, sys, tenantID.Get())
		if err != nil {
			return err
		}
		if len(mems) != 0 {
			return errors.New("expected 0 memberships after remove")
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

func TestDeclarative_UserRoundTrip(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "user@example.com", "Test User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if user.Email != "user@example.com" {
			return errors.New("Email mismatch")
		}
		if user.DisplayName != "Test User" {
			return errors.New("DisplayName mismatch")
		}
		if user.EmailVerified {
			return errors.New("EmailVerified should be false")
		}
		if user.CreatedAt.IsZero() {
			return errors.New("CreatedAt is zero")
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
			return errors.New("Email mismatch after change")
		}
		if user.EmailVerified {
			return errors.New("EmailVerified should be false after change")
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
			return errors.New("EmailVerified should be true")
		}
		return nil
	})
}

func TestDeclarative_UserDisplayNameChange(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "display@example.com", "Original",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeDisplayNameCmd(
		userStreamID, "Updated Name",
	)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if user.DisplayName != "Updated Name" {
			return errors.New("DisplayName mismatch")
		}
		return nil
	})
}

func TestDeclarative_UserCredentials(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "cred@example.com", "Cred User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))

	credID := []byte{0xAA, 0xBB, 0xCC}
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddCredentialCmd(
		userStreamID, identitymodel.WebAuthnCredential{
			CredentialCore: identitymodel.CredentialCore{
				ID:              credID,
				PublicKey:       []byte{0x01, 0x02, 0x03},
				AttestationType: "none",
				Transports:      []string{"internal"},
				Name:            "My Passkey",
			},
		},
	)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if len(user.Credentials) != 1 {
			return errors.New("expected 1 credential")
		}
		if user.Credentials[0].AttestationType != "none" {
			return errors.New("AttestationType mismatch")
		}
		if user.Credentials[0].Name != "My Passkey" {
			return errors.New("Name mismatch")
		}
		return nil
	})

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRemoveCredentialCmd(userStreamID, credID)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if len(user.Credentials) != 0 {
			return errors.New("expected 0 credentials after remove")
		}
		return nil
	})
}

func TestDeclarative_UserTOTP(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "totp@example.com", "TOTP User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewEnableTOTPCmd(
		userStreamID, []byte("secret-key"),
	)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if !user.TOTPEnabled {
			return errors.New("TOTP should be enabled")
		}
		return nil
	})

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewDisableTOTPCmd(userStreamID)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if user.TOTPEnabled {
			return errors.New("TOTP should be disabled")
		}
		return nil
	})
}

func TestDeclarative_UserExternalAccounts(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "ext@example.com", "Ext User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewLinkExternalAccountCmd(
		userStreamID, "github", "gh-123", "ext@github.com", "Ext User",
	)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if len(user.ExternalAccounts) != 1 {
			return errors.New("expected 1 external account")
		}
		if user.ExternalAccounts[0].Provider != "github" {
			return errors.New("Provider mismatch")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		byExt, err := systemadapter.FindUserByExternalAccount(ctx, sys, "github", "gh-123")
		if err != nil {
			return err
		}
		if byExt.ID != userStreamID.String() {
			return errors.New("user ID mismatch from external account lookup")
		}
		return nil
	})

	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewUnlinkExternalAccountCmd(
		userStreamID, "github", "gh-123",
	)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if len(user.ExternalAccounts) != 0 {
			return errors.New("expected 0 external accounts after unlink")
		}
		return nil
	})
}

func TestDeclarative_UserDelete(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "delete@example.com", "Delete Me",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewDeleteUserCmd(userStreamID, "test")))

	eventually(t, 5*time.Second, func() error {
		_, err := systemadapter.FindUserByID(ctx, sys, userStreamID.String())
		if err == nil {
			return errors.New("user should be deleted")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		all, err := systemadapter.AllUsers(ctx, sys)
		if err != nil {
			return err
		}
		if len(all) != 0 {
			return errors.New("expected 0 users after delete")
		}
		return nil
	})
}

func TestDeclarative_AllUsers(t *testing.T) {
	ctx := context.Background()
	sys := setupDeclarativeSystem(t)
	defer func() { _ = sys.Close() }()

	for i := 0; i < 3; i++ {
		uid := id.NewStreamID()
		must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
			uid, "user%d@example.com", "User",
			[]identitymodel.Role{identitymodel.RoleUser},
		)))
	}

	eventually(t, 5*time.Second, func() error {
		all, err := systemadapter.AllUsers(ctx, sys)
		if err != nil {
			return err
		}
		if len(all) != 3 {
			return errors.New("expected 3 users")
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Authz
// ---------------------------------------------------------------------------

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

	eventually(t, 5*time.Second, func() error {
		allowed, err := systemadapter.Enforce(ctx, sys, userStreamID.String(), tenantID.Get(), "manage")
		if err != nil {
			return err
		}
		if !allowed {
			return errors.New("admin should be allowed to manage")
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
			return errors.New("viewer should NOT be allowed to manage")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		allowed, err := systemadapter.Enforce(ctx, sys, plainStreamID.String(), tenantID.Get(), "view")
		if err != nil {
			return err
		}
		if !allowed {
			return errors.New("viewer should be allowed to view")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		entries, err := systemadapter.FindPolicies(ctx, sys, userStreamID.String(), tenantID.Get())
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return errors.New("expected at least 1 policy entry")
		}
		return nil
	})

	userPolID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userPolID, "pol@example.com", "Pol User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	eventually(t, 5*time.Second, func() error {
		pol, err := systemadapter.FindPolicyByStreamID(ctx, sys, userPolID.String())
		if err != nil {
			return err
		}
		if pol.Subject != userPolID.String() {
			return errors.New("policy subject mismatch")
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// AuditLog
// ---------------------------------------------------------------------------

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

	eventually(t, 5*time.Second, func() error {
		entries, err := systemadapter.AuditEntries(ctx, sys)
		if err != nil {
			return err
		}
		if len(entries) < 2 {
			return errors.New("expected >= 2 audit entries")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		forUser, err := systemadapter.AuditEntriesFor(ctx, sys, userStreamID.String())
		if err != nil {
			return err
		}
		if len(forUser) < 2 {
			return errors.New("expected >= 2 entries for user")
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
				return errors.New("OccurredAt is zero")
			}
		}
		if !foundRegister {
			return errors.New("UserRegistered entry missing")
		}
		if !foundChangeEmail {
			return errors.New("EmailChanged entry missing")
		}
		return nil
	})

	eventually(t, 5*time.Second, func() error {
		recent, err := systemadapter.RecentAuditEntries(ctx, sys, 1)
		if err != nil {
			return err
		}
		if len(recent) != 1 {
			return errors.New("expected 1 recent entry")
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Sanity
// ---------------------------------------------------------------------------

func TestDeclarative_AllProjectionNames(t *testing.T) {
	decls := systemadapter.DeclarativeProjections()
	if len(decls) < 10 {
		t.Fatalf("expected at least 10 projection declarations, got %d", len(decls))
	}
}
