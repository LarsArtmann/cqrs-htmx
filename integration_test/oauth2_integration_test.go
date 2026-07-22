package integration_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	usermgmtoauth2 "github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

const (
	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
	testRedirectURL  = "http://localhost:8080/callback"
	testGitHub       = "github"
	testUserEmail    = "test@example.com"
)

// githubProviderConfig returns a ProviderConfig pointing at the given server URL.
func githubProviderConfig(serverURL string) usermgmtoauth2.ProviderConfig {
	return usermgmtoauth2.ProviderConfig{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURL:  testRedirectURL,
		AuthURL:      serverURL + "/auth",
		TokenURL:     serverURL + "/token",
		UserInfoURL:  serverURL + "/userinfo",
	}
}

// newOAuth2Service creates a Service wired with the given OAuth2 providers.
func newOAuth2Service(t *testing.T, providers map[string]usermgmtoauth2.ProviderConfig) *usermgmt.Service {
	t.Helper()
	provider, err := usermgmtoauth2.New(context.Background(), usermgmtoauth2.Config{
		Providers: providers,
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
	return svc
}

// beginLoginAndExtractState calls BeginOAuthLogin and extracts the state param.
func beginLoginAndExtractState(t *testing.T, svc *usermgmt.Service, provider string) string {
	t.Helper()
	resp, err := svc.BeginOAuthLogin(context.Background(), provider)
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}
	parsedURL, err := url.Parse(resp.RedirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	state := parsedURL.Query().Get("state")
	if state == "" {
		t.Fatal("redirect URL missing state parameter")
	}
	return state
}

// TestService_OAuth2_BeginLogin_Integration verifies the cross-module
// flow: Service.BeginOAuthLogin → generateOAuth2State →
// oauth2.Provider.BeginLogin (PKCE + redirect URL).
func TestService_OAuth2_BeginLogin_Integration(t *testing.T) {
	t.Parallel()

	svc := newOAuth2Service(t, map[string]usermgmtoauth2.ProviderConfig{
		testGitHub: githubProviderConfig("https://github.com"),
	})

	resp, err := svc.BeginOAuthLogin(context.Background(), testGitHub)
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}
	if resp.RedirectURL == "" {
		t.Fatal("expected non-empty redirect URL")
	}
	if !strings.Contains(resp.RedirectURL, "client_id="+testClientID) {
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

	svc := newOAuth2Service(t, map[string]usermgmtoauth2.ProviderConfig{
		"google": { //nolint:gosec // test fixture
			ClientID:     testClientID,
			ClientSecret: testClientSecret,
			RedirectURL:  testRedirectURL,
			AuthURL:      "https://accounts.google.com/o/oauth2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		},
	})

	_, err := svc.BeginOAuthLogin(context.Background(), "nonexistent")
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
func newMockOAuth2Server(t *testing.T, userInfo map[string]any) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "test-auth-code" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.MarshalWrite(w, map[string]string{"error": "invalid_grant"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, map[string]string{
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
		_ = json.MarshalWrite(w, userInfo)
	})

	server := httptest.NewServer(mux)
	return server.URL, server.Close
}

// TestService_OAuth2_FinishLogin_Integration is the critical cross-module test:
// Service.FinishOAuthLogin → OAuth2StateStore.Consume → Provider.FinishLogin
// (mock token exchange + userinfo) → matchOrCreateUser → session creation.
func TestService_OAuth2_FinishLogin_Integration(t *testing.T) {
	t.Parallel()

	const testEmail = "integration-test@example.com"
	serverURL, cleanup := newMockOAuth2Server(t, map[string]any{
		"id":             "github-sub-42",
		"email":          testEmail, //nolint:goconst // mock JSON key
		"name":           "Integration Test User",
		"email_verified": true,
	})
	defer cleanup()

	svc := newOAuth2Service(t, map[string]usermgmtoauth2.ProviderConfig{
		testGitHub: githubProviderConfig(serverURL),
	})

	state := beginLoginAndExtractState(t, svc, testGitHub)

	result, err := svc.FinishOAuthLogin(
		context.Background(), testGitHub, "test-auth-code", state,
	)
	if err != nil {
		t.Fatalf("FinishOAuthLogin: %v", err)
	}
	if result.User == nil {
		t.Fatal("expected non-nil user")
	}
	if result.User.Email != testEmail {
		t.Errorf("user email = %q, want %q", result.User.Email, testEmail)
	}
	if result.Session == nil || result.Session.Token == "" {
		t.Error("expected non-nil session with non-empty token")
	}
}

// TestService_OAuth2_FinishLogin_RejectsInvalidState verifies that FinishOAuthLogin
// rejects a state token that was never stored (CSRF protection).
func TestService_OAuth2_FinishLogin_RejectsInvalidState(t *testing.T) {
	t.Parallel()

	serverURL, cleanup := newMockOAuth2Server(t, map[string]any{
		"id": "1", "email": testUserEmail,
	})
	defer cleanup()

	svc := newOAuth2Service(t, map[string]usermgmtoauth2.ProviderConfig{
		testGitHub: githubProviderConfig(serverURL),
	})

	_, err := svc.FinishOAuthLogin(
		context.Background(), testGitHub, "test-auth-code", "never-stored-state",
	)
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
		"id": "1", "email": testUserEmail,
	})
	defer cleanup()

	svc := newOAuth2Service(t, map[string]usermgmtoauth2.ProviderConfig{
		testGitHub: githubProviderConfig(serverURL),
		"google": {
			ClientID:     "test-client-id-2",
			ClientSecret: "test-client-secret-2",
			RedirectURL:  testRedirectURL,
			AuthURL:      serverURL + "/auth",
			TokenURL:     serverURL + "/token",
			UserInfoURL:  serverURL + "/userinfo",
		},
	})

	state := beginLoginAndExtractState(t, svc, testGitHub)

	_, err := svc.FinishOAuthLogin(
		context.Background(), "google", "test-auth-code", state,
	)
	if err == nil {
		t.Fatal("expected error for provider mismatch")
	}
}
