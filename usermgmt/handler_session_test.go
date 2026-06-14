package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewSessionMiddleware_Cookie(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	reg, _ := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "mw@t.com", Password: "secret12",
	})

	called := false
	w := wrapSessionMiddleware(
		t,
		svc,
		reg.Session.Token,
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			called = true
			user, ok := UserFromContext(r.Context())
			if !ok || user == nil {
				t.Error("expected user in context")
			} else if user.ID != NewUserID("u1") {
				t.Errorf("expected user ID u1, got %s", user.ID)
			}
		}),
	)
	if !called {
		t.Error("expected handler called")
	}
	_ = w
}

func TestNewSessionMiddleware_BearerToken(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	reg, _ := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "bt@t.com", Password: "secret12",
	})

	h := NewSessionMiddleware(svc, "session_token")(
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
	h.ServeHTTP(w, req)
}

func TestNewSessionMiddleware_NoToken(t *testing.T) {
	svc := newTestServiceWithAuthz(t)

	called := false
	wrapSessionMiddleware(
		t,
		svc,
		"",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			_, ok := UserFromContext(r.Context())
			if ok {
				t.Error("expected no user in context")
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
	if !called {
		t.Error("expected handler called")
	}
}
