package identitymodel

import errorfamily "github.com/larsartmann/go-error-family"

// Domain error sentinels for identity management.
//
// These errors are classified using errorfamily (Rejection/Conflict/Transient)
// without any HTTP status dependency. The infrastructure layer (usermgmt, root
// cqrshtmx) wraps them with HTTP status codes as needed via
// cqrshtmx.WithHTTPStatus.
//
// Error codes retain the "usermgmt." prefix for backward compatibility with
// existing consumers that match on error codes.
var (
	// ErrUserNotFound is returned when a user cannot be located by ID.
	ErrUserNotFound = errorfamily.NewRejection("usermgmt.user_not_found", "user not found")
	// ErrUserIDExists is returned when attempting to create a user with an ID that already exists.
	ErrUserIDExists = errorfamily.NewRejection("usermgmt.user_id_exists", "user ID already exists")
	// ErrEmailExists is returned when attempting to register an email that is already taken.
	ErrEmailExists = errorfamily.NewConflict("usermgmt.email_exists", "email already registered")
	// ErrInvalidCredentials is returned when WebAuthn login fails.
	ErrInvalidCredentials = errorfamily.NewRejection("usermgmt.invalid_credentials", "invalid credentials")
	// ErrNoCredentials is returned when a user has no registered WebAuthn credentials.
	ErrNoCredentials = errorfamily.NewRejection("usermgmt.no_credentials", "user has no registered credentials")
	// ErrWebAuthnNotConfigured is returned when WebAuthn is required but not configured.
	ErrWebAuthnNotConfigured = errorfamily.NewRejection(
		"usermgmt.webauthn_not_configured",
		"WebAuthn is not configured",
	)
	// ErrSessionDataNotFound is returned when WebAuthn session data is not found or expired.
	ErrSessionDataNotFound = errorfamily.NewRejection(
		"usermgmt.session_data_not_found",
		"WebAuthn session data not found or expired",
	)
	// ErrSessionNotFound is returned when a session token does not match any stored session.
	ErrSessionNotFound = errorfamily.NewRejection("usermgmt.session_not_found", "session not found")
	// ErrSessionExpired is returned when a session token has passed its expiration time.
	ErrSessionExpired = errorfamily.NewRejection("usermgmt.session_expired", "session expired")
	// ErrForbidden is returned when an authorization check denies access.
	ErrForbidden = errorfamily.NewRejection("forbidden", "access denied")
	// ErrUnauthorized is returned when authentication is required but missing or invalid.
	ErrUnauthorized = errorfamily.NewRejection("unauthorized", "authentication required")
	// ErrValidation is returned when input validation fails (e.g. invalid email).
	ErrValidation = errorfamily.NewRejection("usermgmt.validation", "validation failed")
	// ErrAccountLocked is returned when login is rejected because the account exceeded the
	// maximum allowed failed attempts.
	ErrAccountLocked = errorfamily.NewRejection(
		"usermgmt.account_locked", "account locked due to too many failed attempts",
	)
	// ErrEnforcerNotInitialized is returned when an Authz operation is called before
	// the Casbin enforcer has been set up.
	ErrEnforcerNotInitialized = errorfamily.NewRejection(
		"usermgmt.enforcer_not_initialized", "authorization enforcer not initialized",
	)
	// ErrEmailVerificationNotConfigured is returned when email verification is
	// used without being configured in ServiceConfig.
	ErrEmailVerificationNotConfigured = errorfamily.NewTransient(
		"usermgmt.email_verification_not_configured", "email verification is not configured",
	)
	// ErrInvalidVerificationToken is returned when a verification token is
	// invalid, already used, or expired.
	ErrInvalidVerificationToken = errorfamily.NewRejection(
		"usermgmt.invalid_verification_token",
		"verification token is invalid or expired",
	)
	// ErrEmailAlreadyVerified is returned when verification is requested for
	// an already-verified email.
	ErrEmailAlreadyVerified = errorfamily.NewConflict("usermgmt.email_already_verified", "email is already verified")
	// ErrTOTPNotConfigured is returned when TOTP MFA is used without being configured.
	ErrTOTPNotConfigured = errorfamily.NewTransient("usermgmt.totp_not_configured", "TOTP is not configured")
	// ErrTOTPAlreadyEnabled is returned when TOTP setup is requested for a user who already has it.
	ErrTOTPAlreadyEnabled = errorfamily.NewConflict(
		"usermgmt.totp_already_enabled",
		"TOTP is already enabled for this user",
	)
	// ErrTOTPNotEnabled is returned when TOTP verification is requested for a user without TOTP.
	ErrTOTPNotEnabled = errorfamily.NewRejection("usermgmt.totp_not_enabled", "TOTP is not enabled for this user")
	// ErrInvalidTOTPCode is returned when the provided TOTP code is invalid.
	ErrInvalidTOTPCode = errorfamily.NewRejection("usermgmt.invalid_totp_code", "invalid TOTP code")
	// ErrTOTPSetupExpired is returned when the pending TOTP setup has expired.
	ErrTOTPSetupExpired = errorfamily.NewRejection(
		"usermgmt.totp_setup_expired",
		"TOTP setup has expired, please try again",
	)
	// ErrOAuthNotConfigured is returned when OAuth2 is used without being configured.
	ErrOAuthNotConfigured = errorfamily.NewTransient("usermgmt.oauth_not_configured", "OAuth2 is not configured")
	// ErrOAuthProviderNotFound is returned when the requested provider is not configured.
	ErrOAuthProviderNotFound = errorfamily.NewRejection(
		"usermgmt.oauth_provider_not_found",
		"OAuth2 provider not found",
	)
	// ErrOAuthInvalidState is returned when the state token is invalid, expired, or missing.
	ErrOAuthInvalidState = errorfamily.NewRejection(
		"usermgmt.oauth_invalid_state",
		"OAuth2 state token is invalid or expired",
	)
	// ErrOAuthTokenExchange is returned when exchanging the authorization code for a token fails.
	ErrOAuthTokenExchange = errorfamily.NewRejection(
		"usermgmt.oauth_token_exchange_failed",
		"OAuth2 token exchange failed",
	)
	// ErrExternalAccountAlreadyLinked is returned when an external account
	// (provider+subject pair) is already linked to a different user.
	ErrExternalAccountAlreadyLinked = errorfamily.NewConflict(
		"usermgmt.external_account_linked_to_other",
		"external account is already linked to another user",
	)
)
