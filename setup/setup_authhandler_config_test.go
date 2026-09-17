package setup_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Black-box coverage for Config.AuthHandlerConfig (M2 of the 2026-09-17
// gap-bundle plan): the seam must make usermgmt's HTTP-layer hardening
// reachable through the bundle — rate limits answer 429, OAuth2 redirects
// flow through, the cookie contract stays enforced — while nil stays
// byte-identical to the pre-seam behavior.
//
// Response bodies are always drained before Close: an unread body forces the
// HTTP client to drop the connection, and a fresh connection means a fresh
// source port — which the per-IP keyed rate limiters would count as a new
// client, deflating the budget assertions below.

func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func newAuthSeamBundle(t *testing.T, cfg setup.Config) (*setup.Bundle, *httptest.Server) {
	t.Helper()

	bundle, err := setup.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	mux := http.NewServeMux()
	bundle.Mount(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return bundle, server
}

// TestNew_NilAuthHandlerConfig_ByteIdenticalBehavior pins the zero value: the
// default bundle still writes the flattened CookieName cookie on registration
// and applies no rate limiting (five rapid register attempts, zero 429s).
func TestNew_NilAuthHandlerConfig_ByteIdenticalBehavior(t *testing.T) {
	t.Parallel()

	_, server := newAuthSeamBundle(t, setup.Config{Title: "Nil Seam"})

	body := []byte(`{"email":"dev@example.com","display_name":"Dev"}`)

	statuses := make([]int, 0, 5)
	var cookieName string

	for range 5 {
		resp, err := server.Client().Post(server.URL+"/auth/register", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST /auth/register: %v", err)
		}
		drainAndClose(resp)

		statuses = append(statuses, resp.StatusCode)

		if cookieName == "" && len(resp.Cookies()) > 0 {
			cookieName = resp.Cookies()[0].Name
		}
	}

	for _, status := range statuses {
		if status == http.StatusTooManyRequests {
			t.Fatalf("nil AuthHandlerConfig must not rate limit, got statuses %v", statuses)
		}
	}

	if cookieName != "session" {
		t.Errorf("nil AuthHandlerConfig must write the flattened cookie name %q, got %q", "session", cookieName)
	}
}

// TestNew_AuthHandlerConfig_WebAuthnRateLimitReturns429 proves the seam
// threads usermgmt.HandlerConfig far enough to protect the passwordless
// ceremonies: with a two-request budget, the third rapid WebAuthn attempt is
// rejected by the limiter before any handler logic runs.
func TestNew_AuthHandlerConfig_WebAuthnRateLimitReturns429(t *testing.T) {
	t.Parallel()

	_, server := newAuthSeamBundle(t, setup.Config{
		Title: "Rate Limited",
		AuthHandlerConfig: &usermgmt.HandlerConfig{
			WebAuthnRateLimit: usermgmt.RateLimitConfig{
				Enabled:     true,
				MaxRequests: 2,
				Window:      time.Minute,
			},
		},
	})

	postBegin := func() int {
		t.Helper()

		resp, err := server.Client().Post(
			server.URL+"/auth/webauthn/login/begin",
			"application/json",
			strings.NewReader("{}"),
		)
		if err != nil {
			t.Fatalf("POST /auth/webauthn/login/begin: %v", err)
		}
		drainAndClose(resp)

		return resp.StatusCode
	}

	first, second, third := postBegin(), postBegin(), postBegin()
	if third != http.StatusTooManyRequests {
		t.Errorf("third WebAuthn attempt must be rate limited (429), got %d (%d, %d before)", third, first, second)
	}

	if first == http.StatusTooManyRequests || second == http.StatusTooManyRequests {
		t.Errorf("first two attempts are within budget, got %d, %d", first, second)
	}
}

// TestNew_AuthHandlerConfig_OAuth2RedirectsFlowThrough proves the redirect
// URL knobs reach the constructed AuthHandler. The error URL is the
// provider-free observable (a failed callback redirects to it); the success
// URL rides the same HandlerConfig struct through the same seam.
func TestNew_AuthHandlerConfig_OAuth2RedirectsFlowThrough(t *testing.T) {
	t.Parallel()

	_, server := newAuthSeamBundle(t, setup.Config{
		Title: "OAuth Redirects",
		AuthHandlerConfig: &usermgmt.HandlerConfig{
			OAuth2SuccessURL: "/after-login",
			OAuth2ErrorURL:   "/oauth-error",
		},
	})

	// The default client follows redirects; the assertion needs the raw 302.
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	resp, err := noRedirect.Get(server.URL + "/auth/oauth/github/callback?code=fake&state=bogus")
	if err != nil {
		t.Fatalf("GET /auth/oauth/github/callback: %v", err)
	}
	drainAndClose(resp)

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("failed OAuth2 callback must redirect (302) when OAuth2ErrorURL is set, got %d", resp.StatusCode)
	}

	if location := resp.Header.Get("Location"); !strings.HasPrefix(location, "/oauth-error?error=") {
		t.Errorf("redirect must target OAuth2ErrorURL with the error query param, got %q", location)
	}
}

// TestNew_AuthHandlerConfig_CookieNameConflictRejected pins the conflict
// rule: a mismatched CookieName inside AuthHandlerConfig is rejected at New
// with both names in the message.
func TestNew_AuthHandlerConfig_CookieNameConflictRejected(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:      "Cookie Conflict",
		CookieName: "session",
		AuthHandlerConfig: &usermgmt.HandlerConfig{
			CookieName: "other-cookie",
		},
	})
	if err == nil {
		t.Fatal("mismatched AuthHandlerConfig.CookieName must be rejected at New")
	}

	for _, want := range []string{`"other-cookie"`, `"session"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("rejection must name both cookie names (want %q): %v", want, err)
		}
	}
}

// TestNew_AuthHandlerConfig_EmptyCookieNameInheritsFlattened proves the
// inherit rule end to end: an AuthHandlerConfig without a cookie name still
// writes the flattened cookie on registration.
func TestNew_AuthHandlerConfig_EmptyCookieNameInheritsFlattened(t *testing.T) {
	t.Parallel()

	_, server := newAuthSeamBundle(t, setup.Config{
		Title:      "Inherits",
		CookieName: "my-session",
		AuthHandlerConfig: &usermgmt.HandlerConfig{
			OAuth2SuccessURL: "/after-login",
		},
	})

	resp, err := server.Client().Post(
		server.URL+"/auth/register",
		"application/json",
		strings.NewReader(`{"email":"dev@example.com","display_name":"Dev"}`),
	)
	if err != nil {
		t.Fatalf("POST /auth/register: %v", err)
	}
	drainAndClose(resp)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("registration should succeed, got %d", resp.StatusCode)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "my-session" {
			return
		}
	}

	t.Errorf("auth handler must write the inherited cookie %q, got cookies %v", "my-session", resp.Cookies())
}

// TestNew_AuthHandlerConfig_ComposesWithServiceConfig proves the seam is
// orthogonal to the service-source precedence: it does not describe service
// construction, so it must NOT be rejected as a conflict alongside
// ServiceConfig.
func TestNew_AuthHandlerConfig_ComposesWithServiceConfig(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:         "Composed",
		ServiceConfig: &usermgmt.ServiceConfig{},
		AuthHandlerConfig: &usermgmt.HandlerConfig{
			WebAuthnRateLimit: usermgmt.RateLimitConfig{
				Enabled:     true,
				MaxRequests: 2,
				Window:      time.Minute,
			},
		},
	})
	if err != nil {
		t.Fatalf("AuthHandlerConfig must compose with ServiceConfig: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Auth == nil {
		t.Fatal("bundle must still construct the auth handler")
	}
}
