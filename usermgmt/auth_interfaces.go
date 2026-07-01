package usermgmt

import (
	"context"
	"encoding/json"
)

// Auth strategy interfaces — v4 Sollbruchstelle seam design.
//
// These interfaces define the contract between the core Service and the three
// optional auth strategy implementations (TOTP, WebAuthn, OAuth2). In v3, the
// Service struct holds concrete types (*webauthn.WebAuthn, *TOTPConfig,
// oauth2Providers) directly, forcing every consumer to pull in all auth deps.
//
// In v4, these interfaces will allow consumers to import only the auth
// strategies they need:
//
//	import usermgmt           // core user management
//	import usermgmt/webauthn  // passkey auth (pulls go-webauthn)
//	// consumer does NOT import usermgmt/totp or usermgmt/oauth2
//
// The interfaces use json.RawMessage and []byte for ceremony payloads to avoid
// importing go-webauthn's protocol types in the core package.
//
// See docs/modularization/2026-07-01_SOLLBRUCHSTELLEN.html for the full analysis.

// TOTPVerifier defines the TOTP multi-factor authentication contract.
// Implemented by usermgmt/totp.DefaultTOTP (v4).
type TOTPVerifier interface {
	// EnableTOTP generates a new TOTP secret for the user and returns the setup
	// response containing the secret and QR code URI.
	EnableTOTP(ctx context.Context, userID UserID) (*TOTPSetupResponse, error)

	// VerifyTOTPSetup confirms the TOTP setup by verifying a code from the
	// user's authenticator app.
	VerifyTOTPSetup(ctx context.Context, userID UserID, code string) error

	// VerifyTOTP checks a TOTP code against the user's stored secret.
	VerifyTOTP(ctx context.Context, userID UserID, code string) error

	// DisableTOTP removes TOTP for the user. Requires a valid code to prevent
	// MFA stripping.
	DisableTOTP(ctx context.Context, userID UserID, code string) error
}

// WebAuthnProvider defines the WebAuthn (passkey) authentication contract.
// Implemented by usermgmt/webauthn.DefaultProvider (v4).
//
// Ceremony payloads use json.RawMessage to avoid importing go-webauthn's
// protocol types in the core package. The implementation handles serialization.
type WebAuthnProvider interface {
	// BeginRegistration starts the credential registration ceremony.
	// Returns the CredentialCreation options (as JSON) and a session key.
	BeginRegistration(ctx context.Context, userID UserID) (options json.RawMessage, sessionKey string, err error)

	// FinishRegistration completes the credential registration ceremony.
	// The body is the raw assertion response from the authenticator.
	FinishRegistration(ctx context.Context, userID UserID, body []byte, credentialName string) error

	// BeginLogin starts the authentication ceremony.
	// Returns the CredentialAssertion options (as JSON) and a session key.
	BeginLogin(ctx context.Context, email string) (options json.RawMessage, sessionKey string, err error)

	// FinishLogin completes the authentication ceremony.
	// The body is the raw assertion response from the authenticator.
	// Returns the authentication result containing the user ID and session.
	FinishLogin(ctx context.Context, userID UserID, body []byte) (*AuthResult, error)
}

// OAuth2Provider defines the OAuth2/OIDC authentication contract.
// Implemented by usermgmt/oauth2.DefaultProvider (v4).
type OAuth2Provider interface {
	// BeginLogin starts the OAuth2 authorization code flow for the given provider.
	// Returns the redirect URL to which the user's browser should be sent.
	BeginLogin(ctx context.Context, provider string) (redirectURL string, err error)

	// FinishLogin completes the OAuth2 flow by exchanging the authorization
	// code for an access token and fetching user info.
	// Returns the external account info for user matching/creation.
	FinishLogin(ctx context.Context, provider, code, state string) (*ExternalAccount, error)
}

// Compile-time assertion: Service already satisfies TOTPVerifier today.
// This proves the interface is implementable and locks the contract.
// WebAuthnProvider and OAuth2Provider are v4 target shapes — the current
// Service methods use *http.Request and protocol-specific response types
// that will be replaced by json.RawMessage/[]byte in v4 sub-packages.
var _ TOTPVerifier = (*Service)(nil)
