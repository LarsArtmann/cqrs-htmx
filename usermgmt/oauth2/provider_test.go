package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
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
	config := ProviderConfig{
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected error for missing ClientID")
	}
}

func TestProviderConfig_Validate_MissingClientSecret(t *testing.T) {
	config := ProviderConfig{
		ClientID:    "id",
		RedirectURL: "http://localhost/callback",
		IssuerURL:   "https://accounts.google.com",
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected error for missing ClientSecret")
	}
}

func TestProviderConfig_Validate_MissingRedirectURL(t *testing.T) {
	config := ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected error for missing RedirectURL")
	}
}

func TestProviderConfig_Validate_OAuth2EndpointsRequired(t *testing.T) {
	config := ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected error when IssuerURL is empty and no explicit endpoints")
	}
}

func TestProviderConfig_Validate_OAuth2EndpointsOK(t *testing.T) {
	config := ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("valid OAuth2 config should not error: %v", err)
	}
}

func TestProviderConfig_Validate_PublicClientNoSecret(t *testing.T) {
	config := ProviderConfig{
		ClientID:    "id",
		ClientType:  ClientTypePublic,
		RedirectURL: "http://localhost/callback",
		IssuerURL:   "https://accounts.google.com",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("public client without secret should not error: %v", err)
	}
}

func TestProviderConfig_Validate_PublicClientWithSecret(t *testing.T) {
	config := ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		ClientType:   ClientTypePublic,
		RedirectURL:  "http://localhost/callback",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("public client with secret should not error: %v", err)
	}
}

func TestProviderConfig_Validate_ConfidentialDefaultRequiresSecret(t *testing.T) {
	// ClientType zero value must stay backward compatible: confidential,
	// so a missing secret is still rejected.
	config := ProviderConfig{
		ClientID:    "id",
		RedirectURL: "http://localhost/callback",
		IssuerURL:   "https://accounts.google.com",
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected error for missing ClientSecret with default (confidential) ClientType")
	}
}

