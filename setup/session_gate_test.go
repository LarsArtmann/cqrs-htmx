package setup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// newAuthedBundle builds a default bundle (memory stores) and registers a user
// through the service, returning a fully-mounted handler and the session
// token issued at registration.
func newAuthedBundle(t *testing.T) (http.Handler, string) {
	t.Helper()
	bundle, err := setup.New(setup.Config{Title: "Session Gate"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	reg, err := bundle.Service.Register(context.Background(), usermgmt.RegisterRequest{
		ID: usermgmt.NewUserID("gatem1"), Email: "gatem1@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return bundle.Handler(http.NewServeMux()), reg.Session.Token
}

// --- M9: the auth subset is reachable end-to-end through Bundle.Handler ---

func TestBundleHandler_MeAndCredentials_ReachableWithSessionCookie(t *testing.T) {
	handler, token := newAuthedBundle(t)

	for _, tc := range []struct {
		name string
		path string
	}{
		{"me", "/auth/me"},
		{"credentials", "/auth/credentials"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: "session", Value: token})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d (valid cookie through full bundle), body: %s",
					w.Code, http.StatusOK, w.Body.String())
			}
		})
	}
}

func TestBundleHandler_AuthSubset_FailClosedWithoutCookie(t *testing.T) {
	handler, _ := newAuthedBundle(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (no cookie, fail-closed)", w.Code, http.StatusUnauthorized)
	}
}

// --- M8.3: exported session gates ---

func gateChain(bundle *setup.Bundle, gate func(http.Handler) http.Handler) http.Handler {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return bundle.SessionMiddleware()(gate(ok))
}

func bundleForGateTest(t *testing.T) *setup.Bundle {
	t.Helper()
	bundle, err := setup.New(setup.Config{Title: "Gates"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })
	return bundle
}

func TestRequireSession_BlocksWithoutUser_PassesWithUser(t *testing.T) {
	bundle := bundleForGateTest(t)
	handler := gateChain(bundle, setup.RequireSession)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/private", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no user: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	reg, err := bundle.Service.Register(context.Background(), usermgmt.RegisterRequest{
		ID: usermgmt.NewUserID("gate1"), Email: "gate1@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: reg.Session.Token})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("with user: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireSessionRedirect_BlocksWithoutUser_PassesWithUser(t *testing.T) {
	bundle := bundleForGateTest(t)
	handler := gateChain(bundle, setup.RequireSessionRedirect("/login"))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/private", nil))
	if w.Code != http.StatusSeeOther {
		t.Errorf("no user: status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}

	reg, err := bundle.Service.Register(context.Background(), usermgmt.RegisterRequest{
		ID: usermgmt.NewUserID("gate2"), Email: "gate2@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: reg.Session.Token})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("with user: status = %d, want %d", w.Code, http.StatusOK)
	}
}
