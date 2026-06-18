package integration_test

import (
	"context"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v2"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
)

// registerTestUser creates a fresh user with a unique email and the given
// cqrshtmx UserID. Returns the registration response.
func registerTestUser(
	t *testing.T,
	svc *usermgmt.Service,
	uid cqrshtmx.UserID,
	email string,
) *usermgmt.RegisterResponse {
	t.Helper()
	ctx := context.Background()
	reg, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    usermgmt.NewUserID(uid.String()),
		Email: email,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return reg
}

func TestIntegration_UserIDExtraction_Bridge(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	ctx := context.Background()
	uid := cqrshtmx.NewUserID()
	reg := registerTestUser(t, svc, uid, "bridge@test.com")

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
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	ctx := context.Background()
	uid := cqrshtmx.NewUserID()
	reg := registerTestUser(t, svc, uid, "bridge2@test.com")

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	if user.ID.Get() != uid.String() {
		t.Errorf("expected %q, got %q", uid.String(), user.ID.Get())
	}
}
