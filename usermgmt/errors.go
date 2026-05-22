package usermgmt

import "github.com/cockroachdb/errors"

var (
	// ErrUserNotFound is returned when a user cannot be located by ID.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserIDExists is returned when attempting to create a user with an ID that already exists.
	ErrUserIDExists = errors.New("user ID already exists")
	// ErrEmailExists is returned when attempting to register an email that is already taken.
	ErrEmailExists = errors.New("email already registered")
	// ErrInvalidCredentials is returned when login fails due to wrong email or password.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrSessionNotFound is returned when a session token does not match any stored session.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired is returned when a session token has passed its expiration time.
	ErrSessionExpired = errors.New("session expired")
	// ErrForbidden is returned when an authorization check denies access.
	ErrForbidden = errors.New("access denied")
	// ErrUnauthorized is returned when authentication is required but missing or invalid.
	ErrUnauthorized = errors.New("authentication required")
	// ErrValidation is returned when input validation fails (e.g. invalid email, short password).
	ErrValidation = errors.New("validation failed")
	// ErrAccountLocked is returned when login is rejected because the account exceeded the
	// maximum allowed failed login attempts.
	ErrAccountLocked = errors.New("account locked due to too many failed attempts")
)
