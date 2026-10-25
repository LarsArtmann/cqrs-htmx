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
	"sort"
	"strconv"

	"github.com/coreos/go-oidc/v3/oidc"
	errorfamily "github.com/larsartmann/go-error-family"
	"golang.org/x/oauth2"
)

// ClientType distinguishes how a client authenticates with the provider.
// Public clients (RFC 7636 PKCE, OAuth 2.1) have no client secret; the
// authorization-code exchange is secured by the PKCE verifier instead.
type ClientType string

const (
	// ClientTypeConfidential is the default: the client holds a secret that
	// authenticates the token exchange. Backward compatible with the previous
	// behavior, where ClientSecret was always required.
	ClientTypeConfidential ClientType = "confidential"
	// ClientTypePublic is for clients without a secure secret storage location
	// (browser SPAs, native apps, CLI tools). PKCE S256 is always enforced,
	// so the code exchange remains protected.
	ClientTypePublic ClientType = "public"
)

// ProviderConfig configures a single OAuth2/OIDC identity provider.
//
// For OIDC providers (Google, Microsoft, etc.), set IssuerURL — the provider's
// discovery endpoint will be queried at startup to fill in endpoints automatically.
//
// For pure OAuth2 providers (GitHub without OIDC), set AuthURL, TokenURL, and
// UserInfoURL explicitly.
//
// ClientType defaults to ClientTypeConfidential. Set ClientTypePublic to use
// a secret-less (PKCE-only) client — ClientSecret is then optional.
type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	ClientType   ClientType
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
	if c.ClientType == "" {
		c.ClientType = ClientTypeConfidential
	}
	switch c.ClientType {
	case ClientTypeConfidential, ClientTypePublic:
	default:
		return errorfamily.NewRejection(
			"oauth2.invalid_client_type",
			"oauth2 provider: ClientType must be ClientTypeConfidential or ClientTypePublic",
		)
	}
	if c.ClientType == ClientTypeConfidential && c.ClientSecret == "" {
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
//
// INTENTIONAL DUPLICATION (Sollbruchstelle seam): this auth-strategy module
// must NOT import core usermgmt (see AGENTS.md dependency direction). The JSON
// tags are the real contract — they are kept identical to OAuth2UserInfo on
// purpose so the two modules stay decoupled. PreferredUsername is an additive
// optional field (omitempty): usermgmt's OAuth2UserInfo ignores it, while
// standalone consumers (e.g. dnsblockd) can read it from the FinishLogin JSON.
type userInfo struct {
	Subject           string `json:"subject"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	DisplayName       string `json:"display_name"`
	PreferredUsername string `json:"preferred_username,omitempty"`
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
func New(ctx context.Context, config Config) (*Provider, error) {
	providers := make(map[string]*initializedProvider, len(config.Providers))
	for name, provCfg := range config.Providers {
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

func initProvider(ctx context.Context, name string, config ProviderConfig) (*initializedProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}

	p := &initializedProvider{ //nolint:exhaustruct // fields set conditionally below
		name: name,
		config: &oauth2.Config{ //nolint:exhaustruct // Endpoint set below
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       scopes,
		},
	}

	if config.IssuerURL != "" {
		oidcProv, err := oidc.NewProvider(ctx, config.IssuerURL)
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
			&oidc.Config{ClientID: config.ClientID}, //nolint:exhaustruct // only ClientID needed
		)
		p.config.Endpoint = oidcProv.Endpoint()
	} else {
		p.config.Endpoint = oauth2.Endpoint{ //nolint:exhaustruct // only AuthURL+TokenURL needed
			AuthURL:  config.AuthURL,
			TokenURL: config.TokenURL,
		}
		p.userInfoURL = config.UserInfoURL
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

// Names returns the sorted names of all configured providers.
// Used by consumers to auto-discover which OAuth2/OIDC providers are available
// (e.g. to auto-populate sign-in buttons).
func (p *Provider) Names() []string {
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// BeginLogin generates PKCE and an OIDC nonce, then builds the authorization
// URL for the given provider. state is the CSRF state token (generated by
// the Service).
// Returns the redirect URL, the PKCE verifier, and the OIDC nonce (both
// secrets are stored by the Service and presented again at FinishLogin; the
// nonce binds the ID token to this authorization request, per OIDC Core
// §3.1.2.1 — replayed or injected tokens carry a different nonce and are
// rejected at verification).
func (p *Provider) BeginLogin(_ context.Context, providerName, state string) (string, string, string, error) {
	prov, err := p.get(providerName)
	if err != nil {
		return "", "", "", errorfamily.WrapRejection(err, "oauth2.begin_login", "begin login").
			WithContext("provider", providerName).
			WithContext("state", state)
	}

	pkceVerifier := oauth2.GenerateVerifier()
	nonce := oauth2.GenerateVerifier()
	redirectURL := prov.config.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(pkceVerifier),
		oauth2.SetAuthURLParam("nonce", nonce),
	)
	return redirectURL, pkceVerifier, nonce, nil
}

// FinishLogin exchanges the authorization code for tokens and extracts user info.
// Returns the user info as JSON (subject, email, email_verified, display_name,
// preferred_username). For pure OAuth2 providers whose userinfo response has no
// preferred_username (e.g. GitHub), the login field is used as a fallback.
func (p *Provider) FinishLogin(ctx context.Context, providerName, code, pkceVerifier, nonce string) ([]byte, error) {
	prov, err := p.get(providerName)
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "oauth2.finish_login", "finish login").
			WithContext("provider", providerName)
	}

	info, _, err := prov.exchangeAndExtractUser(ctx, code, pkceVerifier, nonce)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Classify(err), "oauth2.finish_login", "finish login").
			WithContext("provider", providerName)
	}

	data, err := json.Marshal(info)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "oauth2.marshal_userinfo", "marshal user info").
			WithContext("provider", providerName)
	}
	return data, nil
}

// FinishLoginWithToken exchanges the authorization code for tokens and extracts
// user info, additionally returning the verified raw ID token (OIDC providers
// only) so consumers can read provider-specific claims without a library
// round-trip. For non-OIDC providers the raw ID token is empty.
//
// The user info JSON is identical to [Provider.FinishLogin]'s return value; the
// raw ID token is an additive escape hatch and is never part of the JSON
// contract consumed by usermgmt.
func (p *Provider) FinishLoginWithToken(
	ctx context.Context, providerName, code, pkceVerifier, nonce string,
) ([]byte, string, error) {
	prov, err := p.get(providerName)
	if err != nil {
		return nil, "", errorfamily.WrapRejection(err, "oauth2.finish_login_with_token", "finish login with token").
			WithContext("provider", providerName)
	}

	info, rawIDToken, err := prov.exchangeAndExtractUser(ctx, code, pkceVerifier, nonce)
	if err != nil {
		return nil, "", errorfamily.Wrapf(err, errorfamily.Classify(err), "oauth2.finish_login_with_token", "finish login with token").
			WithContext("provider", providerName)
	}

	data, err := json.Marshal(info)
	if err != nil {
		return nil, "", errorfamily.WrapInfrastructure(err, "oauth2.marshal_userinfo", "marshal user info").
			WithContext("provider", providerName)
	}
	return data, rawIDToken, nil
}

func (p *initializedProvider) exchangeAndExtractUser(
	ctx context.Context, code, pkceVerifier, nonce string,
) (userInfo, string, error) {
	token, err := p.config.Exchange(
		ctx, code, oauth2.VerifierOption(pkceVerifier),
	)
	if err != nil {
		// No code/pkce_verifier context: both are live credentials (RFC
		// 7636 §7.1) — a code+verifier pair read from the logs of a failed
		// exchange can still be redeemed.
		return userInfo{}, "", errorfamily.WrapTransient(err, "oauth2.token_exchange", "exchange code")
	}

	var info userInfo
	var rawIDToken string
	if p.oidcProvider != nil {
		info, rawIDToken, err = p.extractFromIDToken(ctx, token, nonce)
	} else {
		info, err = p.fetchUserInfo(ctx, token)
	}
	if err != nil {
		return userInfo{}, "", err
	}
	// An empty subject means the provider returned neither "sub" nor a usable
	// "id" — the external-account identity key would degrade to email-only,
	// silently linking every email-sharing account to one user. Fail loudly
	// at the contract seam instead.
	if info.Subject == "" {
		return userInfo{}, "", errorfamily.NewRejection(
			"oauth2.empty_subject",
			"provider returned neither a subject (sub) nor a numeric id claim",
		).WithContext("provider", p.name)
	}

	return info, rawIDToken, nil
}

// extractFromIDToken verifies the provider's id_token and extracts claims.
// Its errors deliberately carry NO rawIDToken/claim context: the raw token is
// a live credential that must never be echoed into an error, so the three
// returns below suppress erraudit (LIVE-SECRET class) instead of enriching.
func (p *initializedProvider) extractFromIDToken(
	ctx context.Context, token *oauth2.Token, nonce string,
) (userInfo, string, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return userInfo{}, "", errorfamily.NewTransient( //nolint:erraudit // LIVE-SECRET: id_token is a live credential
			"oauth2.id_token_missing",
			"id_token missing from token response",
		)
	}
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return userInfo{}, "", errorfamily.WrapTransient(
			err,
			"oauth2.verify_id_token",
			"verify id_token",
		) //nolint:erraudit // LIVE-SECRET: id_token is a live credential
	}
	var claims struct {
		Sub               string `json:"sub"`
		Nonce             string `json:"nonce"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return userInfo{}, "", errorfamily.WrapTransient(
			err,
			"oauth2.extract_claims",
			"extract id_token claims",
		) //nolint:erraudit // LIVE-SECRET: id_token is a live credential
	}
	// OIDC Core §3.1.3.7: when a nonce was sent in the authorization request,
	// the id_token MUST carry it back unchanged. A mismatched (or missing)
	// nonce means the token was not issued for this authorization request —
	// reject rather than trust the claims.
	if nonce != "" && claims.Nonce != nonce {
		return userInfo{}, "", errorfamily.NewRejection(
			"oauth2.nonce_mismatch",
			"id_token nonce does not match the authorization request",
		).WithContext("provider", p.name) //nolint:erraudit // LIVE-SECRET: nonce values must not be logged
	}
	return userInfo{
		Subject:           claims.Sub,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified,
		DisplayName:       claims.Name,
		PreferredUsername: claims.PreferredUsername,
	}, rawIDToken, nil
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
		ID                jsontext.Value `json:"id"`
		Sub               string         `json:"sub"`
		Email             string         `json:"email"`
		Name              string         `json:"name"`
		Login             string         `json:"login"`
		PreferredUsername string         `json:"preferred_username"`
		EmailVerified     bool           `json:"email_verified"`
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return userInfo{}, errorfamily.WrapTransient(err, "oauth2.read_userinfo", "read userinfo response")
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return userInfo{}, errorfamily.WrapTransient(err, "oauth2.decode_userinfo", "decode userinfo response")
	}
	subject := raw.Sub
	if subject == "" && len(raw.ID) > 0 {
		// ID may be a JSON string ("12345") or number (12345).
		// jsontext.Value.String() returns raw JSON including quotes,
		// so we must properly decode the value.
		if err := json.Unmarshal(raw.ID, &subject); err != nil {
			var num int64
			if err2 := json.Unmarshal(raw.ID, &num); err2 != nil {
				return userInfo{}, errorfamily.WrapTransient(
					err2, "oauth2.decode_userinfo_id", "decode userinfo ID field")
			}
			subject = strconv.FormatInt(num, 10)
		}
	}
	name := raw.Name
	if name == "" {
		name = raw.Login
	}
	preferredUsername := raw.PreferredUsername
	if preferredUsername == "" {
		preferredUsername = raw.Login
	}
	return userInfo{
		Subject:           subject,
		Email:             raw.Email,
		EmailVerified:     raw.EmailVerified,
		DisplayName:       name,
		PreferredUsername: preferredUsername,
	}, nil
}
