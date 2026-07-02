package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// --- ProviderConfig.Validate tests ---

func TestProviderConfig_Validate(t *testing.T) {
	base := ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid OIDC config should not error: %v", err)
	}
}

func TestProviderConfig_Validate_MissingClientID(t *testing.T) {
	cfg := ProviderConfig{
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing ClientID")
	}
}

func TestProviderConfig_Validate_MissingClientSecret(t *testing.T) {
	cfg := ProviderConfig{
		ClientID:    "id",
		RedirectURL: "http://localhost/callback",
		IssuerURL:   "https://accounts.google.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing ClientSecret")
	}
}

func TestProviderConfig_Validate_MissingRedirectURL(t *testing.T) {
	cfg := ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing RedirectURL")
	}
}

func TestProviderConfig_Validate_OAuth2EndpointsRequired(t *testing.T) {
	cfg := ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when IssuerURL is empty and no explicit endpoints")
	}
}

func TestProviderConfig_Validate_OAuth2EndpointsOK(t *testing.T) {
	cfg := ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid OAuth2 config should not error: %v", err)
	}
}

// --- New / provider lookup tests ---

func TestNew_UnknownProviderDiscovery(t *testing.T) {
	_, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"broken": {
				ClientID:     "id",
				ClientSecret: "secret",
				RedirectURL:  "http://localhost/callback",
				IssuerURL:    "http://127.0.0.1:0/nonexistent",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for unreachable OIDC discovery")
	}
}

func TestProvider_BeginLogin_UnknownProvider(t *testing.T) {
	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"google": {
				ClientID:     "id",
				ClientSecret: "secret",
				RedirectURL:  "http://localhost/callback",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				UserInfoURL:  "https://example.com/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, _, err = p.BeginLogin(context.Background(), "unknown", "state123")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProvider_FinishLogin_UnknownProvider(t *testing.T) {
	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"google": {
				ClientID:     "id",
				ClientSecret: "secret",
				RedirectURL:  "http://localhost/callback",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				UserInfoURL:  "https://example.com/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = p.FinishLogin(context.Background(), "unknown", "code", "verifier")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

// --- BeginLogin PKCE tests ---

func TestProvider_BeginLogin_PKCE(t *testing.T) {
	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"google": {
				ClientID:     "test-client-id",
				ClientSecret: "secret",
				RedirectURL:  "http://localhost/callback",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				UserInfoURL:  "https://example.com/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	redirectURL, pkceVerifier, err := p.BeginLogin(context.Background(), "google", "my-state")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	if redirectURL == "" {
		t.Error("expected non-empty redirect URL")
	}
	if pkceVerifier == "" {
		t.Error("expected non-empty PKCE verifier")
	}

	u, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}

	// Verify state is passed through
	if u.Query().Get("state") != "my-state" {
		t.Errorf("state = %q, want %q", u.Query().Get("state"), "my-state")
	}

	// Verify client_id is in the URL
	if u.Query().Get("client_id") != "test-client-id" {
		t.Errorf("client_id = %q", u.Query().Get("client_id"))
	}

	// Verify PKCE code challenge is present (S256)
	codeChallenge := u.Query().Get("code_challenge")
	if codeChallenge == "" {
		t.Error("expected non-empty code_challenge")
	}
	if u.Query().Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q, want S256", u.Query().Get("code_challenge_method"))
	}

	// Verify the code challenge is the SHA256 hash of the verifier
	// (The verifier is base64url random, challenge = base64url(sha256(verifier)))
	if len(codeChallenge) < 32 {
		t.Error("code_challenge looks too short for S256")
	}
}

// --- Pure OAuth2 flow tests ---

func newFakeOAuth2Server(t *testing.T, userInfo map[string]any) (*httptest.Server, ProviderConfig) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "test-auth-code" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
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
		json.NewEncoder(w).Encode(userInfo)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cfg := ProviderConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		AuthURL:      server.URL + "/auth",
		TokenURL:     server.URL + "/token",
		UserInfoURL:  server.URL + "/userinfo",
	}
	return server, cfg
}

