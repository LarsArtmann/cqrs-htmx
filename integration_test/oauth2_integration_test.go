package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	usermgmtoauth2 "github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestService_OAuth2_BeginLogin_Integration verifies the cross-module
// flow: Service.BeginOAuthLogin → generateOAuth2State →
// oauth2.Provider.BeginLogin (PKCE + redirect URL).
//
// This proves the OAuth2Provider interface contract works end-to-end
// through the module boundary without needing a running OAuth2 server.
// The Provider is initialized with explicit endpoints (no OIDC discovery).
func TestService_OAuth2_BeginLogin_Integration(t *testing.T) {
	t.Parallel()

	provider, err := usermgmtoauth2.New(context.Background(), usermgmtoauth2.Config{
		Providers: map[string]usermgmtoauth2.ProviderConfig{
			"github": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:8080/callback",
				AuthURL:      "https://github.com/login/oauth/authorize",
				TokenURL:     "https://github.com/login/oauth/access_token",
				UserInfoURL:  "https://api.github.com/user",
			},
		},
	})
	if err != nil {
		t.Fatalf("oauth2.New: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		OAuth2: provider,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	// BeginOAuthLogin should return a redirect URL with PKCE params
	resp, err := svc.BeginOAuthLogin(context.Background(), "github")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}

	if resp.RedirectURL == "" {
		t.Fatal("expected non-empty redirect URL")
	}
	if !strings.Contains(resp.RedirectURL, "client_id=test-client-id") {
		t.Errorf("redirect URL missing client_id: %s", resp.RedirectURL)
	}
	if !strings.Contains(resp.RedirectURL, "code_challenge=") {
		t.Error("redirect URL missing PKCE code_challenge")
	}
	if !strings.Contains(resp.RedirectURL, "state=") {
		t.Error("redirect URL missing state parameter")
	}
}

// TestService_OAuth2_UnknownProvider verifies error handling for
// a provider name that doesn't exist in the configuration.
func TestService_OAuth2_UnknownProvider(t *testing.T) {
	t.Parallel()

	provider, err := usermgmtoauth2.New(context.Background(), usermgmtoauth2.Config{
		Providers: map[string]usermgmtoauth2.ProviderConfig{
			"google": {
				ClientID:     "test-id",
				ClientSecret: "test-secret",
				RedirectURL:  "http://localhost:8080/callback",
				AuthURL:      "https://accounts.google.com/o/oauth2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("oauth2.New: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		OAuth2: provider,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	_, err = svc.BeginOAuthLogin(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for unknown OAuth2 provider")
	}
}

// TestService_OAuth2_NilProvider_Guards verifies that a Service without
// an OAuth2 provider returns clear errors, not panics.
func TestService_OAuth2_NilProvider_Guards(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	_, err = svc.BeginOAuthLogin(context.Background(), "google")
	if err == nil {
		t.Error("expected error when OAuth2 is not configured (BeginOAuthLogin)")
	}
}

// newMockOAuth2Server creates a fake OAuth2 provider server for FinishLogin tests.
// It serves token + userinfo endpoints that return predictable responses.
func newMockOAuth2Server(t *testing.T, userInfo map[string]any) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "test-auth-code" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
		})
	})

	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userInfo)
	})

	server := httptest.NewServer(mux)
	return server.URL, server.Close
}

