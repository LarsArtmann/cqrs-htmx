package usermgmt

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSession_Valid_Expired(t *testing.T) {
	s, err := NewSession(NewUserID("u1"), time.Hour)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	s.ExpiresAt = time.Now().Add(-time.Hour)
	if s.Valid(s.Token) {
		t.Error("expected expired session to be invalid")
	}
}

func TestAuthz_EnforceEx_Denied(t *testing.T) {
	a := newTestAuthz(t)
	result, err := a.EnforceEx("nobody", "dom", "secret", ActionRead)
	if err != nil {
		t.Fatalf("EnforceEx: %v", err)
	}
	if result.Allowed {
		t.Error("expected denied")
	}
}

func TestAuthz_Authorize_Allowed(t *testing.T) {
	a := newTestAuthz(t)
	_ = a.AddGroupPolicy(GroupPolicy{Subject: "admin1", Role: RoleAdmin, Domain: "d1"})

	if err := a.Authorize("admin1", "d1", "anything", ActionAll); err != nil {
		t.Errorf("expected allowed, got: %v", err)
	}
}

func TestAuthz_Apply_RemoveAndAddPolicies(t *testing.T) {
	a := newTestAuthz(t)
	_ = a.AddPolicy(Policy{"*", "*", "res.get", ActionRead, EffectAllow})

	err := a.Apply(PolicyUpdate{
		RemovePolicies: []Policy{{"*", "*", "res.get", ActionRead, EffectAllow}},
		AddPolicies:    []Policy{{"*", "*", "res.put", ActionExecute, EffectAllow}},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	ok, _ := a.Enforce("anyone", "dom", "res.get", ActionRead)
	if ok {
		t.Error("expected removed policy to deny access")
	}

	ok, _ = a.Enforce("anyone", "dom", "res.put", ActionExecute)
	if !ok {
		t.Error("expected added policy to allow access")
	}
}

func TestNewAuthz_WithGroups(t *testing.T) {
	_, err := NewAuthz(EnforcerConfig{
		Groups: []GroupPolicy{
			{Subject: "p1", Role: RoleOwner, Domain: "g1"},
		},
	})
	if err != nil {
		t.Fatalf("NewAuthz with groups: %v", err)
	}
}

func TestNewAuthz_InvalidModel(t *testing.T) {
	_, err := NewAuthz(EnforcerConfig{ModelString: "not a valid model"})
	if err == nil {
		t.Error("expected error for invalid model")
	}
}

func TestNewAuthz_EmptyModelString(t *testing.T) {
	a, err := NewAuthz(EnforcerConfig{ModelString: ""})
	if err != nil {
		t.Fatalf("NewAuthz with empty model string: %v", err)
	}
	if a == nil {
		t.Error("expected non-nil Authz with default model fallback")
	}
}

func TestAccountLockout_RecordFailure_ResetsExpiredLockout(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: 50 * time.Millisecond})

	l.RecordFailure("a@b.com")
	l.RecordFailure("a@b.com")
	if !l.IsLocked("a@b.com") {
		t.Fatal("expected locked")
	}

	time.Sleep(100 * time.Millisecond)

	if l.IsLocked("a@b.com") {
		t.Error("expected lockout to have expired")
	}

	if l.RecordFailure("a@b.com") {
		t.Error("expected not locked on first failure after expiry reset")
	}
}

func TestService_Authenticate_InvalidToken(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	_, err := svc.Authenticate(context.Background(), "nonexistent-token")
	assertErrorIs(t, err, ErrUnauthorized, "ErrUnauthorized")
}

func TestService_Register_DisplayNameTooLong(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	longName := strings.Repeat("x", 101)
	_, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "long@test.com", Password: "secret12",
		DisplayName: longName,
	})
	assertErrorIs(t, err, ErrValidation, "ErrValidation for long display name")
}

func TestStore_Save_EmailTakenByOtherUser(t *testing.T) {
	store := NewInMemoryUserStore()
	u1 := NewUser(NewUserID("u1"), "a@b.com", "One")
	_ = store.Create(u1)
	u2 := NewUser(NewUserID("u2"), "c@d.com", "Two")
	_ = store.Create(u2)

	u2.Email = "a@b.com"
	if err := store.Save(u2); !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestHandlers_Register_EmptyBody(t *testing.T) {
	_, mux := setupMux(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty registration, got %d", w.Code)
	}
}

func TestHandlers_Me_NilUser(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: false})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated /me, got %d", w.Code)
	}
}

func TestHandlers_Logout_DeletedSession(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: false})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	reg := registerTestUser(t, svc, "u1", "lo@test.com", "secret12")

	_ = svc.sessions.Delete(reg.Session.Token)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent logout, got %d", w.Code)
	}
}

func TestNewService_WithLogger(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		Lockout:    NewAccountLockout(),
	})
	if err != nil {
		t.Fatalf("NewService with lockout: %v", err)
	}
	if svc == nil {
		t.Error("expected non-nil service")
	}
}

func TestUserFromContext_NilContext(t *testing.T) {
	user, ok := UserFromContext(nil) //nolint:staticcheck // intentionally testing nil-safety
	if ok || user != nil {
		t.Error("expected nil/false from nil context")
	}
}
