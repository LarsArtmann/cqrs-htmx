package usermgmt

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// defaultOAuthStateTTL is the default lifetime of an OAuth2 state token.
const defaultOAuthStateTTL = 10 * time.Minute

// oauthStateEvictionInterval is how often expired OAuth state tokens are cleaned up.
const oauthStateEvictionInterval = 5 * time.Minute

// OAuth2ProviderConfig configures a single OAuth2/OIDC identity provider.
//
// For OIDC providers (Google, Microsoft, etc.), set IssuerURL — the provider's
// discovery endpoint will be queried at startup to fill in endpoints automatically.
//
// For pure OAuth2 providers (GitHub without OIDC), set AuthURL, TokenURL, and
// UserInfoURL explicitly.
type OAuth2ProviderConfig struct {
	// ClientID is the OAuth2 client ID registered with the provider.
	ClientID string
	// ClientSecret is the OAuth2 client secret.
	ClientSecret string
	// RedirectURL is the callback URL (e.g., "https://myapp.com/auth/oauth/google/callback").
	RedirectURL string
	// Scopes are the OAuth2 scopes to request. Defaults to ["openid", "email", "profile"].
	// Ignored if empty.
	Scopes []string
	// IssuerURL is the OIDC discovery URL (e.g., "https://accounts.google.com").
	// When set, OIDC discovery is used: endpoints are auto-discovered and ID tokens
	// are verified. When empty, AuthURL/TokenURL/UserInfoURL must be set.
	IssuerURL string
	// AuthURL is the authorization endpoint. Required when IssuerURL is empty.
	AuthURL string
	// TokenURL is the token endpoint. Required when IssuerURL is empty.
	TokenURL string
	// UserInfoURL is the userinfo endpoint for fetching user details when no
	// ID token is available (pure OAuth2 providers). Required when IssuerURL is empty.
	UserInfoURL string
}

// Validate returns an error if the configuration is incomplete or inconsistent.
func (c OAuth2ProviderConfig) Validate() error {
	if c.ClientID == "" {
		return errors.New("oauth2 provider: ClientID is required")
	}
	if c.ClientSecret == "" {
		return errors.New("oauth2 provider: ClientSecret is required")
	}
	if c.RedirectURL == "" {
		return errors.New("oauth2 provider: RedirectURL is required")
	}
	if c.IssuerURL == "" {
		if c.AuthURL == "" || c.TokenURL == "" || c.UserInfoURL == "" {
			return errors.New(
				"oauth2 provider: when IssuerURL is empty, AuthURL, TokenURL, and UserInfoURL are required",
			)
		}
	}
	return nil
}

// OAuth2Config configures multi-provider OAuth2/OIDC authentication.
type OAuth2Config struct {
	// Providers maps provider names ("google", "github", etc.) to their configs.
	Providers map[string]OAuth2ProviderConfig
	// StateTTL is the lifetime of CSRF state tokens. Defaults to 10 minutes.
	StateTTL time.Duration
}

// oauth2Provider is an initialized provider with discovered endpoints.
type oauth2Provider struct {
	name         string
	config       *oauth2.Config
	oidcProvider *oidc.Provider        // nil for non-OIDC providers
	verifier     *oidc.IDTokenVerifier // nil for non-OIDC providers
	userInfoURL  string                // non-empty for non-OIDC providers
}

// oauth2UserInfo holds the normalized user information extracted from either
// an OIDC ID token or a UserInfo endpoint response.
type oauth2UserInfo struct {
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
}

// initOAuth2Provider discovers OIDC endpoints (or uses explicit OAuth2 endpoints)
// and returns an initialized provider.
func initOAuth2Provider(ctx context.Context, name string, cfg OAuth2ProviderConfig) (*oauth2Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("init oauth2 provider %q: %w", name, err)
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"} //nolint:goconst // OAuth scope name
	}

	p := &oauth2Provider{ //nolint:exhaustruct // fields set conditionally below
		name: name,
		config: &oauth2.Config{ //nolint:exhaustruct // Endpoint set below
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
		},
	}

	if cfg.IssuerURL != "" {
		oidcProv, err := oidc.NewProvider(ctx, cfg.IssuerURL)
		if err != nil {
			return nil, fmt.Errorf("discover oidc provider %q: %w", name, err)
		}
		p.oidcProvider = oidcProv
		p.verifier = oidcProv.Verifier(
			&oidc.Config{ClientID: cfg.ClientID}, //nolint:exhaustruct // only ClientID needed
		)
		p.config.Endpoint = oidcProv.Endpoint()
	} else {
		p.config.Endpoint = oauth2.Endpoint{ //nolint:exhaustruct // only AuthURL+TokenURL needed
			AuthURL:  cfg.AuthURL,
			TokenURL: cfg.TokenURL,
		}
		p.userInfoURL = cfg.UserInfoURL
	}

	return p, nil
}

