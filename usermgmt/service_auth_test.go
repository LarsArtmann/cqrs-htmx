package usermgmt

import (
	"context"
	"testing"
	"time"
)

func TestService_Authenticate(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.ID != NewUserID("user-1") {
		t.Errorf("expected user ID user-1, got %s", user.ID)
	}
}

func TestService_Authenticate_ExpiredSession(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "u1", "exp@test.com", "secret12")

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
	svc, ctx, reg := newTestServiceWithUser(t, "u1", "del@test.com", "secret12")

	svc.users.Delete(context.Background(), NewUserID("u1")) //nolint:errcheck // test cleanup

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")
}
