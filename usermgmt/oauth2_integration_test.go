package usermgmt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fakeOAuth2Provider is an httptest-based mock OAuth2 provider for integration tests.
type fakeOAuth2Provider struct {
	server       *httptest.Server
	authURL      string
	tokenURL     string
	userInfoURL  string
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
}

// newFakeOAuth2Provider creates a fake OAuth2 provider with working token + userinfo endpoints.
func newFakeOAuth2Provider(t *testing.T) *fakeOAuth2Provider {
	t.Helper()
	mux := http.NewServeMux()
	prov := &fakeOAuth2Provider{
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		redirectURL:  "http://localhost:8080/auth/oauth/fake/callback",
		scopes:       []string{"email", "profile"},
	}

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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             "12345",
			"email":          "fakeuser@example.com",
			"name":           "Fake User",
			"email_verified": true,
		})
	})

	prov.server = httptest.NewServer(mux)
	prov.authURL = prov.server.URL + "/auth"
	prov.tokenURL = prov.server.URL + "/token"
	prov.userInfoURL = prov.server.URL + "/userinfo"

	t.Cleanup(prov.server.Close)
	return prov
}

// toOAuth2ProviderConfig returns the fake provider's config for ServiceConfig.
func (p *fakeOAuth2Provider) toOAuth2ProviderConfig() OAuth2ProviderConfig {
	return OAuth2ProviderConfig{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  p.redirectURL,
		Scopes:       p.scopes,
		AuthURL:      p.authURL,
		TokenURL:     p.tokenURL,
		UserInfoURL:  p.userInfoURL,
	}
}

