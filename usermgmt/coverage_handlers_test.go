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

func TestHandlers_Register_Success(t *testing.T) {
	_, mux := setupMux(t)

	w := postJSON(t, mux, "/auth/register",
		`{"id":"u2","email":"reg@test.com","display_name":"Reg"}`)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 for register, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandlers_Register_ValidationError(t *testing.T) {
	assertValidationBadRequest(t, "/auth/register", `{"id":"","email":"bad"}`)
}

func TestHandlers_Register_WithCustomSessionMaxAge(t *testing.T) {
	registerWithSessionMaxAge(t, "sma1", "sma@test.com", 7200)
}

// assertValidationBadRequest POSTs a JSON body to the given path and asserts
// the handler returns 400. Used by validation-error coverage tests.
func assertValidationBadRequest(t *testing.T, path, body string) {
	t.Helper()
	_, mux := setupMux(t)
	w := postJSON(t, mux, path, body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for validation error, got %d", w.Code)
	}
}

func TestHandlers_Logout_Success(t *testing.T) {
	svc, mux := setupMux(t)

	reg := registerTestUser(t, svc, "u1", "logout@test.com")

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for logout, got %d", w.Code)
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
	})
	_ = svc

	sessions2 := &mockSessionStore{}
	svc2, _ := NewService(ServiceConfig{
		SessionStore: sessions2,
	})
	reg := registerTestUser(t, svc2, "lse1", "lse@test.com")

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

func TestHandlers_Logout_WithTimeout(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool), Timeout: 5 * time.Second})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	reg := registerTestUser(t, svc, "lot", "lot@test.com")

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

	reg := registerTestUser(t, svc, "lc1", "lc@test.com")

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assertCookie(t, w, "session_token", func(c *http.Cookie) bool {
		return c.MaxAge == -1 && c.Secure
	})
}

func TestHandlers_Logout_DeletedSession(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	reg := registerTestUser(t, svc, "u1", "lo@test.com")

	_ = svc.sessions.Delete(context.Background(), reg.Session.Token)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: reg.Session.Token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent logout, got %d", w.Code)
	}
}

func TestHandlers_Me_WithUser(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	reg := registerTestUser(t, svc, "u1", "me@test.com")

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = req.WithContext(WithUser(req.Context(), reg.User))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /me with user, got %d", w.Code)
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

func TestUserFromContext_NilContext(t *testing.T) {
	var nilCtx context.Context // intentionally testing nil-safety
	_ = nilCtx
	user, ok := UserFromContext(nilCtx)
	if ok || user != nil {
		t.Error("expected nil/false from nil context")
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
