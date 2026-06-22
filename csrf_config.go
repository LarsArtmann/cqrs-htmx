package cqrshtmx

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/justinas/nosurf"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

const (
	defaultCSRFCookieName = "csrf_token"
	defaultCSRFHeaderName = "X-CSRF-Token"
	defaultCSRFFieldName  = "csrf_token"
	defaultCSRFMaxAge     = 24 * time.Hour
)

// ForbiddenErrorHandler is a minimal CSRFConfig.ErrorHandler that responds
// with HTTP 403 Forbidden and no body. Useful for tests and for consumers who
// want to handle CSRF failures via a separate middleware (e.g., HTMX-aware
// error responses) rather than rendering the underlying nosurf error.
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    CSRF: cqrshtmx.CSRFConfig{
//	        ErrorHandler: cqrshtmx.ForbiddenErrorHandler,
//	    },
//	})
func ForbiddenErrorHandler(w http.ResponseWriter, _ *http.Request, _ error) {
	w.WriteHeader(http.StatusForbidden)
}

// ErrCSRFInvalid is returned when a CSRF token is missing, malformed, or does not match.
// Uses justinas/nosurf under the hood for token generation and validation.
var ErrCSRFInvalid = event.NewRejection("csrf_invalid", "invalid or missing CSRF token").WithCause(ErrForbidden)

// CSRFConfig configures CSRF protection.
//
// All fields are optional; zero values use secure defaults.
// Uses justinas/nosurf internally for token generation, masking (BREACH mitigation),
// cookie management, and validation.
type CSRFConfig struct {
	// CookieName is the name of the CSRF cookie.
	// Default: "csrf_token"
	CookieName string

	// HeaderName is the request header containing the CSRF token.
	// HTMX sends this header when configured with hx-headers.
	// Default: "X-CSRF-Token"
	HeaderName string

	// FieldName is the form field name containing the CSRF token.
	// Checked as fallback when the header is not present.
	// Default: "csrf_token"
	FieldName string

	// MaxAge is the cookie max age.
	// Default: 24 hours
	MaxAge time.Duration

	// Secure sets the Secure flag on the cookie.
	// Default: false (auto-detected from request scheme)
	Secure bool

	// SameSite sets the SameSite attribute on the cookie.
	// Default: http.SameSiteLaxMode
	SameSite http.SameSite

	// Domain sets the cookie domain.
	// Default: "" (host-only cookie)
	Domain string

	// Path sets the cookie path.
	// Default: "/"
	Path string

	// TrustedOrigins configures origins allowed for cross-domain CSRF.
	// Default: nil (same-origin only)
	TrustedOrigins []string

	// TrustedProxies lists the IP addresses (or CIDR-notation networks) of
	// reverse proxies that may strip/forward X-Forwarded-* and similar headers.
	// Used by the plaintext-HTTP origin bypass: a request with no Origin/
	// Referer/Sec-Fetch-Site header is only auto-marked as same-origin when
	// the RemoteAddr is one of these trusted proxies (or loopback).
	// Examples: []string{"10.0.0.1", "192.168.1.0/24"}.
	// Use TrustedProxiesCIDR for parsed CIDR networks.
	TrustedProxies []string

	// TrustedProxiesCIDR is the parsed form of TrustedProxies CIDR entries.
	// TrustedProxies strings that are not valid CIDR networks are skipped
	// (logged at startup).
	TrustedProxiesCIDR []*net.IPNet

	// ErrorHandler is called when CSRF validation fails.
	// Default: writes 403 Forbidden with plain text
	ErrorHandler ErrorHandler
}

func (c *CSRFConfig) cookieName() string {
	if c.CookieName != "" {
		return c.CookieName
	}
	return defaultCSRFCookieName
}

func (c *CSRFConfig) headerName() string {
	if c.HeaderName != "" {
		return c.HeaderName
	}
	return defaultCSRFHeaderName
}

func (c *CSRFConfig) fieldName() string {
	if c.FieldName != "" {
		return c.FieldName
	}
	return defaultCSRFFieldName
}

func (c *CSRFConfig) maxAge() time.Duration {
	if c.MaxAge > 0 {
		return c.MaxAge
	}
	return defaultCSRFMaxAge
}

func (c *CSRFConfig) path() string {
	if c.Path != "" {
		return c.Path
	}
	return "/"
}

// Validate checks the CSRF configuration for common misconfigurations.
// Returns a non-nil error if the config would produce insecure or broken behavior.
// Call this in production startup code to fail fast on misconfiguration.
func (c *CSRFConfig) Validate() error {
	if c.SameSite == http.SameSiteNoneMode && !c.Secure {
		return event.NewInfrastructure("csrf_samesite_insecure", "SameSite=None requires Secure=true").
			WithCause(ErrCSRFConfig)
	}

	for _, origin := range c.TrustedOrigins {
		if origin == "" || origin == "*" {
			return event.NewInfrastructure("csrf_unsafe_origin",
				fmt.Sprintf("TrustedOrigins contains unsafe entry %q — use specific domain names only",
					origin)).WithCause(ErrCSRFConfig)
		}
	}

	if !c.Secure {
		slog.Warn("cqrs-htmx: CSRFConfig.Validate: Secure is false — CSRF cookies will be sent over plain HTTP",
			slog.String("hint", "set Secure=true in production"))
	}

	// Parse TrustedProxies CIDR entries. Plain strings (non-CIDR) are kept
	// verbatim in TrustedProxies; CIDR strings are moved to TrustedProxiesCIDR.
	c.TrustedProxiesCIDR = nil
	for _, p := range c.TrustedProxies {
		if p == "" {
			return event.NewInfrastructure("csrf_unsafe_proxy",
				"TrustedProxies contains empty entry").WithCause(ErrCSRFConfig)
		}
		if strings.Contains(p, "/") {
			_, ipnet, err := net.ParseCIDR(p)
			if err != nil {
				return event.NewInfrastructure("csrf_invalid_cidr",
					fmt.Sprintf("TrustedProxies contains invalid CIDR %q: %v", p, err)).
					WithCause(ErrCSRFConfig)
			}
			c.TrustedProxiesCIDR = append(c.TrustedProxiesCIDR, ipnet)
		}
	}

	return nil
}

// configureNosurfHandler applies CSRFConfig settings to a nosurf handler.
func configureNosurfHandler(handler *nosurf.CSRFHandler, cfg CSRFConfig) {
	//nolint:gosec,exhaustruct // HttpOnly=false required for double-submit; http.Cookie has many optional fields
	cookie := http.Cookie{
		Name:     cfg.cookieName(),
		Path:     cfg.path(),
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: cfg.SameSite,
		MaxAge:   int(cfg.maxAge().Seconds()),
	}
	if cfg.Domain != "" {
		cookie.Domain = cfg.Domain
	}
	handler.SetBaseCookie(cookie)

	handler.SetIsTLSFunc(func(r *http.Request) bool {
		return r.TLS != nil
	})

	if len(cfg.TrustedOrigins) > 0 {
		origins, err := nosurf.StaticOrigins(cfg.TrustedOrigins...)
		if err != nil {
			slog.Error(
				"cqrs-htmx: invalid TrustedOrigins",
				slog.String("error", err.Error()),
			)
		} else {
			handler.SetIsAllowedOriginFunc(origins)
		}
	}

	failureHandler := cfg.ErrorHandler
	if failureHandler == nil {
		failureHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", ContentTypePlain)
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(err.Error())) //nolint:gosec // text/plain prevents HTML rendering
		}
	}

	handler.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reason := nosurf.Reason(r); reason != nil {
			slog.Warn(
				"cqrs-htmx: CSRF validation failed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("reason", reason.Error()),
			)
		}
		failureHandler(w, r, ErrCSRFInvalid)
	}))
}
