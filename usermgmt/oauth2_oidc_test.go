package usermgmt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// fakeOIDCProvider is an httptest-based mock OIDC provider with real JWT signing.
// It serves discovery, JWKS, token, and userinfo endpoints.
type fakeOIDCProvider struct {
	server *httptest.Server
	issuer string
	signer jose.Signer
	jwks   jose.JSONWebKeySet
	// idTokenClaims controls the claims placed in the signed ID token.
	// Tests can mutate this between logins (e.g., to change email).
	idTokenClaims map[string]any
}

// newFakeOIDCProvider creates a mock OIDC provider with a real RSA signing key.
func newFakeOIDCProvider(t *testing.T) *fakeOIDCProvider {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	keyID := "test-oidc-key"
	jwk := jose.JSONWebKey{
		Key:       privKey.Public(),
		KeyID:     keyID,
		Algorithm: string(jose.RS256),
	}

	opts := &jose.SignerOptions{}
	opts = opts.WithHeader("kid", keyID)
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privKey},
		opts,
	)
	if err != nil {
		t.Fatalf("create jose signer: %v", err)
	}

	prov := &fakeOIDCProvider{
		signer: signer,
		jwks:   jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}},
		idTokenClaims: map[string]any{
			"sub":            "oidc-sub-67890",
			"email":          "oidcuser@example.com",
			"email_verified": true,
			"name":           "OIDC User",
		},
	}

	mux := http.NewServeMux()
	prov.server = httptest.NewServer(mux)
	prov.issuer = prov.server.URL
	prov.registerHandlers(mux)

	t.Cleanup(prov.server.Close)
	return prov
}

// registerHandlers sets up the discovery, JWKS, and token endpoints on the mux.
func (p *fakeOIDCProvider) registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                p.issuer,
			"authorization_endpoint":                p.issuer + "/auth",
			"token_endpoint":                        p.issuer + "/token",
			"jwks_uri":                              p.issuer + "/jwks",
			"userinfo_endpoint":                     p.issuer + "/userinfo",
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
		})
	})

	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p.jwks)
	})

	mux.HandleFunc("POST /token", p.handleToken)
}

// handleToken exchanges the auth code for a signed JWT ID token.
func (p *fakeOIDCProvider) handleToken(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.PostFormValue("code") != "test-auth-code" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"}) //nolint:errchkjson // test code
		return
	}

	now := time.Now()
	claims := map[string]any{
		"iss": p.issuer,
		"aud": "test-client-id",
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	}
	for k, v := range p.idTokenClaims {
		claims[k] = v
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	signed, err := p.signer.Sign(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	idToken, err := signed.CompactSerialize()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{ //nolint:errchkjson // test code
		"access_token": "test-access-token",
		"token_type":   "Bearer",
		"id_token":     idToken,
	})
}

func (p *fakeOIDCProvider) toOIDCConfig() OAuth2ProviderConfig {
	return OAuth2ProviderConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/auth/oauth/oidc/callback",
		Scopes:       []string{"openid", "email", "profile"},
		IssuerURL:    p.issuer,
	}
}

