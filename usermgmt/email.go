package usermgmt

import (
	"fmt"
	"net/mail"
	"strings"
)

// Email is a validated, normalized email address.
// Use ParseEmail or MustParseEmail to construct from user input.
// The zero value (empty string) represents "no email" (invalid).
type Email string

// ParseEmail validates and normalizes a raw email string.
// Trims whitespace, lowercases, and verifies RFC 5322 format.
// Returns ErrValidation if the email is empty or invalid.
func ParseEmail(raw string) (Email, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "", fmt.Errorf("%w: email is required", ErrValidation)
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", fmt.Errorf("%w: invalid email %q: %s", ErrValidation, s, err)
	}
	if len(addr.Address) > maxEmailLength {
		return "", fmt.Errorf("%w: email too long (max %d)", ErrValidation, maxEmailLength)
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
