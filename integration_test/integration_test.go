package integration_test

import (
	"context"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/cqrs-htmx/v2/usermgmt"
)

func TestIntegration_UserIDExtraction_Bridge(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{BcryptCost: 4})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	uid := cqrshtmx.NewUserID()
	ctx := context.Background()
	reg, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:       usermgmt.NewUserID(uid.String()),
		Email:    "bridge@test.com",
		Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	userIDStr := user.ID.Get()
	parsedID, parseErr := cqrshtmx.ParseUserID(userIDStr)
	if parseErr != nil {
		t.Fatalf("ParseUserID(%q): %v", userIDStr, parseErr)
	}

	ctx = cqrshtmx.WithUserID(ctx, parsedID)
	extracted := cqrshtmx.UserIDFromContext(ctx)
	if extracted.String() != userIDStr {
		t.Errorf("expected %q, got %q", userIDStr, extracted.String())
	}
}

func TestIntegration_UserIDFromRequest_Bridge(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{BcryptCost: 4})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	uid := cqrshtmx.NewUserID()
	ctx := context.Background()
	reg, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:       usermgmt.NewUserID(uid.String()),
		Email:    "bridge2@test.com",
		Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	if user.ID.Get() != uid.String() {
		t.Errorf("expected %q, got %q", uid.String(), user.ID.Get())
	}
}
