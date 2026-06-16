package usermgmt

import "github.com/larsartmann/go-cqrs-lite/event/v2"

var (
	// ErrUserNotFound is returned when a user cannot be located by ID.
	ErrUserNotFound = event.NewRejection("usermgmt.user_not_found", "user not found")
	// ErrUserIDExists is returned when attempting to create a user with an ID that already exists.
	ErrUserIDExists = event.NewRejection("usermgmt.user_id_exists", "user ID already exists")
	// ErrEmailExists is returned when attempting to register an email that is already taken.
	ErrEmailExists = event.NewRejection("usermgmt.email_exists", "email already registered")
	// ErrInvalidCredentials is returned when WebAuthn login fails.
	ErrInvalidCredentials = event.NewRejection(
		"usermgmt.invalid_credentials",
		"invalid credentials",
	)
	// ErrNoCredentials is returned when a user has no registered WebAuthn credentials.
	ErrNoCredentials = event.NewRejection(
		"usermgmt.no_credentials",
		"user has no registered credentials",
	)
	// ErrWebAuthnNotConfigured is returned when WebAuthn is required but not configured.
	ErrWebAuthnNotConfigured = event.NewRejection(
		"usermgmt.webauthn_not_configured",
		"WebAuthn is not configured",
	)
	// ErrSessionDataNotFound is returned when WebAuthn session data is not found or expired.
	ErrSessionDataNotFound = event.NewRejection(
		"usermgmt.session_data_not_found",
		"WebAuthn session data not found or expired",
	)
	// ErrSessionNotFound is returned when a session token does not match any stored session.
	ErrSessionNotFound = event.NewRejection("usermgmt.session_not_found", "session not found")
	// ErrSessionExpired is returned when a session token has passed its expiration time.
	ErrSessionExpired = event.NewRejection("usermgmt.session_expired", "session expired")
	// ErrForbidden is returned when an authorization check denies access.
	ErrForbidden = event.NewRejection("usermgmt.forbidden", "access denied")
	// ErrUnauthorized is returned when authentication is required but missing or invalid.
	ErrUnauthorized = event.NewRejection("usermgmt.unauthorized", "authentication required")
	// ErrValidation is returned when input validation fails (e.g. invalid email).
	ErrValidation = event.NewRejection("usermgmt.validation", "validation failed")
	// ErrAccountLocked is returned when login is rejected because the account exceeded the
	// maximum allowed failed attempts.
	ErrAccountLocked = event.NewRejection(
		"usermgmt.account_locked",
		"account locked due to too many failed attempts",
	)
	// ErrEnforcerNotInitialized is returned when an Authz operation is called before
	// the Casbin enforcer has been set up.
	ErrEnforcerNotInitialized = event.NewRejection(
		"usermgmt.enforcer_not_initialized",
		"authorization enforcer not initialized",
	)
	// ErrEmailVerificationNotConfigured is returned when email verification is
	// used without being configured in ServiceConfig.
	ErrEmailVerificationNotConfigured = event.NewRejection(
		"usermgmt.email_verification_not_configured",
		"email verification is not configured",
	)
	// ErrInvalidVerificationToken is returned when a verification token is
	// invalid, already used, or expired.
	ErrInvalidVerificationToken = event.NewRejection(
		"usermgmt.invalid_verification_token",
		"verification token is invalid or expired",
	)
	// ErrEmailAlreadyVerified is returned when verification is requested for
	// an already-verified email.
	ErrEmailAlreadyVerified = event.NewRejection(
		"usermgmt.email_already_verified",
		"email is already verified",
	)
)
