package loginpage

import (
	"strings"

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

// safeRedirectPath returns path if it is a safe same-origin redirect target
// (starts with "/" but not "//"), otherwise "/". This prevents open-redirect
// attacks from user-controlled input.
func safeRedirectPath(path string) string {
	if path == "" || path[0] != '/' || len(path) > 1 && path[1] == '/' {
		return "/"
	}
	return path
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
