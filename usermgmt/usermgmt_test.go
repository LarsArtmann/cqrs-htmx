package usermgmt

import (
	"errors"
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	u := NewUser("user-1", "test@example.com", "Test User")
	if u.ID != "user-1" {
		t.Errorf("expected ID user-1, got %s", u.ID)
	}
	if u.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", u.Email)
	}
	if len(u.Roles) != 1 || u.Roles[0] != RoleViewer {
		t.Errorf("expected default role [viewer], got %v", u.Roles)
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestUser_Password(t *testing.T) {
	u := NewUser("user-1", "test@example.com", "Test User")
	if err := u.SetPassword("secret123"); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}
	if u.PasswordHash == "" {
		t.Error("expected non-empty password hash")
	}
	if !u.CheckPassword("secret123") {
		t.Error("expected password to match")
	}
	if u.CheckPassword("wrong") {
		t.Error("expected wrong password not to match")
	}
}

func TestUser_Roles(t *testing.T) {
	u := NewUser("user-1", "test@example.com", "Test User")
	u.AddRole(RoleAdmin)
	if !u.HasRole(RoleAdmin) {
		t.Error("expected admin role")
	}
	if !u.HasRole(RoleViewer) {
		t.Error("expected viewer role still present")
	}
	u.AddRole(RoleAdmin)
	if len(u.Roles) != 2 {
		t.Errorf("expected 2 roles after duplicate add, got %d", len(u.Roles))
	}
	u.RemoveRole(RoleAdmin)
	if u.HasRole(RoleAdmin) {
		t.Error("expected admin role removed")
	}
}

func TestNewEnforcer(t *testing.T) {
	e, err := NewEnforcer()
	if err != nil {
		t.Fatalf("NewEnforcer failed: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil enforcer")
	}
}

func TestEnforcer_DefaultPolicies(t *testing.T) {
	e, err := NewEnforcer()
	if err != nil {
		t.Fatalf("NewEnforcer failed: %v", err)
	}

	tests := []struct {
		sub, obj, act string
		expected      bool
	}{
		{"admin", "users", "read", true},
		{"admin", "anything", "whatever", true},
		{"user", "users", "read", true},
		{"user", "users", "delete", false},
		{"viewer", "profile", "read", true},
		{"viewer", "users", "update", false},
		{"unknown", "users", "read", false},
	}

	for _, tt := range tests {
		ok, err := e.Enforce(tt.sub, tt.obj, tt.act)
		if err != nil {
			t.Errorf("Enforce(%s, %s, %s) error: %v", tt.sub, tt.obj, tt.act, err)
			continue
		}
		if ok != tt.expected {
			t.Errorf("Enforce(%s, %s, %s) = %v, want %v", tt.sub, tt.obj, tt.act, ok, tt.expected)
		}
	}
}

func TestAssignRole(t *testing.T) {
	e, err := NewEnforcer()
	if err != nil {
		t.Fatalf("NewEnforcer failed: %v", err)
	}

	if err := AssignRole(e, "user-1", RoleAdmin); err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}

	ok, err := e.Enforce("user-1", "anything", "whatever")
	if err != nil {
		t.Fatalf("Enforce error: %v", err)
	}
	if !ok {
		t.Error("expected user-1 with admin role to have access")
	}
}

func TestRevokeRole(t *testing.T) {
	e, err := NewEnforcer()
	if err != nil {
		t.Fatalf("NewEnforcer failed: %v", err)
	}

	_ = AssignRole(e, "user-1", RoleAdmin)
	_ = RevokeRole(e, "user-1", RoleAdmin)

	ok, _ := e.Enforce("user-1", "anything", "whatever")
	if ok {
		t.Error("expected user-1 without admin role to be denied")
	}
}