// newOAuth2Service creates a Service with the fake OAuth2 provider configured.
func newOAuth2Service(t *testing.T, prov *fakeOAuth2Provider) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{
		OAuth2Config: &OAuth2Config{
			Providers: map[string]OAuth2ProviderConfig{
				"fake": prov.toOAuth2ProviderConfig(),
			},
			StateTTL: 5 * time.Minute,
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	return svc
}

func TestBeginOAuthLogin_ReturnsRedirectURL(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	resp, err := svc.BeginOAuthLogin(context.Background(), "fake")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}
	if resp.RedirectURL == "" {
		t.Fatal("expected non-empty redirect URL")
	}

	u, err := url.Parse(resp.RedirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	if u.Query().Get("client_id") != prov.clientID {
		t.Errorf("client_id = %q", u.Query().Get("client_id"))
	}
	if u.Query().Get("response_type") != "code" {
		t.Errorf("response_type = %q", u.Query().Get("response_type"))
	}
	if u.Query().Get("code_challenge_method") != "S256" {
		t.Errorf("missing PKCE S256 challenge")
	}
}

func TestFinishOAuthLogin_AutoRegister(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	begin, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u, _ := url.Parse(begin.RedirectURL)
	state := u.Query().Get("state")

	resp, err := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", state)
	if err != nil {
		t.Fatalf("FinishOAuthLogin: %v", err)
	}
	if resp.User == nil || resp.Session == nil {
		t.Fatal("expected user and session")
	}
	if resp.User.Email != "fakeuser@example.com" {
		t.Errorf("Email = %q", resp.User.Email)
	}
	if len(resp.User.ExternalAccounts) != 1 {
		t.Fatalf("expected 1 external account, got %d", len(resp.User.ExternalAccounts))
	}
	ea := resp.User.ExternalAccounts[0]
	if ea.Provider != "fake" || ea.Subject != "12345" {
		t.Errorf("account = %q/%q", ea.Provider, ea.Subject)
	}
	if !resp.User.EmailVerified {
		t.Error("expected EmailVerified = true from OAuth provider")
	}
}

func TestFinishOAuthLogin_LinkExistingUser(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	// Register a user with the same email first
	_, err := svc.Register(context.Background(), RegisterRequest{
		ID:          NewUserID("01J0000000000000000000000"),
		Email:       "fakeuser@example.com",
		DisplayName: "Existing User",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Now login via OAuth2 — should link to existing user
	begin, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u, _ := url.Parse(begin.RedirectURL)
	state := u.Query().Get("state")

	resp, err := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", state)
	if err != nil {
		t.Fatalf("FinishOAuthLogin: %v", err)
	}
	if resp.User == nil {
		t.Fatal("expected user")
	}
	if resp.User.Email != "fakeuser@example.com" {
		t.Errorf("Email = %q", resp.User.Email)
	}
	if len(resp.User.ExternalAccounts) != 1 {
		t.Fatalf("expected 1 external account, got %d", len(resp.User.ExternalAccounts))
	}
}

func TestFinishOAuthLogin_DuplicateReLogin(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	// First login (auto-registers)
	begin1, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u1, _ := url.Parse(begin1.RedirectURL)
	state1 := u1.Query().Get("state")
	resp1, _ := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", state1)
	if resp1 == nil {
		t.Fatal("first login failed")
	}

	// Second login with same email — should not error (idempotent link)
	begin2, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u2, _ := url.Parse(begin2.RedirectURL)
	state2 := u2.Query().Get("state")
	resp2, err := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", state2)
	if err != nil {
		t.Fatalf("second login failed: %v", err)
	}
	if resp2.User.ID.Get() != resp1.User.ID.Get() {
		t.Fatal("second login created a different user")
	}
	if len(resp2.User.ExternalAccounts) != 1 {
		t.Fatalf("expected still 1 external account, got %d", len(resp2.User.ExternalAccounts))
	}
}

func TestFinishOAuthLogin_InvalidState(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	_, err := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", "bad-state")
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
}

func TestBeginOAuthLogin_UnknownProvider(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	_, err := svc.BeginOAuthLogin(context.Background(), "unknown")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestUnlinkExternalAccount(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	// First login to create user + link
	begin, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u, _ := url.Parse(begin.RedirectURL)
	state := u.Query().Get("state")
	resp, _ := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", state)
	if resp == nil {
		t.Fatal("login failed")
	}

	// Add a WebAuthn credential so unlinking is allowed
	aggID, _ := aggIDFromUser(resp.User.ID)
	_ = svc.dispatcher.Dispatch(context.Background(), NewAddCredentialCmd(aggID, WebAuthnCredential{
		ID:              []byte{1, 2, 3},
		PublicKey:       []byte{4, 5, 6},
		AttestationType: "none",
	}))

	// Unlink should succeed
	if err := svc.UnlinkExternalAccount(context.Background(), resp.User.ID, "fake"); err != nil {
		t.Fatalf("UnlinkExternalAccount: %v", err)
	}

	user, ok := svc.readModel.FindByUserID(resp.User.ID)
	if !ok {
		t.Fatal("user not found after unlink")
	}
	if len(user.ExternalAccounts) != 0 {
		t.Fatalf("expected 0 external accounts, got %d", len(user.ExternalAccounts))
	}
}

func TestUnlinkExternalAccount_NotFound(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	err := svc.UnlinkExternalAccount(context.Background(), NewUserID("01J0000000000000000000000"), "fake")
	if err == nil {
		t.Fatal("expected error for unlinking non-existent user")
	}
}

func TestOAuth2Provider_Exchange_InvalidCode(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	begin, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u, _ := url.Parse(begin.RedirectURL)
	state := u.Query().Get("state")

	_, err := svc.FinishOAuthLogin(context.Background(), "fake", "wrong-code", state)
	if err == nil {
		t.Fatal("expected error for invalid code")
	}
}

func TestOAuth2Provider_Exchange_NoEmail(t *testing.T) {
	t.Helper()
	mux := http.NewServeMux()
	prov := &fakeOAuth2Provider{
		clientID:     "test-id",
		clientSecret: "test-secret",
		redirectURL:  "http://localhost/callback",
		scopes:       []string{"email"},
	}
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "token",
			"token_type":   "Bearer",
		})
	})
	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":   "99",
			"name": "No Email User",
		})
	})
	prov.server = httptest.NewServer(mux)
	prov.authURL = prov.server.URL + "/auth"
	prov.tokenURL = prov.server.URL + "/token"
	prov.userInfoURL = prov.server.URL + "/userinfo"
	t.Cleanup(prov.server.Close)

	svc, err := NewService(ServiceConfig{
		OAuth2Config: &OAuth2Config{
			Providers: map[string]OAuth2ProviderConfig{
				"noemail": prov.toOAuth2ProviderConfig(),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	begin, _ := svc.BeginOAuthLogin(context.Background(), "noemail")
	u, _ := url.Parse(begin.RedirectURL)
	state := u.Query().Get("state")

	_, err = svc.FinishOAuthLogin(context.Background(), "noemail", "test-auth-code", state)
	if err == nil {
		t.Fatal("expected error when provider returns no email")
	}
}
