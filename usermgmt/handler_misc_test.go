package usermgmt

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
		{ErrUserIDExists, http.StatusNotFound},
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

func TestErrorStatus_Default(t *testing.T) {
	got := errorStatus(errors.New("some unknown error"))
	if got != http.StatusInternalServerError {
		t.Errorf("expected 500 for unknown error, got %d", got)
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

func TestUserFromContextOr_WithUser(t *testing.T) {
	user := &User{ID: NewUserID("real")}
	ctx := WithUser(context.Background(), user)
	fallback := &User{ID: NewUserID("fallback")}
	result := UserFromContextOr(ctx, fallback)
	if result.ID != NewUserID("real") {
		t.Errorf("expected real user, got %s", result.ID)
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

	meHandler := NewSessionMiddleware(svc, "session_token")(
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
	assertStatusCode(t, meW, http.StatusOK)

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: "session_token", Value: regResp.Session.Token})
	logoutW := httptest.NewRecorder()
	mux.ServeHTTP(logoutW, logoutReq)
	assertStatusCode(t, logoutW, http.StatusOK)
}

func TestNewAuthHandler_SessionMaxAge(t *testing.T) {
	registerWithSessionMaxAge(t, "u1", "maxage@test.com", "secret12", 3600)
}

func TestNewAuthHandler_CustomCookieName(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{CookieName: "my_session", Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	w := postJSON(t, mux, "/auth/register",
		`{"id":"u1","email":"cookie@test.com","password":"secret12"}`)
	assertStatusCode(t, w, http.StatusCreated)
	assertCookie(t, w, "my_session", func(c *http.Cookie) bool { return c.Value != "" })
}

func TestInMemoryUserStore_Count(t *testing.T) {
	store := NewInMemoryUserStore()
	if store.Count() != 0 {
		t.Fatalf("expected 0, got %d", store.Count())
	}
	uid := NewUserID("01H4Z000000000000000000000")
	user := NewUser(uid, "test@example.com", "Test")
	if err := user.SetPasswordWithCost("password123", 4); err != nil {
		t.Fatalf("SetPasswordWithCost: %v", err)
	}
	if err := store.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if store.Count() != 1 {
		t.Fatalf("expected 1, got %d", store.Count())
	}
}

func TestInMemorySessionStore_Count(t *testing.T) {
	store := NewInMemorySessionStore()
	if store.Count() != 0 {
		t.Fatalf("expected 0, got %d", store.Count())
	}
	uid := NewUserID("01H4Z000000000000000000000")
	s, err := store.Create(context.Background(), uid, time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if store.Count() != 1 {
		t.Fatalf("expected 1, got %d", store.Count())
	}
	if err := store.Delete(context.Background(), s.Token); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if store.Count() != 0 {
		t.Fatalf("expected 0, got %d", store.Count())
	}
}
