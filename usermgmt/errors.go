package usermgmt

import "github.com/cockroachdb/errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrForbidden          = errors.New("access denied")
	ErrUnauthorized       = errors.New("authentication required")
	ErrValidation         = errors.New("validation failed")
	ErrAccountLocked      = errors.New("account locked due to too many failed attempts")
)
