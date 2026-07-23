package identitymodel

import (
	"net/mail"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const maxEmailLength = 254 // RFC 5321 max

// Email is a validated, normalized email address.
// Use ParseEmail or MustParseEmail to construct from user input.
type Email string

// ParseEmail validates and normalizes a raw email string.
func ParseEmail(raw string) (Email, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "", errorfamily.WrapRejection(ErrValidation, "usermgmt.email.required", "email is required")
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", errorfamily.Wrapf(ErrValidation, event.Rejection, "usermgmt.email.invalid", "invalid email %q", s)
	}
	if len(addr.Address) > maxEmailLength {
		return "", errorfamily.Wrapf(
			ErrValidation,
			event.Rejection,
			"usermgmt.email.too_long",
			"email too long (max %d)",
			maxEmailLength,
		)
	}
	return Email(addr.Address), nil
}

// MustParseEmail panics if the email is invalid. For use in tests and constants.
func MustParseEmail(raw string) Email {
	email, err := ParseEmail(raw)
	if err != nil {
		panic(err)
	}
	return email
}

// String returns the email as a plain string.
func (e Email) String() string { return string(e) }
