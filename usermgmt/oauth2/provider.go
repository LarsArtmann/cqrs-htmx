// Package oauth2 provides OAuth2/OIDC authentication using the golang.org/x/oauth2
// and coreos/go-oidc libraries.
//
// It implements the usermgmt.OAuth2Provider interface via structural typing —
// no import of the core usermgmt package is required. Consumers inject a
// *Provider into usermgmt.ServiceConfig.OAuth2 to enable OAuth2/OIDC login
// with external identity providers (Google, GitHub, Microsoft, etc.).
package oauth2

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	errorfamily "github.com/larsartmann/go-error-family"
	"golang.org/x/oauth2"
)

// ProviderConfig configures a single OAuth2/OIDC identity provider.
//
// For OIDC providers (Google, Microsoft, etc.), set IssuerURL — the provider's
// discovery endpoint will be queried at startup to fill in endpoints automatically.
//
// For pure OAuth2 providers (GitHub without OIDC), set AuthURL, TokenURL, and
// UserInfoURL explicitly.
type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	IssuerURL    string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

// Validate returns an error if the configuration is incomplete or inconsistent.
func (c ProviderConfig) Validate() error {
	if c.ClientID == "" {
		return errorfamily.NewRejection("oauth2.client_id_required", "oauth2 provider: ClientID is required")
	}
	if c.ClientSecret == "" {
		return errorfamily.NewRejection("oauth2.client_secret_required", "oauth2 provider: ClientSecret is required")
	}
	if c.RedirectURL == "" {
		return errorfamily.NewRejection("oauth2.redirect_url_required", "oauth2 provider: RedirectURL is required")
	}
	if c.IssuerURL == "" {
		if c.AuthURL == "" || c.TokenURL == "" || c.UserInfoURL == "" {
			return errorfamily.NewRejection(
				"oauth2.endpoints_required",
				"oauth2 provider: when IssuerURL is empty, AuthURL, TokenURL, and UserInfoURL are required",
			)
		}
	}
	return nil
}

// Config configures multi-provider OAuth2/OIDC authentication.
type Config struct {
	Providers map[string]ProviderConfig
}

