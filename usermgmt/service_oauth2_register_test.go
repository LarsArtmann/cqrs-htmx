package usermgmt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// perProviderOAuth2Stub is an OAuth2Provider stub that derives subject and
// email from the provider name, so distinct provider names simulate distinct
// identities (and therefore distinct users) in tests.
type perProviderOAuth2Stub struct{}

func (perProviderOAuth2Stub) BeginLogin(_ context.Context, provider, state string) (string, string, error) {
	return "https://provider.example.com/auth?state=" + state + "&provider=" + provider, "pkce", nil
}

func (perProviderOAuth2Stub) FinishLogin(_ context.Context, provider, _, _ string) ([]byte, error) {
	return []byte(fmt.Sprintf(
		`{"subject":"sub-%s","email":"%s@oauth.test","email_verified":true,"display_name":"%s"}`,
		provider, provider, provider,
	)), nil
}

// runTestOAuth2Login runs the BeginLogin/FinishLogin round-trip for the given
// provider. It is safe to call from goroutines (no *testing.T).
func runTestOAuth2Login(svc *Service, provider string) (*FinishOAuthLoginResponse, error) {
	resp, err := svc.BeginOAuthLogin(context.Background(), provider)
	if err != nil {
		return nil, fmt.Errorf("BeginOAuthLogin(%s): %w", provider, err)
	}
	state, ok := stateFromRedirectURL(resp.RedirectURL)
	if !ok {
		return nil, fmt.Errorf("no state in redirect URL %q", resp.RedirectURL)
	}
	return svc.FinishOAuthLogin(context.Background(), provider, "code", state)
}

// stateFromRedirectURL extracts the state query parameter from a stub redirect
// URL. Returns false when absent.
func stateFromRedirectURL(rawURL string) (string, bool) {
	idx := strings.LastIndex(rawURL, "state=")
	if idx < 0 {
		return "", false
	}
	state := rawURL[idx+6:]
	if amp := strings.Index(state, "&"); amp >= 0 {
		state = state[:amp]
	}
	return state, true
}

func newOAuth2RegisterTestService(t *testing.T, maxUsers int) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{OAuth2: perProviderOAuth2Stub{}, MaxUsers: maxUsers})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	return svc
}

func TestFinishOAuthLogin_MaxUsersReached_RejectsAutoProvisioning(t *testing.T) {
	t.Parallel()
	svc := newOAuth2RegisterTestService(t, 1)
	registerTestUser(t, svc, "u1", "first@example.com")

	_, err := runTestOAuth2Login(svc, "github")
	assertErrorIs(t, err, ErrRegistrationClosed, "first-login auto-provisioning must be rejected at MaxUsers")
	if got := svc.readModel.Count(); got != 1 {
		t.Errorf("read model count = %d, want 1 (no user may be created by the rejected login)", got)
	}
}

func TestFinishOAuthLogin_MaxUsersReached_ExistingEmailStillLogsIn(t *testing.T) {
	t.Parallel()
	svc := newOAuth2RegisterTestService(t, 1)
	registerTestUser(t, svc, "u1", "github@oauth.test")

	resp, err := runTestOAuth2Login(svc, "github")
	if err != nil {
		t.Fatalf("FinishOAuthLogin for existing email must not be gated: %v", err)
	}
	if resp.User == nil || resp.User.Email != "github@oauth.test" {
		t.Fatalf("unexpected user in response: %+v", resp.User)
	}
	if got := svc.readModel.Count(); got != 1 {
		t.Errorf("read model count = %d, want 1 (email match must link, not create)", got)
	}
}

func TestFinishOAuthLogin_MaxUsersReached_ExistingExternalAccountStillLogsIn(t *testing.T) {
	t.Parallel()
	svc := newOAuth2RegisterTestService(t, 1)

	if _, err := runTestOAuth2Login(svc, "github"); err != nil {
		t.Fatalf("first login auto-provisions the sole user: %v", err)
	}
	if _, err := runTestOAuth2Login(svc, "github"); err != nil {
		t.Fatalf("second login must match by external account, not be gated: %v", err)
	}
	if got := svc.readModel.Count(); got != 1 {
		t.Errorf("read model count = %d, want 1 (repeat login must not create users)", got)
	}
}

func TestFinishOAuthLogin_MaxUsersZero_UnlimitedAutoProvisioning(t *testing.T) {
	t.Parallel()
	svc := newOAuth2RegisterTestService(t, 0)

	if _, err := runTestOAuth2Login(svc, "github"); err != nil {
		t.Fatalf("first login: %v", err)
	}
	if _, err := runTestOAuth2Login(svc, "google"); err != nil {
		t.Fatalf("second login: %v", err)
	}
	if got := svc.readModel.Count(); got != 2 {
		t.Errorf("read model count = %d, want 2", got)
	}
}

func TestFinishOAuthLogin_MaxUsersTwo_AllowsThird(t *testing.T) {
	t.Parallel()
	svc := newOAuth2RegisterTestService(t, 2)

	if _, err := runTestOAuth2Login(svc, "github"); err != nil {
		t.Fatalf("first login: %v", err)
	}
	if _, err := runTestOAuth2Login(svc, "google"); err != nil {
		t.Fatalf("second login: %v", err)
	}
	_, err := runTestOAuth2Login(svc, "gitlab")
	assertErrorIs(t, err, ErrRegistrationClosed, "third auto-provisioning must be rejected at MaxUsers=2")
}

// TestRegister_MixedConcurrentRegistrations_RespectMaxUsers hammers Register
// and OAuth2 first-login auto-provisioning concurrently. Regardless of
// interleaving, exactly MaxUsers users may be created — the shared
// registration mutex serializes the count-check-then-dispatch window.
func TestRegister_MixedConcurrentRegistrations_RespectMaxUsers(t *testing.T) {
	t.Parallel()
	const maxUsers = 2
	svc := newOAuth2RegisterTestService(t, maxUsers)
	ctx := context.Background()

	const workers = 16
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	created, rejected := 0, 0

	for i := range workers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var err error
			if i%2 == 0 {
				_, err = svc.Register(ctx, RegisterRequest{
					Email: fmt.Sprintf("racer%d@example.com", i),
				})
			} else {
				_, err = runTestOAuth2Login(svc, fmt.Sprintf("provider%d", i))
			}
			resultsMu.Lock()
			defer resultsMu.Unlock()
			switch {
			case err == nil:
				created++
			case errors.Is(err, ErrRegistrationClosed):
				rejected++
			default:
				t.Errorf("worker %d: unexpected error: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	if created != maxUsers {
		t.Errorf("created = %d, want %d", created, maxUsers)
	}
	if rejected != workers-maxUsers {
		t.Errorf("rejected = %d, want %d", rejected, workers-maxUsers)
	}
	if got := svc.readModel.Count(); got != maxUsers {
		t.Errorf("read model count = %d, want %d", got, maxUsers)
	}
}
