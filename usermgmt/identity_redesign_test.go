package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBeginImpersonation_RequiresReason(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	caller := NewUserID("admin-1")
	target := NewUserID("user-1")

	_, err := svc.BeginImpersonation(ctx, caller, target, "")
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestBeginImpersonation_PreventsSelfImpersonation(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	caller := NewUserID("admin-1")

	_, err := svc.BeginImpersonation(ctx, caller, caller, "testing")
	if err == nil {
		t.Fatal("expected error for self-impersonation")
	}
}

func TestBeginImpersonation_RequiresSuperAdminRole(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Register a regular user (no super_admin role)
	caller := registerIdentityTestUser(t, svc, "admin@test.com")
	target := registerIdentityTestUser(t, svc, "target@test.com")

	_, err := svc.BeginImpersonation(ctx, caller, target, "audit-investigation")
	if err == nil {
		t.Fatal("expected error when caller lacks super_admin role")
	}
}

func TestBeginImpersonation_TargetMustExist(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Register a super_admin caller
	callerID := registerIdentityTestUser(t, svc, "super@test.com")
	// Grant super_admin role
	grantSuperAdmin(t, svc, callerID)

	// Non-existent target
	nonexistentTarget := NewUserID("01JXNONEXISTENTUSER000")

	_, err := svc.BeginImpersonation(ctx, callerID, nonexistentTarget, "investigation")
	if err == nil {
		t.Fatal("expected error for non-existent target")
	}
}

func TestBeginImpersonation_Success_CreatesImpersonationSession(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	callerID := registerIdentityTestUser(t, svc, "super@test.com")
	grantSuperAdmin(t, svc, callerID)

	targetID := registerIdentityTestUser(t, svc, "target@test.com")

	session, err := svc.BeginImpersonation(ctx, callerID, targetID, "security-audit")
	if err != nil {
		t.Fatalf("BeginImpersonation: %v", err)
	}
	if session == nil {
		t.Fatal("session is nil")
	}
	if session.Token == "" {
		t.Error("session token is empty")
	}
	if session.UserID != targetID {
		t.Errorf("session.UserID = %q, want %q", session.UserID, targetID)
	}

	// Verify session origin is Impersonation
	imp, ok := session.Origin.(Impersonation)
	if !ok {
		t.Fatal("session origin is not Impersonation")
	}
	if imp.By != ActorIDFromUser(callerID) {
		t.Errorf("impersonator = %q, want %q", imp.By, ActorIDFromUser(callerID))
	}
	if imp.Reason != "security-audit" {
		t.Errorf("reason = %q", imp.Reason)
	}
}

func TestEndImpersonation_DeletesSession(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	callerID := registerIdentityTestUser(t, svc, "super@test.com")
	grantSuperAdmin(t, svc, callerID)
	targetID := registerIdentityTestUser(t, svc, "target@test.com")

	session, err := svc.BeginImpersonation(ctx, callerID, targetID, "test")
	if err != nil {
		t.Fatalf("BeginImpersonation: %v", err)
	}

	if err := svc.EndImpersonation(ctx, session.Token); err != nil {
		t.Fatalf("EndImpersonation: %v", err)
	}

	// Verify session is gone
	_, err = svc.sessions.Find(ctx, session.Token)
	if err == nil {
		t.Error("session should be deleted after EndImpersonation")
	}
}

func TestEndImpersonation_RejectsNonImpersonationSession(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Create a regular (non-impersonation) session
	userID := registerIdentityTestUser(t, svc, "normal@test.com")
	session, err := svc.createSession(ctx, userID)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}

	err = svc.EndImpersonation(ctx, session.Token)
	if err == nil {
		t.Error("expected error when ending non-impersonation session")
	}
}

// --- API Token Middleware Tests ---

func TestAPITokenMiddleware_NoToken_Passes(t *testing.T) {
	svc := setupTestService(t)
	mw := NewAPITokenMiddleware(svc)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler should be called when no token is present")
	}
}

func TestAPITokenMiddleware_InvalidToken_Passes(t *testing.T) {
	svc := setupTestService(t)
	mw := NewAPITokenMiddleware(svc)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler should be called even with invalid token (pass-through)")
	}
}

func TestRequireBot_RejectsWithoutBot(t *testing.T) {
	handler := RequireBot(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("handler should not be called without bot")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

// --- Helpers ---

func setupTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{
		TokenPepper: TokenPepper("test-pepper-32-bytes-long-xxxxxxx"),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	return svc
}

func registerIdentityTestUser(t *testing.T, svc *Service, email string) UserID {
	t.Helper()
	ctx := context.Background()
	resp, err := svc.Register(ctx, RegisterRequest{
		ID:    NewUserID("01JX" + email[:4] + "TESTUSER0000000000"),
		Email: email,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return resp.User.ID
}

func grantSuperAdmin(t *testing.T, svc *Service, userID UserID) {
	t.Helper()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	if err := svc.authz.AddGroupPolicy(GroupPolicy{
		Subject: aggID.String(),
		Role:    RoleSuperAdmin,
		Domain:  aggID.String(),
	}); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}
}
