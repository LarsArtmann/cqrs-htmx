package usermgmt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServiceConfig() ServiceConfig {
	return ServiceConfig{
		BcryptCost: minBcryptCost,
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithConfig(t, newTestServiceConfig())
}

func newTestServiceWithUser(
	t *testing.T,
	id, email, password string,
) (*Service, context.Context, *RegisterResponse) {
	t.Helper()
	svc := newTestService(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, id, email, password)
	return svc, ctx, reg
}

func addTestGroupPolicy(t *testing.T, a *Authz, gp GroupPolicy) {
	t.Helper()
	if err := a.AddGroupPolicy(gp); err != nil {
		t.Fatalf("AddGroupPolicy: %v", err)
	}
}

func assertEnforce(t *testing.T, a *Authz, sub, dom, obj string, act Action, wantAllowed bool) {
	t.Helper()
	ok, err := a.Enforce(sub, dom, obj, act)
	if err != nil {
		t.Fatalf("Enforce(%s, %s, %s, %s) error: %v", sub, dom, obj, act, err)
	}
	if ok != wantAllowed {
		t.Errorf("Enforce(%s, %s, %s, %s) = %v, want %v", sub, dom, obj, act, ok, wantAllowed)
	}
}

func postJSON(t *testing.T, mux *http.ServeMux, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func newTestServiceWithConfig(t *testing.T, cfg ServiceConfig) *Service {
	t.Helper()
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func newTestServiceWithAuthz(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithConfig(t, ServiceConfig{
		Authz:      newTestAuthz(t),
		BcryptCost: minBcryptCost,
	})
}

func assertErrorIs(t *testing.T, err, target error, msg string) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("expected %v, got %v (%s)", target, err, msg)
	}
}

func registerTestUser(t *testing.T, svc *Service, id, email, password string) *RegisterResponse {
	t.Helper()
	resp, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID(id), Email: email, Password: password,
	})
	if err != nil {
		t.Fatalf("registerTestUser %s: %v", id, err)
	}
	return resp
}

// registerWithSessionMaxAge builds an auth handler with the given session max age,
// registers a user, and asserts the cookie's MaxAge matches.
func registerWithSessionMaxAge(t *testing.T, id, email, password string, maxAge int) {
	t.Helper()
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool), SessionMaxAge: maxAge})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := fmt.Sprintf(`{"id":%q,"email":%q,"password":%q}`, id, email, password)
	w := postJSON(t, mux, "/auth/register", body)
	assertStatusCode(t, w, http.StatusCreated)
	assertCookie(t, w, "session_token", func(c *http.Cookie) bool { return c.MaxAge == maxAge })
}
