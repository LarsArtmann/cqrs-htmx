package usermgmt

import (
	"context"
	"errors"
	"testing"
)

func TestDeleteUser_RevokesSessions(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	reg := registerTestUser(t, svc, "sr1", "sr1@test.com")

	// Verify session is valid
	if _, err := svc.Authenticate(context.Background(), reg.Session.Token); err != nil {
		t.Fatalf("pre-delete Authenticate: %v", err)
	}

	// Delete user — should revoke sessions
	if err := svc.DeleteUser(context.Background(), reg.User.ID, "test"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	// Verify old session is no longer valid
	_, err := svc.Authenticate(context.Background(), reg.Session.Token)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized after user deletion, got %v", err)
	}
}

func TestDeleteUser_SessionDeleteFailure_LoggingOnly(t *testing.T) {
	svc := newTestServiceWithConfig(t, ServiceConfig{
		SessionStore: failingDeleteByUserIDSessionStore("store unavailable"),
	})
	reg := registerTestUser(t, svc, "sr2", "sr2@test.com")

	// Should still succeed even if session deletion fails
	if err := svc.DeleteUser(context.Background(), reg.User.ID, "test"); err != nil {
		t.Fatalf("DeleteUser should succeed despite session store failure: %v", err)
	}
}
