package integration_test

import (
	"context"
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
