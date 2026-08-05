package usermgmt

import (
	"net/http"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// All domain error sentinels now live in identity-model (domain-only, no HTTP
// dependency). Here we re-export them, wrapping those that need a non-default
// HTTP status with cqrshtmx.WithHTTPStatus. Errors whose family default is
// already correct (Rejection→400, Conflict→409, Transient→503) are aliased
// directly.
//
// ErrForbidden and ErrUnauthorized are NOT wrapped because cqrshtmx.MapError
// already maps codes "forbidden"→403 and "unauthorized"→401 via
// authStatusFromErrorCode.
var (
	// ErrUserNotFound is returned when a user cannot be located by ID.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrUserNotFound = cqrshtmx.WithHTTPStatus(identitymodel.ErrUserNotFound, http.StatusNotFound)
	// ErrUserIDExists is returned when attempting to create a user with an ID that already exists.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrUserIDExists = cqrshtmx.WithHTTPStatus(identitymodel.ErrUserIDExists, http.StatusNotFound)
	// ErrEmailExists is returned when attempting to register an email that is already taken.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrEmailExists = cqrshtmx.WithHTTPStatus(identitymodel.ErrEmailExists, http.StatusConflict)
	// ErrInvalidCredentials is returned when WebAuthn login fails.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrInvalidCredentials = cqrshtmx.WithHTTPStatus(identitymodel.ErrInvalidCredentials, http.StatusUnauthorized)
	// ErrNoCredentials is returned when a user has no registered WebAuthn credentials.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrNoCredentials = cqrshtmx.WithHTTPStatus(identitymodel.ErrNoCredentials, http.StatusNotFound)
	// ErrWebAuthnNotConfigured is returned when WebAuthn is required but not configured.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrWebAuthnNotConfigured = cqrshtmx.WithHTTPStatus(identitymodel.ErrWebAuthnNotConfigured, http.StatusUnauthorized)
	// ErrSessionDataNotFound is returned when WebAuthn session data is not found or expired.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrSessionDataNotFound = cqrshtmx.WithHTTPStatus(identitymodel.ErrSessionDataNotFound, http.StatusUnauthorized)
	// ErrSessionNotFound is returned when a session token does not match any stored session.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrSessionNotFound = cqrshtmx.WithHTTPStatus(identitymodel.ErrSessionNotFound, http.StatusNotFound)
	// ErrSessionExpired is returned when a session token has passed its expiration time.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrSessionExpired = cqrshtmx.WithHTTPStatus(identitymodel.ErrSessionExpired, http.StatusUnauthorized)
	// ErrForbidden is returned when an authorization check denies access.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrForbidden = identitymodel.ErrForbidden
	// ErrUnauthorized is returned when authentication is required but missing or invalid.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrUnauthorized = identitymodel.ErrUnauthorized
	// ErrValidation is returned when input validation fails (e.g. invalid email).
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrValidation = identitymodel.ErrValidation
	// ErrAccountLocked is returned when login is rejected because the account exceeded the
	// maximum allowed failed attempts.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrAccountLocked = cqrshtmx.WithHTTPStatus(identitymodel.ErrAccountLocked, http.StatusTooManyRequests)
	// ErrEnforcerNotInitialized is returned when an Authz operation is called before
	// the Casbin enforcer has been set up.
	ErrEnforcerNotInitialized = cqrshtmx.WithHTTPStatus(
		identitymodel.ErrEnforcerNotInitialized, http.StatusInternalServerError,
	)
	// ErrEmailVerificationNotConfigured is returned when email verification is
	// used without being configured in ServiceConfig.
	ErrEmailVerificationNotConfigured = cqrshtmx.WithHTTPStatus(
		identitymodel.ErrEmailVerificationNotConfigured,
		http.StatusServiceUnavailable,
	)
	// ErrInvalidVerificationToken is returned when a verification token is
	// invalid, already used, or expired.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrInvalidVerificationToken = identitymodel.ErrInvalidVerificationToken
	// ErrEmailAlreadyVerified is returned when verification is requested for
	// an already-verified email.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrEmailAlreadyVerified = cqrshtmx.WithHTTPStatus(identitymodel.ErrEmailAlreadyVerified, http.StatusConflict)
	// ErrTOTPNotConfigured is returned when TOTP MFA is used without being configured.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrTOTPNotConfigured = cqrshtmx.WithHTTPStatus(identitymodel.ErrTOTPNotConfigured, http.StatusServiceUnavailable)
	// ErrTOTPAlreadyEnabled is returned when TOTP setup is requested for a user who already has it.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrTOTPAlreadyEnabled = cqrshtmx.WithHTTPStatus(identitymodel.ErrTOTPAlreadyEnabled, http.StatusConflict)
	// ErrTOTPNotEnabled is returned when TOTP verification is requested for a user without TOTP.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrTOTPNotEnabled = identitymodel.ErrTOTPNotEnabled
	// ErrInvalidTOTPCode is returned when the provided TOTP code is invalid.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrInvalidTOTPCode = cqrshtmx.WithHTTPStatus(identitymodel.ErrInvalidTOTPCode, http.StatusUnauthorized)
	// ErrTOTPSetupExpired is returned when the pending TOTP setup has expired.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrTOTPSetupExpired = identitymodel.ErrTOTPSetupExpired
	// ErrOAuthNotConfigured is returned when OAuth2 is used without being configured.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrOAuthNotConfigured = cqrshtmx.WithHTTPStatus(identitymodel.ErrOAuthNotConfigured, http.StatusServiceUnavailable)
	// ErrOAuthProviderNotFound is returned when the requested provider is not configured.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrOAuthProviderNotFound = cqrshtmx.WithHTTPStatus(identitymodel.ErrOAuthProviderNotFound, http.StatusNotFound)
	// ErrOAuthInvalidState is returned when the state token is invalid, expired, or missing.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrOAuthInvalidState = identitymodel.ErrOAuthInvalidState
	// ErrOAuthTokenExchange is returned when exchanging the authorization code for a token fails.
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ErrOAuthTokenExchange = identitymodel.ErrOAuthTokenExchange
	// ErrExternalAccountAlreadyLinked is returned when an external account
	// (provider+subject pair) is already linked to a different user.
	ErrExternalAccountAlreadyLinked = cqrshtmx.WithHTTPStatus(
		identitymodel.ErrExternalAccountAlreadyLinked,
		http.StatusConflict,
	)
)
