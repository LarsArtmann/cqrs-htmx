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
	_ = store.Create(context.Background(), u1)
	u2 := NewUser(NewUserID("u2"), "c@d.com", "Two")
	_ = store.Create(context.Background(), u2)

	u2.Email = "a@b.com"
	if err := store.Save(context.Background(), u2); !errors.Is(err, ErrEmailExists) {
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

	_ = svc.sessions.Delete(context.Background(), reg.Session.Token)

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
	var nilCtx context.Context // intentionally testing nil-safety
	_ = nilCtx
	user, ok := UserFromContext(nilCtx)
	if ok || user != nil {
		t.Error("expected nil/false from nil context")
	}
}

func TestPolicyWrapErr(t *testing.T) {
	p := Policy{RoleAdmin, "domain1", "resource", ActionExecute, EffectAllow}
	got := policyWrapErr("test msg", p)
	if got != "test msg {admin, domain1, resource, execute, allow}" {
		t.Errorf("unexpected policyWrapErr output: %s", got)
	}
}

func TestHandlers_Login_Success(t *testing.T) {
	svc, mux := setupMux(t)

	registerTestUser(t, svc, "u1", "login@test.com", "secret12")

	w := postJSON(t, mux, "/auth/login", `{"email":"login@test.com","password":"secret12"}`)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for login, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Set-Cookie") == "" {
		t.Error("expected Set-Cookie header after login")
	}
}

func TestHandlers_Register_Success(t *testing.T) {
	_, mux := setupMux(t)

	w := postJSON(t, mux, "/auth/register",
		`{"id":"u2","email":"reg@test.com","password":"secret12","display_name":"Reg"}`)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 for register, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandlers_Logout_Success(t *testing.T) {
	svc, mux := setupMux(t)

	reg := registerTestUser(t, svc, "u1", "logout@test.com", "secret12")

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for logout, got %d", w.Code)
	}
}

func TestHandlers_Me_WithUser(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: false})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	reg := registerTestUser(t, svc, "u1", "me@test.com", "secret12")

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = req.WithContext(WithUser(req.Context(), reg.User))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /me with user, got %d", w.Code)
	}
}

func TestHandlers_Me_NilUserInContext(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: false})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = req.WithContext(WithUser(req.Context(), nil))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for nil user in context, got %d", w.Code)
	}
}

func TestAuthz_EnforceEx_Error(t *testing.T) {
	a, _ := NewAuthz(EnforcerConfig{
		ModelString: defaultModel,
		Policies:    []Policy{},
	})

	result, err := a.EnforceEx("nobody", "dom", "secret", ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected denied for nobody")
	}
	if result.Subject != "nobody" || result.Domain != "dom" {
		t.Errorf("unexpected result fields: %+v", result)
	}
}

func TestAuthz_Apply_RemovePolicies(t *testing.T) {
	a := newTestAuthz(
		t,
		Policy{"*", "*", "res.action", ActionExecute, EffectAllow},
	)

	if err := a.Apply(PolicyUpdate{
		RemovePolicies: []Policy{{"*", "*", "res.action", ActionExecute, EffectAllow}},
	}); err != nil {
		t.Fatalf("Apply remove policies: %v", err)
	}

	ok, _ := a.Enforce("anyone", "dom", "res.action", ActionExecute)
	if ok {
		t.Error("expected denied after removing policy")
	}
}

func TestService_Login_AccountLocked(t *testing.T) {
	svc, _ := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		Lockout:    NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: time.Hour}),
	})
	ctx := context.Background()

	registerTestUser(t, svc, "u1", "locked@test.com", "secret12")

	//nolint:errcheck // intentionally triggering lockout
	svc.Login(ctx, LoginRequest{Email: "locked@test.com", Password: "wrong1"})
	//nolint:errcheck // intentionally triggering lockout
	svc.Login(ctx, LoginRequest{Email: "locked@test.com", Password: "wrong2"})

	_, err := svc.Login(ctx, LoginRequest{Email: "locked@test.com", Password: "secret12"})
	assertErrorIs(t, err, ErrAccountLocked, "ErrAccountLocked")
}

func TestService_Login_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Login(
		context.Background(),
		LoginRequest{Email: "nobody@test.com", Password: "secret12"},
	)
	assertErrorIs(t, err, ErrInvalidCredentials, "ErrInvalidCredentials")
}

func TestService_Register_DuplicateUserID(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	registerTestUser(t, svc, "u1", "first@test.com", "secret12")

	_, err := svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u1"), Email: "second@test.com", Password: "secret12",
	})
	assertErrorIs(t, err, ErrUserIDExists, "ErrUserIDExists for duplicate user ID")
}

func TestSessionMiddleware_WithCookie(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	reg := registerTestUser(t, svc, "u1", "mid@test.com", "secret12")

	var called bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /protected", func(w http.ResponseWriter, r *http.Request) {
		called = true
		user, ok := UserFromContext(r.Context())
		if !ok || user == nil {
			t.Error("expected user in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := NewSessionMiddleware(svc, "session_token")(mux)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called")
	}
}

func TestSessionMiddleware_WithBearerToken(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	reg := registerTestUser(t, svc, "u1", "bearer@test.com", "secret12")

	var called bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /protected", func(_ http.ResponseWriter, r *http.Request) {
		called = true
		user, ok := UserFromContext(r.Context())
		if !ok || user == nil {
			t.Error("expected user in context")
		}
	})

	handler := NewSessionMiddleware(svc, "session_token")(mux)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+reg.Session.Token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called")
	}
}

func TestSessionMiddleware_InvalidToken(t *testing.T) {
	svc := newTestServiceWithAuthz(t)

	var called bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /protected", func(_ http.ResponseWriter, r *http.Request) {
		called = true
		_, ok := UserFromContext(r.Context())
		if ok {
			t.Error("expected no user in context with invalid token")
		}
	})

	handler := NewSessionMiddleware(svc, "session_token")(mux)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "invalid-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called even with invalid token")
	}
}

func TestUserFromContextOr(t *testing.T) {
	fallback := &User{ID: NewUserID("fallback")}
	result := UserFromContextOr(context.Background(), fallback)
	if result.ID != NewUserID("fallback") {
		t.Error("expected fallback user")
	}

	ctx := WithUser(context.Background(), &User{ID: NewUserID("real")})
	result = UserFromContextOr(ctx, fallback)
	if result.ID != NewUserID("real") {
		t.Error("expected real user from context")
	}
}

func TestNewAuthHandler_Defaults(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc)
	if h.cookieName != "session_token" {
		t.Errorf("expected default cookie name, got %q", h.cookieName)
	}
	if !h.secure {
		t.Error("expected secure=true by default")
	}
}

func TestNewAuthHandler_TimeoutPropagated(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Timeout: 5 * time.Second})
	if h.timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", h.timeout)
	}
}