// userInfo is the normalized user information extracted from either an OIDC ID
// token or a UserInfo endpoint response. Serialized to JSON and returned to
// the Service, which deserializes it into usermgmt.OAuth2UserInfo.
type userInfo struct {
	Subject       string `json:"subject"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	DisplayName   string `json:"display_name"`
}

// initializedProvider is a provider with discovered endpoints.
type initializedProvider struct {
	name         string
	config       *oauth2.Config
	oidcProvider *oidc.Provider        // nil for non-OIDC providers
	verifier     *oidc.IDTokenVerifier // nil for non-OIDC providers
	userInfoURL  string                // non-empty for non-OIDC providers
}

// Provider manages multiple OAuth2/OIDC providers and implements the
// usermgmt.OAuth2Provider interface via structural typing.
type Provider struct {
	providers map[string]*initializedProvider
}

// New initializes all configured providers (performing OIDC discovery if needed)
// and returns a Provider ready for use.
func New(ctx context.Context, cfg Config) (*Provider, error) {
	providers := make(map[string]*initializedProvider, len(cfg.Providers))
	for name, provCfg := range cfg.Providers {
		prov, err := initProvider(ctx, name, provCfg)
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Classify(err),
				"oauth2.init_provider",
				"init provider %q",
				name,
			)
		}
		providers[name] = prov
	}
	return &Provider{providers: providers}, nil
}

func initProvider(ctx context.Context, name string, cfg ProviderConfig) (*initializedProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}

	p := &initializedProvider{ //nolint:exhaustruct // fields set conditionally below
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
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Transient,
				"oauth2.oidc_discovery",
				"discover oidc provider %q",
				name,
			)
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

func (p *Provider) get(name string) (*initializedProvider, error) {
	prov, ok := p.providers[name]
	if !ok {
		return nil, errorfamily.Newf(errorfamily.Rejection, "oauth2.provider_not_found", "provider %q not found", name)
	}
	return prov, nil
}

// BeginLogin generates PKCE and builds the authorization URL for the given provider.
// state is the CSRF state token (generated by the Service).
// Returns the redirect URL and PKCE verifier (stored by the Service).
func (p *Provider) BeginLogin(_ context.Context, providerName, state string) (string, string, error) {
	prov, err := p.get(providerName)
	if err != nil {
		return "", "", errorfamily.WrapRejection(err, "oauth2.begin_login", "begin login").
			WithContext("provider", providerName).
			WithContext("state", state)
	}

	pkceVerifier := oauth2.GenerateVerifier()
	redirectURL := prov.config.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(pkceVerifier),
	)
	return redirectURL, pkceVerifier, nil
}

// FinishLogin exchanges the authorization code for tokens and extracts user info.
// Returns the user info as JSON (subject, email, email_verified, display_name).
func (p *Provider) FinishLogin(ctx context.Context, providerName, code, pkceVerifier string) ([]byte, error) {
	prov, err := p.get(providerName)
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "oauth2.finish_login", "finish login").
			WithContext("provider", providerName).
			WithContext("code", code).
			WithContext("pkce_verifier", pkceVerifier)
	}

	info, err := prov.exchangeAndExtractUser(ctx, code, pkceVerifier)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Classify(err), "oauth2.finish_login", "finish login").
			WithContext("provider", providerName).
			WithContext("pkce_verifier", pkceVerifier)
	}

	data, err := json.Marshal(info)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "oauth2.marshal_userinfo", "marshal user info").
			WithContext("provider", providerName).
			WithContext("code", code).
			WithContext("pkce_verifier", pkceVerifier)
	}
	return data, nil
}

func (p *initializedProvider) exchangeAndExtractUser(
	ctx context.Context, code, pkceVerifier string,
) (userInfo, error) {
	token, err := p.config.Exchange(
		ctx, code, oauth2.VerifierOption(pkceVerifier),
	)
	if err != nil {
		return userInfo{}, errorfamily.WrapTransient(err, "oauth2.token_exchange", "exchange code").
			WithContext("code", code).
			WithContext("pkce_verifier", pkceVerifier)
	}

	if p.oidcProvider != nil {
		return p.extractFromIDToken(ctx, token)
	}
	return p.fetchUserInfo(ctx, token)
}

func (p *initializedProvider) extractFromIDToken(
	ctx context.Context, token *oauth2.Token,
) (userInfo, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return userInfo{}, errorfamily.NewTransient("oauth2.id_token_missing", "id_token missing from token response")
	}
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return userInfo{}, errorfamily.WrapTransient(err, "oauth2.verify_id_token", "verify id_token")
	}
	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return userInfo{}, errorfamily.WrapTransient(err, "oauth2.extract_claims", "extract id_token claims")
	}
	return userInfo{
		Subject:       claims.Sub,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		DisplayName:   claims.Name,
	}, nil
}

func (p *initializedProvider) fetchUserInfo(ctx context.Context, token *oauth2.Token) (userInfo, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return userInfo{}, errorfamily.WrapInfrastructure(
			err,
			"oauth2.userinfo_request_create",
			"create userinfo request",
		)
	}
	resp, err := client.Do(req)
	if err != nil {
		return userInfo{}, errorfamily.WrapTransient(err, "oauth2.fetch_userinfo", "fetch userinfo")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return userInfo{}, errorfamily.Newf(
			errorfamily.Transient,
			"oauth2.userinfo_failed",
			"userinfo returned %d: %s",
			resp.StatusCode,
			string(body),
		)
	}
	// GitHub uses "id" as subject and "login" as display name
	var raw struct {
		ID            jsontext.Value `json:"id"`
		Sub           string         `json:"sub"`
		Email         string         `json:"email"`
		Name          string         `json:"name"`
		Login         string         `json:"login"`
		EmailVerified bool           `json:"email_verified"`
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return userInfo{}, errorfamily.WrapTransient(err, "oauth2.read_userinfo", "read userinfo response")
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return userInfo{}, errorfamily.WrapTransient(err, "oauth2.decode_userinfo", "decode userinfo response")
	}
	subject := raw.Sub
	if subject == "" {
		subject = raw.ID.String()
	}
	name := raw.Name
	if name == "" {
		name = raw.Login
	}
	return userInfo{
		Subject:       subject,
		Email:         raw.Email,
		EmailVerified: raw.EmailVerified,
		DisplayName:   name,
	}, nil
}
