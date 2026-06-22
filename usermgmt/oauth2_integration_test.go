package usermgmt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// requireTestAuthCode writes an OAuth2 invalid_grant error response (RFC 6749
// §5.2) and returns false if r does not carry the expected test auth code.
// Returns true when the caller should proceed with token issuance. Shared by
// the pure-OAuth2 and OIDC mock token endpoints so the rejection shape cannot
// drift between them.
func requireTestAuthCode(w http.ResponseWriter, r *http.Request) bool {
	if r.PostFormValue("code") != "test-auth-code" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"}) //nolint:errchkjson // test code
		return false
	}
	return true
}

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
	// userInfo controls what the /userinfo endpoint returns. Tests can mutate
	// this between logins to simulate provider-side changes (e.g. email change).
	userInfo map[string]any
}

// newFakeOAuth2Provider creates a fake OAuth2 provider with working token + userinfo endpoints.
func newFakeOAuth2Provider(t *testing.T) *fakeOAuth2Provider {
	t.Helper()
	prov := &fakeOAuth2Provider{
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		redirectURL:  "http://localhost:8080/auth/oauth/fake/callback",
		scopes:       []string{"email", "profile"},
		userInfo: map[string]any{
			"id":             "12345",
			"email":          "fakeuser@example.com",
			"name":           "Fake User",
			"email_verified": true,
		},
	}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if !requireTestAuthCode(w, r) {
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
		_ = json.NewEncoder(w).Encode(prov.userInfo)
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
	if resp2.User.ID.Get().String() != resp1.User.ID.Get().String() {
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
		credentialCore: credentialCore{
			ID:              []byte{1, 2, 3},
			PublicKey:       []byte{4, 5, 6},
			AttestationType: "none",
		},
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

// TestFinishOAuthLogin_SubjectMatchOnRelogin verifies that when a user's email
// changes at the provider, re-login still finds them by their stable provider+subject.
func TestFinishOAuthLogin_SubjectMatchOnRelogin(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	// First login — auto-registers with subject "12345", email "fakeuser@example.com"
	begin1, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u1, _ := url.Parse(begin1.RedirectURL)
	resp1, err := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", u1.Query().Get("state"))
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	originalID := resp1.User.ID.Get().String()

	// Provider changes the user's email (same subject "12345")
	prov.userInfo["email"] = "changed@example.com"

	// Second login — must find the same user by subject, NOT create a new one
	begin2, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u2, _ := url.Parse(begin2.RedirectURL)
	resp2, err := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", u2.Query().Get("state"))
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if resp2.User.ID.Get().String() != originalID {
		t.Fatalf("expected same user ID %s, got %s (subject matching failed)",
			originalID, resp2.User.ID.Get().String())
	}
}

// TestLinkExternalAccount_AlreadyLinkedToOtherUser verifies that linking a
// provider+subject that belongs to a different user is rejected.
func TestLinkExternalAccount_AlreadyLinkedToOtherUser(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	// OAuth login auto-registers user with subject "12345"
	begin, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u, _ := url.Parse(begin.RedirectURL)
	resp, err := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", u.Query().Get("state"))
	if err != nil {
		t.Fatalf("first login: %v", err)
	}

	// Register a second user with a different email
	_, err = svc.Register(context.Background(), RegisterRequest{
		ID:          NewUserID("01J0000000000000000000001"),
		Email:       "other@example.com",
		DisplayName: "Other User",
	})
	if err != nil {
		t.Fatalf("Register second user: %v", err)
	}

	// Try to link "fake"/"12345" (which belongs to user 1) to user 2
	otherUserID := NewUserID("01J0000000000000000000001")
	err = svc.linkExternalAccount(context.Background(), otherUserID, "fake", oauth2UserInfo{
		Subject:       "12345",
		Email:         "other@example.com",
		EmailVerified: true,
	})
	if !errors.Is(err, ErrExternalAccountAlreadyLinked) {
		t.Fatalf("expected ErrExternalAccountAlreadyLinked, got: %v", err)
	}

	// Verify user 1 still has the link, user 2 does not
	user1, _ := svc.readModel.FindByUserID(resp.User.ID)
	if len(user1.ExternalAccounts) != 1 {
		t.Errorf("user1 should still have 1 external account, got %d", len(user1.ExternalAccounts))
	}
	user2, _ := svc.readModel.FindByUserID(otherUserID)
	if len(user2.ExternalAccounts) != 0 {
		t.Errorf("user2 should have 0 external accounts, got %d", len(user2.ExternalAccounts))
	}
}

// TestReadModel_FindByExternalAccount tests the read model's external account index directly.
func TestReadModel_FindByExternalAccount(t *testing.T) {
	prov := newFakeOAuth2Provider(t)
	svc := newOAuth2Service(t, prov)

	// No match initially
	_, ok := svc.readModel.FindByExternalAccount("fake", "12345")
	if ok {
		t.Fatal("expected no match before any login")
	}

	// Login to create + link
	begin, _ := svc.BeginOAuthLogin(context.Background(), "fake")
	u, _ := url.Parse(begin.RedirectURL)
	resp, _ := svc.FinishOAuthLogin(context.Background(), "fake", "test-auth-code", u.Query().Get("state"))

	// Now should find by provider+subject
	found, ok := svc.readModel.FindByExternalAccount("fake", "12345")
	if !ok {
		t.Fatal("expected to find user by external account after login")
	}
	if found.ID.Get().String() != resp.User.ID.Get().String() {
		t.Errorf("found wrong user: got %s, want %s", found.ID.Get().String(), resp.User.ID.Get().String())
	}

	// Different subject should not match
	_, ok = svc.readModel.FindByExternalAccount("fake", "99999")
	if ok {
		t.Fatal("expected no match for unknown subject")
	}

	// Different provider should not match
	_, ok = svc.readModel.FindByExternalAccount("google", "12345")
	if ok {
		t.Fatal("expected no match for unknown provider")
	}
}

// TestMultiProvider_Login verifies that logging in via different providers
// with the same email links both to the same user.
func TestMultiProvider_Login(t *testing.T) {
	google := newFakeOAuth2Provider(t)
	github := newFakeOAuth2Provider(t)

	svc, err := NewService(ServiceConfig{
		OAuth2Config: &OAuth2Config{
			Providers: map[string]OAuth2ProviderConfig{
				"google": google.toOAuth2ProviderConfig(),
				"github": github.toOAuth2ProviderConfig(),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	// Both providers return the same email but different subjects
	google.userInfo["email"] = "shared@example.com"
	google.userInfo["sub"] = "google-sub-001"
	github.userInfo["email"] = "shared@example.com"
	github.userInfo["sub"] = "github-sub-002"
	github.userInfo["id"] = "67890"

	// Login via Google → auto-registers
	beginG, _ := svc.BeginOAuthLogin(context.Background(), "google")
	uG, _ := url.Parse(beginG.RedirectURL)
	respG, err := svc.FinishOAuthLogin(context.Background(), "google", "test-auth-code", uG.Query().Get("state"))
	if err != nil {
		t.Fatalf("google login: %v", err)
	}
	if len(respG.User.ExternalAccounts) != 1 {
		t.Fatalf("expected 1 external account after google, got %d", len(respG.User.ExternalAccounts))
	}

	// Login via GitHub (same email) → should LINK to existing user
	beginH, _ := svc.BeginOAuthLogin(context.Background(), "github")
	uH, _ := url.Parse(beginH.RedirectURL)
	respH, err := svc.FinishOAuthLogin(context.Background(), "github", "test-auth-code", uH.Query().Get("state"))
	if err != nil {
		t.Fatalf("github login: %v", err)
	}
	if respH.User.ID.Get().String() != respG.User.ID.Get().String() {
		t.Fatal("expected github login to link to same user as google")
	}
	if len(respH.User.ExternalAccounts) != 2 {
		t.Fatalf("expected 2 external accounts, got %d", len(respH.User.ExternalAccounts))
	}

	// Verify both providers are present
	providers := map[string]bool{}
	for _, ea := range respH.User.ExternalAccounts {
		providers[ea.Provider] = true
	}
	if !providers["google"] || !providers["github"] {
		t.Errorf("expected both google and github, got %v", providers)
	}
}