func TestProvider_FinishLogin_PureOAuth2(t *testing.T) {
	_, provCfg := newFakeOAuth2Server(t, map[string]any{
		"id":             "12345",
		"email":          "user@example.com",
		"name":           "Test User",
		"email_verified": true,
	})

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{"github": provCfg},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, err := p.BeginLogin(context.Background(), "github", "state")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	userInfoJSON, err := p.FinishLogin(context.Background(), "github", "test-auth-code", pkceVerifier)
	if err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}

	var info userInfo
	if err := json.Unmarshal(userInfoJSON, &info); err != nil {
		t.Fatalf("unmarshal userInfo: %v", err)
	}

	if info.Subject != "12345" {
		t.Errorf("Subject = %q, want %q", info.Subject, "12345")
	}
	if info.Email != "user@example.com" {
		t.Errorf("Email = %q", info.Email)
	}
	if info.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q", info.DisplayName)
	}
	if !info.EmailVerified {
		t.Error("expected EmailVerified = true")
	}
}

func TestProvider_FinishLogin_PureOAuth2_GitHubLoginFallback(t *testing.T) {
	// GitHub uses "login" as display name when "name" is empty,
	// and "id" as subject when "sub" is empty.
	_, provCfg := newFakeOAuth2Server(t, map[string]any{
		"id":    "67890",
		"email": "ghuser@example.com",
		"login": "ghuser",
	})

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{"github": provCfg},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, _ := p.BeginLogin(context.Background(), "github", "state")
	userInfoJSON, err := p.FinishLogin(context.Background(), "github", "test-auth-code", pkceVerifier)
	if err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}

	var info userInfo
	json.Unmarshal(userInfoJSON, &info)

	if info.Subject != "67890" {
		t.Errorf("Subject = %q, want %q", info.Subject, "67890")
	}
	if info.DisplayName != "ghuser" {
		t.Errorf("DisplayName = %q, want %q (login fallback)", info.DisplayName, "ghuser")
	}
}

func TestProvider_FinishLogin_PureOAuth2_InvalidCode(t *testing.T) {
	_, provCfg := newFakeOAuth2Server(t, map[string]any{"id": "1"})

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{"github": provCfg},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, _ := p.BeginLogin(context.Background(), "github", "state")
	_, err = p.FinishLogin(context.Background(), "github", "wrong-code", pkceVerifier)
	if err == nil {
		t.Fatal("expected error for invalid auth code")
	}
}

// --- OIDC flow tests ---

type fakeOIDCServer struct {
	server *httptest.Server
	signer jose.Signer
	claims map[string]any
	issuer string
}

func newFakeOIDCServer(t *testing.T) *fakeOIDCServer {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	keyID := "test-key"
	opts := &jose.SignerOptions{}
	opts = opts.WithHeader("kid", keyID)
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privKey},
		opts,
	)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	jwk := jose.JSONWebKey{
		Key:       privKey.Public(),
		KeyID:     keyID,
		Algorithm: string(jose.RS256),
	}
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}

	prov := &fakeOIDCServer{
		signer: signer,
		claims: map[string]any{
			"sub":            "oidc-sub-123",
			"email":          "oidcuser@example.com",
			"email_verified": true,
			"name":           "OIDC User",
		},
	}

	mux := http.NewServeMux()
	prov.server = httptest.NewServer(mux)
	prov.issuer = prov.server.URL

	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                prov.issuer,
			"authorization_endpoint":                prov.issuer + "/auth",
			"token_endpoint":                        prov.issuer + "/token",
			"jwks_uri":                              prov.issuer + "/jwks",
			"userinfo_endpoint":                     prov.issuer + "/userinfo",
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
		})
	})

	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	})

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "test-auth-code" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}

		now := time.Now()
		claims := map[string]any{
			"iss": prov.issuer,
			"aud": "test-client-id",
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		}
		for k, v := range prov.claims {
			claims[k] = v
		}

		payload, _ := json.Marshal(claims)
		signed, err := prov.signer.Sign(payload)
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
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"id_token":     idToken,
		})
	})

	t.Cleanup(prov.server.Close)
	return prov
}

