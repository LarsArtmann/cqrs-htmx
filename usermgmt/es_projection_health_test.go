package usermgmt_test

import (
	"context"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func TestDefaultEventCatalog_HasAllEvents(t *testing.T) {
	catalog := usermgmt.DefaultEventCatalog()
	events := catalog.Events()

	if len(events) < 21 {
		t.Errorf("catalog has %d events, want at least 21", len(events))
	}

	seen := make(map[string]bool)
	for _, e := range events {
		seen[e.Type] = true
	}

	for _, want := range []string{
		"UserRegistered", "RolesUpdated", "EmailChanged", "DisplayNameChanged",
		"UserDeleted", "CredentialAdded", "CredentialRemoved", "EmailVerified",
		"TOTPEnabled", "TOTPDisabled", "ExternalAccountLinked", "ExternalAccountUnlinked",
		"MemberAdded", "MemberRolesChanged", "MemberRemoved",
		"TenantCreated", "TenantSuspended", "TenantReactivated", "TenantDeleted",
		"BotRegistered", "BotDeleted",
	} {
		if !seen[want] {
			t.Errorf("event %q not found in catalog", want)
		}
	}
}

func TestDefaultEventCatalog_JSONSerialization(t *testing.T) {
	catalog := usermgmt.DefaultEventCatalog()
	data, err := catalog.JSON()
	if err != nil {
		t.Fatalf("JSON() returned error: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("JSON output is empty")
	}

	t.Logf("catalog JSON is %d bytes", len(data))
}

func TestDefaultEventCatalog_SchemaVersionIsCurrent(t *testing.T) {
	catalog := usermgmt.DefaultEventCatalog()
	events := catalog.Events()

	for _, e := range events {
		if e.SchemaVersion != 2 {
			t.Errorf("event %q has schema_version %d, want 2", e.Type, e.SchemaVersion)
		}
	}
}

func TestService_ProjectionStatuses(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	defer svc.Close()

	statuses := svc.ProjectionStatuses()
	if len(statuses) == 0 {
		t.Error("ProjectionStatuses() returned empty slice")
	}

	for _, s := range statuses {
		if s.Name == "" {
			t.Error("projection status has empty name")
		}
		if s.Status == "" {
			t.Errorf("projection %q has empty status", s.Name)
		}
	}
}

func TestService_SatisfiesProjectionStatusProvider(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	defer svc.Close()

	var provider cqrshtmx.ProjectionStatusProvider = svc
	statuses := provider.ProjectionStatuses()

	if len(statuses) == 0 {
		t.Error("ProjectionStatusProvider interface returned empty")
	}
}

func TestService_EventCatalog(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	defer svc.Close()

	catalog := svc.EventCatalog()
	if catalog == nil {
		t.Fatal("EventCatalog() returned nil")
	}

	events := catalog.Events()
	if len(events) < 21 {
		t.Errorf("catalog has %d events, want at least 21", len(events))
	}
}

func TestService_RebuildProjection(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	defer svc.Close()

	ctx := context.Background()

	// Register a user so events exist in the store.
	resp, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    usermgmt.SyntheticUserID("test-user"),
		Email: "test@example.com",
	})
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	// Rebuild the user-read-model projection.
	err = svc.RebuildProjection(ctx, "user-read-model")
	if err != nil {
		t.Fatalf("RebuildProjection error: %v", err)
	}

	// After rebuild, the read model should still contain the user.
	user, err := svc.GetUser(ctx, resp.User.ID)
	if err != nil {
		t.Fatalf("GetUser error after rebuild: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("user email after rebuild = %q, want %q", user.Email, "test@example.com")
	}
}

func TestService_RebuildProjection_UnknownProjection(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	defer svc.Close()

	err = svc.RebuildProjection(context.Background(), "nonexistent-projection")
	if err == nil {
		t.Error("RebuildProjection with unknown name should return error")
	}
}
