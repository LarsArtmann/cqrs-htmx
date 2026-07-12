package usermgmt

import (
	"context"
	"errors"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// failingOAuth2Provider is a configurable stub that fails at specific points
// to test that error context (provider name) is properly attached.
type failingOAuth2Provider struct {
	beginErr     error
	finishErr    error
	finishResult []byte
}

func (p *failingOAuth2Provider) BeginLogin(_ context.Context, _, state string) (string, string, error) {
	if p.beginErr != nil {
		return "", "", p.beginErr
	}
	return "https://example.com/auth?state=" + state, "verifier", nil
}

func (p *failingOAuth2Provider) FinishLogin(_ context.Context, _, _, _ string) ([]byte, error) {
	if p.finishErr != nil {
		return nil, p.finishErr
	}
	if p.finishResult != nil {
		return p.finishResult, nil
	}
	return []byte(`{"subject":"sub","email":"test@example.com","email_verified":true}`), nil
}

func newOAuth2ServiceForErrorTest(t *testing.T, provider OAuth2Provider) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{OAuth2: provider})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	return svc
}

// assertProviderContext extracts the errorfamily.Error from err and verifies
// that the "provider" context key equals the expected value.
func assertProviderContext(t *testing.T, err error, wantProvider string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ef *errorfamily.Error
	if !errors.As(err, &ef) {
		t.Fatalf("error is not *errorfamily.Error: %T — %v", err, err)
	}
	got := ef.ContextValue("provider")
	if got != wantProvider {
		t.Errorf("error context provider = %q, want %q (error: %v)", got, wantProvider, err)
	}
}

func TestBeginOAuthLogin_BeginLoginError_HasProviderContext(t *testing.T) {
	t.Parallel()
	provider := &failingOAuth2Provider{
		beginErr: errors.New("connection refused"),
	}
	svc := newOAuth2ServiceForErrorTest(t, provider)

	_, err := svc.BeginOAuthLogin(context.Background(), "github")
	assertProviderContext(t, err, "github")
}

func TestFinishOAuthLogin_StateConsumeError_HasProviderContext(t *testing.T) {
	t.Parallel()
	svc := newOAuth2ServiceForErrorTest(t, &failingOAuth2Provider{})

	_, err := svc.FinishOAuthLogin(context.Background(), "github", "code", "never-stored")
	assertProviderContext(t, err, "github")
}

func TestFinishOAuthLogin_ProviderMismatch_HasProviderContext(t *testing.T) {
	t.Parallel()
	svc := newOAuth2ServiceForErrorTest(t, &failingOAuth2Provider{})

	// Begin with "google" to store the state
	resp, err := svc.BeginOAuthLogin(context.Background(), "google")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}

	// Extract the state from the redirect URL
	state := resp.RedirectURL
	if idx := strings.LastIndex(state, "state="); idx >= 0 {
		state = state[idx+6:]
	}

	// Finish with "github" — should fail with provider mismatch
	_, err = svc.FinishOAuthLogin(context.Background(), "github", "code", state)
	assertProviderContext(t, err, "github")
}

func TestFinishOAuthLogin_TokenExchangeError_HasProviderContext(t *testing.T) {
	t.Parallel()
	provider := &failingOAuth2Provider{
		finishErr: errors.New("token endpoint unreachable"),
	}
	svc := newOAuth2ServiceForErrorTest(t, provider)

	resp, err := svc.BeginOAuthLogin(context.Background(), "google")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}
	state := extractStateFromURL(t, resp.RedirectURL)

	_, err = svc.FinishOAuthLogin(context.Background(), "google", "code", state)
	assertProviderContext(t, err, "google")
}

func TestFinishOAuthLogin_UserInfoUnmarshalError_HasProviderContext(t *testing.T) {
	t.Parallel()
	provider := &failingOAuth2Provider{
		finishResult: []byte(`{invalid json`),
	}
	svc := newOAuth2ServiceForErrorTest(t, provider)

	resp, err := svc.BeginOAuthLogin(context.Background(), "google")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}
	state := extractStateFromURL(t, resp.RedirectURL)

	_, err = svc.FinishOAuthLogin(context.Background(), "google", "code", state)
	assertProviderContext(t, err, "google")
}

func TestFinishOAuthLogin_NoEmailError_HasProviderContext(t *testing.T) {
	t.Parallel()
	provider := &failingOAuth2Provider{
		finishResult: []byte(`{"subject":"sub","email":"","email_verified":false}`),
	}
	svc := newOAuth2ServiceForErrorTest(t, provider)

	resp, err := svc.BeginOAuthLogin(context.Background(), "google")
	if err != nil {
		t.Fatalf("BeginOAuthLogin: %v", err)
	}
	state := extractStateFromURL(t, resp.RedirectURL)

	_, err = svc.FinishOAuthLogin(context.Background(), "google", "code", state)
	assertProviderContext(t, err, "google")
}

// extractStateFromURL pulls the state query param from a redirect URL.
func extractStateFromURL(t *testing.T, rawURL string) string {
	t.Helper()
	idx := strings.LastIndex(rawURL, "state=")
	if idx < 0 {
		t.Fatal("no state= in redirect URL")
	}
	state := rawURL[idx+6:]
	if amp := strings.Index(state, "&"); amp >= 0 {
		state = state[:amp]
	}
	return state
}
