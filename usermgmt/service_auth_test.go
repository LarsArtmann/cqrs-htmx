package usermgmt

import (
	"testing"
	"time"
)

func TestService_Authenticate(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "user-1", "a@b.com")

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.ID != NewUserID("user-1") {
		t.Errorf("expected user ID user-1, got %s", user.ID)
	}
}

func TestService_Authenticate_ExpiredSession(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "u1", "exp@test.com")

	sessions, ok := svc.sessions.(*InMemorySessionStore)
	if !ok {
		t.Fatal("expected *InMemorySessionStore")
	}
	sessions.mu.Lock()
	for _, s := range sessions.sessions {
		s.ExpiresAt = time.Now().Add(-time.Hour)
	}
	sessions.mu.Unlock()

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	assertErrorIs(t, err, ErrSessionExpired, "ErrSessionExpired")
}

func TestService_Authenticate_UserDeleted(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "u1", "del@test.com")

	if err := svc.DeleteUser(ctx, NewUserID("u1"), "test cleanup"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	assertErrorIs(t, err, ErrUnauthorized, "ErrUnauthorized (session revoked)")
}
