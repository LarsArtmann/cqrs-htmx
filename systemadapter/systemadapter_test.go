package systemadapter_test

import (
	"context"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestDomainConfig_RegisterUserEndToEnd(t *testing.T) {
	ctx := context.Background()

	domain := systemadapter.DomainConfig()
	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New failed: %v", err)
	}
	defer sys.Close()

	pl, err := systemadapter.NewProjectionLayer(sys)
	if err != nil {
		t.Fatalf("NewProjectionLayer failed: %v", err)
	}
	defer pl.Stop()

	if err := pl.Start(ctx); err != nil {
		t.Fatalf("ProjectionLayer.Start failed: %v", err)
	}

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

	domain := systemadapter.DomainConfig()
	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New failed: %v", err)
	}
	defer sys.Close()

	pl, err := systemadapter.NewProjectionLayer(sys)
	if err != nil {
		t.Fatalf("NewProjectionLayer failed: %v", err)
	}
	defer pl.Stop()

	if err := pl.Start(ctx); err != nil {
		t.Fatalf("ProjectionLayer.Start failed: %v", err)
	}

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

	// AuditLog only handles user events, so register a user to check it
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
