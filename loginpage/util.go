package loginpage

import (
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// errConfig returns a Rejection-family error for invalid configuration.
func errConfig(msg string) error {
	return errorfamily.NewRejection("loginpage.config", msg)
}

// trimTrailingSlash removes trailing "/" so prefixes never end with one.
func trimTrailingSlash(s string) string {
	return strings.TrimRight(s, "/")
}

// safeRedirectPath delegates to the shared cqrshtmx.SafeRedirectPath to
// prevent open-redirect attacks from user-controlled input.
func safeRedirectPath(path string) string {
	return cqrshtmx.SafeRedirectPath(path)
}

// firstRune returns the first Unicode code point of s as a string, or "?" if
// s is empty. Used for the favicon initial.
func firstRune(s string) string {
	if s == "" {
		return "?"
	}
	r := []rune(s)
	return string(r[0])
}
