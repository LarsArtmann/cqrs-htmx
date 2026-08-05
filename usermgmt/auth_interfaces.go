package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
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
//
// The canonical definitions live in identity-model (the domain source of
// truth) and are re-exported here as type aliases. See AGENTS.md
// "identity-model is the domain source of truth".

// TOTPProvider provides TOTP secret generation and code validation.
// Implemented by github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4.Provider.
// The interface uses only primitive types so the implementation module does
// not need to import core usermgmt.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type TOTPProvider = identitymodel.TOTPProvider

// WebAuthnProvider defines the WebAuthn (passkey) authentication contract.
// Implemented by usermgmt/webauthn.Provider (v4).
//
// All parameters use []byte / json.RawMessage to avoid importing go-webauthn's
// protocol types in the core package. The implementation handles serialization.
//
// userJSON contains serialized user data (id, email, display_name, credentials).
// options contains the ceremony options as JSON (CredentialCreation / CredentialAssertion).
// sessionData contains opaque serialized session state (stored by the Service).
// credentialJSON contains the new credential data as JSON.
// body contains the raw authenticator response body.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type WebAuthnProvider = identitymodel.WebAuthnProvider

// OAuth2Provider defines the OAuth2/OIDC authentication contract.
// Implemented by usermgmt/oauth2.Provider (v4).
//
// All parameters use primitive types so the implementation module does not
// import core usermgmt. The provider generates PKCE internally.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type OAuth2Provider = identitymodel.OAuth2Provider

// OAuth2UserInfo is the normalized user information returned by an OAuth2
// provider after token exchange. Used by the Service to deserialize the
// provider's JSON return value.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type OAuth2UserInfo = identitymodel.OAuth2UserInfo