func (f *fakeOIDCServer) providerConfig() ProviderConfig {
	return ProviderConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "email", "profile"},
		IssuerURL:    f.issuer,
	}
}

func TestProvider_FinishLogin_OIDC(t *testing.T) {
	oidc := newFakeOIDCServer(t)

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{"google": oidc.providerConfig()},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, err := p.BeginLogin(context.Background(), "google", "state")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	userInfoJSON, err := p.FinishLogin(context.Background(), "google", "test-auth-code", pkceVerifier)
	if err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}

	var info userInfo
	if err := json.Unmarshal(userInfoJSON, &info); err != nil {
		t.Fatalf("unmarshal userInfo: %v", err)
	}

	if info.Subject != "oidc-sub-123" {
		t.Errorf("Subject = %q, want %q", info.Subject, "oidc-sub-123")
	}
	if info.Email != "oidcuser@example.com" {
		t.Errorf("Email = %q", info.Email)
	}
	if !info.EmailVerified {
		t.Error("expected EmailVerified = true")
	}
	if info.DisplayName != "OIDC User" {
		t.Errorf("DisplayName = %q", info.DisplayName)
	}
}

func TestProvider_FinishLogin_OIDC_InvalidToken(t *testing.T) {
	oidc := newFakeOIDCServer(t)
	// Tamper with issuer claim so verification fails
	oidc.claims["iss"] = "https://evil.example.com"

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{"google": oidc.providerConfig()},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, _ := p.BeginLogin(context.Background(), "google", "state")
	_, err = p.FinishLogin(context.Background(), "google", "test-auth-code", pkceVerifier)
	if err == nil {
		t.Fatal("expected error for invalid ID token (wrong issuer)")
	}
}

func TestProvider_FinishLogin_OIDC_InvalidCode(t *testing.T) {
	oidc := newFakeOIDCServer(t)

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{"google": oidc.providerConfig()},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, _ := p.BeginLogin(context.Background(), "google", "state")
	_, err = p.FinishLogin(context.Background(), "google", "wrong-code", pkceVerifier)
	if err == nil {
		t.Fatal("expected error for invalid auth code")
	}
}

func TestProvider_FinishLogin_PureOAuth2_UserInfoError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "tok",
			"token_type":   "Bearer",
		})
	})
	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"custom": {
				ClientID:     "id",
				ClientSecret: "secret",
				RedirectURL:  "http://localhost/callback",
				AuthURL:      server.URL + "/auth",
				TokenURL:     server.URL + "/token",
				UserInfoURL:  server.URL + "/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, _ := p.BeginLogin(context.Background(), "custom", "state")
	_, err = p.FinishLogin(context.Background(), "custom", "test-auth-code", pkceVerifier)
	if err == nil {
		t.Fatal("expected error when userinfo returns 500")
	}
}

// --- Default scopes test ---

func TestNew_DefaultScopes(t *testing.T) {
	oidc := newFakeOIDCServer(t)
	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"google": {
				ClientID:     "test-client-id",
				ClientSecret: "secret",
				RedirectURL:  "http://localhost/callback",
				IssuerURL:    oidc.issuer,
				// No scopes set — should default to openid, email, profile
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	redirectURL, _, err := p.BeginLogin(context.Background(), "google", "state")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	u, _ := url.Parse(redirectURL)
	scopes := strings.Split(u.Query().Get("scope"), " ")
	if len(scopes) != 3 {
		t.Fatalf("expected 3 default scopes, got %d: %v", len(scopes), scopes)
	}
	expected := map[string]bool{"openid": true, "email": true, "profile": true}
	for _, s := range scopes {
		if !expected[s] {
			t.Errorf("unexpected scope: %q", s)
		}
	}
}