func newOIDCService(t *testing.T, prov *fakeOIDCProvider) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{
		OAuth2Config: &OAuth2Config{
			Providers: map[string]OAuth2ProviderConfig{
				"oidc": prov.toOIDCConfig(),
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

// TestOIDC_LoginViaIDToken verifies the full OIDC flow: discovery → PKCE →
// token exchange with signed JWT → ID token verification → user creation.
func TestOIDC_LoginViaIDToken(t *testing.T) {
	prov := newFakeOIDCProvider(t)
	svc := newOIDCService(t, prov)

	begin, err := svc.BeginOAuthLogin(context.Background(), "oidc")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}
	u, _ := url.Parse(begin.RedirectURL)
	state := u.Query().Get("state")

	resp, err := svc.FinishOAuthLogin(context.Background(), "oidc", "test-auth-code", state)
	if err != nil {
		t.Fatalf("FinishOAuthLogin: %v", err)
	}
	if resp.User == nil {
		t.Fatal("expected user")
	}
	if resp.User.Email != "oidcuser@example.com" {
		t.Errorf("Email = %q, want oidcuser@example.com", resp.User.Email)
	}
	if len(resp.User.ExternalAccounts) != 1 {
		t.Fatalf("expected 1 external account, got %d", len(resp.User.ExternalAccounts))
	}
	ea := resp.User.ExternalAccounts[0]
	if ea.Provider != "oidc" || ea.Subject != "oidc-sub-67890" {
		t.Errorf("account = %q/%q", ea.Provider, ea.Subject)
	}
	if !resp.User.EmailVerified {
		t.Error("expected EmailVerified from OIDC email_verified claim")
	}
}

// TestOIDC_RelognWithChangedEmail verifies subject-first matching through OIDC.
func TestOIDC_RelognWithChangedEmail(t *testing.T) {
	prov := newFakeOIDCProvider(t)
	svc := newOIDCService(t, prov)

	// First login
	begin1, _ := svc.BeginOAuthLogin(context.Background(), "oidc")
	u1, _ := url.Parse(begin1.RedirectURL)
	resp1, _ := svc.FinishOAuthLogin(context.Background(), "oidc", "test-auth-code", u1.Query().Get("state"))
	originalID := resp1.User.ID.Get()

	// Change email at provider (same subject)
	prov.idTokenClaims["email"] = "changed-oidc@example.com"

	// Second login — same subject, different email → same user
	begin2, _ := svc.BeginOAuthLogin(context.Background(), "oidc")
	u2, _ := url.Parse(begin2.RedirectURL)
	resp2, err := svc.FinishOAuthLogin(context.Background(), "oidc", "test-auth-code", u2.Query().Get("state"))
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if resp2.User.ID.Get() != originalID {
		t.Fatalf("expected same user, got different ID")
	}
}

// TestOIDC_InvalidIDToken verifies that a token with wrong issuer is rejected.
func TestOIDC_InvalidIDToken(t *testing.T) {
	prov := newFakeOIDCProvider(t)
	svc := newOIDCService(t, prov)

	// Tamper with the issuer claim so verification fails
	prov.idTokenClaims["iss"] = "https://evil.example.com"

	begin, _ := svc.BeginOAuthLogin(context.Background(), "oidc")
	u, _ := url.Parse(begin.RedirectURL)
	state := u.Query().Get("state")

	_, err := svc.FinishOAuthLogin(context.Background(), "oidc", "test-auth-code", state)
	if err == nil {
		t.Fatal("expected error for invalid ID token (wrong issuer)")
	}
}

// TestInitOAuth2Provider_OIDCDiscovery verifies that OIDC discovery
// correctly initializes the provider endpoints and verifier.
func TestInitOAuth2Provider_OIDCDiscovery(t *testing.T) {
	prov := newFakeOIDCProvider(t)

	p, err := initOAuth2Provider(context.Background(), "google", prov.toOIDCConfig())
	if err != nil {
		t.Fatalf("initOAuth2Provider: %v", err)
	}
	if p.oidcProvider == nil {
		t.Error("expected oidcProvider to be discovered")
	}
	if p.verifier == nil {
		t.Error("expected ID token verifier to be initialized")
	}
	if p.userInfoURL != "" {
		t.Error("expected empty userInfoURL for OIDC provider (uses ID token instead)")
	}
	if p.config.Endpoint.AuthURL != prov.issuer+"/auth" {
		t.Errorf("AuthURL = %q, want %q", p.config.Endpoint.AuthURL, prov.issuer+"/auth")
	}
	if p.config.Endpoint.TokenURL != prov.issuer+"/token" {
		t.Errorf("TokenURL = %q, want %q", p.config.Endpoint.TokenURL, prov.issuer+"/token")
	}
}

// TestInitOAuth2Provider_PureOAuth2 verifies that explicit endpoints work
// without OIDC discovery.
func TestInitOAuth2Provider_PureOAuth2(t *testing.T) {
	cfg := OAuth2ProviderConfig{
		ClientID:     "test",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
	}
	p, err := initOAuth2Provider(context.Background(), "github", cfg)
	if err != nil {
		t.Fatalf("initOAuth2Provider: %v", err)
	}
	if p.oidcProvider != nil {
		t.Error("expected nil oidcProvider for pure OAuth2")
	}
	if p.verifier != nil {
		t.Error("expected nil verifier for pure OAuth2")
	}
	if p.userInfoURL != "https://example.com/userinfo" {
		t.Errorf("userInfoURL = %q", p.userInfoURL)
	}
}
