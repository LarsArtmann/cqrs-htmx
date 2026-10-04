package loginpage

import (
	"strings"
	"unicode"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// errConfig returns a Rejection-family error for invalid configuration.
func errConfig(msg string) error {
	return errorfamily.NewRejection("loginpage.config", msg)
}

// validateAccentColor rejects values that cannot be safely embedded in the
// page's raw <style> block or the SVG data-URI favicon. Legitimate CSS colors
// (hex, rgb()/hsl()/color-mix() notation, var() references) never contain
// markup or quote characters; anything that does would be an injection attempt.
func validateAccentColor(accent string) error {
	if strings.ContainsAny(accent, `<>&"'\`+"`") {
		return errConfig(
			"Config.AccentColor must be a CSS color without markup or quote characters (< > & \" ' \\ `)",
		)
	}
	return nil
}

// validateStylesheetURL ensures Config.CSSPath is a stylesheet reference, not
// a scheme the browser would execute (e.g. javascript:). Only root-relative
// paths and absolute http(s) URLs are accepted.
func validateStylesheetURL(cssPath string) error {
	if cssPath == "" ||
		strings.HasPrefix(cssPath, "/") ||
		strings.HasPrefix(cssPath, "http://") ||
		strings.HasPrefix(cssPath, "https://") {
		return nil
	}
	return errConfig(
		`Config.CSSPath must be a root-relative path ("/css/app.css") or an absolute http(s) URL`,
	)
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

// firstRune returns the first letter or number in s as a string, or "?" if s
// contains none. Used for the favicon initial: SVG favicon text cannot render
// emoji/symbol glyphs reliably (they show as tofu boxes in many browsers), so
// a leading emoji like "🚀X" falls through to the first printable letter "X"
// instead of the raw first code point.
func firstRune(s string) string {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return string(r)
		}
	}
	return "?"
}
