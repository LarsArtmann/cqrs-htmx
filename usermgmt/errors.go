package usermgmt

import (
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// Each sentinel is a classified event.Error (Rejection/Conflict family) wrapped
// with cqrshtmx.WithHTTPStatus when its correct HTTP status differs from the
// family default (Rejection→400, Conflict→409). Carrying the status on the
// error lets cqrshtmx.MapError derive it directly, so this module no longer
// maintains a parallel error→status switch (the former errorStatus split brain).
// errors.Is still matches: the wrapper Unwraps to the event.Error, which
// compares by code+family.
var (
	// ErrUserNotFound is returned when a user cannot be located by ID.
	ErrUserNotFound = notFound("usermgmt.user_not_found", "user not found")
	// ErrUserIDExists is returned when attempting to create a user with an ID that already exists.
	ErrUserIDExists = notFound("usermgmt.user_id_exists", "user ID already exists")
	// ErrEmailExists is returned when attempting to register an email that is already taken.
	ErrEmailExists = conflict("usermgmt.email_exists", "email already registered")
	// ErrInvalidCredentials is returned when WebAuthn login fails.
	ErrInvalidCredentials = unauthorized("usermgmt.invalid_credentials", "invalid credentials")
	// ErrNoCredentials is returned when a user has no registered WebAuthn credentials.
	ErrNoCredentials = notFound("usermgmt.no_credentials", "user has no registered credentials")
	// ErrWebAuthnNotConfigured is returned when WebAuthn is required but not configured.
	ErrWebAuthnNotConfigured = unauthorized("usermgmt.webauthn_not_configured", "WebAuthn is not configured")
	// ErrSessionDataNotFound is returned when WebAuthn session data is not found or expired.
	ErrSessionDataNotFound = unauthorized(
		"usermgmt.session_data_not_found",
		"WebAuthn session data not found or expired",
	)
	// ErrSessionNotFound is returned when a session token does not match any stored session.
	ErrSessionNotFound = notFound("usermgmt.session_not_found", "session not found")
	// ErrSessionExpired is returned when a session token has passed its expiration time.
	ErrSessionExpired = unauthorized("usermgmt.session_expired", "session expired")
	// ErrForbidden is returned when an authorization check denies access.
	// Uses the same code as root cqrshtmx.ErrForbidden for cross-module compatibility.
	ErrForbidden = forbidden()
	// ErrUnauthorized is returned when authentication is required but missing or invalid.
	// Uses the same code as root cqrshtmx.ErrUnauthorized for cross-module compatibility.
	ErrUnauthorized = unauthorized(cqrshtmx.CodeUnauthorized, "authentication required")
	// ErrValidation is returned when input validation fails (e.g. invalid email).
	// Rejection family default (400) is correct — no status override needed.
	ErrValidation = event.NewRejection("usermgmt.validation", "validation failed")
	// ErrAccountLocked is returned when login is rejected because the account exceeded the
	// maximum allowed failed attempts.
	ErrAccountLocked = withStatus(
		"usermgmt.account_locked", "account locked due to too many failed attempts",
		http.StatusTooManyRequests,
	)
	// ErrEnforcerNotInitialized is returned when an Authz operation is called before
	// the Casbin enforcer has been set up.
	ErrEnforcerNotInitialized = withStatus(
		"usermgmt.enforcer_not_initialized", "authorization enforcer not initialized",
		http.StatusInternalServerError,
	)
	// ErrEmailVerificationNotConfigured is returned when email verification is
	// used without being configured in ServiceConfig.
	ErrEmailVerificationNotConfigured = serviceUnavailable(
		"usermgmt.email_verification_not_configured", "email verification is not configured",
	)
	// ErrInvalidVerificationToken is returned when a verification token is
	// invalid, already used, or expired. Rejection family default (400) is correct.
	ErrInvalidVerificationToken = event.NewRejection(
		"usermgmt.invalid_verification_token",
		"verification token is invalid or expired",
	)

	// ErrEmailAlreadyVerified is returned when verification is requested for
	// an already-verified email.
	ErrEmailAlreadyVerified = conflict("usermgmt.email_already_verified", "email is already verified")
	// ErrTOTPNotConfigured is returned when TOTP MFA is used without being configured.
	ErrTOTPNotConfigured = serviceUnavailable("usermgmt.totp_not_configured", "TOTP is not configured")
	// ErrTOTPAlreadyEnabled is returned when TOTP setup is requested for a user who already has it.
	ErrTOTPAlreadyEnabled = conflict("usermgmt.totp_already_enabled", "TOTP is already enabled for this user")
	// ErrTOTPNotEnabled is returned when TOTP verification is requested for a user without TOTP.
	// Rejection family default (400) is correct.
	ErrTOTPNotEnabled = event.NewRejection("usermgmt.totp_not_enabled", "TOTP is not enabled for this user")
	// ErrInvalidTOTPCode is returned when the provided TOTP code is invalid.
	ErrInvalidTOTPCode = unauthorized("usermgmt.invalid_totp_code", "invalid TOTP code")
	// ErrTOTPSetupExpired is returned when the pending TOTP setup has expired.
	// Rejection family default (400) is correct.
	ErrTOTPSetupExpired = event.NewRejection("usermgmt.totp_setup_expired", "TOTP setup has expired, please try again")
	// ErrOAuthNotConfigured is returned when OAuth2 is used without being configured.
	ErrOAuthNotConfigured = serviceUnavailable("usermgmt.oauth_not_configured", "OAuth2 is not configured")
	// ErrOAuthProviderNotFound is returned when the requested provider is not configured.
	ErrOAuthProviderNotFound = notFound("usermgmt.oauth_provider_not_found", "OAuth2 provider not found")
	// ErrOAuthInvalidState is returned when the state token is invalid, expired, or missing.
	// Rejection family default (400) is correct.
	ErrOAuthInvalidState = event.NewRejection(
		"usermgmt.oauth_invalid_state",
		"OAuth2 state token is invalid or expired",
	)
	// ErrOAuthTokenExchange is returned when exchanging the authorization code for a token fails.
	// Rejection family default (400) is correct.
	ErrOAuthTokenExchange = event.NewRejection("usermgmt.oauth_token_exchange_failed", "OAuth2 token exchange failed")
	// ErrExternalAccountAlreadyLinked is returned when an external account
	// (provider+subject pair) is already linked to a different user.
	// Conflict family default (409) is correct — no status override needed.
	ErrExternalAccountAlreadyLinked = event.NewConflict(
		"usermgmt.external_account_linked_to_other",
		"external account is already linked to another user",
	)
)

// withStatus wraps a Rejection-classified error carrying an explicit HTTP status.
// Used when the correct status differs from the Rejection default (400).
func withStatus(code, msg string, status int) error {
	return cqrshtmx.WithHTTPStatus(event.NewRejection(code, msg), status) //nolint:wrapcheck // wrapping boundary
}

// Semantic helpers for the common non-default statuses, keeping the sentinel
// declarations above readable and self-documenting.
func unauthorized(code, msg string) error {
	return withStatus(code, msg, http.StatusUnauthorized)
}

func forbidden() error {
	return withStatus(cqrshtmx.CodeForbidden, "access denied", http.StatusForbidden)
}

func notFound(code, msg string) error {
	return withStatus(code, msg, http.StatusNotFound)
}

func conflict(code, msg string) error {
	return withStatus(code, msg, http.StatusConflict)
}

func serviceUnavailable(code, msg string) error {
	return withStatus(code, msg, http.StatusServiceUnavailable)
}
