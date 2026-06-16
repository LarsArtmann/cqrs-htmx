package usermgmt

import (
	"context"
	"errors"
	"testing"
)

func TestUpdateRoles_RevokesSessions(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	reg := registerTestUser(t, svc, "sr1", "sr1@test.com")

	// Verify session is valid
	if _, err := svc.Authenticate(context.Background(), reg.Session.Token); err != nil {
		t.Fatalf("pre-update Authenticate: %v", err)
	}

	// Update roles - should revoke sessions
	if err := svc.UpdateRoles(
		context.Background(), reg.User.ID, []Role{RoleAdmin}, "test",
	); err != nil {
		t.Fatalf("UpdateRoles: %v", err)
	}

	// Verify old session is no longer valid
	_, err := svc.Authenticate(context.Background(), reg.Session.Token)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized after role update, got %v", err)
	}
}

func TestUpdateRoles_SessionDeleteFailure(t *testing.T) {
	svc := newTestServiceWithConfig(t, ServiceConfig{
		SessionStore: &mockSessionStore{
			DeleteByUserIDFn: func(context.Context, UserID) error {
				return errors.New("store unavailable")
			},
		},
	})
	reg := registerTestUser(t, svc, "sr2", "sr2@test.com")

	// Should still succeed even if session deletion fails
	if err := svc.UpdateRoles(
		context.Background(), reg.User.ID, []Role{RoleUser}, "test",
	); err != nil {
		t.Fatalf("UpdateRoles should succeed even when session deletion fails: %v", err)
	}
}
