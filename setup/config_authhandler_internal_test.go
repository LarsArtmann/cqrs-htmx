package setup

import (
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Tests pinning resolveAuthHandlerConfig / validateAuthHandlerConfig (M2 of
// the 2026-09-17 gap-bundle plan): the AuthHandlerConfig seam must be a pure
// opt-in — nil is byte-identical to the historical one-field literal, and an
// explicit value is copied with only the cookie name defaulted.

func TestResolveAuthHandlerConfig_NilIsTheHistoricalLiteral(t *testing.T) {
	t.Parallel()

	got := resolveAuthHandlerConfig(Config{CookieName: "session"})

	// HandlerConfig embeds a func field, so it is not struct-comparable —
	// pin every non-func field plus the nil func explicitly.
	if got.CookieName != "session" {
		t.Errorf("CookieName: got %q, want %q", got.CookieName, "session")
	}

	if got.Secure != nil {
		t.Errorf("Secure: got %v, want nil (NewAuthHandler applies the default)", *got.Secure)
	}

	if got.SessionMaxAge != 0 || got.Timeout != 0 {
		t.Errorf("SessionMaxAge/Timeout: got %d/%d, want 0/0", got.SessionMaxAge, got.Timeout)
	}

	for name, limit := range map[string]usermgmt.RateLimitConfig{
		"RegistrationRateLimit": got.RegistrationRateLimit,
		"ImportRateLimit":       got.ImportRateLimit,
		"TOTPRateLimit":         got.TOTPRateLimit,
		"VerificationRateLimit": got.VerificationRateLimit,
		"WebAuthnRateLimit":     got.WebAuthnRateLimit,
		"OAuthRateLimit":        got.OAuthRateLimit,
	} {
		if limit.Enabled || limit.MaxRequests != 0 || limit.Window != 0 {
			t.Errorf("%s: got %+v, want the zero value", name, limit)
		}
	}

	if got.OAuth2SuccessURL != "" || got.OAuth2ErrorURL != "" {
		t.Errorf("OAuth2 URLs: got %q/%q, want empty", got.OAuth2SuccessURL, got.OAuth2ErrorURL)
	}

	if got.ImportExportAuthorizer != nil {
		t.Error("ImportExportAuthorizer: got non-nil, want nil (NewAuthHandler applies the default)")
	}
}

func TestResolveAuthHandlerConfig_EmptyCookieNameInherits(t *testing.T) {
	t.Parallel()

	got := resolveAuthHandlerConfig(Config{
		CookieName: "my-session",
		AuthHandlerConfig: &usermgmt.HandlerConfig{
			OAuth2SuccessURL: "/after-login",
		},
	})

	if got.CookieName != "my-session" {
		t.Errorf("empty CookieName inside AuthHandlerConfig must inherit Config.CookieName, got %q", got.CookieName)
	}

	if got.OAuth2SuccessURL != "/after-login" {
		t.Errorf("AuthHandlerConfig fields must pass through, got OAuth2SuccessURL %q", got.OAuth2SuccessURL)
	}
}

func TestResolveAuthHandlerConfig_ExplicitCookieNameWins(t *testing.T) {
	t.Parallel()

	got := resolveAuthHandlerConfig(Config{
		CookieName:        "session",
		AuthHandlerConfig: &usermgmt.HandlerConfig{CookieName: "session"},
	})

	if got.CookieName != "session" {
		t.Errorf("matching CookieName must be kept, got %q", got.CookieName)
	}
}

func TestResolveAuthHandlerConfig_ReturnsACopy(t *testing.T) {
	t.Parallel()

	supplied := &usermgmt.HandlerConfig{Timeout: 1}
	resolved := resolveAuthHandlerConfig(Config{AuthHandlerConfig: supplied})

	resolved.Timeout = 2
	if supplied.Timeout != 1 {
		t.Error("resolveAuthHandlerConfig must copy: mutating the result must not affect the caller's struct")
	}

	supplied.OAuth2SuccessURL = "/mutated-after"
	if resolved.OAuth2SuccessURL == "/mutated-after" {
		t.Error("resolveAuthHandlerConfig must copy: mutating the caller's struct must not affect the bundle")
	}
}

func TestValidateAuthHandlerConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{name: "nil passes", cfg: Config{CookieName: "session"}},
		{
			name:    "matching cookie name passes",
			cfg:     Config{CookieName: "session", AuthHandlerConfig: &usermgmt.HandlerConfig{CookieName: "session"}},
			wantErr: false,
		},
		{
			name:    "empty cookie name inherits (passes)",
			cfg:     Config{CookieName: "session", AuthHandlerConfig: &usermgmt.HandlerConfig{}},
			wantErr: false,
		},
		{
			name: "mismatched cookie name rejected",
			cfg: Config{
				CookieName:        "session",
				AuthHandlerConfig: &usermgmt.HandlerConfig{CookieName: "other-cookie"},
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.validateAuthHandlerConfig()
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateAuthHandlerConfig() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
