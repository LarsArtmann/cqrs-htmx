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
	return ServiceConfig{}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithConfig(t, newTestServiceConfig())
}

func newTestServiceWithUser(
	t *testing.T,
	id, email string,
) (*Service, context.Context, *RegisterResponse) {
	t.Helper()
	svc := newTestService(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, id, email)
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
		Authz: newTestAuthz(t),
	})
}

func localTestWebAuthnConfig() *WebAuthnConfig {
	return &WebAuthnConfig{
		RPID:          "localhost",
		RPDisplayName: "Test",
		RPOrigins:     []string{"https://localhost"},
	}
}

func newWebAuthnTestServiceWithConfig(t *testing.T, cfg *WebAuthnConfig) *Service {
	t.Helper()
	svc := newTestServiceWithConfig(t, ServiceConfig{WebAuthnConfig: cfg})
	t.Cleanup(svc.Stop)
	return svc
}

func assertErrorIs(t *testing.T, err, target error, msg string) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("expected %v, got %v (%s)", target, err, msg)
	}
}

// addTestCredential registers a credential on the given service for the
// given user, failing the test on error. Used to set up WebAuthn
// state for tests that exercise credential-dependent code paths.
func addTestCredential(t *testing.T, svc *Service, userID UserID, cred WebAuthnCredential) {
	t.Helper()
	if err := svc.AddCredential(context.Background(), userID, cred); err != nil {
		t.Fatalf("AddCredential: %v", err)
	}
}

func registerTestUser(t *testing.T, svc *Service, id, email string) *RegisterResponse {
	t.Helper()
	resp, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID(id), Email: email,
	})
	if err != nil {
		t.Fatalf("registerTestUser %s: %v", id, err)
	}
	return resp
}

func grantTestRole(t *testing.T, svc *Service, userID UserID, role Role) {
	t.Helper()
	actor := ActorIDFromUser(userID)
	tenant := NewTenantID(userID.Get().String()) // self-scoped domain (matches legacy Casbin behavior)
	ctx := context.Background()
	if err := svc.dispatcher.Dispatch(ctx, NewAddMemberCmd(actor, tenant, []Role{role})); err != nil {
		t.Fatalf("grantTestRole AddMember: %v", err)
	}
}

func assertChangeEmailError(t *testing.T, svc *Service, userID UserID, email string) {
	t.Helper()
	if err := svc.ChangeEmail(context.Background(), userID, email); err == nil {
		t.Fatal("expected error for ChangeEmail")
	}
}

func registerWithSessionMaxAge(t *testing.T, id, email string, maxAge int) {
	t.Helper()
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool), SessionMaxAge: maxAge})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := fmt.Sprintf(`{"id":%q,"email":%q}`, NewUserID(id).Get().String(), email)
	w := postJSON(t, mux, "/auth/register", body)
	assertStatusCode(t, w, http.StatusCreated)
	assertCookie(t, w, "session_token", func(c *http.Cookie) bool { return c.MaxAge == maxAge })
}
