package systemadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // register sqlite driver
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// memoryDeployment returns a DeploymentConfig using the in-memory driver,
// suitable for fast unit tests.
func memoryDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
		},
	}
}

// sqliteDeployment returns a DeploymentConfig using the SQLite in-memory driver.
// The DSN uses mode=memory&cache=shared so connections in the same process
// share the same database.
func sqliteDeployment(t *testing.T) system.DeploymentConfig {
	t.Helper()

	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {
				Driver:  "sqlite",
				DSN:     fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()),
				Pragmas: []string{"journal_mode=wal"},
			},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}
}

// setupTestSystem creates a system + projection layer for testing.
// The caller must defer pl.Stop() and sys.Close().
func setupTestSystem(t *testing.T, deployment system.DeploymentConfig,
) (*system.System, *systemadapter.ProjectionLayer) {
	t.Helper()

	ctx := context.Background()

	sys, err := system.New(ctx, systemadapter.DomainConfig(), deployment)
	if err != nil {
		t.Fatalf("system.New failed: %v", err)
	}

	pl, err := systemadapter.NewProjectionLayer(sys)
	if err != nil {
		_ = sys.Close()
		t.Fatalf("NewProjectionLayer failed: %v", err)
	}

	if err := pl.Start(ctx); err != nil {
		_ = sys.Close()
		t.Fatalf("ProjectionLayer.Start failed: %v", err)
	}

	return sys, pl
}

