package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newOAuth2TestService(t *testing.T) *Service {
	t.Helper()
	svc := newTestServiceWithConfig(t, ServiceConfig{
		Authz:  newTestAuthz(t),
		OAuth2: testOAuth2Provider{},
	})
	t.Cleanup(svc.Stop)
	return svc
}

// --- handleOAuth2Begin tests ---

func TestHandler_OAuth2Begin_Success(t *testing.T) {
	svc := newOAuth2TestService(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/begin", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusFound)

	loc := w.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected non-empty Location header")
	}
	if !strings.HasPrefix(loc, "https://") {
		t.Errorf("expected HTTPS redirect URL, got %q", loc)
	}
}

func TestHandler_OAuth2Callback_MissingCodeAndState(t *testing.T) {
	svc := newOAuth2TestService(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/callback", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusBadRequest)
}

func TestHandler_OAuth2Callback_InvalidState(t *testing.T) {
	svc := newOAuth2TestService(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/callback?code=fake&state=never-stored", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusFound {
		t.Errorf("expected 400 or 302 (error redirect), got %d", w.Code)
	}
}

// TestHandler_OAuth2Callback_RegistrationClosed_Returns403 verifies that a
// first-login auto-provisioning attempt past ServiceConfig.MaxUsers surfaces
// as HTTP 403 (via the ErrRegistrationClosed WithHTTPStatus carrier), not 500.
func TestHandler_OAuth2Callback_RegistrationClosed_Returns403(t *testing.T) {
	svc := newTestServiceWithConfig(t, ServiceConfig{
		Authz:    newTestAuthz(t),
		OAuth2:   testOAuth2Provider{},
		MaxUsers: 1,
	})
	t.Cleanup(svc.Stop)
	registerTestUser(t, svc, "u1", "first@example.com")

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	beginReq := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/begin", nil)
	beginW := httptest.NewRecorder()
	mux.ServeHTTP(beginW, beginReq)
	assertStatusCode(t, beginW, http.StatusFound)
	parsedURL, err := url.Parse(beginW.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	state := parsedURL.Query().Get("state")
	if state == "" {
		t.Fatal("redirect URL missing state")
	}

	cbReq := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/callback?code=test-code&state="+state, nil)
	cbW := httptest.NewRecorder()
	mux.ServeHTTP(cbW, cbReq)
	assertStatusCode(t, cbW, http.StatusForbidden)
	if got := svc.readModel.Count(); got != 1 {
		t.Errorf("read model count = %d, want 1 (no user created by rejected login)", got)
	}
}

func TestHandler_OAuth2Callback_Success(t *testing.T) {
	svc := newOAuth2TestService(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	// Step 1: Begin login to get a valid state
	beginReq := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/begin", nil)
	beginW := httptest.NewRecorder()
	mux.ServeHTTP(beginW, beginReq)
	assertStatusCode(t, beginW, http.StatusFound)

	// Extract state from the Location header redirect URL
	parsedURL, err := url.Parse(beginW.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	state := parsedURL.Query().Get("state")
	if state == "" {
		t.Fatal("redirect URL missing state")
	}

	// Step 2: Complete callback with valid state + code
	cbReq := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/callback?code=test-code&state="+state, nil)
	cbW := httptest.NewRecorder()
	mux.ServeHTTP(cbW, cbReq)
	assertStatusCode(t, cbW, http.StatusOK)

	// Verify session cookie was set
	cookies := cbW.Result().Cookies()
	var hasSession bool
	for _, c := range cookies {
		if c.Name == defaultCookieName && c.Value != "" {
			hasSession = true
		}
	}
	if !hasSession {
		t.Error("expected session cookie to be set")
	}
}

func TestHandler_OAuth2Callback_ErrorRedirect(t *testing.T) {
	svc := newOAuth2TestService(t)
	h := NewAuthHandler(svc, HandlerConfig{
		Secure:         new(bool),
		OAuth2ErrorURL: "http://localhost:3000/login-error",
	})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/callback?code=fake&state=bogus", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusFound)

	loc := w.Header().Get("Location")
	if loc == "" {
		t.Error("expected redirect Location header")
	}
	if !strings.Contains(loc, "error=") {
		t.Errorf("expected error query param in redirect: %s", loc)
	}
}

func TestHandler_OAuth2Callback_SuccessRedirect(t *testing.T) {
	svc := newOAuth2TestService(t)
	h := NewAuthHandler(svc, HandlerConfig{
		Secure:           new(bool),
		OAuth2SuccessURL: "http://localhost:3000/dashboard",
	})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	// Begin login
	beginReq := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/begin", nil)
	beginW := httptest.NewRecorder()
	mux.ServeHTTP(beginW, beginReq)
	parsedURL, _ := url.Parse(beginW.Header().Get("Location"))
	state := parsedURL.Query().Get("state")

	// Complete callback — should redirect to success URL
	cbReq := httptest.NewRequest(http.MethodGet, "/auth/oauth/github/callback?code=test-code&state="+state, nil)
	cbW := httptest.NewRecorder()
	mux.ServeHTTP(cbW, cbReq)
	assertStatusCode(t, cbW, http.StatusFound)
	if loc := cbW.Header().Get("Location"); loc != "http://localhost:3000/dashboard" {
		t.Errorf("expected redirect to success URL, got %q", loc)
	}
}

// --- handleOAuth2Unlink tests ---

func TestHandler_OAuth2Unlink_Unauthenticated(t *testing.T) {
	svc := newOAuth2TestService(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/github/unlink", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusUnauthorized)
}

func TestHandler_OAuth2Unlink_NotLinked(t *testing.T) {
	svc := newOAuth2TestService(t)
	registerTestUser(t, svc, "u1", "unlink@test.com")

	user, err := svc.GetUser(context.Background(), NewUserID("u1"))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/github/unlink", nil)
	req = req.WithContext(WithUser(context.Background(), user))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("expected 200, 400, or 404 for unlink-not-linked, got %d", w.Code)
	}
}

func TestHandler_OAuth2Unlink_Success(t *testing.T) {
	svc := newOAuth2TestService(t)
	registerTestUser(t, svc, "u1", "unlink-ok@test.com")

	// Link TWO external accounts so unlinking one doesn't trigger the last-auth-method guard
	ctx := context.Background()
	userID := NewUserID("u1")
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	if err := svc.dispatcher.Dispatch(ctx, NewLinkExternalAccountCmd(
		aggID, "github", "gh-sub-1", "unlink-ok@test.com", "Test User",
	)); err != nil {
		t.Fatalf("dispatch LinkExternalAccount github: %v", err)
	}
	if err := svc.dispatcher.Dispatch(ctx, NewLinkExternalAccountCmd(
		aggID, "google", "goog-sub-1", "unlink-ok@test.com", "Test User",
	)); err != nil {
		t.Fatalf("dispatch LinkExternalAccount google: %v", err)
	}

	// Verify the external account was linked
	user, err := svc.GetUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if len(user.ExternalAccounts) == 0 {
		t.Fatal("expected at least 1 external account after link")
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterOAuth2Routes(mux)

	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/github/unlink", nil)
	req = req.WithContext(WithUser(ctx, user))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusOK)

	// Verify the external account was removed
	user2, _ := svc.GetUser(ctx, userID)
	for _, ea := range user2.ExternalAccounts {
		if ea.Provider == "github" {
			t.Error("github external account should have been unlinked")
		}
	}
}
