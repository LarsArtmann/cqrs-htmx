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

	conflicting := NewUser(NewUserID("u2"), "a@b.com", "Two")
	if err := store.Save(context.Background(), conflicting); !errors.Is(err, ErrEmailExists) {
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
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
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
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
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
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
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
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
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

func TestService_Login_StoreError(t *testing.T) {
	users := &mockUserStore{
		FindByEmailFn: func(_ context.Context, _ string) (*User, error) {
			return nil, errors.New("db connection lost")
		},
	}
	svc, _ := NewService(ServiceConfig{
		UserStore:  users,
		BcryptCost: minBcryptCost,
	})

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "any@test.com",
		Password: "secret12",
	})
	if errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("store errors must return transient, not ErrInvalidCredentials")
	}
	if err == nil {
		t.Fatal("expected error when store fails")
	}
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

func TestService_Register_RollbackOnGroupPolicyFailure(t *testing.T) {
	store := NewInMemoryUserStore()
	svc, err := NewService(ServiceConfig{
		UserStore:  store,
		BcryptCost: minBcryptCost,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// Use an Authz with nil enforcer to force AddGroupPolicy to fail.
	svc.authz = &Authz{enforcer: nil}

	ctx := context.Background()
	uid := NewUserID("rollback-user")

	_, regErr := svc.Register(ctx, RegisterRequest{
		ID:       uid,
		Email:    "rollback@test.com",
		Password: "secret12",
	})

	if regErr == nil {
		t.Fatal("expected Register to fail when AddGroupPolicy fails")
	}

	store.mu.RLock()
	afterUsers := len(store.users)
	store.mu.RUnlock()
	if afterUsers != 0 {
		t.Errorf("expected user to be rolled back, but %d users remain",
			afterUsers)
	}
}

func TestService_Register_RollbackOnSessionFailure(t *testing.T) {
	store := NewInMemoryUserStore()
	sessions := &mockSessionStore{
		CreateFn: func(_ context.Context, _ UserID, _ time.Duration) (*Session, error) {
			return nil, errors.New("session creation failed")
		},
	}

	svc, err := NewService(ServiceConfig{
		UserStore:    store,
		SessionStore: sessions,
		BcryptCost:   minBcryptCost,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	ctx := context.Background()
	uid := NewUserID("rollback-session")

	_, regErr := svc.Register(ctx, RegisterRequest{
		ID:       uid,
		Email:    "rollsession@test.com",
		Password: "secret12",
	})
	if regErr == nil {
		t.Fatal("expected error when session creation fails")
	}

	if _, err := store.FindByID(ctx, uid); !errors.Is(err, ErrUserNotFound) {
		t.Error("expected user to be rolled back after session failure")
	}
}

func TestService_Logout_StoreError(t *testing.T) {
	sessions := &mockSessionStore{
		DeleteFn: func(_ context.Context, _ string) error {
			return errors.New("db connection lost")
		},
	}
	svc, _ := NewService(ServiceConfig{
		SessionStore: sessions,
		BcryptCost:   minBcryptCost,
	})

	err := svc.Logout(context.Background(), "some-token")
	if err == nil {
		t.Fatal("expected error from Logout when store fails")
	}
}

func TestService_GetUser_NotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.GetUser(context.Background(), NewUserID("nonexistent"))
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestService_GetUser_StoreError(t *testing.T) {
	svc, _ := NewService(ServiceConfig{
		UserStore: &mockUserStore{
			FindByIDFn: func(_ context.Context, _ UserID) (*User, error) {
				return nil, errors.New("db error")
			},
		},
		BcryptCost: minBcryptCost,
	})
	_, err := svc.GetUser(context.Background(), NewUserID("u1"))
	if err == nil {
		t.Fatal("expected error from GetUser when store fails")
	}
}

func TestService_ChangePassword_WrongOld(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "cp1", "cp@test.com", "secret12")
	err := svc.ChangePassword(ctx, NewUserID("cp1"), "wrong-old", "newpass123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_ChangePassword_NewTooShort(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "cp2", "cp2@test.com", "secret12")
	err := svc.ChangePassword(ctx, NewUserID("cp2"), "secret12", "short")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_ChangePassword_NewTooLong(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "cp3", "cp3@test.com", "secret12")
	longPass := strings.Repeat("a", 129)
	err := svc.ChangePassword(ctx, NewUserID("cp3"), "secret12", longPass)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_ChangePassword_StoreError(t *testing.T) {
	svc := newTestService(t)
	err := svc.ChangePassword(context.Background(), NewUserID("ghost"), "old", "newpass123")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestService_UpdateRoles_StoreError(t *testing.T) {
	svc := newTestService(t)
	err := svc.UpdateRoles(context.Background(), NewUserID("ghost"), []Role{RoleAdmin}, "dom")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestService_Login_LockoutReset(t *testing.T) {
	lockout := NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: time.Hour})
	svc, _ := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		Lockout:    lockout,
	})
	ctx := context.Background()
	registerTestUser(t, svc, "lr1", "lr@test.com", "secret12")

	_, _ = svc.Login(ctx, LoginRequest{Email: "lr@test.com", Password: "wrong1"})
	_, err := svc.Login(ctx, LoginRequest{Email: "lr@test.com", Password: "secret12"})
	if err != nil {
		t.Errorf("expected successful login after one failure, got %v", err)
	}
}

func TestRecordFailure_TriggersLock(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 3, Duration: time.Hour})
	if l.RecordFailure("a@b.com") {
		t.Error("expected not locked on first failure")
	}
	if l.RecordFailure("a@b.com") {
		t.Error("expected not locked on second failure")
	}
	if !l.RecordFailure("a@b.com") {
		t.Error("expected locked on third failure (threshold)")
	}
}

func TestNewService_ZeroBcryptCost(t *testing.T) {
	svc, err := NewService(ServiceConfig{BcryptCost: 0})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.bcryptCost != defaultBcryptCost {
		t.Errorf("expected default cost %d, got %d", defaultBcryptCost, svc.bcryptCost)
	}
}

func TestNewService_CustomSessionTTL(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		SessionTTL: 2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.sessionTTL != 2*time.Hour {
		t.Errorf("expected 2h TTL, got %v", svc.sessionTTL)
	}
}

func TestNewService_NilAuthz(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.authz == nil {
		t.Error("expected default authz to be created")
	}
}

func TestHandlers_Login_AccountLocked(t *testing.T) {
	svc, _ := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		Lockout:    NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: time.Hour}),
	})
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	registerTestUser(t, svc, "hl1", "hl@test.com", "secret12")

	w := postJSON(t, mux, "/auth/login", `{"email":"hl@test.com","password":"wrong"}`)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}

	w = postJSON(t, mux, "/auth/login", `{"email":"hl@test.com","password":"secret12"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for locked account, got %d", w.Code)
	}
}

func TestService_Authenticate_SessionExpired(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "se1", "se@test.com", "secret12")

	reg.Session.ExpiresAt = time.Now().Add(-time.Hour)
	store, ok := svc.sessions.(*InMemorySessionStore)
	if !ok {
		t.Fatal("expected InMemorySessionStore")
	}
	store.mu.Lock()
	store.sessions[reg.Session.Token] = reg.Session
	store.mu.Unlock()

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

func TestService_Authenticate_UserGone(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "ud1", "ud@test.com", "secret12")

	userStore, ok := svc.users.(*InMemoryUserStore)
	if !ok {
		t.Fatal("expected InMemoryUserStore")
	}
	_ = userStore.Delete(ctx, reg.User.ID)

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestHandlers_Logout_StoreError(t *testing.T) {
	sessions := &mockSessionStore{
		DeleteFn: func(_ context.Context, _ string) error {
			return errors.New("db error")
		},
	}
	svc, _ := NewService(ServiceConfig{
		SessionStore: sessions,
		BcryptCost:   minBcryptCost,
	})
	_ = svc

	sessions2 := &mockSessionStore{}
	svc2, _ := NewService(ServiceConfig{
		SessionStore: sessions2,
		BcryptCost:   minBcryptCost,
	})
	reg := registerTestUser(t, svc2, "lse1", "lse@test.com", "secret12")

	svc.sessions = sessions

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for store error, got %d", w.Code)
	}
}

func TestHandlers_Register_ValidationError(t *testing.T) {
	_, mux := setupMux(t)
	w := postJSON(t, mux, "/auth/register", `{"id":"","email":"bad","password":"short"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for validation error, got %d", w.Code)
	}
}