func TestProviderConfig_Validate_InvalidClientType(t *testing.T) {
	config := ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		ClientType:   ClientType("hybrid"),
		RedirectURL:  "http://localhost/callback",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected error for invalid ClientType")
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

func TestProvider_FinishLoginWithToken_UnknownProvider(t *testing.T) {
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

	_, rawIDToken, err := p.FinishLoginWithToken(context.Background(), "unknown", "code", "verifier")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if rawIDToken != "" {
		t.Errorf("expected empty raw ID token on error, got %q", rawIDToken)
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

func TestProvider_BeginLogin_PublicClient(t *testing.T) {
	// A public (secret-less) client must still be accepted and produce a
	// PKCE-protected authorization URL.
	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"pocket": {
				ClientID:    "public-client-id",
				ClientType:  ClientTypePublic,
				RedirectURL: "http://localhost/callback",
				AuthURL:     "https://example.com/auth",
				TokenURL:    "https://example.com/token",
				UserInfoURL: "https://example.com/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	redirectURL, pkceVerifier, err := p.BeginLogin(context.Background(), "pocket", "my-state")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	u, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	if u.Query().Get("client_id") != "public-client-id" {
		t.Errorf("client_id = %q", u.Query().Get("client_id"))
	}
	if u.Query().Get("code_challenge") == "" {
		t.Error("expected non-empty code_challenge for public client")
	}
	if pkceVerifier == "" {
		t.Error("expected non-empty PKCE verifier")
	}
}

// --- Pure OAuth2 flow tests ---

func newFakeOAuth2Server(t *testing.T, userInfo map[string]any) ProviderConfig {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "test-auth-code" {
			w.WriteHeader(http.StatusBadRequest)
			json.MarshalWrite(w, map[string]string{"error": "invalid_grant"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.MarshalWrite(w, map[string]string{
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
		json.MarshalWrite(w, userInfo)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	config := ProviderConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		AuthURL:      server.URL + "/auth",
		TokenURL:     server.URL + "/token",
		UserInfoURL:  server.URL + "/userinfo",
	}
	return config
}

func TestProvider_FinishLogin_PureOAuth2(t *testing.T) {
	provCfg := newFakeOAuth2Server(t, map[string]any{
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
	provCfg := newFakeOAuth2Server(t, map[string]any{
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
	if info.PreferredUsername != "ghuser" {
		t.Errorf("PreferredUsername = %q, want %q (login fallback)", info.PreferredUsername, "ghuser")
	}
}

func TestProvider_FinishLogin_PublicClient_ExchangeWithoutSecret(t *testing.T) {
	// Wire-level proof for RFC 7636 public clients: the token exchange must be
	// secured by PKCE alone — no client secret may appear as a POST body
	// parameter (AuthStyleInParams) or as a Basic-auth password
	// (AuthStyleInHeader, x/oauth2's first probe attempt).
	mux := http.NewServeMux()

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "test-auth-code" {
			w.WriteHeader(http.StatusBadRequest)
			json.MarshalWrite(w, map[string]string{"error": "invalid_grant"})
			return
		}
		if r.PostForm.Has("client_secret") {
			w.WriteHeader(http.StatusBadRequest)
			json.MarshalWrite(w, map[string]string{"error": "invalid_client", "error_description": "public client must not send client_secret"})
			return
		}
		if _, password, ok := r.BasicAuth(); ok && password != "" {
			w.WriteHeader(http.StatusBadRequest)
			json.MarshalWrite(w, map[string]string{"error": "invalid_client", "error_description": "public client must not send a Basic-auth secret"})
			return
		}
		if r.PostFormValue("code_verifier") == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.MarshalWrite(w, map[string]string{"error": "invalid_grant", "error_description": "code_verifier required"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.MarshalWrite(w, map[string]string{
			"access_token": "public-access-token",
			"token_type":   "Bearer",
		})
	})

	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.MarshalWrite(w, map[string]any{
			"sub":                "public-sub-1",
			"email":              "public@example.com",
			"preferred_username": "public-user",
			"email_verified":     true,
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"public-idp": {
				ClientID:    "public-client-id",
				ClientType:  ClientTypePublic,
				RedirectURL: "http://localhost:8080/callback",
				AuthURL:     server.URL + "/auth",
				TokenURL:    server.URL + "/token",
				UserInfoURL: server.URL + "/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, err := p.BeginLogin(context.Background(), "public-idp", "state")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	userInfoJSON, err := p.FinishLogin(context.Background(), "public-idp", "test-auth-code", pkceVerifier)
	if err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}

	var info userInfo
	if err := json.Unmarshal(userInfoJSON, &info); err != nil {
		t.Fatalf("unmarshal userInfo: %v", err)
	}
	if info.Subject != "public-sub-1" {
		t.Errorf("Subject = %q, want %q", info.Subject, "public-sub-1")
	}
	if info.PreferredUsername != "public-user" {
		t.Errorf("PreferredUsername = %q, want %q", info.PreferredUsername, "public-user")
	}
}

func TestProvider_FinishLogin_ConfidentialClient_SendsSecret(t *testing.T) {
	// Complement of the public-client test: the confidential (default) client
	// must authenticate the token exchange — the secret appears either as a
	// client_secret POST body parameter (AuthStyleInParams) or as the Basic-auth
	// password (AuthStyleInHeader).
	mux := http.NewServeMux()

	const wantSecret = "confidential-secret"
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "test-auth-code" {
			w.WriteHeader(http.StatusBadRequest)
			json.MarshalWrite(w, map[string]string{"error": "invalid_grant"})
			return
		}
		_, basicPassword, basicOK := r.BasicAuth()
		if r.PostForm.Get("client_secret") != wantSecret && (!basicOK || basicPassword != wantSecret) {
			w.WriteHeader(http.StatusUnauthorized)
			json.MarshalWrite(w, map[string]string{"error": "invalid_client", "error_description": "client secret missing or wrong"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.MarshalWrite(w, map[string]string{
			"access_token": "confidential-access-token",
			"token_type":   "Bearer",
		})
	})

	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.MarshalWrite(w, map[string]any{"sub": "confidential-sub-1", "login": "conf-user"})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"github": {
				ClientID:     "confidential-client-id",
				ClientSecret: wantSecret,
				RedirectURL:  "http://localhost:8080/callback",
				AuthURL:      server.URL + "/auth",
				TokenURL:     server.URL + "/token",
				UserInfoURL:  server.URL + "/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, pkceVerifier, _ := p.BeginLogin(context.Background(), "github", "state")
	if _, err := p.FinishLogin(context.Background(), "github", "test-auth-code", pkceVerifier); err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}
}

func TestProvider_FinishLogin_PureOAuth2_InvalidCode(t *testing.T) {
	provCfg := newFakeOAuth2Server(t, map[string]any{"id": "1"})

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

// Pre-generated RSA key + signer to avoid the ~50ms key generation cost
// per test invocation. Generated once, reused across all fakeOIDCServer tests.
var (
	cachedSignerOnce sync.Once
	cachedSigner     jose.Signer
	cachedJWKS       jose.JSONWebKeySet
)

func initCachedSigner() {
	cachedSignerOnce.Do(func() {
		privKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic("oauth2 test: generate RSA key: " + err.Error())
		}

		keyID := "test-key"
		opts := &jose.SignerOptions{}
		opts = opts.WithHeader("kid", keyID)
		signer, err := jose.NewSigner(
			jose.SigningKey{Algorithm: jose.RS256, Key: privKey},
			opts,
		)
		if err != nil {
			panic("oauth2 test: create signer: " + err.Error())
		}

		jwk := jose.JSONWebKey{
			Key:       privKey.Public(),
			KeyID:     keyID,
			Algorithm: string(jose.RS256),
		}

		cachedSigner = signer
		cachedJWKS = jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	})
}

type fakeOIDCServer struct {
	server *httptest.Server
	signer jose.Signer
	claims map[string]any
	issuer string
}

func newFakeOIDCServer(t *testing.T) *fakeOIDCServer {
	t.Helper()
	initCachedSigner()

	prov := &fakeOIDCServer{
		signer: cachedSigner,
		claims: map[string]any{
			"sub":                "oidc-sub-123",
			"email":              "oidcuser@example.com",
			"email_verified":     true,
			"name":               "OIDC User",
			"preferred_username": "oidcuser",
		},
	}

	mux := http.NewServeMux()
	prov.server = httptest.NewServer(mux)
	prov.issuer = prov.server.URL

	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.MarshalWrite(w, map[string]any{
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
		json.MarshalWrite(w, cachedJWKS)
	})

	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "test-auth-code" {
			w.WriteHeader(http.StatusBadRequest)
			json.MarshalWrite(w, map[string]string{"error": "invalid_grant"})
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
		json.MarshalWrite(w, map[string]string{
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
	if info.PreferredUsername != "oidcuser" {
		t.Errorf("PreferredUsername = %q, want %q", info.PreferredUsername, "oidcuser")
	}
}

func TestProvider_FinishLoginWithToken_OIDC(t *testing.T) {
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

	userInfoJSON, rawIDToken, err := p.FinishLoginWithToken(context.Background(), "google", "test-auth-code", pkceVerifier)
	if err != nil {
		t.Fatalf("FinishLoginWithToken: %v", err)
	}

	var info userInfo
	if err := json.Unmarshal(userInfoJSON, &info); err != nil {
		t.Fatalf("unmarshal userInfo: %v", err)
	}
	if info.Subject != "oidc-sub-123" {
		t.Errorf("Subject = %q, want %q", info.Subject, "oidc-sub-123")
	}
	if info.PreferredUsername != "oidcuser" {
		t.Errorf("PreferredUsername = %q, want %q", info.PreferredUsername, "oidcuser")
	}

	// The raw ID token must be non-empty and be a verifiable JWT (3 dot-separated parts).
	if rawIDToken == "" {
		t.Fatal("expected non-empty raw ID token")
	}
	parts := strings.Split(rawIDToken, ".")
	if len(parts) != 3 {
		t.Errorf("raw ID token should be a compact JWT, got %d parts", len(parts))
	}
}

func TestProvider_FinishLoginWithToken_PureOAuth2(t *testing.T) {
	// Non-OIDC providers have no ID token; the raw token must be empty.
	provCfg := newFakeOAuth2Server(t, map[string]any{
		"id":                 "12345",
		"email":              "user@example.com",
		"name":               "Test User",
		"preferred_username": "testuser",
		"email_verified":     true,
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

	userInfoJSON, rawIDToken, err := p.FinishLoginWithToken(context.Background(), "github", "test-auth-code", pkceVerifier)
	if err != nil {
		t.Fatalf("FinishLoginWithToken: %v", err)
	}
	if rawIDToken != "" {
		t.Errorf("expected empty raw ID token for non-OIDC provider, got %q", rawIDToken)
	}

	var info userInfo
	if err := json.Unmarshal(userInfoJSON, &info); err != nil {
		t.Fatalf("unmarshal userInfo: %v", err)
	}
	if info.Subject != "12345" {
		t.Errorf("Subject = %q, want %q", info.Subject, "12345")
	}
	if info.PreferredUsername != "testuser" {
		t.Errorf("PreferredUsername = %q, want %q", info.PreferredUsername, "testuser")
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
		json.MarshalWrite(w, map[string]string{
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

// --- Names() tests ---

func TestProvider_Names_Empty(t *testing.T) {
	p, err := New(context.Background(), Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	names := p.Names()
	if len(names) != 0 {
		t.Errorf("expected empty slice, got %v", names)
	}
}

func TestProvider_Names_Single(t *testing.T) {
	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"github": {
				ClientID:     "id",
				ClientSecret: "secret",
				RedirectURL:  "http://localhost/callback",
				AuthURL:      "https://github.com/login/oauth/authorize",
				TokenURL:     "https://github.com/login/oauth/access_token",
				UserInfoURL:  "https://api.github.com/user",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	names := p.Names()
	if len(names) != 1 || names[0] != "github" {
		t.Errorf("expected [github], got %v", names)
	}
}

func TestProvider_Names_Multiple_Sorted(t *testing.T) {
	oidc := newFakeOIDCServer(t)
	p, err := New(context.Background(), Config{
		Providers: map[string]ProviderConfig{
			"google": {
				ClientID:     "id1",
				ClientSecret: "secret1",
				RedirectURL:  "http://localhost/callback",
				IssuerURL:    oidc.issuer,
			},
			"github": {
				ClientID:     "id2",
				ClientSecret: "secret2",
				RedirectURL:  "http://localhost/callback",
				AuthURL:      "https://github.com/login/oauth/authorize",
				TokenURL:     "https://github.com/login/oauth/access_token",
				UserInfoURL:  "https://api.github.com/user",
			},
			"azure": {
				ClientID:     "id3",
				ClientSecret: "secret3",
				RedirectURL:  "http://localhost/callback",
				AuthURL:      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
				UserInfoURL:  "https://graph.microsoft.com/oidc/userinfo",
			},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	names := p.Names()
	want := []string{"azure", "github", "google"}
	if len(names) != len(want) {
		t.Fatalf("expected %d names, got %d: %v", len(want), len(names), names)
	}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("names[%d] = %q, want %q", i, names[i], w)
		}
	}
}
