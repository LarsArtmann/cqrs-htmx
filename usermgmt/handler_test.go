package usermgmt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithConfig(t, ServiceConfig{
		Authz:      newTestAuthz(t),
		BcryptCost: minBcryptCost,
	})
}

func setupMux(t *testing.T) (*Service, *http.ServeMux) {
	t.Helper()
	svc := newTestService(t)
	h := NewAuthHandlers(svc, HandlerConfig{Secure: false})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return svc, mux
}

func registerUser(t *testing.T, mux *http.ServeMux) *RegisterResponse {
	t.Helper()
	body := `{"id":"u1","email":"a@b.com","password":"secret12","display_name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp RegisterResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
	}
	return &resp
}

func TestHandlers_Register(t *testing.T) {
	_, mux := setupMux(t)

	resp := registerUser(t, mux)
	if resp.User.ID != NewUserID("u1") {
		t.Errorf("expected user ID u1, got %s", resp.User.ID)
	}

	found := false
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	_ = found
}

func TestHandlers_Register_SetsCookie(t *testing.T) {
	_, mux := setupMux(t)
	body := `{"id":"u1","email":"cookie@test.com","password":"secret12"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_token" && c.Value != "" && c.HttpOnly &&
			c.SameSite == http.SameSiteStrictMode &&
			!c.Secure {
			found = true
		}
	}
	if !found {
		t.Error("expected session_token cookie with correct attributes")
	}
}

func TestHandlers_Register_DuplicateEmail(t *testing.T) {
	_, mux := setupMux(t)
	registerUser(t, mux)

	body := `{"id":"u2","email":"a@b.com","password":"secret12"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestHandlers_Login(t *testing.T) {
	_, mux := setupMux(t)
	registerUser(t, mux)

	body := `{"email":"a@b.com","password":"secret12"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.User.ID != NewUserID("u1") {
		t.Errorf("expected user ID u1, got %s", resp.User.ID)
	}
}

func TestHandlers_Login_WrongPassword(t *testing.T) {
	_, mux := setupMux(t)
	registerUser(t, mux)

	body := `{"email":"a@b.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandlers_Logout(t *testing.T) {
	_, mux := setupMux(t)
	resp := registerUser(t, mux)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: resp.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	cleared := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_token" && c.MaxAge == -1 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("expected cleared session_token cookie")
	}
}

func TestHandlers_Logout_NoSession(t *testing.T) {
	_, mux := setupMux(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandlers_Me_Unauthenticated(t *testing.T) {
	_, mux := setupMux(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandlers_BadRequestBody(t *testing.T) {
	_, mux := setupMux(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSessionMiddleware_Cookie(t *testing.T) {
	svc := newTestService(t)
	reg, _ := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "mw@t.com", Password: "secret12",
	})

	called := false
	handler := SessionMiddleware(
		svc,
		"session_token",
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			user, ok := UserFromContext(r.Context())
			if !ok || user == nil {
				t.Error("expected user in context")
			} else if user.ID != NewUserID("u1") {
				t.Errorf("expected user ID u1, got %s", user.ID)
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler called")
	}
}

func TestSessionMiddleware_BearerToken(t *testing.T) {
	svc := newTestService(t)
	reg, _ := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "bt@t.com", Password: "secret12",
	})

	handler := SessionMiddleware(
		svc,
		"session_token",
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || user == nil {
				t.Fatal("expected user in context")
			}
			if user.ID != NewUserID("u1") {
				t.Errorf("expected u1, got %s", user.ID)
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+reg.Session.Token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestSessionMiddleware_NoToken(t *testing.T) {
	svc := newTestService(t)

	called := false
	handler := SessionMiddleware(
		svc,
		"session_token",
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			_, ok := UserFromContext(r.Context())
			if ok {
				t.Error("expected no user in context")
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler called")
	}
}

func TestErrorStatus(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{ErrInvalidCredentials, http.StatusUnauthorized},
		{ErrUnauthorized, http.StatusUnauthorized},
		{ErrSessionExpired, http.StatusUnauthorized},
		{ErrEmailExists, http.StatusConflict},
		{ErrForbidden, http.StatusForbidden},
		{ErrUserNotFound, http.StatusNotFound},
		{ErrSessionNotFound, http.StatusNotFound},
		{ErrValidation, http.StatusBadRequest},
		{ErrAccountLocked, http.StatusTooManyRequests},
	}
	for _, tt := range tests {
		got := errorStatus(tt.err)
		if got != tt.expected {
			t.Errorf("errorStatus(%v) = %d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestUserFromContext_Nil(t *testing.T) {
	user, ok := UserFromContext(context.TODO())
	if ok || user != nil {
		t.Error("expected nil/false from empty context")
	}
}

func TestUserFromContextOr_Fallback(t *testing.T) {
	fallback := &User{ID: NewUserID("fallback")}
	result := UserFromContextOr(context.TODO(), fallback)
	if result != fallback {
		t.Error("expected fallback")
	}
}

func TestExtractToken_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if token := extractToken(req, "session_token"); token != "" {
		t.Errorf("expected empty token, got %s", token)
	}
}

func TestUserIDFromRequest(t *testing.T) {
	t.Run("returns ID when user in context", func(t *testing.T) {
		user := &User{ID: NewUserID("u1")}
		ctx := WithUser(context.Background(), user)
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		if got := UserIDFromRequest(req); got != "u1" {
			t.Errorf("expected u1, got %s", got)
		}
	})

	t.Run("returns empty when no user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if got := UserIDFromRequest(req); got != "" {
			t.Errorf("expected empty, got %s", got)
		}
	})
}

func TestHandlers_FullFlow(t *testing.T) {
	svc, mux := setupMux(t)

	regResp := registerUser(t, mux)

	meHandler := SessionMiddleware(
		svc,
		"session_token",
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || user == nil {
				t.Fatal("no user in context")
			}
			writeJSON(w, http.StatusOK, user)
		}),
	)

	meReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meReq.AddCookie(&http.Cookie{Name: "session_token", Value: regResp.Session.Token})
	meW := httptest.NewRecorder()
	meHandler.ServeHTTP(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Errorf("me: expected 200, got %d", meW.Code)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: "session_token", Value: regResp.Session.Token})
	logoutW := httptest.NewRecorder()
	mux.ServeHTTP(logoutW, logoutReq)

	if logoutW.Code != http.StatusOK {
		t.Errorf("logout: expected 200, got %d", logoutW.Code)
	}
}

func TestNewAuthHandlers_SessionMaxAge(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandlers(svc, HandlerConfig{Secure: false, SessionMaxAge: 3600})

	body := `{"id":"u1","email":"maxage@test.com","password":"secret12"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_token" && c.MaxAge == 3600 {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie with MaxAge=3600")
	}
}

func TestNewAuthHandlers_CustomCookieName(t *testing.T) {
	svc := newTestService(t)
	h := NewAuthHandlers(svc, HandlerConfig{CookieName: "my_session", Secure: false})

	body := `{"id":"u1","email":"cookie@test.com","password":"secret12"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "my_session" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie named 'my_session'")
	}
}