func TestInMemoryUserStore(t *testing.T) {
	store := NewInMemoryUserStore()

	u := NewUser("user-1", "test@example.com", "Test User")
	_ = u.SetPassword("pass")

	if err := store.Save(u); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	found, err := store.FindByID("user-1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", found.Email)
	}

	byEmail, err := store.FindByEmail("test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if byEmail.ID != "user-1" {
		t.Errorf("expected ID user-1, got %s", byEmail.ID)
	}

	_, err = store.FindByID("nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	if err := store.Delete("user-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.FindByID("user-1")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after delete, got %v", err)
	}
}

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	session, err := store.Create("user-1", time.Hour)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if session.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", session.UserID)
	}

	found, err := store.Find(session.Token)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if found.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", found.UserID)
	}

	_, err = store.Find("nonexistent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}

	if err := store.Delete(session.Token); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.Find(session.Token)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestSession_Validity(t *testing.T) {
	session, err := NewSession("user-1", time.Hour)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	if session.IsExpired() {
		t.Error("expected session not to be expired")
	}
	if !session.Valid(session.Token) {
		t.Error("expected session to be valid with correct token")
	}
	if session.Valid("wrong-token") {
		t.Error("expected session to be invalid with wrong token")
	}
}

func TestService_Register(t *testing.T) {
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	resp, err := svc.Register(RegisterRequest{
		ID:          "user-1",
		Email:       "test@example.com",
		Password:    "secret123",
		DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if resp.User.ID != "user-1" {
		t.Errorf("expected user ID user-1, got %s", resp.User.ID)
	}
	if resp.Session == nil {
		t.Error("expected non-nil session")
	}
	if !resp.User.HasRole(RoleUser) {
		t.Error("expected user role")
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc, _ := NewService(ServiceConfig{})
	_, _ = svc.Register(RegisterRequest{ID: "u1", Email: "a@b.com", Password: "pass"})

	_, err := svc.Register(RegisterRequest{ID: "u2", Email: "a@b.com", Password: "pass"})
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestService_Login(t *testing.T) {
	svc, _ := NewService(ServiceConfig{})
	_, _ = svc.Register(RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret"})

	resp, err := svc.Login(LoginRequest{Email: "a@b.com", Password: "secret"})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if resp.User.ID != "user-1" {
		t.Errorf("expected user ID user-1, got %s", resp.User.ID)
	}
	if resp.Session == nil {
		t.Error("expected non-nil session")
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc, _ := NewService(ServiceConfig{})
	_, _ = svc.Register(RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret"})

	_, err := svc.Login(LoginRequest{Email: "a@b.com", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_Authenticate(t *testing.T) {
	svc, _ := NewService(ServiceConfig{})
	reg, _ := svc.Register(RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret"})

	user, err := svc.Authenticate(reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("expected user ID user-1, got %s", user.ID)
	}
}

func TestService_Logout(t *testing.T) {
	svc, _ := NewService(ServiceConfig{})
	reg, _ := svc.Register(RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret"})

	if err := svc.Logout(reg.Session.Token); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	_, err := svc.Authenticate(reg.Session.Token)
	if err == nil {
		t.Error("expected error after logout")
	}
}

func TestService_Authorize(t *testing.T) {
	svc, _ := NewService(ServiceConfig{})
	_, _ = svc.Register(RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret"})

	ok, err := svc.Authorize("user-1", "users", "read")
	if err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}
	if !ok {
		t.Error("expected user to be authorized to read users")
	}

	ok, _ = svc.Authorize("user-1", "users", "delete")
	if ok {
		t.Error("expected user NOT to be authorized to delete users")
	}
}

func TestService_UpdateRoles(t *testing.T) {
	svc, _ := NewService(ServiceConfig{})
	_, _ = svc.Register(RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret"})

	if err := svc.UpdateRoles("user-1", []string{RoleAdmin}); err != nil {
		t.Fatalf("UpdateRoles failed: %v", err)
	}

	ok, _ := svc.Authorize("user-1", "anything", "whatever")
	if !ok {
		t.Error("expected admin user to have full access")
	}

	user, _ := svc.GetUser("user-1")
	if !user.HasRole(RoleAdmin) {
		t.Error("expected admin role in user object")
	}
}
