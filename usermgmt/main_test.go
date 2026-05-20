package usermgmt

import (
	"context"
	"errors"
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