// TestProjectionLayer_CustomStores verifies that WithCheckpointStore and
// WithDeadLetterStore are honored: the layer builds, starts, drains a real
// command into the read model, and stops cleanly with injected stores.
func TestProjectionLayer_CustomStores(t *testing.T) {
	ctx := context.Background()

	sys, err := system.New(ctx, systemadapter.DomainConfig(), memoryDeployment())
	if err != nil {
		t.Fatalf("system.New failed: %v", err)
	}

	defer func() { _ = sys.Close() }()

	cpStore := memory.NewMemoryCheckpointStore()
	dlqStore := projectionhost.NewMemoryDeadLetterStore()

	pl, err := systemadapter.NewProjectionLayer(sys,
		systemadapter.WithCheckpointStore(cpStore),
		systemadapter.WithDeadLetterStore(dlqStore),
	)
	if err != nil {
		t.Fatalf("NewProjectionLayer with custom stores failed: %v", err)
	}

	defer func() { _ = pl.Stop() }()

	if err := pl.Start(ctx); err != nil {
		t.Fatalf("ProjectionLayer.Start failed: %v", err)
	}

	streamID := id.NewStreamID()
	cmd := identitymodel.NewRegisterUserCmd(streamID, "custom-stores@example.com", "Custom Stores", nil)

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		t.Fatalf("Dispatch RegisterUser failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	user, ok := pl.User.FindByEmail("custom-stores@example.com")
	if !ok {
		t.Fatal("custom-stores user not found in read model after drain")
	}

	if user.ID.String() != streamID.String() {
		t.Errorf("user ID = %q, want %q", user.ID.String(), streamID.String())
	}
}

func TestDomainConfig_RegisterUserEndToEnd(t *testing.T) {
	ctx := context.Background()
	sys, pl := setupTestSystem(t, memoryDeployment())

	defer func() { _ = sys.Close() }()
	defer func() { _ = pl.Stop() }()

	streamID := id.NewStreamID()
	cmd := identitymodel.NewRegisterUserCmd(
		streamID, "test@example.com", "Test User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		t.Fatalf("Dispatch RegisterUser failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	user, ok := pl.User.FindByID(streamID)
	if !ok {
		t.Fatal("user not found in read model after RegisterUser command")
	}

	if user.Email != "test@example.com" {
		t.Errorf("email = %q, want %q", user.Email, "test@example.com")
	}

	if user.DisplayName != "Test User" {
		t.Errorf("displayName = %q, want %q", user.DisplayName, "Test User")
	}
}

func TestEventTypeDecoder_All21EventTypesRegistered(t *testing.T) {
	decoder := systemadapter.EventTypeDecoder()

	expectedTypes := []string{
		string(identitymodel.EventUserRegistered),
		string(identitymodel.EventRolesUpdated),
		string(identitymodel.EventEmailChanged),
		string(identitymodel.EventDisplayNameChanged),
		string(identitymodel.EventUserDeleted),
		string(identitymodel.EventCredentialAdded),
		string(identitymodel.EventCredentialRemoved),
		string(identitymodel.EventEmailVerified),
		string(identitymodel.EventTOTPEnabled),
		string(identitymodel.EventTOTPDisabled),
		string(identitymodel.EventExternalAccountLinked),
		string(identitymodel.EventExternalAccountUnlinked),
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

	registered := make(map[string]bool)
	for _, et := range decoder.EventTypes() {
		registered[et] = true
	}

	for _, expected := range expectedTypes {
		if !registered[expected] {
			t.Errorf("event type %q not registered in TypeDecoder", expected)
		}
	}

	if len(decoder.EventTypes()) != len(expectedTypes) {
		t.Errorf("EventTypes() count = %d, want %d", len(decoder.EventTypes()), len(expectedTypes))
	}
}

func TestDomainConfig_TenantAndAuditLog(t *testing.T) {
	ctx := context.Background()
	sys, pl := setupTestSystem(t, memoryDeployment())

	defer func() { _ = sys.Close() }()
	defer func() { _ = pl.Stop() }()

	tenantStreamID := id.NewStreamID()

	tenantCmd := identitymodel.NewCreateTenantCmd(tenantStreamID, "acme", "Acme Corp")
	if err := sys.CommandDispatcher().Dispatch(ctx, tenantCmd); err != nil {
		t.Fatalf("Dispatch CreateTenant failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	tenant, ok := pl.Tenant.FindByID(tenantStreamID)
	if !ok {
		t.Fatal("tenant not found in read model")
	}

	if tenant.Name != "acme" {
		t.Errorf("tenant name = %q, want %q", tenant.Name, "acme")
	}

	// AuditLog only handles user events, so register a user to check it.
	userStreamID := id.NewStreamID()

	userCmd := identitymodel.NewRegisterUserCmd(userStreamID, "test@example.com", "Test", nil)
	if err := sys.CommandDispatcher().Dispatch(ctx, userCmd); err != nil {
		t.Fatalf("Dispatch RegisterUser failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	auditEntries := pl.AuditLog.Entries()
	if len(auditEntries) == 0 {
		t.Error("no audit log entries recorded")
	}
}

func TestDomainConfig_MembershipCommands(t *testing.T) {
	ctx := context.Background()
	sys, pl := setupTestSystem(t, memoryDeployment())

	defer func() { _ = sys.Close() }()
	defer func() { _ = pl.Stop() }()

	// Create prerequisites: a user and a tenant.
	userID := id.NewStreamID()
	tenantID := id.NewStreamID()

	userCmd := identitymodel.NewRegisterUserCmd(userID, "member@example.com", "Member", nil)
	if err := sys.CommandDispatcher().Dispatch(ctx, userCmd); err != nil {
		t.Fatalf("Dispatch RegisterUser failed: %v", err)
	}

	tenantCmd := identitymodel.NewCreateTenantCmd(tenantID, "corp", "Corp")
	if err := sys.CommandDispatcher().Dispatch(ctx, tenantCmd); err != nil {
		t.Fatalf("Dispatch CreateTenant failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	// Add the user as a member of the tenant with admin role.
	adminUserID, err := identitymodel.ParseUserID(userID.String())
	if err != nil {
		t.Fatalf("ParseUserID failed: %v", err)
	}

	actorID := identitymodel.ActorIDFromUser(adminUserID)
	memberCmd := identitymodel.NewAddMemberCmd(
		actorID, identitymodel.NewTenantID(tenantID.String()),
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)
	if err := sys.CommandDispatcher().Dispatch(ctx, memberCmd); err != nil {
		t.Fatalf("Dispatch AddMember failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	// Verify membership in the read model via aggregate ID (the command's StreamID).
	membership, ok := pl.Membership.FindByAggregateID(memberCmd.StreamID())
	if !ok {
		t.Fatal("membership not found in read model after AddMember command")
	}

	if membership.ActorID != actorID {
		t.Errorf("membership actorID = %v, want %v", membership.ActorID, actorID)
	}

	if !membership.HasRole(identitymodel.RoleAdmin) {
		t.Errorf("membership should have admin role, roles = %v", membership.Roles)
	}

	// Verify lookup by tenant.
	byTenant := pl.Membership.FindByTenant(tenantID.String())
	if len(byTenant) != 1 {
		t.Errorf("FindByTenant returned %d memberships, want 1", len(byTenant))
	}
}

func TestDomainConfig_BotCommands(t *testing.T) {
	ctx := context.Background()
	sys, pl := setupTestSystem(t, memoryDeployment())

	defer func() { _ = sys.Close() }()
	defer func() { _ = pl.Stop() }()

	// Register a user as the bot owner.
	ownerID := id.NewStreamID()
	userCmd := identitymodel.NewRegisterUserCmd(ownerID, "owner@example.com", "Owner", nil)
	if err := sys.CommandDispatcher().Dispatch(ctx, userCmd); err != nil {
		t.Fatalf("Dispatch RegisterUser failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	// Register a bot.
	botID := id.NewStreamID()
	tokenHash := []byte{0x01, 0x02, 0x03}
	scopes := []string{"read:users", "write:users"}

	ownerUserID, err := identitymodel.ParseUserID(ownerID.String())
	if err != nil {
		t.Fatalf("ParseUserID failed: %v", err)
	}

	botCmd := identitymodel.NewRegisterBotCmd(botID, "ci-bot", ownerUserID, tokenHash, scopes)
	if err := sys.CommandDispatcher().Dispatch(ctx, botCmd); err != nil {
		t.Fatalf("Dispatch RegisterBot failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	// Verify bot in the read model.
	bot, ok := pl.Bot.FindByID(botCmd.StreamID())
	if !ok {
		t.Fatal("bot not found in read model after RegisterBot command")
	}

	if bot.Name != "ci-bot" {
		t.Errorf("bot name = %q, want %q", bot.Name, "ci-bot")
	}

	if bot.OwnerID != ownerUserID {
		t.Errorf("bot ownerID = %v, want %v", bot.OwnerID, ownerID)
	}

	// Verify lookup by owner.
	byOwner := pl.Bot.FindByOwner(ownerUserID)
	if len(byOwner) != 1 {
		t.Errorf("FindByOwner returned %d bots, want 1", len(byOwner))
	}
}

func TestDomainConfig_CasbinProjection(t *testing.T) {
	ctx := context.Background()
	sys, pl := setupTestSystem(t, memoryDeployment())

	defer func() { _ = sys.Close() }()
	defer func() { _ = pl.Stop() }()

	// Register a user with admin role — CasbinProjection will add
	// role assignments to the enforcer.
	adminID := id.NewStreamID()
	adminCmd := identitymodel.NewRegisterUserCmd(
		adminID, "admin@example.com", "Admin",
		[]identitymodel.Role{identitymodel.RoleAdmin},
	)
	if err := sys.CommandDispatcher().Dispatch(ctx, adminCmd); err != nil {
		t.Fatalf("Dispatch RegisterUser(admin) failed: %v", err)
	}

	// Register a user with no roles — should be denied.
	plainID := id.NewStreamID()
	plainCmd := identitymodel.NewRegisterUserCmd(plainID, "plain@example.com", "Plain", nil)
	if err := sys.CommandDispatcher().Dispatch(ctx, plainCmd); err != nil {
		t.Fatalf("Dispatch RegisterUser(plain) failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	// Admin should be able to read resources in their own domain.
	// CasbinProjection adds roles with domain=subject (user's own ID).
	allowed, err := pl.Authz.Enforce(adminID.String(), adminID.String(), "resource", identitymodel.ActionRead)
	if err != nil {
		t.Fatalf("Enforce(admin) failed: %v", err)
	}

	if !allowed {
		t.Error("admin user should be allowed to read resources in their own domain")
	}

	// Plain user should be denied (no roles assigned).
	denied, err := pl.Authz.Enforce(plainID.String(), plainID.String(), "resource", identitymodel.ActionRead)
	if err != nil {
		t.Fatalf("Enforce(plain) failed: %v", err)
	}

	if denied {
		t.Error("plain user without roles should be denied")
	}
}

func TestDomainConfig_SQLiteDeployment(t *testing.T) {
	ctx := context.Background()
	sys, pl := setupTestSystem(t, sqliteDeployment(t))

	defer func() { _ = sys.Close() }()
	defer func() { _ = pl.Stop() }()

	// Full CQRS round-trip: dispatch RegisterUser, drain, query read model.
	streamID := id.NewStreamID()
	cmd := identitymodel.NewRegisterUserCmd(
		streamID, "sqlite@example.com", "SQLite User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		t.Fatalf("Dispatch RegisterUser failed: %v", err)
	}

	if err := pl.WaitForDrain(5 * time.Second); err != nil {
		t.Fatalf("WaitForDrain failed: %v", err)
	}

	user, ok := pl.User.FindByID(streamID)
	if !ok {
		t.Fatal("user not found in read model after RegisterUser (SQLite)")
	}

	if user.Email != "sqlite@example.com" {
		t.Errorf("email = %q, want %q", user.Email, "sqlite@example.com")
	}

	// Verify events persisted to the SQLite journal by loading the stream.
	events, err := sys.EventStore().Load(ctx, id.NewStreamRef(identitymodel.AggregateTypeUser, streamID))
	if err != nil {
		t.Fatalf("EventStore.Load failed: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("no events loaded from SQLite journal")
	}
}