// TestService_OAuth2_FinishLogin_Integration is the critical cross-module test:
// Service.FinishOAuthLogin → OAuth2StateStore.Consume → Provider.FinishLogin
// (mock token exchange + userinfo) → matchOrCreateUser → session creation.
//
// This proves the FULL OAuth2 login flow works end-to-end through the module
// boundary: a real oauth2.Provider hitting a mock OAuth2 server, wired through
// the usermgmt.Service which handles user matching/registration + sessions.
func TestService_OAuth2_FinishLogin_Integration(t *testing.T) {
	t.Parallel()

	serverURL, cleanup := newMockOAuth2Server(t, map[string]any{
		"id":             "github-sub-42",
		"email":          "integration-test@example.com",
		"name":           "Integration Test User",
		"email_verified": true,
	})
	defer cleanup()

	provider, err := usermgmtoauth2.New(context.Background(), usermgmtoauth2.Config{
		Providers: map[string]usermgmtoauth2.ProviderConfig{
			"github": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:8080/callback",
				AuthURL:      serverURL + "/auth",
				TokenURL:     serverURL + "/token",
				UserInfoURL:  serverURL + "/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("oauth2.New: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		OAuth2: provider,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	// Step 1: Begin login to get a valid state token
	beginResp, err := svc.BeginOAuthLogin(context.Background(), "github")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}

	// Extract state from the redirect URL
	parsedURL, err := url.Parse(beginResp.RedirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	state := parsedURL.Query().Get("state")
	if state == "" {
		t.Fatal("redirect URL missing state parameter")
	}

	// Step 2: Finish login with the state + fake auth code
	result, err := svc.FinishOAuthLogin(context.Background(), "github", "test-auth-code", state)
	if err != nil {
		t.Fatalf("FinishOAuthLogin: %v", err)
	}

	// Step 3: Verify user was created correctly
	if result.User == nil {
		t.Fatal("expected non-nil user")
	}
	if result.User.Email != "integration-test@example.com" {
		t.Errorf("user email = %q, want %q", result.User.Email, "integration-test@example.com")
	}
	if result.User.DisplayName != "Integration Test User" {
		t.Errorf("user display name = %q, want %q", result.User.DisplayName, "Integration Test User")
	}
	if result.Session == nil {
		t.Fatal("expected non-nil session")
	}
	if result.Session.Token == "" {
		t.Error("expected non-empty session token")
	}
}

// TestService_OAuth2_FinishLogin_RejectsInvalidState verifies that FinishOAuthLogin
// rejects a state token that was never stored (CSRF protection).
func TestService_OAuth2_FinishLogin_RejectsInvalidState(t *testing.T) {
	t.Parallel()

	serverURL, cleanup := newMockOAuth2Server(t, map[string]any{
		"id":    "1",
		"email": "test@example.com",
	})
	defer cleanup()

	provider, err := usermgmtoauth2.New(context.Background(), usermgmtoauth2.Config{
		Providers: map[string]usermgmtoauth2.ProviderConfig{
			"github": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:8080/callback",
				AuthURL:      serverURL + "/auth",
				TokenURL:     serverURL + "/token",
				UserInfoURL:  serverURL + "/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("oauth2.New: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		OAuth2: provider,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	_, err = svc.FinishOAuthLogin(context.Background(), "github", "test-auth-code", "never-stored-state")
	if err == nil {
		t.Fatal("expected error for invalid state token")
	}
}

// TestService_OAuth2_FinishLogin_RejectsProviderMismatch verifies that
// FinishOAuthLogin rejects when the callback provider differs from the
// provider stored in the state token.
func TestService_OAuth2_FinishLogin_RejectsProviderMismatch(t *testing.T) {
	t.Parallel()

	serverURL, cleanup := newMockOAuth2Server(t, map[string]any{
		"id":    "1",
		"email": "test@example.com",
	})
	defer cleanup()

	provider, err := usermgmtoauth2.New(context.Background(), usermgmtoauth2.Config{
		Providers: map[string]usermgmtoauth2.ProviderConfig{
			"github": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:8080/callback",
				AuthURL:      serverURL + "/auth",
				TokenURL:     serverURL + "/token",
				UserInfoURL:  serverURL + "/userinfo",
			},
			"google": {
				ClientID:     "test-client-id-2",
				ClientSecret: "test-client-secret-2",
				RedirectURL:  "http://localhost:8080/callback",
				AuthURL:      serverURL + "/auth",
				TokenURL:     serverURL + "/token",
				UserInfoURL:  serverURL + "/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("oauth2.New: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		OAuth2: provider,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	// Begin with "github"
	beginResp, err := svc.BeginOAuthLogin(context.Background(), "github")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}
	parsedURL, _ := url.Parse(beginResp.RedirectURL)
	state := parsedURL.Query().Get("state")

	// Try to finish with "google" — should fail
	_, err = svc.FinishOAuthLogin(context.Background(), "google", "test-auth-code", state)
	if err == nil {
		t.Fatal("expected error for provider mismatch")
	}
}