// authCodeURL builds the authorization redirect URL with state and PKCE.
func (p *oauth2Provider) authCodeURL(state, pkceVerifier string) string {
	return p.config.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(pkceVerifier),
	)
}

// exchangeAndExtractUser exchanges the authorization code for tokens, then
// extracts user information from either the OIDC ID token or the UserInfo endpoint.
func (p *oauth2Provider) exchangeAndExtractUser(
	ctx context.Context, code, pkceVerifier string,
) (oauth2UserInfo, error) {
	token, err := p.config.Exchange(
		ctx, code, oauth2.VerifierOption(pkceVerifier),
	)
	if err != nil {
		return oauth2UserInfo{}, fmt.Errorf("exchange oauth2 code: %w", err)
	}

	if p.oidcProvider != nil {
		return p.extractFromIDToken(ctx, token)
	}
	return p.fetchUserInfo(ctx, token)
}

func (p *oauth2Provider) extractFromIDToken(
	ctx context.Context, token *oauth2.Token,
) (oauth2UserInfo, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return oauth2UserInfo{}, errors.New("id_token missing from oauth2 token response")
	}
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return oauth2UserInfo{}, fmt.Errorf("verify id_token: %w", err)
	}
	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return oauth2UserInfo{}, fmt.Errorf("extract id_token claims: %w", err)
	}
	return oauth2UserInfo{
		Subject:       claims.Sub,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		DisplayName:   claims.Name,
	}, nil
}

func (p *oauth2Provider) fetchUserInfo(ctx context.Context, token *oauth2.Token) (oauth2UserInfo, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return oauth2UserInfo{}, fmt.Errorf("create userinfo request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return oauth2UserInfo{}, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return oauth2UserInfo{}, fmt.Errorf(
			"userinfo returned %d: %s", resp.StatusCode, string(body),
		)
	}
	// GitHub uses "id" as subject and "login" as display name
	var raw struct {
		ID      json.Number `json:"id"`
		Sub     string      `json:"sub"`
		Email   string      `json:"email"`
		Name    string      `json:"name"`
		Login   string      `json:"login"`
		Verifie bool        `json:"email_verified"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&raw); err != nil {
		return oauth2UserInfo{}, fmt.Errorf("decode userinfo response: %w", err)
	}
	subject := raw.Sub
	if subject == "" {
		subject = raw.ID.String()
	}
	name := raw.Name
	if name == "" {
		name = raw.Login
	}
	return oauth2UserInfo{
		Subject:       subject,
		Email:         raw.Email,
		EmailVerified: raw.Verifie,
		DisplayName:   name,
	}, nil
}

// --- oauth2StateStore (mirrors verificationTokenStore) ---

type oauth2StateEntry struct {
	provider     string
	pkceVerifier string
	expiresAt    time.Time
}

type oauth2StateStore struct {
	mu     sync.Mutex
	states map[string]oauth2StateEntry
}

func newOAuth2StateStore() *oauth2StateStore {
	return &oauth2StateStore{ //nolint:exhaustruct // mu is zero-value (sync.Mutex)
		states: make(map[string]oauth2StateEntry),
	}
}

// Save generates a random state token, stores it with the provider name and
// PKCE verifier, and returns the state token.
func (s *oauth2StateStore) Save(provider, pkceVerifier string, ttl time.Duration) (string, error) {
	state, err := generateOAuth2State()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.states[state] = oauth2StateEntry{
		provider:     provider,
		pkceVerifier: pkceVerifier,
		expiresAt:    time.Now().Add(ttl),
	}
	s.mu.Unlock()
	return state, nil
}

// Consume validates and deletes the state token (one-time use).
// Returns the stored provider name and PKCE verifier.
func (s *oauth2StateStore) Consume(state string) (provider, pkceVerifier string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.states[state]
	if !ok {
		return "", "", ErrOAuthInvalidState
	}
	delete(s.states, state)
	if time.Now().After(entry.expiresAt) {
		return "", "", ErrOAuthInvalidState
	}
	return entry.provider, entry.pkceVerifier, nil
}

// EvictExpired removes all expired state tokens. Returns the count evicted.
func (s *oauth2StateStore) EvictExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for state, entry := range s.states {
		if now.After(entry.expiresAt) {
			delete(s.states, state)
			count++
		}
	}
	return count
}

func (s *oauth2StateStore) startEviction() (stop func()) {
	return startPeriodicEviction(s.EvictExpired, oauthStateEvictionInterval)
}

func generateOAuth2State() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate oauth2 state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
