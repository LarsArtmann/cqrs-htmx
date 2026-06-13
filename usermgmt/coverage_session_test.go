package usermgmt

import (
	"net/http"
	"net/http/httptest"
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