func TestHandlers_Login_ValidationError(t *testing.T) {
	_, mux := setupMux(t)
	w := postJSON(t, mux, "/auth/login", `{"email":"","password":""}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for validation error, got %d", w.Code)
	}
}

func TestHandlers_Logout_WithTimeout(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool), Timeout: 5 * time.Second})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	reg := registerTestUser(t, svc, "lot", "lot@test.com", "secret12")

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for logout with timeout, got %d", w.Code)
	}
}

func TestHandlers_Logout_ClearCookie(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: func() *bool { v := true; return &v }()})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	reg := registerTestUser(t, svc, "lc1", "lc@test.com", "secret12")

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assertCookie(t, w, "session_token", func(c *http.Cookie) bool {
		return c.MaxAge == -1 && c.Secure
	})
}

func TestHandlers_Register_WithCustomSessionMaxAge(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool), SessionMaxAge: 7200})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/register",
		`{"id":"sma1","email":"sma@test.com","password":"secret12"}`)
	assertStatusCode(t, w, http.StatusCreated)
	assertCookie(t, w, "session_token", func(c *http.Cookie) bool { return c.MaxAge == 7200 })
}

func TestEvictExpired_NoExpired(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()
	_, _ = store.Create(ctx, NewUserID("u1"), time.Hour)

	evicted := store.EvictExpired()
	if evicted != 0 {
		t.Errorf("expected 0 evictions, got %d", evicted)
	}
}

func TestEvictExpired_WithExpired(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()
	s, _ := store.Create(ctx, NewUserID("u1"), time.Millisecond)

	s.ExpiresAt = time.Now().Add(-time.Hour)
	store.mu.Lock()
	store.sessions[s.Token] = s
	store.mu.Unlock()

	time.Sleep(2 * time.Millisecond)
	evicted := store.EvictExpired()
	if evicted != 1 {
		t.Errorf("expected 1 eviction, got %d", evicted)
	}
}

func TestEvictStale_NoStale(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 5, Duration: time.Hour})
	l.RecordFailure("active@test.com")

	evicted := l.EvictStale()
	if evicted != 0 {
		t.Errorf("expected 0 evictions, got %d", evicted)
	}
}

func TestEvictStale_WithStale(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 1, Duration: 50 * time.Millisecond})
	l.RecordFailure("stale@test.com")

	time.Sleep(100 * time.Millisecond)
	evicted := l.EvictStale()
	if evicted != 1 {
		t.Errorf("expected 1 eviction, got %d", evicted)
	}
}

func TestAccountLockout_Reset(t *testing.T) {
	l := NewAccountLockout(LockoutConfig{MaxAttempts: 2, Duration: time.Hour})
	l.RecordFailure("r@test.com")
	l.Reset("r@test.com")
	if l.IsLocked("r@test.com") {
		t.Error("expected not locked after reset")
	}
}

func TestAccountLockout_IsLocked_NotLocked(t *testing.T) {
	l := NewAccountLockout()
	if l.IsLocked("never@test.com") {
		t.Error("expected not locked for unknown email")
	}
}
