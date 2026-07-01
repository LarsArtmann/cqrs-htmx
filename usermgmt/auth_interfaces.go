package usermgmt

import (
	"context"
	"encoding/json"
)

// Auth strategy interfaces — v4 Sollbruchstelle seam design.
//
// These interfaces define the contract between the core Service and the three
// optional auth strategy implementations (TOTP, WebAuthn, OAuth2). In v4,
// consumers import only the auth strategies they need:
//
//	import usermgmt           // core user management
//	import usermgmt/totp      // TOTP MFA (pulls pquerna/otp)
//	import usermgmt/webauthn  // passkey auth (pulls go-webauthn)
//	// consumer does NOT import usermgmt/oauth2 if they don't need it
//
// The interfaces use primitive types ([]byte, string) so the implementations
// can live in separate modules without importing core usermgmt.

// TOTPProvider provides TOTP secret generation and code validation.
// Implemented by github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4.Provider.
// The interface uses only primitive types so the implementation module does
// not need to import core usermgmt.
type TOTPProvider interface {
	// GenerateSecret creates a new TOTP secret for the given account name.
	// Returns the raw secret bytes (for storage), the base32-encoded secret
	// (for display to the user), and the otpauth:// URI (for QR codes).
	GenerateSecret(accountName string) (rawSecret []byte, base32Secret, otpauthURI string, err error)

	// ValidateCode checks if the given code is valid for the provided raw
	// secret bytes.
	ValidateCode(rawSecret []byte, code string) bool
}

// WebAuthnProvider defines the WebAuthn (passkey) authentication contract.
// Implemented by usermgmt/webauthn.Provider (v4).
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
// Implemented by usermgmt/oauth2.Provider (v4).
type OAuth2Provider interface {
	// BeginLogin starts the OAuth2 authorization code flow for the given provider.
	// Returns the redirect URL to which the user's browser should be sent.
	BeginLogin(ctx context.Context, provider string) (redirectURL string, err error)

	// FinishLogin completes the OAuth2 flow by exchanging the authorization
	// code for an access token and fetching user info.
	// Returns the external account info for user matching/creation.
	FinishLogin(ctx context.Context, provider, code, state string) (*ExternalAccount, error)
}
